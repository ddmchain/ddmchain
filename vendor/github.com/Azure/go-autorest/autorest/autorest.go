
package autorest

import (
	"net/http"
	"time"
)

const (

	HeaderLocation = "Location"

	HeaderRetryAfter = "Retry-After"
)

func ResponseHasStatusCode(resp *http.Response, codes ...int) bool {
	return containsInt(codes, resp.StatusCode)
}

func GetLocation(resp *http.Response) string {
	return resp.Header.Get(HeaderLocation)
}

func GetRetryAfter(resp *http.Response, defaultDelay time.Duration) time.Duration {
	retry := resp.Header.Get(HeaderRetryAfter)
	if retry == "" {
		return defaultDelay
	}

	d, err := time.ParseDuration(retry + "s")
	if err != nil {
		return defaultDelay
	}

	return d
}

func NewPollingRequest(resp *http.Response, cancel <-chan struct{}) (*http.Request, error) {
	location := GetLocation(resp)
	if location == "" {
		return nil, NewErrorWithResponse("autorest", "NewPollingRequest", resp, "Location header missing from response that requires polling")
	}

	req, err := Prepare(&http.Request{Cancel: cancel},
		AsGet(),
		WithBaseURL(location))
	if err != nil {
		return nil, NewErrorWithError(err, "autorest", "NewPollingRequest", nil, "Failure creating poll request to %s", location)
	}

	return req, nil
}
