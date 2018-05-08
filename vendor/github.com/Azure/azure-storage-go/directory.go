package storage

import (
	"encoding/xml"
	"net/http"
	"net/url"
)

type Directory struct {
	fsc        *FileServiceClient
	Metadata   map[string]string
	Name       string `xml:"Name"`
	parent     *Directory
	Properties DirectoryProperties
	share      *Share
}

type DirectoryProperties struct {
	LastModified string `xml:"Last-Modified"`
	Etag         string `xml:"Etag"`
}

type ListDirsAndFilesParameters struct {
	Marker     string
	MaxResults uint
	Timeout    uint
}

type DirsAndFilesListResponse struct {
	XMLName     xml.Name    `xml:"EnumerationResults"`
	Xmlns       string      `xml:"xmlns,attr"`
	Marker      string      `xml:"Marker"`
	MaxResults  int64       `xml:"MaxResults"`
	Directories []Directory `xml:"Entries>Directory"`
	Files       []File      `xml:"Entries>File"`
	NextMarker  string      `xml:"NextMarker"`
}

func (d *Directory) buildPath() string {
	path := ""
	current := d
	for current.Name != "" {
		path = "/" + current.Name + path
		current = current.parent
	}
	return d.share.buildPath() + path
}

func (d *Directory) Create() error {

	if d.parent == nil {
		return nil
	}

	headers, err := d.fsc.createResource(d.buildPath(), resourceDirectory, nil, mergeMDIntoExtraHeaders(d.Metadata, nil), []int{http.StatusCreated})
	if err != nil {
		return err
	}

	d.updateEtagAndLastModified(headers)
	return nil
}

func (d *Directory) CreateIfNotExists() (bool, error) {

	if d.parent == nil {
		return false, nil
	}

	resp, err := d.fsc.createResourceNoClose(d.buildPath(), resourceDirectory, nil, nil)
	if resp != nil {
		defer readAndCloseBody(resp.body)
		if resp.statusCode == http.StatusCreated || resp.statusCode == http.StatusConflict {
			if resp.statusCode == http.StatusCreated {
				d.updateEtagAndLastModified(resp.headers)
				return true, nil
			}

			return false, d.FetchAttributes()
		}
	}

	return false, err
}

func (d *Directory) Delete() error {
	return d.fsc.deleteResource(d.buildPath(), resourceDirectory)
}

func (d *Directory) DeleteIfExists() (bool, error) {
	resp, err := d.fsc.deleteResourceNoClose(d.buildPath(), resourceDirectory)
	if resp != nil {
		defer readAndCloseBody(resp.body)
		if resp.statusCode == http.StatusAccepted || resp.statusCode == http.StatusNotFound {
			return resp.statusCode == http.StatusAccepted, nil
		}
	}
	return false, err
}

func (d *Directory) Exists() (bool, error) {
	exists, headers, err := d.fsc.resourceExists(d.buildPath(), resourceDirectory)
	if exists {
		d.updateEtagAndLastModified(headers)
	}
	return exists, err
}

func (d *Directory) FetchAttributes() error {
	headers, err := d.fsc.getResourceHeaders(d.buildPath(), compNone, resourceDirectory, http.MethodHead)
	if err != nil {
		return err
	}

	d.updateEtagAndLastModified(headers)
	d.Metadata = getMetadataFromHeaders(headers)

	return nil
}

func (d *Directory) GetDirectoryReference(name string) *Directory {
	return &Directory{
		fsc:    d.fsc,
		Name:   name,
		parent: d,
		share:  d.share,
	}
}

func (d *Directory) GetFileReference(name string) *File {
	return &File{
		fsc:    d.fsc,
		Name:   name,
		parent: d,
		share:  d.share,
	}
}

func (d *Directory) ListDirsAndFiles(params ListDirsAndFilesParameters) (*DirsAndFilesListResponse, error) {
	q := mergeParams(params.getParameters(), getURLInitValues(compList, resourceDirectory))

	resp, err := d.fsc.listContent(d.buildPath(), q, nil)
	if err != nil {
		return nil, err
	}

	defer resp.body.Close()
	var out DirsAndFilesListResponse
	err = xmlUnmarshal(resp.body, &out)
	return &out, err
}

func (d *Directory) SetMetadata() error {
	headers, err := d.fsc.setResourceHeaders(d.buildPath(), compMetadata, resourceDirectory, mergeMDIntoExtraHeaders(d.Metadata, nil))
	if err != nil {
		return err
	}

	d.updateEtagAndLastModified(headers)
	return nil
}

func (d *Directory) updateEtagAndLastModified(headers http.Header) {
	d.Properties.Etag = headers.Get("Etag")
	d.Properties.LastModified = headers.Get("Last-Modified")
}

func (d *Directory) URL() string {
	return d.fsc.client.getEndpoint(fileServiceName, d.buildPath(), url.Values{})
}
