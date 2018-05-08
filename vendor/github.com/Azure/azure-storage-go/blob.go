package storage

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Blob struct {
	Name       string         `xml:"Name"`
	Properties BlobProperties `xml:"Properties"`
	Metadata   BlobMetadata   `xml:"Metadata"`
}

type BlobMetadata map[string]string

type blobMetadataEntries struct {
	Entries []blobMetadataEntry `xml:",any"`
}
type blobMetadataEntry struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

func (bm *BlobMetadata) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var entries blobMetadataEntries
	if err := d.DecodeElement(&entries, &start); err != nil {
		return err
	}
	for _, entry := range entries.Entries {
		if *bm == nil {
			*bm = make(BlobMetadata)
		}
		(*bm)[strings.ToLower(entry.XMLName.Local)] = entry.Value
	}
	return nil
}

func (bm BlobMetadata) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	entries := make([]blobMetadataEntry, 0, len(bm))
	for k, v := range bm {
		entries = append(entries, blobMetadataEntry{
			XMLName: xml.Name{Local: http.CanonicalHeaderKey(k)},
			Value:   v,
		})
	}
	return enc.EncodeElement(blobMetadataEntries{
		Entries: entries,
	}, start)
}

type BlobProperties struct {
	LastModified          string   `xml:"Last-Modified"`
	Etag                  string   `xml:"Etag"`
	ContentMD5            string   `xml:"Content-MD5"`
	ContentLength         int64    `xml:"Content-Length"`
	ContentType           string   `xml:"Content-Type"`
	ContentEncoding       string   `xml:"Content-Encoding"`
	CacheControl          string   `xml:"Cache-Control"`
	ContentLanguage       string   `xml:"Cache-Language"`
	BlobType              BlobType `xml:"x-ms-blob-blob-type"`
	SequenceNumber        int64    `xml:"x-ms-blob-sequence-number"`
	CopyID                string   `xml:"CopyId"`
	CopyStatus            string   `xml:"CopyStatus"`
	CopySource            string   `xml:"CopySource"`
	CopyProgress          string   `xml:"CopyProgress"`
	CopyCompletionTime    string   `xml:"CopyCompletionTime"`
	CopyStatusDescription string   `xml:"CopyStatusDescription"`
	LeaseStatus           string   `xml:"LeaseStatus"`
	LeaseState            string   `xml:"LeaseState"`
}

type BlobHeaders struct {
	ContentMD5      string `header:"x-ms-blob-content-md5"`
	ContentLanguage string `header:"x-ms-blob-content-language"`
	ContentEncoding string `header:"x-ms-blob-content-encoding"`
	ContentType     string `header:"x-ms-blob-content-type"`
	CacheControl    string `header:"x-ms-blob-cache-control"`
}

type BlobType string

const (
	BlobTypeBlock  BlobType = "BlockBlob"
	BlobTypePage   BlobType = "PageBlob"
	BlobTypeAppend BlobType = "AppendBlob"
)

type PageWriteType string

const (
	PageWriteTypeUpdate PageWriteType = "update"
	PageWriteTypeClear  PageWriteType = "clear"
)

const (
	blobCopyStatusPending = "pending"
	blobCopyStatusSuccess = "success"
	blobCopyStatusAborted = "aborted"
	blobCopyStatusFailed  = "failed"
)

const (
	leaseHeaderPrefix = "x-ms-lease-"
	headerLeaseID     = "x-ms-lease-id"
	leaseAction       = "x-ms-lease-action"
	leaseBreakPeriod  = "x-ms-lease-break-period"
	leaseDuration     = "x-ms-lease-duration"
	leaseProposedID   = "x-ms-proposed-lease-id"
	leaseTime         = "x-ms-lease-time"

	acquireLease = "acquire"
	renewLease   = "renew"
	changeLease  = "change"
	releaseLease = "release"
	breakLease   = "break"
)

type BlockListType string

const (
	BlockListTypeAll         BlockListType = "all"
	BlockListTypeCommitted   BlockListType = "committed"
	BlockListTypeUncommitted BlockListType = "uncommitted"
)

const (
	MaxBlobBlockSize = 100 * 1024 * 1024
	MaxBlobPageSize  = 4 * 1024 * 1024
)

type BlockStatus string

const (
	BlockStatusUncommitted BlockStatus = "Uncommitted"
	BlockStatusCommitted   BlockStatus = "Committed"
	BlockStatusLatest      BlockStatus = "Latest"
)

type Block struct {
	ID     string
	Status BlockStatus
}

type BlockListResponse struct {
	XMLName           xml.Name        `xml:"BlockList"`
	CommittedBlocks   []BlockResponse `xml:"CommittedBlocks>Block"`
	UncommittedBlocks []BlockResponse `xml:"UncommittedBlocks>Block"`
}

type BlockResponse struct {
	Name string `xml:"Name"`
	Size int64  `xml:"Size"`
}

type GetPageRangesResponse struct {
	XMLName  xml.Name    `xml:"PageList"`
	PageList []PageRange `xml:"PageRange"`
}

type PageRange struct {
	Start int64 `xml:"Start"`
	End   int64 `xml:"End"`
}

var (
	errBlobCopyAborted    = errors.New("storage: blob copy is aborted")
	errBlobCopyIDMismatch = errors.New("storage: blob copy id is a mismatch")
)

func (b BlobStorageClient) BlobExists(container, name string) (bool, error) {
	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), url.Values{})
	headers := b.client.getStandardHeaders()
	resp, err := b.client.exec(http.MethodHead, uri, headers, nil, b.auth)
	if resp != nil {
		defer readAndCloseBody(resp.body)
		if resp.statusCode == http.StatusOK || resp.statusCode == http.StatusNotFound {
			return resp.statusCode == http.StatusOK, nil
		}
	}
	return false, err
}

func (b BlobStorageClient) GetBlobURL(container, name string) string {
	if container == "" {
		container = "$root"
	}
	return b.client.getEndpoint(blobServiceName, pathForResource(container, name), url.Values{})
}

func (b BlobStorageClient) GetBlob(container, name string) (io.ReadCloser, error) {
	resp, err := b.getBlobRange(container, name, "", nil)
	if err != nil {
		return nil, err
	}

	if err := checkRespCode(resp.statusCode, []int{http.StatusOK}); err != nil {
		return nil, err
	}
	return resp.body, nil
}

func (b BlobStorageClient) GetBlobRange(container, name, bytesRange string, extraHeaders map[string]string) (io.ReadCloser, error) {
	resp, err := b.getBlobRange(container, name, bytesRange, extraHeaders)
	if err != nil {
		return nil, err
	}

	if err := checkRespCode(resp.statusCode, []int{http.StatusPartialContent}); err != nil {
		return nil, err
	}
	return resp.body, nil
}

func (b BlobStorageClient) getBlobRange(container, name, bytesRange string, extraHeaders map[string]string) (*storageResponse, error) {
	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), url.Values{})

	extraHeaders = b.client.protectUserAgent(extraHeaders)
	headers := b.client.getStandardHeaders()
	if bytesRange != "" {
		headers["Range"] = fmt.Sprintf("bytes=%s", bytesRange)
	}

	for k, v := range extraHeaders {
		headers[k] = v
	}

	resp, err := b.client.exec(http.MethodGet, uri, headers, nil, b.auth)
	if err != nil {
		return nil, err
	}
	return resp, err
}

func (b BlobStorageClient) leaseCommonPut(container string, name string, headers map[string]string, expectedStatus int) (http.Header, error) {
	params := url.Values{"comp": {"lease"}}
	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), params)

	resp, err := b.client.exec(http.MethodPut, uri, headers, nil, b.auth)
	if err != nil {
		return nil, err
	}
	defer readAndCloseBody(resp.body)

	if err := checkRespCode(resp.statusCode, []int{expectedStatus}); err != nil {
		return nil, err
	}

	return resp.headers, nil
}

func (b BlobStorageClient) SnapshotBlob(container string, name string, timeout int, extraHeaders map[string]string) (snapshotTimestamp *time.Time, err error) {
	extraHeaders = b.client.protectUserAgent(extraHeaders)
	headers := b.client.getStandardHeaders()
	params := url.Values{"comp": {"snapshot"}}

	if timeout > 0 {
		params.Add("timeout", strconv.Itoa(timeout))
	}

	for k, v := range extraHeaders {
		headers[k] = v
	}

	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), params)
	resp, err := b.client.exec(http.MethodPut, uri, headers, nil, b.auth)
	if err != nil || resp == nil {
		return nil, err
	}

	defer readAndCloseBody(resp.body)

	if err := checkRespCode(resp.statusCode, []int{http.StatusCreated}); err != nil {
		return nil, err
	}

	snapshotResponse := resp.headers.Get(http.CanonicalHeaderKey("x-ms-snapshot"))
	if snapshotResponse != "" {
		snapshotTimestamp, err := time.Parse(time.RFC3339, snapshotResponse)
		if err != nil {
			return nil, err
		}

		return &snapshotTimestamp, nil
	}

	return nil, errors.New("Snapshot not created")
}

func (b BlobStorageClient) AcquireLease(container string, name string, leaseTimeInSeconds int, proposedLeaseID string) (returnedLeaseID string, err error) {
	headers := b.client.getStandardHeaders()
	headers[leaseAction] = acquireLease

	if leaseTimeInSeconds == -1 {

	} else if leaseTimeInSeconds > 60 || b.client.apiVersion < "2012-02-12" {
		leaseTimeInSeconds = 60
	} else if leaseTimeInSeconds < 15 {
		leaseTimeInSeconds = 15
	}

	headers[leaseDuration] = strconv.Itoa(leaseTimeInSeconds)

	if proposedLeaseID != "" {
		headers[leaseProposedID] = proposedLeaseID
	}

	respHeaders, err := b.leaseCommonPut(container, name, headers, http.StatusCreated)
	if err != nil {
		return "", err
	}

	returnedLeaseID = respHeaders.Get(http.CanonicalHeaderKey(headerLeaseID))

	if returnedLeaseID != "" {
		return returnedLeaseID, nil
	}

	return "", errors.New("LeaseID not returned")
}

func (b BlobStorageClient) BreakLease(container string, name string) (breakTimeout int, err error) {
	headers := b.client.getStandardHeaders()
	headers[leaseAction] = breakLease
	return b.breakLeaseCommon(container, name, headers)
}

func (b BlobStorageClient) BreakLeaseWithBreakPeriod(container string, name string, breakPeriodInSeconds int) (breakTimeout int, err error) {
	headers := b.client.getStandardHeaders()
	headers[leaseAction] = breakLease
	headers[leaseBreakPeriod] = strconv.Itoa(breakPeriodInSeconds)
	return b.breakLeaseCommon(container, name, headers)
}

func (b BlobStorageClient) breakLeaseCommon(container string, name string, headers map[string]string) (breakTimeout int, err error) {

	respHeaders, err := b.leaseCommonPut(container, name, headers, http.StatusAccepted)
	if err != nil {
		return 0, err
	}

	breakTimeoutStr := respHeaders.Get(http.CanonicalHeaderKey(leaseTime))
	if breakTimeoutStr != "" {
		breakTimeout, err = strconv.Atoi(breakTimeoutStr)
		if err != nil {
			return 0, err
		}
	}

	return breakTimeout, nil
}

func (b BlobStorageClient) ChangeLease(container string, name string, currentLeaseID string, proposedLeaseID string) (newLeaseID string, err error) {
	headers := b.client.getStandardHeaders()
	headers[leaseAction] = changeLease
	headers[headerLeaseID] = currentLeaseID
	headers[leaseProposedID] = proposedLeaseID

	respHeaders, err := b.leaseCommonPut(container, name, headers, http.StatusOK)
	if err != nil {
		return "", err
	}

	newLeaseID = respHeaders.Get(http.CanonicalHeaderKey(headerLeaseID))
	if newLeaseID != "" {
		return newLeaseID, nil
	}

	return "", errors.New("LeaseID not returned")
}

func (b BlobStorageClient) ReleaseLease(container string, name string, currentLeaseID string) error {
	headers := b.client.getStandardHeaders()
	headers[leaseAction] = releaseLease
	headers[headerLeaseID] = currentLeaseID

	_, err := b.leaseCommonPut(container, name, headers, http.StatusOK)
	if err != nil {
		return err
	}

	return nil
}

func (b BlobStorageClient) RenewLease(container string, name string, currentLeaseID string) error {
	headers := b.client.getStandardHeaders()
	headers[leaseAction] = renewLease
	headers[headerLeaseID] = currentLeaseID

	_, err := b.leaseCommonPut(container, name, headers, http.StatusOK)
	if err != nil {
		return err
	}

	return nil
}

func (b BlobStorageClient) GetBlobProperties(container, name string) (*BlobProperties, error) {
	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), url.Values{})

	headers := b.client.getStandardHeaders()
	resp, err := b.client.exec(http.MethodHead, uri, headers, nil, b.auth)
	if err != nil {
		return nil, err
	}
	defer readAndCloseBody(resp.body)

	if err = checkRespCode(resp.statusCode, []int{http.StatusOK}); err != nil {
		return nil, err
	}

	var contentLength int64
	contentLengthStr := resp.headers.Get("Content-Length")
	if contentLengthStr != "" {
		contentLength, err = strconv.ParseInt(contentLengthStr, 0, 64)
		if err != nil {
			return nil, err
		}
	}

	var sequenceNum int64
	sequenceNumStr := resp.headers.Get("x-ms-blob-sequence-number")
	if sequenceNumStr != "" {
		sequenceNum, err = strconv.ParseInt(sequenceNumStr, 0, 64)
		if err != nil {
			return nil, err
		}
	}

	return &BlobProperties{
		LastModified:          resp.headers.Get("Last-Modified"),
		Etag:                  resp.headers.Get("Etag"),
		ContentMD5:            resp.headers.Get("Content-MD5"),
		ContentLength:         contentLength,
		ContentEncoding:       resp.headers.Get("Content-Encoding"),
		ContentType:           resp.headers.Get("Content-Type"),
		CacheControl:          resp.headers.Get("Cache-Control"),
		ContentLanguage:       resp.headers.Get("Content-Language"),
		SequenceNumber:        sequenceNum,
		CopyCompletionTime:    resp.headers.Get("x-ms-copy-completion-time"),
		CopyStatusDescription: resp.headers.Get("x-ms-copy-status-description"),
		CopyID:                resp.headers.Get("x-ms-copy-id"),
		CopyProgress:          resp.headers.Get("x-ms-copy-progress"),
		CopySource:            resp.headers.Get("x-ms-copy-source"),
		CopyStatus:            resp.headers.Get("x-ms-copy-status"),
		BlobType:              BlobType(resp.headers.Get("x-ms-blob-type")),
		LeaseStatus:           resp.headers.Get("x-ms-lease-status"),
		LeaseState:            resp.headers.Get("x-ms-lease-state"),
	}, nil
}

func (b BlobStorageClient) SetBlobProperties(container, name string, blobHeaders BlobHeaders) error {
	params := url.Values{"comp": {"properties"}}
	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), params)
	headers := b.client.getStandardHeaders()

	extraHeaders := headersFromStruct(blobHeaders)

	for k, v := range extraHeaders {
		headers[k] = v
	}

	resp, err := b.client.exec(http.MethodPut, uri, headers, nil, b.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)

	return checkRespCode(resp.statusCode, []int{http.StatusOK})
}

func (b BlobStorageClient) SetBlobMetadata(container, name string, metadata map[string]string, extraHeaders map[string]string) error {
	params := url.Values{"comp": {"metadata"}}
	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), params)
	metadata = b.client.protectUserAgent(metadata)
	extraHeaders = b.client.protectUserAgent(extraHeaders)
	headers := b.client.getStandardHeaders()
	for k, v := range metadata {
		headers[userDefinedMetadataHeaderPrefix+k] = v
	}

	for k, v := range extraHeaders {
		headers[k] = v
	}

	resp, err := b.client.exec(http.MethodPut, uri, headers, nil, b.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)

	return checkRespCode(resp.statusCode, []int{http.StatusOK})
}

func (b BlobStorageClient) GetBlobMetadata(container, name string) (map[string]string, error) {
	params := url.Values{"comp": {"metadata"}}
	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), params)
	headers := b.client.getStandardHeaders()

	resp, err := b.client.exec(http.MethodGet, uri, headers, nil, b.auth)
	if err != nil {
		return nil, err
	}
	defer readAndCloseBody(resp.body)

	if err := checkRespCode(resp.statusCode, []int{http.StatusOK}); err != nil {
		return nil, err
	}

	metadata := make(map[string]string)
	for k, v := range resp.headers {

		k = strings.ToLower(k)
		if len(v) == 0 || !strings.HasPrefix(k, strings.ToLower(userDefinedMetadataHeaderPrefix)) {
			continue
		}

		k = k[len(userDefinedMetadataHeaderPrefix):]
		metadata[k] = v[len(v)-1]
	}
	return metadata, nil
}

func (b BlobStorageClient) CreateBlockBlob(container, name string) error {
	return b.CreateBlockBlobFromReader(container, name, 0, nil, nil)
}

func (b BlobStorageClient) CreateBlockBlobFromReader(container, name string, size uint64, blob io.Reader, extraHeaders map[string]string) error {
	path := fmt.Sprintf("%s/%s", container, name)
	uri := b.client.getEndpoint(blobServiceName, path, url.Values{})
	extraHeaders = b.client.protectUserAgent(extraHeaders)
	headers := b.client.getStandardHeaders()
	headers["x-ms-blob-type"] = string(BlobTypeBlock)
	headers["Content-Length"] = fmt.Sprintf("%d", size)

	for k, v := range extraHeaders {
		headers[k] = v
	}

	resp, err := b.client.exec(http.MethodPut, uri, headers, blob, b.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

func (b BlobStorageClient) PutBlock(container, name, blockID string, chunk []byte) error {
	return b.PutBlockWithLength(container, name, blockID, uint64(len(chunk)), bytes.NewReader(chunk), nil)
}

func (b BlobStorageClient) PutBlockWithLength(container, name, blockID string, size uint64, blob io.Reader, extraHeaders map[string]string) error {
	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), url.Values{"comp": {"block"}, "blockid": {blockID}})
	extraHeaders = b.client.protectUserAgent(extraHeaders)
	headers := b.client.getStandardHeaders()
	headers["x-ms-blob-type"] = string(BlobTypeBlock)
	headers["Content-Length"] = fmt.Sprintf("%v", size)

	for k, v := range extraHeaders {
		headers[k] = v
	}

	resp, err := b.client.exec(http.MethodPut, uri, headers, blob, b.auth)
	if err != nil {
		return err
	}

	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

func (b BlobStorageClient) PutBlockList(container, name string, blocks []Block) error {
	blockListXML := prepareBlockListRequest(blocks)

	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), url.Values{"comp": {"blocklist"}})
	headers := b.client.getStandardHeaders()
	headers["Content-Length"] = fmt.Sprintf("%v", len(blockListXML))

	resp, err := b.client.exec(http.MethodPut, uri, headers, strings.NewReader(blockListXML), b.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

func (b BlobStorageClient) GetBlockList(container, name string, blockType BlockListType) (BlockListResponse, error) {
	params := url.Values{"comp": {"blocklist"}, "blocklisttype": {string(blockType)}}
	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), params)
	headers := b.client.getStandardHeaders()

	var out BlockListResponse
	resp, err := b.client.exec(http.MethodGet, uri, headers, nil, b.auth)
	if err != nil {
		return out, err
	}
	defer resp.body.Close()

	err = xmlUnmarshal(resp.body, &out)
	return out, err
}

func (b BlobStorageClient) PutPageBlob(container, name string, size int64, extraHeaders map[string]string) error {
	path := fmt.Sprintf("%s/%s", container, name)
	uri := b.client.getEndpoint(blobServiceName, path, url.Values{})
	extraHeaders = b.client.protectUserAgent(extraHeaders)
	headers := b.client.getStandardHeaders()
	headers["x-ms-blob-type"] = string(BlobTypePage)
	headers["x-ms-blob-content-length"] = fmt.Sprintf("%v", size)

	for k, v := range extraHeaders {
		headers[k] = v
	}

	resp, err := b.client.exec(http.MethodPut, uri, headers, nil, b.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)

	return checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

func (b BlobStorageClient) PutPage(container, name string, startByte, endByte int64, writeType PageWriteType, chunk []byte, extraHeaders map[string]string) error {
	path := fmt.Sprintf("%s/%s", container, name)
	uri := b.client.getEndpoint(blobServiceName, path, url.Values{"comp": {"page"}})
	extraHeaders = b.client.protectUserAgent(extraHeaders)
	headers := b.client.getStandardHeaders()
	headers["x-ms-blob-type"] = string(BlobTypePage)
	headers["x-ms-page-write"] = string(writeType)
	headers["x-ms-range"] = fmt.Sprintf("bytes=%v-%v", startByte, endByte)
	for k, v := range extraHeaders {
		headers[k] = v
	}
	var contentLength int64
	var data io.Reader
	if writeType == PageWriteTypeClear {
		contentLength = 0
		data = bytes.NewReader([]byte{})
	} else {
		contentLength = int64(len(chunk))
		data = bytes.NewReader(chunk)
	}
	headers["Content-Length"] = fmt.Sprintf("%v", contentLength)

	resp, err := b.client.exec(http.MethodPut, uri, headers, data, b.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)

	return checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

func (b BlobStorageClient) GetPageRanges(container, name string) (GetPageRangesResponse, error) {
	path := fmt.Sprintf("%s/%s", container, name)
	uri := b.client.getEndpoint(blobServiceName, path, url.Values{"comp": {"pagelist"}})
	headers := b.client.getStandardHeaders()

	var out GetPageRangesResponse
	resp, err := b.client.exec(http.MethodGet, uri, headers, nil, b.auth)
	if err != nil {
		return out, err
	}
	defer resp.body.Close()

	if err = checkRespCode(resp.statusCode, []int{http.StatusOK}); err != nil {
		return out, err
	}
	err = xmlUnmarshal(resp.body, &out)
	return out, err
}

func (b BlobStorageClient) PutAppendBlob(container, name string, extraHeaders map[string]string) error {
	path := fmt.Sprintf("%s/%s", container, name)
	uri := b.client.getEndpoint(blobServiceName, path, url.Values{})
	extraHeaders = b.client.protectUserAgent(extraHeaders)
	headers := b.client.getStandardHeaders()
	headers["x-ms-blob-type"] = string(BlobTypeAppend)

	for k, v := range extraHeaders {
		headers[k] = v
	}

	resp, err := b.client.exec(http.MethodPut, uri, headers, nil, b.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)

	return checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

func (b BlobStorageClient) AppendBlock(container, name string, chunk []byte, extraHeaders map[string]string) error {
	path := fmt.Sprintf("%s/%s", container, name)
	uri := b.client.getEndpoint(blobServiceName, path, url.Values{"comp": {"appendblock"}})
	extraHeaders = b.client.protectUserAgent(extraHeaders)
	headers := b.client.getStandardHeaders()
	headers["x-ms-blob-type"] = string(BlobTypeAppend)
	headers["Content-Length"] = fmt.Sprintf("%v", len(chunk))

	for k, v := range extraHeaders {
		headers[k] = v
	}

	resp, err := b.client.exec(http.MethodPut, uri, headers, bytes.NewReader(chunk), b.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)

	return checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

func (b BlobStorageClient) CopyBlob(container, name, sourceBlob string) error {
	copyID, err := b.StartBlobCopy(container, name, sourceBlob)
	if err != nil {
		return err
	}

	return b.WaitForBlobCopy(container, name, copyID)
}

func (b BlobStorageClient) StartBlobCopy(container, name, sourceBlob string) (string, error) {
	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), url.Values{})

	headers := b.client.getStandardHeaders()
	headers["x-ms-copy-source"] = sourceBlob

	resp, err := b.client.exec(http.MethodPut, uri, headers, nil, b.auth)
	if err != nil {
		return "", err
	}
	defer readAndCloseBody(resp.body)

	if err := checkRespCode(resp.statusCode, []int{http.StatusAccepted, http.StatusCreated}); err != nil {
		return "", err
	}

	copyID := resp.headers.Get("x-ms-copy-id")
	if copyID == "" {
		return "", errors.New("Got empty copy id header")
	}
	return copyID, nil
}

func (b BlobStorageClient) AbortBlobCopy(container, name, copyID, currentLeaseID string, timeout int) error {
	params := url.Values{"comp": {"copy"}, "copyid": {copyID}}
	if timeout > 0 {
		params.Add("timeout", strconv.Itoa(timeout))
	}

	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), params)
	headers := b.client.getStandardHeaders()
	headers["x-ms-copy-action"] = "abort"

	if currentLeaseID != "" {
		headers[headerLeaseID] = currentLeaseID
	}

	resp, err := b.client.exec(http.MethodPut, uri, headers, nil, b.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)

	if err := checkRespCode(resp.statusCode, []int{http.StatusNoContent}); err != nil {
		return err
	}

	return nil
}

func (b BlobStorageClient) WaitForBlobCopy(container, name, copyID string) error {
	for {
		props, err := b.GetBlobProperties(container, name)
		if err != nil {
			return err
		}

		if props.CopyID != copyID {
			return errBlobCopyIDMismatch
		}

		switch props.CopyStatus {
		case blobCopyStatusSuccess:
			return nil
		case blobCopyStatusPending:
			continue
		case blobCopyStatusAborted:
			return errBlobCopyAborted
		case blobCopyStatusFailed:
			return fmt.Errorf("storage: blob copy failed. Id=%s Description=%s", props.CopyID, props.CopyStatusDescription)
		default:
			return fmt.Errorf("storage: unhandled blob copy status: '%s'", props.CopyStatus)
		}
	}
}

func (b BlobStorageClient) DeleteBlob(container, name string, extraHeaders map[string]string) error {
	resp, err := b.deleteBlob(container, name, extraHeaders)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusAccepted})
}

func (b BlobStorageClient) DeleteBlobIfExists(container, name string, extraHeaders map[string]string) (bool, error) {
	resp, err := b.deleteBlob(container, name, extraHeaders)
	if resp != nil {
		defer readAndCloseBody(resp.body)
		if resp.statusCode == http.StatusAccepted || resp.statusCode == http.StatusNotFound {
			return resp.statusCode == http.StatusAccepted, nil
		}
	}
	return false, err
}

func (b BlobStorageClient) deleteBlob(container, name string, extraHeaders map[string]string) (*storageResponse, error) {
	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), url.Values{})
	extraHeaders = b.client.protectUserAgent(extraHeaders)
	headers := b.client.getStandardHeaders()
	for k, v := range extraHeaders {
		headers[k] = v
	}

	return b.client.exec(http.MethodDelete, uri, headers, nil, b.auth)
}

func pathForBlob(container, name string) string {
	return fmt.Sprintf("/%s/%s", container, name)
}

func pathForResource(container, name string) string {
	if len(name) > 0 {
		return fmt.Sprintf("/%s/%s", container, name)
	}
	return fmt.Sprintf("/%s", container)
}

func (b BlobStorageClient) GetBlobSASURIWithSignedIPAndProtocol(container, name string, expiry time.Time, permissions string, signedIPRange string, HTTPSOnly bool) (string, error) {
	var (
		signedPermissions = permissions
		blobURL           = b.GetBlobURL(container, name)
	)
	canonicalizedResource, err := b.client.buildCanonicalizedResource(blobURL, b.auth)
	if err != nil {
		return "", err
	}

	canonicalizedResource = strings.Replace(canonicalizedResource, "+", "%2b", -1)
	canonicalizedResource, err = url.QueryUnescape(canonicalizedResource)
	if err != nil {
		return "", err
	}

	signedExpiry := expiry.UTC().Format(time.RFC3339)

	signedResource := "c"
	if len(name) > 0 {
		signedResource = "b"
	}

	protocols := "https,http"
	if HTTPSOnly {
		protocols = "https"
	}
	stringToSign, err := blobSASStringToSign(b.client.apiVersion, canonicalizedResource, signedExpiry, signedPermissions, signedIPRange, protocols)
	if err != nil {
		return "", err
	}

	sig := b.client.computeHmac256(stringToSign)
	sasParams := url.Values{
		"sv":  {b.client.apiVersion},
		"se":  {signedExpiry},
		"sr":  {signedResource},
		"sp":  {signedPermissions},
		"sig": {sig},
	}

	if b.client.apiVersion >= "2015-04-05" {
		sasParams.Add("spr", protocols)
		if signedIPRange != "" {
			sasParams.Add("sip", signedIPRange)
		}
	}

	sasURL, err := url.Parse(blobURL)
	if err != nil {
		return "", err
	}
	sasURL.RawQuery = sasParams.Encode()
	return sasURL.String(), nil
}

func (b BlobStorageClient) GetBlobSASURI(container, name string, expiry time.Time, permissions string) (string, error) {
	url, err := b.GetBlobSASURIWithSignedIPAndProtocol(container, name, expiry, permissions, "", false)
	return url, err
}

func blobSASStringToSign(signedVersion, canonicalizedResource, signedExpiry, signedPermissions string, signedIP string, protocols string) (string, error) {
	var signedStart, signedIdentifier, rscc, rscd, rsce, rscl, rsct string

	if signedVersion >= "2015-02-21" {
		canonicalizedResource = "/blob" + canonicalizedResource
	}

	if signedVersion >= "2015-04-05" {
		return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s", signedPermissions, signedStart, signedExpiry, canonicalizedResource, signedIdentifier, signedIP, protocols, signedVersion, rscc, rscd, rsce, rscl, rsct), nil
	}

	if signedVersion >= "2013-08-15" {
		return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s", signedPermissions, signedStart, signedExpiry, canonicalizedResource, signedIdentifier, signedVersion, rscc, rscd, rsce, rscl, rsct), nil
	}

	return "", errors.New("storage: not implemented SAS for versions earlier than 2013-08-15")
}
