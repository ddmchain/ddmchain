package storage

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (

	approximateMessagesCountHeader  = "X-Ms-Approximate-Messages-Count"
	userDefinedMetadataHeaderPrefix = "X-Ms-Meta-"
)

func pathForQueue(queue string) string         { return fmt.Sprintf("/%s", queue) }
func pathForQueueMessages(queue string) string { return fmt.Sprintf("/%s/messages", queue) }
func pathForMessage(queue, name string) string { return fmt.Sprintf("/%s/messages/%s", queue, name) }

type putMessageRequest struct {
	XMLName     xml.Name `xml:"QueueMessage"`
	MessageText string   `xml:"MessageText"`
}

type PutMessageParameters struct {
	VisibilityTimeout int
	MessageTTL        int
}

func (p PutMessageParameters) getParameters() url.Values {
	out := url.Values{}
	if p.VisibilityTimeout != 0 {
		out.Set("visibilitytimeout", strconv.Itoa(p.VisibilityTimeout))
	}
	if p.MessageTTL != 0 {
		out.Set("messagettl", strconv.Itoa(p.MessageTTL))
	}
	return out
}

type GetMessagesParameters struct {
	NumOfMessages     int
	VisibilityTimeout int
}

func (p GetMessagesParameters) getParameters() url.Values {
	out := url.Values{}
	if p.NumOfMessages != 0 {
		out.Set("numofmessages", strconv.Itoa(p.NumOfMessages))
	}
	if p.VisibilityTimeout != 0 {
		out.Set("visibilitytimeout", strconv.Itoa(p.VisibilityTimeout))
	}
	return out
}

type PeekMessagesParameters struct {
	NumOfMessages int
}

func (p PeekMessagesParameters) getParameters() url.Values {
	out := url.Values{"peekonly": {"true"}} 
	if p.NumOfMessages != 0 {
		out.Set("numofmessages", strconv.Itoa(p.NumOfMessages))
	}
	return out
}

type UpdateMessageParameters struct {
	PopReceipt        string
	VisibilityTimeout int
}

func (p UpdateMessageParameters) getParameters() url.Values {
	out := url.Values{}
	if p.PopReceipt != "" {
		out.Set("popreceipt", p.PopReceipt)
	}
	if p.VisibilityTimeout != 0 {
		out.Set("visibilitytimeout", strconv.Itoa(p.VisibilityTimeout))
	}
	return out
}

type GetMessagesResponse struct {
	XMLName           xml.Name             `xml:"QueueMessagesList"`
	QueueMessagesList []GetMessageResponse `xml:"QueueMessage"`
}

type GetMessageResponse struct {
	MessageID       string `xml:"MessageId"`
	InsertionTime   string `xml:"InsertionTime"`
	ExpirationTime  string `xml:"ExpirationTime"`
	PopReceipt      string `xml:"PopReceipt"`
	TimeNextVisible string `xml:"TimeNextVisible"`
	DequeueCount    int    `xml:"DequeueCount"`
	MessageText     string `xml:"MessageText"`
}

type PeekMessagesResponse struct {
	XMLName           xml.Name              `xml:"QueueMessagesList"`
	QueueMessagesList []PeekMessageResponse `xml:"QueueMessage"`
}

type PeekMessageResponse struct {
	MessageID      string `xml:"MessageId"`
	InsertionTime  string `xml:"InsertionTime"`
	ExpirationTime string `xml:"ExpirationTime"`
	DequeueCount   int    `xml:"DequeueCount"`
	MessageText    string `xml:"MessageText"`
}

type QueueMetadataResponse struct {
	ApproximateMessageCount int
	UserDefinedMetadata     map[string]string
}

func (c QueueServiceClient) SetMetadata(name string, metadata map[string]string) error {
	uri := c.client.getEndpoint(queueServiceName, pathForQueue(name), url.Values{"comp": []string{"metadata"}})
	metadata = c.client.protectUserAgent(metadata)
	headers := c.client.getStandardHeaders()
	for k, v := range metadata {
		headers[userDefinedMetadataHeaderPrefix+k] = v
	}

	resp, err := c.client.exec(http.MethodPut, uri, headers, nil, c.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)

	return checkRespCode(resp.statusCode, []int{http.StatusNoContent})
}

func (c QueueServiceClient) GetMetadata(name string) (QueueMetadataResponse, error) {
	qm := QueueMetadataResponse{}
	qm.UserDefinedMetadata = make(map[string]string)
	uri := c.client.getEndpoint(queueServiceName, pathForQueue(name), url.Values{"comp": []string{"metadata"}})
	headers := c.client.getStandardHeaders()
	resp, err := c.client.exec(http.MethodGet, uri, headers, nil, c.auth)
	if err != nil {
		return qm, err
	}
	defer readAndCloseBody(resp.body)

	for k, v := range resp.headers {
		if len(v) != 1 {
			return qm, fmt.Errorf("Unexpected number of values (%d) in response header '%s'", len(v), k)
		}

		value := v[0]

		if k == approximateMessagesCountHeader {
			qm.ApproximateMessageCount, err = strconv.Atoi(value)
			if err != nil {
				return qm, fmt.Errorf("Unexpected value in response header '%s': '%s' ", k, value)
			}
		} else if strings.HasPrefix(k, userDefinedMetadataHeaderPrefix) {
			name := strings.TrimPrefix(k, userDefinedMetadataHeaderPrefix)
			qm.UserDefinedMetadata[strings.ToLower(name)] = value
		}
	}

	return qm, checkRespCode(resp.statusCode, []int{http.StatusOK})
}

func (c QueueServiceClient) CreateQueue(name string) error {
	uri := c.client.getEndpoint(queueServiceName, pathForQueue(name), url.Values{})
	headers := c.client.getStandardHeaders()
	resp, err := c.client.exec(http.MethodPut, uri, headers, nil, c.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

func (c QueueServiceClient) DeleteQueue(name string) error {
	uri := c.client.getEndpoint(queueServiceName, pathForQueue(name), url.Values{})
	resp, err := c.client.exec(http.MethodDelete, uri, c.client.getStandardHeaders(), nil, c.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusNoContent})
}

func (c QueueServiceClient) QueueExists(name string) (bool, error) {
	uri := c.client.getEndpoint(queueServiceName, pathForQueue(name), url.Values{"comp": {"metadata"}})
	resp, err := c.client.exec(http.MethodGet, uri, c.client.getStandardHeaders(), nil, c.auth)
	if resp != nil && (resp.statusCode == http.StatusOK || resp.statusCode == http.StatusNotFound) {
		return resp.statusCode == http.StatusOK, nil
	}

	return false, err
}

func (c QueueServiceClient) PutMessage(queue string, message string, params PutMessageParameters) error {
	uri := c.client.getEndpoint(queueServiceName, pathForQueueMessages(queue), params.getParameters())
	req := putMessageRequest{MessageText: message}
	body, nn, err := xmlMarshal(req)
	if err != nil {
		return err
	}
	headers := c.client.getStandardHeaders()
	headers["Content-Length"] = strconv.Itoa(nn)
	resp, err := c.client.exec(http.MethodPost, uri, headers, body, c.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

func (c QueueServiceClient) ClearMessages(queue string) error {
	uri := c.client.getEndpoint(queueServiceName, pathForQueueMessages(queue), url.Values{})
	resp, err := c.client.exec(http.MethodDelete, uri, c.client.getStandardHeaders(), nil, c.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusNoContent})
}

func (c QueueServiceClient) GetMessages(queue string, params GetMessagesParameters) (GetMessagesResponse, error) {
	var r GetMessagesResponse
	uri := c.client.getEndpoint(queueServiceName, pathForQueueMessages(queue), params.getParameters())
	resp, err := c.client.exec(http.MethodGet, uri, c.client.getStandardHeaders(), nil, c.auth)
	if err != nil {
		return r, err
	}
	defer resp.body.Close()
	err = xmlUnmarshal(resp.body, &r)
	return r, err
}

func (c QueueServiceClient) PeekMessages(queue string, params PeekMessagesParameters) (PeekMessagesResponse, error) {
	var r PeekMessagesResponse
	uri := c.client.getEndpoint(queueServiceName, pathForQueueMessages(queue), params.getParameters())
	resp, err := c.client.exec(http.MethodGet, uri, c.client.getStandardHeaders(), nil, c.auth)
	if err != nil {
		return r, err
	}
	defer resp.body.Close()
	err = xmlUnmarshal(resp.body, &r)
	return r, err
}

func (c QueueServiceClient) DeleteMessage(queue, messageID, popReceipt string) error {
	uri := c.client.getEndpoint(queueServiceName, pathForMessage(queue, messageID), url.Values{
		"popreceipt": {popReceipt}})
	resp, err := c.client.exec(http.MethodDelete, uri, c.client.getStandardHeaders(), nil, c.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusNoContent})
}

func (c QueueServiceClient) UpdateMessage(queue string, messageID string, message string, params UpdateMessageParameters) error {
	uri := c.client.getEndpoint(queueServiceName, pathForMessage(queue, messageID), params.getParameters())
	req := putMessageRequest{MessageText: message}
	body, nn, err := xmlMarshal(req)
	if err != nil {
		return err
	}
	headers := c.client.getStandardHeaders()
	headers["Content-Length"] = fmt.Sprintf("%d", nn)
	resp, err := c.client.exec(http.MethodPut, uri, headers, body, c.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusNoContent})
}
