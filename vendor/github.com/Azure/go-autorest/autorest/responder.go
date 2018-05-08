package autorest

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type Responder interface {
	Respond(*http.Response) error
}

type ResponderFunc func(*http.Response) error

func (rf ResponderFunc) Respond(r *http.Response) error {
	return rf(r)
}

type RespondDecorator func(Responder) Responder

func CreateResponder(decorators ...RespondDecorator) Responder {
	return DecorateResponder(
		Responder(ResponderFunc(func(r *http.Response) error { return nil })),
		decorators...)
}

func DecorateResponder(r Responder, decorators ...RespondDecorator) Responder {
	for _, decorate := range decorators {
		r = decorate(r)
	}
	return r
}

func Respond(r *http.Response, decorators ...RespondDecorator) error {
	if r == nil {
		return nil
	}
	return CreateResponder(decorators...).Respond(r)
}

func ByIgnoring() RespondDecorator {
	return func(r Responder) Responder {
		return ResponderFunc(func(resp *http.Response) error {
			return r.Respond(resp)
		})
	}
}

func ByCopying(b *bytes.Buffer) RespondDecorator {
	return func(r Responder) Responder {
		return ResponderFunc(func(resp *http.Response) error {
			err := r.Respond(resp)
			if err == nil && resp != nil && resp.Body != nil {
				resp.Body = TeeReadCloser(resp.Body, b)
			}
			return err
		})
	}
}

func ByDiscardingBody() RespondDecorator {
	return func(r Responder) Responder {
		return ResponderFunc(func(resp *http.Response) error {
			err := r.Respond(resp)
			if err == nil && resp != nil && resp.Body != nil {
				if _, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
					return fmt.Errorf("Error discarding the response body: %v", err)
				}
			}
			return err
		})
	}
}

func ByClosing() RespondDecorator {
	return func(r Responder) Responder {
		return ResponderFunc(func(resp *http.Response) error {
			err := r.Respond(resp)
			if resp != nil && resp.Body != nil {
				if err := resp.Body.Close(); err != nil {
					return fmt.Errorf("Error closing the response body: %v", err)
				}
			}
			return err
		})
	}
}

func ByClosingIfError() RespondDecorator {
	return func(r Responder) Responder {
		return ResponderFunc(func(resp *http.Response) error {
			err := r.Respond(resp)
			if err != nil && resp != nil && resp.Body != nil {
				if err := resp.Body.Close(); err != nil {
					return fmt.Errorf("Error closing the response body: %v", err)
				}
			}
			return err
		})
	}
}

func ByUnmarshallingJSON(v interface{}) RespondDecorator {
	return func(r Responder) Responder {
		return ResponderFunc(func(resp *http.Response) error {
			err := r.Respond(resp)
			if err == nil {
				b, errInner := ioutil.ReadAll(resp.Body)

				b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))
				if errInner != nil {
					err = fmt.Errorf("Error occurred reading http.Response#Body - Error = '%v'", errInner)
				} else if len(strings.Trim(string(b), " ")) > 0 {
					errInner = json.Unmarshal(b, v)
					if errInner != nil {
						err = fmt.Errorf("Error occurred unmarshalling JSON - Error = '%v' JSON = '%s'", errInner, string(b))
					}
				}
			}
			return err
		})
	}
}

func ByUnmarshallingXML(v interface{}) RespondDecorator {
	return func(r Responder) Responder {
		return ResponderFunc(func(resp *http.Response) error {
			err := r.Respond(resp)
			if err == nil {
				b, errInner := ioutil.ReadAll(resp.Body)
				if errInner != nil {
					err = fmt.Errorf("Error occurred reading http.Response#Body - Error = '%v'", errInner)
				} else {
					errInner = xml.Unmarshal(b, v)
					if errInner != nil {
						err = fmt.Errorf("Error occurred unmarshalling Xml - Error = '%v' Xml = '%s'", errInner, string(b))
					}
				}
			}
			return err
		})
	}
}

func WithErrorUnlessStatusCode(codes ...int) RespondDecorator {
	return func(r Responder) Responder {
		return ResponderFunc(func(resp *http.Response) error {
			err := r.Respond(resp)
			if err == nil && !ResponseHasStatusCode(resp, codes...) {
				derr := NewErrorWithResponse("autorest", "WithErrorUnlessStatusCode", resp, "%v %v failed with %s",
					resp.Request.Method,
					resp.Request.URL,
					resp.Status)
				if resp.Body != nil {
					defer resp.Body.Close()
					b, _ := ioutil.ReadAll(resp.Body)
					derr.ServiceError = b
					resp.Body = ioutil.NopCloser(bytes.NewReader(b))
				}
				err = derr
			}
			return err
		})
	}
}

func WithErrorUnlessOK() RespondDecorator {
	return WithErrorUnlessStatusCode(http.StatusOK)
}

func ExtractHeader(header string, resp *http.Response) []string {
	if resp != nil && resp.Header != nil {
		return resp.Header[http.CanonicalHeaderKey(header)]
	}
	return nil
}

func ExtractHeaderValue(header string, resp *http.Response) string {
	h := ExtractHeader(header, resp)
	if len(h) > 0 {
		return h[0]
	}
	return ""
}
