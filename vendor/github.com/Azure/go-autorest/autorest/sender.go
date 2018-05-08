package autorest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"time"
)

type Sender interface {
	Do(*http.Request) (*http.Response, error)
}

type SenderFunc func(*http.Request) (*http.Response, error)

func (sf SenderFunc) Do(r *http.Request) (*http.Response, error) {
	return sf(r)
}

type SendDecorator func(Sender) Sender

func CreateSender(decorators ...SendDecorator) Sender {
	return DecorateSender(&http.Client{}, decorators...)
}

func DecorateSender(s Sender, decorators ...SendDecorator) Sender {
	for _, decorate := range decorators {
		s = decorate(s)
	}
	return s
}

func Send(r *http.Request, decorators ...SendDecorator) (*http.Response, error) {
	return SendWithSender(&http.Client{}, r, decorators...)
}

func SendWithSender(s Sender, r *http.Request, decorators ...SendDecorator) (*http.Response, error) {
	return DecorateSender(s, decorators...).Do(r)
}

func AfterDelay(d time.Duration) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			if !DelayForBackoff(d, 0, r.Cancel) {
				return nil, fmt.Errorf("autorest: AfterDelay canceled before full delay")
			}
			return s.Do(r)
		})
	}
}

func AsIs() SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			return s.Do(r)
		})
	}
}

func DoCloseIfError() SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			resp, err := s.Do(r)
			if err != nil {
				Respond(resp, ByDiscardingBody(), ByClosing())
			}
			return resp, err
		})
	}
}

func DoErrorIfStatusCode(codes ...int) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			resp, err := s.Do(r)
			if err == nil && ResponseHasStatusCode(resp, codes...) {
				err = NewErrorWithResponse("autorest", "DoErrorIfStatusCode", resp, "%v %v failed with %s",
					resp.Request.Method,
					resp.Request.URL,
					resp.Status)
			}
			return resp, err
		})
	}
}

func DoErrorUnlessStatusCode(codes ...int) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			resp, err := s.Do(r)
			if err == nil && !ResponseHasStatusCode(resp, codes...) {
				err = NewErrorWithResponse("autorest", "DoErrorUnlessStatusCode", resp, "%v %v failed with %s",
					resp.Request.Method,
					resp.Request.URL,
					resp.Status)
			}
			return resp, err
		})
	}
}

func DoPollForStatusCodes(duration time.Duration, delay time.Duration, codes ...int) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (resp *http.Response, err error) {
			resp, err = s.Do(r)

			if err == nil && ResponseHasStatusCode(resp, codes...) {
				r, err = NewPollingRequest(resp, r.Cancel)

				for err == nil && ResponseHasStatusCode(resp, codes...) {
					Respond(resp,
						ByDiscardingBody(),
						ByClosing())
					resp, err = SendWithSender(s, r,
						AfterDelay(GetRetryAfter(resp, delay)))
				}
			}

			return resp, err
		})
	}
}

func DoRetryForAttempts(attempts int, backoff time.Duration) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (resp *http.Response, err error) {
			for attempt := 0; attempt < attempts; attempt++ {
				resp, err = s.Do(r)
				if err == nil {
					return resp, err
				}
				DelayForBackoff(backoff, attempt, r.Cancel)
			}
			return resp, err
		})
	}
}

func DoRetryForStatusCodes(attempts int, backoff time.Duration, codes ...int) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (resp *http.Response, err error) {
			b := []byte{}
			if r.Body != nil {
				b, err = ioutil.ReadAll(r.Body)
				if err != nil {
					return resp, err
				}
			}

			attempts++
			for attempt := 0; attempt < attempts; attempt++ {
				r.Body = ioutil.NopCloser(bytes.NewBuffer(b))
				resp, err = s.Do(r)
				if err != nil || !ResponseHasStatusCode(resp, codes...) {
					return resp, err
				}
				DelayForBackoff(backoff, attempt, r.Cancel)
			}
			return resp, err
		})
	}
}

func DoRetryForDuration(d time.Duration, backoff time.Duration) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (resp *http.Response, err error) {
			end := time.Now().Add(d)
			for attempt := 0; time.Now().Before(end); attempt++ {
				resp, err = s.Do(r)
				if err == nil {
					return resp, err
				}
				DelayForBackoff(backoff, attempt, r.Cancel)
			}
			return resp, err
		})
	}
}

func WithLogging(logger *log.Logger) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			logger.Printf("Sending %s %s", r.Method, r.URL)
			resp, err := s.Do(r)
			if err != nil {
				logger.Printf("%s %s received error '%v'", r.Method, r.URL, err)
			} else {
				logger.Printf("%s %s received %s", r.Method, r.URL, resp.Status)
			}
			return resp, err
		})
	}
}

func DelayForBackoff(backoff time.Duration, attempt int, cancel <-chan struct{}) bool {
	select {
	case <-time.After(time.Duration(backoff.Seconds()*math.Pow(2, float64(attempt))) * time.Second):
		return true
	case <-cancel:
		return false
	}
}
