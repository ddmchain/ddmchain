package storage

import (
	"fmt"
	"net/http"
	"net/url"
)

type BlobStorageClient struct {
	client Client
	auth   authentication
}

func (b *BlobStorageClient) GetServiceProperties() (*ServiceProperties, error) {
	return b.client.getServiceProperties(blobServiceName, b.auth)
}

func (b *BlobStorageClient) SetServiceProperties(props ServiceProperties) error {
	return b.client.setServiceProperties(props, blobServiceName, b.auth)
}

type ListContainersParameters struct {
	Prefix     string
	Marker     string
	Include    string
	MaxResults uint
	Timeout    uint
}

func (b BlobStorageClient) GetContainerReference(name string) Container {
	return Container{
		bsc:  &b,
		Name: name,
	}
}

func (b BlobStorageClient) ListContainers(params ListContainersParameters) (*ContainerListResponse, error) {
	q := mergeParams(params.getParameters(), url.Values{"comp": {"list"}})
	uri := b.client.getEndpoint(blobServiceName, "", q)
	headers := b.client.getStandardHeaders()

	var out ContainerListResponse
	resp, err := b.client.exec(http.MethodGet, uri, headers, nil, b.auth)
	if err != nil {
		return nil, err
	}
	defer resp.body.Close()
	err = xmlUnmarshal(resp.body, &out)

	for i := range out.Containers {
		out.Containers[i].bsc = &b
	}
	return &out, err
}

func (p ListContainersParameters) getParameters() url.Values {
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
