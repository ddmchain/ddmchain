package storage

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type FileServiceClient struct {
	client Client
	auth   authentication
}

type ListSharesParameters struct {
	Prefix     string
	Marker     string
	Include    string
	MaxResults uint
	Timeout    uint
}

type ShareListResponse struct {
	XMLName    xml.Name `xml:"EnumerationResults"`
	Xmlns      string   `xml:"xmlns,attr"`
	Prefix     string   `xml:"Prefix"`
	Marker     string   `xml:"Marker"`
	NextMarker string   `xml:"NextMarker"`
	MaxResults int64    `xml:"MaxResults"`
	Shares     []Share  `xml:"Shares>Share"`
}

type compType string

const (
	compNone       compType = ""
	compList       compType = "list"
	compMetadata   compType = "metadata"
	compProperties compType = "properties"
	compRangeList  compType = "rangelist"
)

func (ct compType) String() string {
	return string(ct)
}

type resourceType string

const (
	resourceDirectory resourceType = "directory"
	resourceFile      resourceType = ""
	resourceShare     resourceType = "share"
)

func (rt resourceType) String() string {
	return string(rt)
}

func (p ListSharesParameters) getParameters() url.Values {
	out := url.Values{}

	if p.Prefix != "" {
		out.Set("prefix", p.Prefix)
	}
	if p.Marker != "" {
		out.Set("marker", p.Marker)
	}
	if p.Include != "" {
		out.Set("include", p.Include)
	}
	if p.MaxResults != 0 {
		out.Set("maxresults", fmt.Sprintf("%v", p.MaxResults))
	}
	if p.Timeout != 0 {
		out.Set("timeout", fmt.Sprintf("%v", p.Timeout))
	}

	return out
}

func (p ListDirsAndFilesParameters) getParameters() url.Values {
	out := url.Values{}

	if p.Marker != "" {
		out.Set("marker", p.Marker)
	}
	if p.MaxResults != 0 {
		out.Set("maxresults", fmt.Sprintf("%v", p.MaxResults))
	}
	if p.Timeout != 0 {
		out.Set("timeout", fmt.Sprintf("%v", p.Timeout))
	}

	return out
}

func getURLInitValues(comp compType, res resourceType) url.Values {
	values := url.Values{}
	if comp != compNone {
		values.Set("comp", comp.String())
	}
	if res != resourceFile {
		values.Set("restype", res.String())
	}
	return values
}

func (f FileServiceClient) GetShareReference(name string) Share {
	return Share{
		fsc:  &f,
		Name: name,
		Properties: ShareProperties{
			Quota: -1,
		},
	}
}

func (f FileServiceClient) ListShares(params ListSharesParameters) (*ShareListResponse, error) {
	q := mergeParams(params.getParameters(), url.Values{"comp": {"list"}})

	var out ShareListResponse
	resp, err := f.listContent("", q, nil)
	if err != nil {
		return nil, err
	}
	defer resp.body.Close()
	err = xmlUnmarshal(resp.body, &out)

	for i := range out.Shares {
		out.Shares[i].fsc = &f
	}
	return &out, err
}

func (f *FileServiceClient) GetServiceProperties() (*ServiceProperties, error) {
	return f.client.getServiceProperties(fileServiceName, f.auth)
}

func (f *FileServiceClient) SetServiceProperties(props ServiceProperties) error {
	return f.client.setServiceProperties(props, fileServiceName, f.auth)
}

func (f FileServiceClient) listContent(path string, params url.Values, extraHeaders map[string]string) (*storageResponse, error) {
	if err := f.checkForStorageEmulator(); err != nil {
		return nil, err
	}

	uri := f.client.getEndpoint(fileServiceName, path, params)
	extraHeaders = f.client.protectUserAgent(extraHeaders)
	headers := mergeHeaders(f.client.getStandardHeaders(), extraHeaders)

	resp, err := f.client.exec(http.MethodGet, uri, headers, nil, f.auth)
	if err != nil {
		return nil, err
	}

	if err = checkRespCode(resp.statusCode, []int{http.StatusOK}); err != nil {
		readAndCloseBody(resp.body)
		return nil, err
	}

	return resp, nil
}

func (f FileServiceClient) resourceExists(path string, res resourceType) (bool, http.Header, error) {
	if err := f.checkForStorageEmulator(); err != nil {
		return false, nil, err
	}

	uri := f.client.getEndpoint(fileServiceName, path, getURLInitValues(compNone, res))
	headers := f.client.getStandardHeaders()

	resp, err := f.client.exec(http.MethodHead, uri, headers, nil, f.auth)
	if resp != nil {
		defer readAndCloseBody(resp.body)
		if resp.statusCode == http.StatusOK || resp.statusCode == http.StatusNotFound {
			return resp.statusCode == http.StatusOK, resp.headers, nil
		}
	}
	return false, nil, err
}

func (f FileServiceClient) createResource(path string, res resourceType, urlParams url.Values, extraHeaders map[string]string, expectedResponseCodes []int) (http.Header, error) {
	resp, err := f.createResourceNoClose(path, res, urlParams, extraHeaders)
	if err != nil {
		return nil, err
	}
	defer readAndCloseBody(resp.body)
	return resp.headers, checkRespCode(resp.statusCode, expectedResponseCodes)
}

func (f FileServiceClient) createResourceNoClose(path string, res resourceType, urlParams url.Values, extraHeaders map[string]string) (*storageResponse, error) {
	if err := f.checkForStorageEmulator(); err != nil {
		return nil, err
	}

	values := getURLInitValues(compNone, res)
	combinedParams := mergeParams(values, urlParams)
	uri := f.client.getEndpoint(fileServiceName, path, combinedParams)
	extraHeaders = f.client.protectUserAgent(extraHeaders)
	headers := mergeHeaders(f.client.getStandardHeaders(), extraHeaders)

	return f.client.exec(http.MethodPut, uri, headers, nil, f.auth)
}

func (f FileServiceClient) getResourceHeaders(path string, comp compType, res resourceType, verb string) (http.Header, error) {
	resp, err := f.getResourceNoClose(path, comp, res, verb, nil)
	if err != nil {
		return nil, err
	}
	defer readAndCloseBody(resp.body)

	if err = checkRespCode(resp.statusCode, []int{http.StatusOK}); err != nil {
		return nil, err
	}

	return resp.headers, nil
}

func (f FileServiceClient) getResourceNoClose(path string, comp compType, res resourceType, verb string, extraHeaders map[string]string) (*storageResponse, error) {
	if err := f.checkForStorageEmulator(); err != nil {
		return nil, err
	}

	params := getURLInitValues(comp, res)
	uri := f.client.getEndpoint(fileServiceName, path, params)
	extraHeaders = f.client.protectUserAgent(extraHeaders)
	headers := mergeHeaders(f.client.getStandardHeaders(), extraHeaders)

	return f.client.exec(verb, uri, headers, nil, f.auth)
}

func (f FileServiceClient) deleteResource(path string, res resourceType) error {
	resp, err := f.deleteResourceNoClose(path, res)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusAccepted})
}

func (f FileServiceClient) deleteResourceNoClose(path string, res resourceType) (*storageResponse, error) {
	if err := f.checkForStorageEmulator(); err != nil {
		return nil, err
	}

	values := getURLInitValues(compNone, res)
	uri := f.client.getEndpoint(fileServiceName, path, values)
	return f.client.exec(http.MethodDelete, uri, f.client.getStandardHeaders(), nil, f.auth)
}

func mergeMDIntoExtraHeaders(metadata, extraHeaders map[string]string) map[string]string {
	if metadata == nil && extraHeaders == nil {
		return nil
	}
	if extraHeaders == nil {
		extraHeaders = make(map[string]string)
	}
	for k, v := range metadata {
		extraHeaders[userDefinedMetadataHeaderPrefix+k] = v
	}
	return extraHeaders
}

func mergeHeaders(headers, extraHeaders map[string]string) map[string]string {
	for k, v := range extraHeaders {
		headers[k] = v
	}
	return headers
}

func (f FileServiceClient) setResourceHeaders(path string, comp compType, res resourceType, extraHeaders map[string]string) (http.Header, error) {
	if err := f.checkForStorageEmulator(); err != nil {
		return nil, err
	}

	params := getURLInitValues(comp, res)
	uri := f.client.getEndpoint(fileServiceName, path, params)
	extraHeaders = f.client.protectUserAgent(extraHeaders)
	headers := mergeHeaders(f.client.getStandardHeaders(), extraHeaders)

	resp, err := f.client.exec(http.MethodPut, uri, headers, nil, f.auth)
	if err != nil {
		return nil, err
	}
	defer readAndCloseBody(resp.body)

	return resp.headers, checkRespCode(resp.statusCode, []int{http.StatusOK})
}

func (f FileServiceClient) getMetadata(path string, res resourceType) (map[string]string, error) {
	if err := f.checkForStorageEmulator(); err != nil {
		return nil, err
	}

	headers, err := f.getResourceHeaders(path, compMetadata, res, http.MethodGet)
	if err != nil {
		return nil, err
	}

	return getMetadataFromHeaders(headers), nil
}

func getMetadataFromHeaders(header http.Header) map[string]string {
	metadata := make(map[string]string)
	for k, v := range header {

		k = strings.ToLower(k)
		if len(v) == 0 || !strings.HasPrefix(k, strings.ToLower(userDefinedMetadataHeaderPrefix)) {
			continue
		}

		k = k[len(userDefinedMetadataHeaderPrefix):]
		metadata[k] = v[len(v)-1]
	}

	if len(metadata) == 0 {
		return nil
	}

	return metadata
}

func (f FileServiceClient) checkForStorageEmulator() error {
	if f.client.accountName == StorageEmulatorAccountName {
		return fmt.Errorf("Error: File service is not currently supported by Azure Storage Emulator")
	}
	return nil
}
