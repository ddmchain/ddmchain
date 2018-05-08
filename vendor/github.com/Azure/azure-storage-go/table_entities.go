package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
)

const (
	partitionKeyNode                    = "PartitionKey"
	rowKeyNode                          = "RowKey"
	tag                                 = "table"
	tagIgnore                           = "-"
	continuationTokenPartitionKeyHeader = "X-Ms-Continuation-Nextpartitionkey"
	continuationTokenRowHeader          = "X-Ms-Continuation-Nextrowkey"
	maxTopParameter                     = 1000
)

type queryTablesResponse struct {
	TableName []struct {
		TableName string `json:"TableName"`
	} `json:"value"`
}

const (
	tableOperationTypeInsert          = iota
	tableOperationTypeUpdate          = iota
	tableOperationTypeMerge           = iota
	tableOperationTypeInsertOrReplace = iota
	tableOperationTypeInsertOrMerge   = iota
)

type tableOperation int

type TableEntity interface {
	PartitionKey() string
	RowKey() string
	SetPartitionKey(string) error
	SetRowKey(string) error
}

type ContinuationToken struct {
	NextPartitionKey string
	NextRowKey       string
}

type getTableEntriesResponse struct {
	Elements []map[string]interface{} `json:"value"`
}

func (c *TableServiceClient) QueryTableEntities(tableName AzureTable, previousContToken *ContinuationToken, retType reflect.Type, top int, query string) ([]TableEntity, *ContinuationToken, error) {
	if top > maxTopParameter {
		return nil, nil, fmt.Errorf("top accepts at maximum %d elements. Requested %d instead", maxTopParameter, top)
	}

	uri := c.client.getEndpoint(tableServiceName, pathForTable(tableName), url.Values{})
	uri += fmt.Sprintf("?$top=%d", top)
	if query != "" {
		uri += fmt.Sprintf("&$filter=%s", url.QueryEscape(query))
	}

	if previousContToken != nil {
		uri += fmt.Sprintf("&NextPartitionKey=%s&NextRowKey=%s", previousContToken.NextPartitionKey, previousContToken.NextRowKey)
	}

	headers := c.getStandardHeaders()

	headers["Content-Length"] = "0"

	resp, err := c.client.execInternalJSON(http.MethodGet, uri, headers, nil, c.auth)

	if err != nil {
		return nil, nil, err
	}

	contToken := extractContinuationTokenFromHeaders(resp.headers)

	defer resp.body.Close()

	if err = checkRespCode(resp.statusCode, []int{http.StatusOK}); err != nil {
		return nil, contToken, err
	}

	retEntries, err := deserializeEntity(retType, resp.body)
	if err != nil {
		return nil, contToken, err
	}

	return retEntries, contToken, nil
}

func (c *TableServiceClient) InsertEntity(table AzureTable, entity TableEntity) error {
	sc, err := c.execTable(table, entity, false, http.MethodPost)
	if err != nil {
		return err
	}

	return checkRespCode(sc, []int{http.StatusCreated})
}

func (c *TableServiceClient) execTable(table AzureTable, entity TableEntity, specifyKeysInURL bool, method string) (int, error) {
	uri := c.client.getEndpoint(tableServiceName, pathForTable(table), url.Values{})
	if specifyKeysInURL {
		uri += fmt.Sprintf("(PartitionKey='%s',RowKey='%s')", url.QueryEscape(entity.PartitionKey()), url.QueryEscape(entity.RowKey()))
	}

	headers := c.getStandardHeaders()

	var buf bytes.Buffer

	if err := injectPartitionAndRowKeys(entity, &buf); err != nil {
		return 0, err
	}

	headers["Content-Length"] = fmt.Sprintf("%d", buf.Len())

	resp, err := c.client.execInternalJSON(method, uri, headers, &buf, c.auth)

	if err != nil {
		return 0, err
	}

	defer resp.body.Close()

	return resp.statusCode, nil
}

func (c *TableServiceClient) UpdateEntity(table AzureTable, entity TableEntity) error {
	sc, err := c.execTable(table, entity, true, http.MethodPut)
	if err != nil {
		return err
	}

	return checkRespCode(sc, []int{http.StatusNoContent})
}

func (c *TableServiceClient) MergeEntity(table AzureTable, entity TableEntity) error {
	sc, err := c.execTable(table, entity, true, "MERGE")
	if err != nil {
		return err
	}

	return checkRespCode(sc, []int{http.StatusNoContent})
}

func (c *TableServiceClient) DeleteEntityWithoutCheck(table AzureTable, entity TableEntity) error {
	return c.DeleteEntity(table, entity, "*")
}

func (c *TableServiceClient) DeleteEntity(table AzureTable, entity TableEntity, ifMatch string) error {
	uri := c.client.getEndpoint(tableServiceName, pathForTable(table), url.Values{})
	uri += fmt.Sprintf("(PartitionKey='%s',RowKey='%s')", url.QueryEscape(entity.PartitionKey()), url.QueryEscape(entity.RowKey()))

	headers := c.getStandardHeaders()

	headers["Content-Length"] = "0"
	headers["If-Match"] = ifMatch

	resp, err := c.client.execInternalJSON(http.MethodDelete, uri, headers, nil, c.auth)

	if err != nil {
		return err
	}
	defer resp.body.Close()

	if err := checkRespCode(resp.statusCode, []int{http.StatusNoContent}); err != nil {
		return err
	}

	return nil
}

func (c *TableServiceClient) InsertOrReplaceEntity(table AzureTable, entity TableEntity) error {
	sc, err := c.execTable(table, entity, true, http.MethodPut)
	if err != nil {
		return err
	}

	return checkRespCode(sc, []int{http.StatusNoContent})
}

func (c *TableServiceClient) InsertOrMergeEntity(table AzureTable, entity TableEntity) error {
	sc, err := c.execTable(table, entity, true, "MERGE")
	if err != nil {
		return err
	}

	return checkRespCode(sc, []int{http.StatusNoContent})
}

func injectPartitionAndRowKeys(entity TableEntity, buf *bytes.Buffer) error {
	if err := json.NewEncoder(buf).Encode(entity); err != nil {
		return err
	}

	dec := make(map[string]interface{})
	if err := json.NewDecoder(buf).Decode(&dec); err != nil {
		return err
	}

	dec[partitionKeyNode] = entity.PartitionKey()
	dec[rowKeyNode] = entity.RowKey()

	numFields := reflect.ValueOf(entity).Elem().NumField()
	for i := 0; i < numFields; i++ {
		f := reflect.ValueOf(entity).Elem().Type().Field(i)

		if f.Tag.Get(tag) == tagIgnore {

			jsonName := f.Name
			if f.Tag.Get("json") != "" {
				jsonName = f.Tag.Get("json")
			}
			delete(dec, jsonName)
		}
	}

	buf.Reset()

	if err := json.NewEncoder(buf).Encode(&dec); err != nil {
		return err
	}

	return nil
}

func deserializeEntity(retType reflect.Type, reader io.Reader) ([]TableEntity, error) {
	buf := new(bytes.Buffer)

	var ret getTableEntriesResponse
	if err := json.NewDecoder(reader).Decode(&ret); err != nil {
		return nil, err
	}

	tEntries := make([]TableEntity, len(ret.Elements))

	for i, entry := range ret.Elements {

		buf.Reset()
		if err := json.NewEncoder(buf).Encode(entry); err != nil {
			return nil, err
		}

		dec := make(map[string]interface{})
		if err := json.NewDecoder(buf).Decode(&dec); err != nil {
			return nil, err
		}

		var pKey, rKey string

		for key, val := range dec {
			switch key {
			case partitionKeyNode:
				pKey = val.(string)
			case rowKeyNode:
				rKey = val.(string)
			}
		}

		delete(dec, partitionKeyNode)
		delete(dec, rowKeyNode)

		buf.Reset()
		if err := json.NewEncoder(buf).Encode(dec); err != nil {
			return nil, err
		}

		tEntries[i] = reflect.New(retType.Elem()).Interface().(TableEntity)

		if err := json.NewDecoder(buf).Decode(&tEntries[i]); err != nil {
			return nil, err
		}

		if err := tEntries[i].SetPartitionKey(pKey); err != nil {
			return nil, err
		}
		if err := tEntries[i].SetRowKey(rKey); err != nil {
			return nil, err
		}
	}

	return tEntries, nil
}

func extractContinuationTokenFromHeaders(h http.Header) *ContinuationToken {
	ct := ContinuationToken{h.Get(continuationTokenPartitionKeyHeader), h.Get(continuationTokenRowHeader)}

	if ct.NextPartitionKey != "" && ct.NextRowKey != "" {
		return &ct
	}
	return nil
}
