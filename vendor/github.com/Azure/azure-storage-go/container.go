package storage

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Container struct {
	bsc        *BlobStorageClient
	Name       string              `xml:"Name"`
	Properties ContainerProperties `xml:"Properties"`
}

func (c *Container) buildPath() string {
	return fmt.Sprintf("/%s", c.Name)
}

type ContainerProperties struct {
	LastModified  string `xml:"Last-Modified"`
	Etag          string `xml:"Etag"`
	LeaseStatus   string `xml:"LeaseStatus"`
	LeaseState    string `xml:"LeaseState"`
	LeaseDuration string `xml:"LeaseDuration"`
}

type ContainerListResponse struct {
	XMLName    xml.Name    `xml:"EnumerationResults"`
	Xmlns      string      `xml:"xmlns,attr"`
	Prefix     string      `xml:"Prefix"`
	Marker     string      `xml:"Marker"`
	NextMarker string      `xml:"NextMarker"`
	MaxResults int64       `xml:"MaxResults"`
	Containers []Container `xml:"Containers>Container"`
}

type BlobListResponse struct {
	XMLName    xml.Name `xml:"EnumerationResults"`
	Xmlns      string   `xml:"xmlns,attr"`
	Prefix     string   `xml:"Prefix"`
	Marker     string   `xml:"Marker"`
	NextMarker string   `xml:"NextMarker"`
	MaxResults int64    `xml:"MaxResults"`
	Blobs      []Blob   `xml:"Blobs>Blob"`

	BlobPrefixes []string `xml:"Blobs>BlobPrefix>Name"`

	Delimiter string `xml:"Delimiter"`
}

type ListBlobsParameters struct {
	Prefix     string
	Delimiter  string
	Marker     string
	Include    string
	MaxResults uint
	Timeout    uint
}

func (p ListBlobsParameters) getParameters() url.Values {
	out := url.Values{}

	if p.Prefix != "" {
		out.Set("prefix", p.Prefix)
	}
	if p.Delimiter != "" {
		out.Set("delimiter", p.Delimiter)
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

type ContainerAccessType string

const (
	ContainerAccessTypePrivate   ContainerAccessType = ""
	ContainerAccessTypeBlob      ContainerAccessType = "blob"
	ContainerAccessTypeContainer ContainerAccessType = "container"
)

type ContainerAccessPolicy struct {
	ID         string
	StartTime  time.Time
	ExpiryTime time.Time
	CanRead    bool
	CanWrite   bool
	CanDelete  bool
}

type ContainerPermissions struct {
	AccessType     ContainerAccessType
	AccessPolicies []ContainerAccessPolicy
}

const (
	ContainerAccessHeader string = "x-ms-blob-public-access"
)

func (c *Container) Create() error {
	resp, err := c.create()
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

func (c *Container) CreateIfNotExists() (bool, error) {
	resp, err := c.create()
	if resp != nil {
		defer readAndCloseBody(resp.body)
		if resp.statusCode == http.StatusCreated || resp.statusCode == http.StatusConflict {
			return resp.statusCode == http.StatusCreated, nil
		}
	}
	return false, err
}

func (c *Container) create() (*storageResponse, error) {
	uri := c.bsc.client.getEndpoint(blobServiceName, c.buildPath(), url.Values{"restype": {"container"}})
	headers := c.bsc.client.getStandardHeaders()
	return c.bsc.client.exec(http.MethodPut, uri, headers, nil, c.bsc.auth)
}

func (c *Container) Exists() (bool, error) {
	uri := c.bsc.client.getEndpoint(blobServiceName, c.buildPath(), url.Values{"restype": {"container"}})
	headers := c.bsc.client.getStandardHeaders()

	resp, err := c.bsc.client.exec(http.MethodHead, uri, headers, nil, c.bsc.auth)
	if resp != nil {
		defer readAndCloseBody(resp.body)
		if resp.statusCode == http.StatusOK || resp.statusCode == http.StatusNotFound {
			return resp.statusCode == http.StatusOK, nil
		}
	}
	return false, err
}

func (c *Container) SetPermissions(permissions ContainerPermissions, timeout int, leaseID string) error {
	params := url.Values{
		"restype": {"container"},
		"comp":    {"acl"},
	}

	if timeout > 0 {
		params.Add("timeout", strconv.Itoa(timeout))
	}

	uri := c.bsc.client.getEndpoint(blobServiceName, c.buildPath(), params)
	headers := c.bsc.client.getStandardHeaders()
	if permissions.AccessType != "" {
		headers[ContainerAccessHeader] = string(permissions.AccessType)
	}

	if leaseID != "" {
		headers[headerLeaseID] = leaseID
	}

	body, length, err := generateContainerACLpayload(permissions.AccessPolicies)
	headers["Content-Length"] = strconv.Itoa(length)

	resp, err := c.bsc.client.exec(http.MethodPut, uri, headers, body, c.bsc.auth)
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)

	if err := checkRespCode(resp.statusCode, []int{http.StatusOK}); err != nil {
		return errors.New("Unable to set permissions")
	}

	return nil
}

func (c *Container) GetPermissions(timeout int, leaseID string) (*ContainerPermissions, error) {
	params := url.Values{
		"restype": {"container"},
		"comp":    {"acl"},
	}

	if timeout > 0 {
		params.Add("timeout", strconv.Itoa(timeout))
	}

	uri := c.bsc.client.getEndpoint(blobServiceName, c.buildPath(), params)
	headers := c.bsc.client.getStandardHeaders()

	if leaseID != "" {
		headers[headerLeaseID] = leaseID
	}

	resp, err := c.bsc.client.exec(http.MethodGet, uri, headers, nil, c.bsc.auth)
	if err != nil {
		return nil, err
	}
	defer resp.body.Close()

	var ap AccessPolicy
	err = xmlUnmarshal(resp.body, &ap.SignedIdentifiersList)
	if err != nil {
		return nil, err
	}
	return buildAccessPolicy(ap, &resp.headers), nil
}

func buildAccessPolicy(ap AccessPolicy, headers *http.Header) *ContainerPermissions {

	containerAccess := headers.Get(http.CanonicalHeaderKey(ContainerAccessHeader))
	permissions := ContainerPermissions{
		AccessType:     ContainerAccessType(containerAccess),
		AccessPolicies: []ContainerAccessPolicy{},
	}

	for _, policy := range ap.SignedIdentifiersList.SignedIdentifiers {
		capd := ContainerAccessPolicy{
			ID:         policy.ID,
			StartTime:  policy.AccessPolicy.StartTime,
			ExpiryTime: policy.AccessPolicy.ExpiryTime,
		}
		capd.CanRead = updatePermissions(policy.AccessPolicy.Permission, "r")
		capd.CanWrite = updatePermissions(policy.AccessPolicy.Permission, "w")
		capd.CanDelete = updatePermissions(policy.AccessPolicy.Permission, "d")

		permissions.AccessPolicies = append(permissions.AccessPolicies, capd)
	}
	return &permissions
}

func (c *Container) Delete() error {
	resp, err := c.delete()
	if err != nil {
		return err
	}
	defer readAndCloseBody(resp.body)
	return checkRespCode(resp.statusCode, []int{http.StatusAccepted})
}

func (c *Container) DeleteIfExists() (bool, error) {
	resp, err := c.delete()
	if resp != nil {
		defer readAndCloseBody(resp.body)
		if resp.statusCode == http.StatusAccepted || resp.statusCode == http.StatusNotFound {
			return resp.statusCode == http.StatusAccepted, nil
		}
	}
	return false, err
}

func (c *Container) delete() (*storageResponse, error) {
	uri := c.bsc.client.getEndpoint(blobServiceName, c.buildPath(), url.Values{"restype": {"container"}})
	headers := c.bsc.client.getStandardHeaders()
	return c.bsc.client.exec(http.MethodDelete, uri, headers, nil, c.bsc.auth)
}

func (c *Container) ListBlobs(params ListBlobsParameters) (BlobListResponse, error) {
	q := mergeParams(params.getParameters(), url.Values{
		"restype": {"container"},
		"comp":    {"list"}},
	)
	uri := c.bsc.client.getEndpoint(blobServiceName, c.buildPath(), q)
	headers := c.bsc.client.getStandardHeaders()

	var out BlobListResponse
	resp, err := c.bsc.client.exec(http.MethodGet, uri, headers, nil, c.bsc.auth)
	if err != nil {
		return out, err
	}
	defer resp.body.Close()

	err = xmlUnmarshal(resp.body, &out)
	return out, err
}

func generateContainerACLpayload(policies []ContainerAccessPolicy) (io.Reader, int, error) {
	sil := SignedIdentifiers{
		SignedIdentifiers: []SignedIdentifier{},
	}
	for _, capd := range policies {
		permission := capd.generateContainerPermissions()
		signedIdentifier := convertAccessPolicyToXMLStructs(capd.ID, capd.StartTime, capd.ExpiryTime, permission)
		sil.SignedIdentifiers = append(sil.SignedIdentifiers, signedIdentifier)
	}
	return xmlMarshal(sil)
}

func (capd *ContainerAccessPolicy) generateContainerPermissions() (permissions string) {

	permissions = ""

	if capd.CanRead {
		permissions += "r"
	}

	if capd.CanWrite {
		permissions += "w"
	}

	if capd.CanDelete {
		permissions += "d"
	}

	return permissions
}
