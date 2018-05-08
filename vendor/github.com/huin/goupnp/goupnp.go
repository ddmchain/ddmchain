
package goupnp

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/html/charset"

	"github.com/huin/goupnp/httpu"
	"github.com/huin/goupnp/ssdp"
)

type ContextError struct {
	Context string
	Err     error
}

func (err ContextError) Error() string {
	return fmt.Sprintf("%s: %v", err.Context, err.Err)
}

type MaybeRootDevice struct {

	Root *RootDevice

	Location *url.URL

	Err error
}

func DiscoverDevices(searchTarget string) ([]MaybeRootDevice, error) {
	httpu, err := httpu.NewHTTPUClient()
	if err != nil {
		return nil, err
	}
	defer httpu.Close()
	responses, err := ssdp.SSDPRawSearch(httpu, string(searchTarget), 2, 3)
	if err != nil {
		return nil, err
	}

	results := make([]MaybeRootDevice, len(responses))
	for i, response := range responses {
		maybe := &results[i]
		loc, err := response.Location()
		if err != nil {
			maybe.Err = ContextError{"unexpected bad location from search", err}
			continue
		}
		maybe.Location = loc
		if root, err := DeviceByURL(loc); err != nil {
			maybe.Err = err
		} else {
			maybe.Root = root
		}
	}

	return results, nil
}

func DeviceByURL(loc *url.URL) (*RootDevice, error) {
	locStr := loc.String()
	root := new(RootDevice)
	if err := requestXml(locStr, DeviceXMLNamespace, root); err != nil {
		return nil, ContextError{fmt.Sprintf("error requesting root device details from %q", locStr), err}
	}
	var urlBaseStr string
	if root.URLBaseStr != "" {
		urlBaseStr = root.URLBaseStr
	} else {
		urlBaseStr = locStr
	}
	urlBase, err := url.Parse(urlBaseStr)
	if err != nil {
		return nil, ContextError{fmt.Sprintf("error parsing location URL %q", locStr), err}
	}
	root.SetURLBase(urlBase)
	return root, nil
}

func requestXml(url string, defaultSpace string, doc interface{}) error {
	timeout := time.Duration(3 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("goupnp: got response status %s from %q",
			resp.Status, url)
	}

	decoder := xml.NewDecoder(resp.Body)
	decoder.DefaultSpace = defaultSpace
	decoder.CharsetReader = charset.NewReaderLabel

	return decoder.Decode(doc)
}
