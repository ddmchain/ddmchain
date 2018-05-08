package autorest

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"runtime"
	"time"
)

const (

	DefaultPollingDelay = 60 * time.Second

	DefaultPollingDuration = 15 * time.Minute

	DefaultRetryAttempts = 3
)

var (

	defaultUserAgent = fmt.Sprintf("Go/%s (%s-%s) go-autorest/%s",
		runtime.Version(),
		runtime.GOARCH,
		runtime.GOOS,
		Version(),
	)

	statusCodesForRetry = []int{
		http.StatusRequestTimeout,      
		http.StatusInternalServerError, 
		http.StatusBadGateway,          
		http.StatusServiceUnavailable,  
		http.StatusGatewayTimeout,      
	}
)

const (
	requestFormat = `HTTP Request Begin ===================================================
%s
===================================================== HTTP Request End
`
	responseFormat = `HTTP Response Begin ===================================================
%s
===================================================== HTTP Response End
`
)

type Response struct {
	*http.Response `json:"-"`
}

type LoggingInspector struct {
	Logger *log.Logger
}

func (li LoggingInspector) WithInspection() PrepareDecorator {
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			var body, b bytes.Buffer

			defer r.Body.Close()

			r.Body = ioutil.NopCloser(io.TeeReader(r.Body, &body))
			if err := r.Write(&b); err != nil {
				return nil, fmt.Errorf("Failed to write response: %v", err)
			}

			li.Logger.Printf(requestFormat, b.String())

			r.Body = ioutil.NopCloser(&body)
			return p.Prepare(r)
		})
	}
}

func (li LoggingInspector) ByInspecting() RespondDecorator {
	return func(r Responder) Responder {
		return ResponderFunc(func(resp *http.Response) error {
			var body, b bytes.Buffer
			defer resp.Body.Close()
			resp.Body = ioutil.NopCloser(io.TeeReader(resp.Body, &body))
			if err := resp.Write(&b); err != nil {
				return fmt.Errorf("Failed to write response: %v", err)
			}

			li.Logger.Printf(responseFormat, b.String())

			resp.Body = ioutil.NopCloser(&body)
			return r.Respond(resp)
		})
	}
}

type Client struct {
	Authorizer        Authorizer
	Sender            Sender
	RequestInspector  PrepareDecorator
	ResponseInspector RespondDecorator

	PollingDelay time.Duration

	PollingDuration time.Duration

	RetryAttempts int

	RetryDuration time.Duration

	UserAgent string

	Jar http.CookieJar
}

func NewClientWithUserAgent(ua string) Client {
	c := Client{
		PollingDelay:    DefaultPollingDelay,
		PollingDuration: DefaultPollingDuration,
		RetryAttempts:   DefaultRetryAttempts,
		RetryDuration:   30 * time.Second,
		UserAgent:       defaultUserAgent,
	}
	c.AddToUserAgent(ua)
	return c
}

func (c *Client) AddToUserAgent(extension string) error {
	if extension != "" {
		c.UserAgent = fmt.Sprintf("%s %s", c.UserAgent, extension)
		return nil
	}
	return fmt.Errorf("Extension was empty, User Agent stayed as %s", c.UserAgent)
}

func (c Client) Do(r *http.Request) (*http.Response, error) {
	if r.UserAgent() == "" {
		r, _ = Prepare(r,
			WithUserAgent(c.UserAgent))
	}
	r, err := Prepare(r,
		c.WithInspection(),
		c.WithAuthorization())
	if err != nil {
		return nil, NewErrorWithError(err, "autorest/Client", "Do", nil, "Preparing request failed")
	}
	resp, err := SendWithSender(c.sender(), r,
		DoRetryForStatusCodes(c.RetryAttempts, c.RetryDuration, statusCodesForRetry...))
	Respond(resp,
		c.ByInspecting())
	return resp, err
}

func (c Client) sender() Sender {
	if c.Sender == nil {
		j, _ := cookiejar.New(nil)
		return &http.Client{Jar: j}
	}
	return c.Sender
}

func (c Client) WithAuthorization() PrepareDecorator {
	return c.authorizer().WithAuthorization()
}

func (c Client) authorizer() Authorizer {
	if c.Authorizer == nil {
		return NullAuthorizer{}
	}
	return c.Authorizer
}

func (c Client) WithInspection() PrepareDecorator {
	if c.RequestInspector == nil {
		return WithNothing()
	}
	return c.RequestInspector
}

func (c Client) ByInspecting() RespondDecorator {
	if c.ResponseInspector == nil {
		return ByIgnoring()
	}
	return c.ResponseInspector
}
