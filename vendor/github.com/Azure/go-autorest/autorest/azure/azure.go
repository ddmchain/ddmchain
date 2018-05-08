
package azure

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/Azure/go-autorest/autorest"
)

const (

	HeaderClientID = "x-ms-client-request-id"

	HeaderReturnClientID = "x-ms-return-client-request-id"

	HeaderRequestID = "x-ms-request-id"
)

type ServiceError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details *[]interface{} `json:"details"`
}

func (se ServiceError) Error() string {
	if se.Details != nil {
		d, err := json.Marshal(*(se.Details))
		if err != nil {
			return fmt.Sprintf("Code=%q Message=%q Details=%v", se.Code, se.Message, *se.Details)
		}
		return fmt.Sprintf("Code=%q Message=%q Details=%v", se.Code, se.Message, string(d))
	}
	return fmt.Sprintf("Code=%q Message=%q", se.Code, se.Message)
}

type RequestError struct {
	autorest.DetailedError

	ServiceError *ServiceError `json:"error"`

	RequestID string
}

func (e RequestError) Error() string {
	return fmt.Sprintf("autorest/azure: Service returned an error. Status=%v %v",
		e.StatusCode, e.ServiceError)
}

func IsAzureError(e error) bool {
	_, ok := e.(*RequestError)
	return ok
}

func NewErrorWithError(original error, packageType string, method string, resp *http.Response, message string, args ...interface{}) RequestError {
	if v, ok := original.(*RequestError); ok {
		return *v
	}

	statusCode := autorest.UndefinedStatusCode
	if resp != nil {
		statusCode = resp.StatusCode
	}
	return RequestError{
		DetailedError: autorest.DetailedError{
			Original:    original,
			PackageType: packageType,
			Method:      method,
			StatusCode:  statusCode,
			Message:     fmt.Sprintf(message, args...),
		},
	}
}

func WithReturningClientID(uuid string) autorest.PrepareDecorator {
	preparer := autorest.CreatePreparer(
		WithClientID(uuid),
		WithReturnClientID(true))

	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err != nil {
				return r, err
			}
			return preparer.Prepare(r)
		})
	}
}

func WithClientID(uuid string) autorest.PrepareDecorator {
	return autorest.WithHeader(HeaderClientID, uuid)
}

func WithReturnClientID(b bool) autorest.PrepareDecorator {
	return autorest.WithHeader(HeaderReturnClientID, strconv.FormatBool(b))
}

func ExtractClientID(resp *http.Response) string {
	return autorest.ExtractHeaderValue(HeaderClientID, resp)
}

func ExtractRequestID(resp *http.Response) string {
	return autorest.ExtractHeaderValue(HeaderRequestID, resp)
}

func WithErrorUnlessStatusCode(codes ...int) autorest.RespondDecorator {
	return func(r autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(resp *http.Response) error {
			err := r.Respond(resp)
			if err == nil && !autorest.ResponseHasStatusCode(resp, codes...) {
				var e RequestError
				defer resp.Body.Close()

				b, decodeErr := autorest.CopyAndDecode(autorest.EncodedAsJSON, resp.Body, &e)
				resp.Body = ioutil.NopCloser(&b)
				if decodeErr != nil {
					return fmt.Errorf("autorest/azure: error response cannot be parsed: %q error: %v", b.String(), decodeErr)
				} else if e.ServiceError == nil {
					e.ServiceError = &ServiceError{Code: "Unknown", Message: "Unknown service error"}
				}

				e.RequestID = ExtractRequestID(resp)
				if e.StatusCode == nil {
					e.StatusCode = resp.StatusCode
				}
				err = &e
			}
			return err
		})
	}
}
