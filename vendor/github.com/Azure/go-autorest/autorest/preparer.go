package autorest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

const (
	mimeTypeJSON     = "application/json"
	mimeTypeFormPost = "application/x-www-form-urlencoded"

	headerAuthorization = "Authorization"
	headerContentType   = "Content-Type"
	headerUserAgent     = "User-Agent"
)

type Preparer interface {
	Prepare(*http.Request) (*http.Request, error)
}

type PreparerFunc func(*http.Request) (*http.Request, error)

func (pf PreparerFunc) Prepare(r *http.Request) (*http.Request, error) {
	return pf(r)
}

type PrepareDecorator func(Preparer) Preparer

func CreatePreparer(decorators ...PrepareDecorator) Preparer {
	return DecoratePreparer(
		Preparer(PreparerFunc(func(r *http.Request) (*http.Request, error) { return r, nil })),
		decorators...)
}

func DecoratePreparer(p Preparer, decorators ...PrepareDecorator) Preparer {
	for _, decorate := range decorators {
		p = decorate(p)
	}
	return p
}

func Prepare(r *http.Request, decorators ...PrepareDecorator) (*http.Request, error) {
	if r == nil {
		return nil, NewError("autorest", "Prepare", "Invoked without an http.Request")
	}
	return CreatePreparer(decorators...).Prepare(r)
}

func WithNothing() PrepareDecorator {
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			return p.Prepare(r)
		})
	}
}

func WithHeader(header string, value string) PrepareDecorator {
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				if r.Header == nil {
					r.Header = make(http.Header)
				}
				r.Header.Set(http.CanonicalHeaderKey(header), value)
			}
			return r, err
		})
	}
}

func WithBearerAuthorization(token string) PrepareDecorator {
	return WithHeader(headerAuthorization, fmt.Sprintf("Bearer %s", token))
}

func AsContentType(contentType string) PrepareDecorator {
	return WithHeader(headerContentType, contentType)
}

func WithUserAgent(ua string) PrepareDecorator {
	return WithHeader(headerUserAgent, ua)
}

func AsFormURLEncoded() PrepareDecorator {
	return AsContentType(mimeTypeFormPost)
}

func AsJSON() PrepareDecorator {
	return AsContentType(mimeTypeJSON)
}

func WithMethod(method string) PrepareDecorator {
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r.Method = method
			return p.Prepare(r)
		})
	}
}

func AsDelete() PrepareDecorator { return WithMethod("DELETE") }

func AsGet() PrepareDecorator { return WithMethod("GET") }

func AsHead() PrepareDecorator { return WithMethod("HEAD") }

func AsOptions() PrepareDecorator { return WithMethod("OPTIONS") }

func AsPatch() PrepareDecorator { return WithMethod("PATCH") }

func AsPost() PrepareDecorator { return WithMethod("POST") }

func AsPut() PrepareDecorator { return WithMethod("PUT") }

func WithBaseURL(baseURL string) PrepareDecorator {
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				var u *url.URL
				if u, err = url.Parse(baseURL); err != nil {
					return r, err
				}
				if u.Scheme == "" {
					err = fmt.Errorf("autorest: No scheme detected in URL %s", baseURL)
				}
				if err == nil {
					r.URL = u
				}
			}
			return r, err
		})
	}
}

func WithCustomBaseURL(baseURL string, urlParameters map[string]interface{}) PrepareDecorator {
	parameters := ensureValueStrings(urlParameters)
	for key, value := range parameters {
		baseURL = strings.Replace(baseURL, "{"+key+"}", value, -1)
	}
	return WithBaseURL(baseURL)
}

func WithFormData(v url.Values) PrepareDecorator {
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				s := v.Encode()
				r.ContentLength = int64(len(s))
				r.Body = ioutil.NopCloser(strings.NewReader(s))
			}
			return r, err
		})
	}
}

func WithMultiPartFormData(formDataParameters map[string]interface{}) PrepareDecorator {
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				for key, value := range formDataParameters {
					if rc, ok := value.(io.ReadCloser); ok {
						var fd io.Writer
						if fd, err = writer.CreateFormFile(key, key); err != nil {
							return r, err
						}
						if _, err = io.Copy(fd, rc); err != nil {
							return r, err
						}
					} else {
						if err = writer.WriteField(key, ensureValueString(value)); err != nil {
							return r, err
						}
					}
				}
				if err = writer.Close(); err != nil {
					return r, err
				}
				if r.Header == nil {
					r.Header = make(http.Header)
				}
				r.Header.Set(http.CanonicalHeaderKey(headerContentType), writer.FormDataContentType())
				r.Body = ioutil.NopCloser(bytes.NewReader(body.Bytes()))
				r.ContentLength = int64(body.Len())
				return r, err
			}
			return r, err
		})
	}
}

func WithFile(f io.ReadCloser) PrepareDecorator {
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				b, err := ioutil.ReadAll(f)
				if err != nil {
					return r, err
				}
				r.Body = ioutil.NopCloser(bytes.NewReader(b))
				r.ContentLength = int64(len(b))
			}
			return r, err
		})
	}
}

func WithBool(v bool) PrepareDecorator {
	return WithString(fmt.Sprintf("%v", v))
}

func WithFloat32(v float32) PrepareDecorator {
	return WithString(fmt.Sprintf("%v", v))
}

func WithFloat64(v float64) PrepareDecorator {
	return WithString(fmt.Sprintf("%v", v))
}

func WithInt32(v int32) PrepareDecorator {
	return WithString(fmt.Sprintf("%v", v))
}

func WithInt64(v int64) PrepareDecorator {
	return WithString(fmt.Sprintf("%v", v))
}

func WithString(v string) PrepareDecorator {
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				r.ContentLength = int64(len(v))
				r.Body = ioutil.NopCloser(strings.NewReader(v))
			}
			return r, err
		})
	}
}

func WithJSON(v interface{}) PrepareDecorator {
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				b, err := json.Marshal(v)
				if err == nil {
					r.ContentLength = int64(len(b))
					r.Body = ioutil.NopCloser(bytes.NewReader(b))
				}
			}
			return r, err
		})
	}
}

func WithPath(path string) PrepareDecorator {
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				if r.URL == nil {
					return r, NewError("autorest", "WithPath", "Invoked with a nil URL")
				}
				if r.URL, err = parseURL(r.URL, path); err != nil {
					return r, err
				}
			}
			return r, err
		})
	}
}

func WithEscapedPathParameters(path string, pathParameters map[string]interface{}) PrepareDecorator {
	parameters := escapeValueStrings(ensureValueStrings(pathParameters))
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				if r.URL == nil {
					return r, NewError("autorest", "WithEscapedPathParameters", "Invoked with a nil URL")
				}
				for key, value := range parameters {
					path = strings.Replace(path, "{"+key+"}", value, -1)
				}
				if r.URL, err = parseURL(r.URL, path); err != nil {
					return r, err
				}
			}
			return r, err
		})
	}
}

func WithPathParameters(path string, pathParameters map[string]interface{}) PrepareDecorator {
	parameters := ensureValueStrings(pathParameters)
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				if r.URL == nil {
					return r, NewError("autorest", "WithPathParameters", "Invoked with a nil URL")
				}
				for key, value := range parameters {
					path = strings.Replace(path, "{"+key+"}", value, -1)
				}

				if r.URL, err = parseURL(r.URL, path); err != nil {
					return r, err
				}
			}
			return r, err
		})
	}
}

func parseURL(u *url.URL, path string) (*url.URL, error) {
	p := strings.TrimRight(u.String(), "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return url.Parse(p + path)
}

func WithQueryParameters(queryParameters map[string]interface{}) PrepareDecorator {
	parameters := ensureValueStrings(queryParameters)
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				if r.URL == nil {
					return r, NewError("autorest", "WithQueryParameters", "Invoked with a nil URL")
				}
				v := r.URL.Query()
				for key, value := range parameters {
					v.Add(key, value)
				}
				r.URL.RawQuery = createQuery(v)
			}
			return r, err
		})
	}
}

type Authorizer interface {
	WithAuthorization() PrepareDecorator
}

type NullAuthorizer struct{}

func (na NullAuthorizer) WithAuthorization() PrepareDecorator {
	return WithNothing()
}
