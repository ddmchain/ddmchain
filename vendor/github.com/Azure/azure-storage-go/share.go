package storage

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type Share struct {
	fsc        *FileServiceClient
	Name       string          `xml:"Name"`
	Properties ShareProperties `xml:"Properties"`
	Metadata   map[string]string
}

type ShareProperties struct {
	LastModified string `xml:"Last-Modified"`
	Etag         string `xml:"Etag"`
	Quota        int    `xml:"Quota"`
}

func (s *Share) buildPath() string {
	return fmt.Sprintf("/%s", s.Name)
}

func (s *Share) Create() error {
	headers, err := s.fsc.createResource(s.buildPath(), resourceShare, nil, mergeMDIntoExtraHeaders(s.Metadata, nil), []int{http.StatusCreated})
	if err != nil {
		return err
	}

	s.updateEtagAndLastModified(headers)
	return nil
}

func (s *Share) CreateIfNotExists() (bool, error) {
	resp, err := s.fsc.createResourceNoClose(s.buildPath(), resourceShare, nil, nil)
	if resp != nil {
		defer readAndCloseBody(resp.body)
		if resp.statusCode == http.StatusCreated || resp.statusCode == http.StatusConflict {
			if resp.statusCode == http.StatusCreated {
				s.updateEtagAndLastModified(resp.headers)
				return true, nil
			}
			return false, s.FetchAttributes()
		}
	}

	return false, err
}

func (s *Share) Delete() error {
	return s.fsc.deleteResource(s.buildPath(), resourceShare)
}

func (s *Share) DeleteIfExists() (bool, error) {
	resp, err := s.fsc.deleteResourceNoClose(s.buildPath(), resourceShare)
	if resp != nil {
		defer readAndCloseBody(resp.body)
		if resp.statusCode == http.StatusAccepted || resp.statusCode == http.StatusNotFound {
			return resp.statusCode == http.StatusAccepted, nil
		}
	}
	return false, err
}

func (s *Share) Exists() (bool, error) {
	exists, headers, err := s.fsc.resourceExists(s.buildPath(), resourceShare)
	if exists {
		s.updateEtagAndLastModified(headers)
		s.updateQuota(headers)
	}
	return exists, err
}

func (s *Share) FetchAttributes() error {
	headers, err := s.fsc.getResourceHeaders(s.buildPath(), compNone, resourceShare, http.MethodHead)
	if err != nil {
		return err
	}

	s.updateEtagAndLastModified(headers)
	s.updateQuota(headers)
	s.Metadata = getMetadataFromHeaders(headers)

	return nil
}

func (s *Share) GetRootDirectoryReference() *Directory {
	return &Directory{
		fsc:   s.fsc,
		share: s,
	}
}

func (s *Share) ServiceClient() *FileServiceClient {
	return s.fsc
}

func (s *Share) SetMetadata() error {
	headers, err := s.fsc.setResourceHeaders(s.buildPath(), compMetadata, resourceShare, mergeMDIntoExtraHeaders(s.Metadata, nil))
	if err != nil {
		return err
	}

	s.updateEtagAndLastModified(headers)
	return nil
}

func (s *Share) SetProperties() error {
	if s.Properties.Quota < 1 || s.Properties.Quota > 5120 {
		return fmt.Errorf("invalid value %v for quota, valid values are [1, 5120]", s.Properties.Quota)
	}

	headers, err := s.fsc.setResourceHeaders(s.buildPath(), compProperties, resourceShare, map[string]string{
		"x-ms-share-quota": strconv.Itoa(s.Properties.Quota),
	})
	if err != nil {
		return err
	}

	s.updateEtagAndLastModified(headers)
	return nil
}

func (s *Share) updateEtagAndLastModified(headers http.Header) {
	s.Properties.Etag = headers.Get("Etag")
	s.Properties.LastModified = headers.Get("Last-Modified")
}

func (s *Share) updateQuota(headers http.Header) {
	quota, err := strconv.Atoi(headers.Get("x-ms-share-quota"))
	if err == nil {
		s.Properties.Quota = quota
	}
}

func (s *Share) URL() string {
	return s.fsc.client.getEndpoint(fileServiceName, s.buildPath(), url.Values{})
}
