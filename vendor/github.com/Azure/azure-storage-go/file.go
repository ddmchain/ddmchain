package storage

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

const fourMB = uint64(4194304)
const oneTB = uint64(1099511627776)

type File struct {
	fsc                *FileServiceClient
	Metadata           map[string]string
	Name               string `xml:"Name"`
	parent             *Directory
	Properties         FileProperties `xml:"Properties"`
	share              *Share
	FileCopyProperties FileCopyState
}

type FileProperties struct {
	CacheControl string `header:"x-ms-cache-control"`
	Disposition  string `header:"x-ms-content-disposition"`
	Encoding     string `header:"x-ms-content-encoding"`
	Etag         string
	Language     string `header:"x-ms-content-language"`
	LastModified string
	Length       uint64 `xml:"Content-Length"`
	MD5          string `header:"x-ms-content-md5"`
	Type         string `header:"x-ms-content-type"`
}

type FileCopyState struct {
	CompletionTime string
	ID             string `header:"x-ms-copy-id"`
	Progress       string
	Source         string
	Status         string `header:"x-ms-copy-status"`
	StatusDesc     string
}

type FileStream struct {
	Body       io.ReadCloser
	ContentMD5 string
}

type FileRequestOptions struct {
	Timeout uint 
}

func (p FileRequestOptions) getParameters() url.Values {
	out := url.Values{}

	if p.Timeout != 0 {
		out.Set("timeout", fmt.Sprintf("%v", p.Timeout))
	}

	return out
}

type FileRanges struct {
	ContentLength uint64
	LastModified  string
	ETag          string
	FileRanges    []FileRange `xml:"Range"`
}

type FileRange struct {
	Start uint64 `xml:"Start"`
	End   uint64 `xml:"End"`
}

func (fr FileRange) String() string {
	return fmt.Sprintf("bytes=%d-%d", fr.Start, fr.End)
}

func (f *File) buildPath() string {
	return f.parent.buildPath() + "/" + f.Name
}

func (f *File) ClearRange(fileRange FileRange) error {
	headers, err := f.modifyRange(nil, fileRange, nil)
	if err != nil {
		return err
	}

	f.updateEtagAndLastModified(headers)
	return nil
}

func (f *File) Create(maxSize uint64) error {
	if maxSize > oneTB {
		return fmt.Errorf("max file size is 1TB")
	}

	extraHeaders := map[string]string{
		"x-ms-content-length": strconv.FormatUint(maxSize, 10),
		"x-ms-type":           "file",
	}

	headers, err := f.fsc.createResource(f.buildPath(), resourceFile, nil, mergeMDIntoExtraHeaders(f.Metadata, extraHeaders), []int{http.StatusCreated})
	if err != nil {
		return err
	}

	f.Properties.Length = maxSize
	f.updateEtagAndLastModified(headers)
	return nil
}

func (f *File) CopyFile(sourceURL string, options *FileRequestOptions) error {
	extraHeaders := map[string]string{
		"x-ms-type":        "file",
		"x-ms-copy-source": sourceURL,
	}

	var parameters url.Values
	if options != nil {
		parameters = options.getParameters()
	}

	headers, err := f.fsc.createResource(f.buildPath(), resourceFile, parameters, mergeMDIntoExtraHeaders(f.Metadata, extraHeaders), []int{http.StatusAccepted})
	if err != nil {
		return err
	}

	f.updateEtagLastModifiedAndCopyHeaders(headers)
	return nil
}

func (f *File) Delete() error {
	return f.fsc.deleteResource(f.buildPath(), resourceFile)
}

func (f *File) DeleteIfExists() (bool, error) {
	resp, err := f.fsc.deleteResourceNoClose(f.buildPath(), resourceFile)
	if resp != nil {
		defer readAndCloseBody(resp.body)
		if resp.statusCode == http.StatusAccepted || resp.statusCode == http.StatusNotFound {
			return resp.statusCode == http.StatusAccepted, nil
		}
	}
	return false, err
}

func (f *File) DownloadRangeToStream(fileRange FileRange, getContentMD5 bool) (fs FileStream, err error) {
	if getContentMD5 && isRangeTooBig(fileRange) {
		return fs, fmt.Errorf("must specify a range less than or equal to 4MB when getContentMD5 is true")
	}

	extraHeaders := map[string]string{
		"Range": fileRange.String(),
	}
	if getContentMD5 == true {
		extraHeaders["x-ms-range-get-content-md5"] = "true"
	}

	resp, err := f.fsc.getResourceNoClose(f.buildPath(), compNone, resourceFile, http.MethodGet, extraHeaders)
	if err != nil {
		return fs, err
	}

	if err = checkRespCode(resp.statusCode, []int{http.StatusOK, http.StatusPartialContent}); err != nil {
		resp.body.Close()
		return fs, err
	}

	fs.Body = resp.body
	if getContentMD5 {
		fs.ContentMD5 = resp.headers.Get("Content-MD5")
	}
	return fs, nil
}

func (f *File) Exists() (bool, error) {
	exists, headers, err := f.fsc.resourceExists(f.buildPath(), resourceFile)
	if exists {
		f.updateEtagAndLastModified(headers)
		f.updateProperties(headers)
	}
	return exists, err
}

func (f *File) FetchAttributes() error {
	headers, err := f.fsc.getResourceHeaders(f.buildPath(), compNone, resourceFile, http.MethodHead)
	if err != nil {
		return err
	}

	f.updateEtagAndLastModified(headers)
	f.updateProperties(headers)
	f.Metadata = getMetadataFromHeaders(headers)
	return nil
}

func isRangeTooBig(fileRange FileRange) bool {
	if fileRange.End-fileRange.Start > fourMB {
		return true
	}

	return false
}

func (f *File) ListRanges(listRange *FileRange) (*FileRanges, error) {
	params := url.Values{"comp": {"rangelist"}}

	var headers map[string]string
	if listRange != nil {
		headers = make(map[string]string)
		headers["Range"] = listRange.String()
	}

	resp, err := f.fsc.listContent(f.buildPath(), params, headers)
	if err != nil {
		return nil, err
	}

	defer resp.body.Close()
	var cl uint64
	cl, err = strconv.ParseUint(resp.headers.Get("x-ms-content-length"), 10, 64)
	if err != nil {
		ioutil.ReadAll(resp.body)
		return nil, err
	}

	var out FileRanges
	out.ContentLength = cl
	out.ETag = resp.headers.Get("ETag")
	out.LastModified = resp.headers.Get("Last-Modified")

	err = xmlUnmarshal(resp.body, &out)
	return &out, err
}

func (f *File) modifyRange(bytes io.Reader, fileRange FileRange, contentMD5 *string) (http.Header, error) {
	if err := f.fsc.checkForStorageEmulator(); err != nil {
		return nil, err
	}
	if fileRange.End < fileRange.Start {
		return nil, errors.New("the value for rangeEnd must be greater than or equal to rangeStart")
	}
	if bytes != nil && isRangeTooBig(fileRange) {
		return nil, errors.New("range cannot exceed 4MB in size")
	}

	uri := f.fsc.client.getEndpoint(fileServiceName, f.buildPath(), url.Values{"comp": {"range"}})

	write := "clear"
	cl := uint64(0)

	if bytes != nil {
		write = "update"
		cl = (fileRange.End - fileRange.Start) + 1
	}

	extraHeaders := map[string]string{
		"Content-Length": strconv.FormatUint(cl, 10),
		"Range":          fileRange.String(),
		"x-ms-write":     write,
	}

	if contentMD5 != nil {
		extraHeaders["Content-MD5"] = *contentMD5
	}

	headers := mergeHeaders(f.fsc.client.getStandardHeaders(), extraHeaders)
	resp, err := f.fsc.client.exec(http.MethodPut, uri, headers, bytes, f.fsc.auth)
	if err != nil {
		return nil, err
	}
	defer readAndCloseBody(resp.body)
	return resp.headers, checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

func (f *File) SetMetadata() error {
	headers, err := f.fsc.setResourceHeaders(f.buildPath(), compMetadata, resourceFile, mergeMDIntoExtraHeaders(f.Metadata, nil))
	if err != nil {
		return err
	}

	f.updateEtagAndLastModified(headers)
	return nil
}

func (f *File) SetProperties() error {
	headers, err := f.fsc.setResourceHeaders(f.buildPath(), compProperties, resourceFile, headersFromStruct(f.Properties))
	if err != nil {
		return err
	}

	f.updateEtagAndLastModified(headers)
	return nil
}

func (f *File) updateEtagAndLastModified(headers http.Header) {
	f.Properties.Etag = headers.Get("Etag")
	f.Properties.LastModified = headers.Get("Last-Modified")
}

func (f *File) updateEtagLastModifiedAndCopyHeaders(headers http.Header) {
	f.Properties.Etag = headers.Get("Etag")
	f.Properties.LastModified = headers.Get("Last-Modified")
	f.FileCopyProperties.ID = headers.Get("X-Ms-Copy-Id")
	f.FileCopyProperties.Status = headers.Get("X-Ms-Copy-Status")
}

func (f *File) updateProperties(header http.Header) {
	size, err := strconv.ParseUint(header.Get("Content-Length"), 10, 64)
	if err == nil {
		f.Properties.Length = size
	}

	f.updateEtagAndLastModified(header)
	f.Properties.CacheControl = header.Get("Cache-Control")
	f.Properties.Disposition = header.Get("Content-Disposition")
	f.Properties.Encoding = header.Get("Content-Encoding")
	f.Properties.Language = header.Get("Content-Language")
	f.Properties.MD5 = header.Get("Content-MD5")
	f.Properties.Type = header.Get("Content-Type")
}

func (f *File) URL() string {
	return f.fsc.client.getEndpoint(fileServiceName, f.buildPath(), url.Values{})
}

func (f *File) WriteRange(bytes io.Reader, fileRange FileRange, contentMD5 *string) error {
	if bytes == nil {
		return errors.New("bytes cannot be nil")
	}

	headers, err := f.modifyRange(bytes, fileRange, contentMD5)
	if err != nil {
		return err
	}

	f.updateEtagAndLastModified(headers)
	return nil
}
