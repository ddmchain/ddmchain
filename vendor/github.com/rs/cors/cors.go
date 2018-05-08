
package cors

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/rs/xhandler"
	"golang.org/x/net/context"
)

type Options struct {

	AllowedOrigins []string

	AllowOriginFunc func(origin string) bool

	AllowedMethods []string

	AllowedHeaders []string

	ExposedHeaders []string

	AllowCredentials bool

	MaxAge int

	OptionsPassthrough bool

	Debug bool
}

type Cors struct {

	Log *log.Logger

	allowedOriginsAll bool

	allowedOrigins []string

	allowedWOrigins []wildcard

	allowOriginFunc func(origin string) bool

	allowedHeadersAll bool

	allowedHeaders []string

	allowedMethods []string

	exposedHeaders    []string
	allowCredentials  bool
	maxAge            int
	optionPassthrough bool
}

func New(options Options) *Cors {
	c := &Cors{
		exposedHeaders:    convert(options.ExposedHeaders, http.CanonicalHeaderKey),
		allowOriginFunc:   options.AllowOriginFunc,
		allowCredentials:  options.AllowCredentials,
		maxAge:            options.MaxAge,
		optionPassthrough: options.OptionsPassthrough,
	}
	if options.Debug {
		c.Log = log.New(os.Stdout, "[cors] ", log.LstdFlags)
	}

	if len(options.AllowedOrigins) == 0 {

		c.allowedOriginsAll = true
	} else {
		c.allowedOrigins = []string{}
		c.allowedWOrigins = []wildcard{}
		for _, origin := range options.AllowedOrigins {

			origin = strings.ToLower(origin)
			if origin == "*" {

				c.allowedOriginsAll = true
				c.allowedOrigins = nil
				c.allowedWOrigins = nil
				break
			} else if i := strings.IndexByte(origin, '*'); i >= 0 {

				w := wildcard{origin[0:i], origin[i+1 : len(origin)]}
				c.allowedWOrigins = append(c.allowedWOrigins, w)
			} else {
				c.allowedOrigins = append(c.allowedOrigins, origin)
			}
		}
	}

	if len(options.AllowedHeaders) == 0 {

		c.allowedHeaders = []string{"Origin", "Accept", "Content-Type"}
	} else {

		c.allowedHeaders = convert(append(options.AllowedHeaders, "Origin"), http.CanonicalHeaderKey)
		for _, h := range options.AllowedHeaders {
			if h == "*" {
				c.allowedHeadersAll = true
				c.allowedHeaders = nil
				break
			}
		}
	}

	if len(options.AllowedMethods) == 0 {

		c.allowedMethods = []string{"GET", "POST"}
	} else {
		c.allowedMethods = convert(options.AllowedMethods, strings.ToUpper)
	}

	return c
}

func Default() *Cors {
	return New(Options{})
}

func (c *Cors) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			c.logf("Handler: Preflight request")
			c.handlePreflight(w, r)

			if c.optionPassthrough {
				h.ServeHTTP(w, r)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		} else {
			c.logf("Handler: Actual request")
			c.handleActualRequest(w, r)
			h.ServeHTTP(w, r)
		}
	})
}

func (c *Cors) HandlerC(h xhandler.HandlerC) xhandler.HandlerC {
	return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			c.logf("Handler: Preflight request")
			c.handlePreflight(w, r)

			if c.optionPassthrough {
				h.ServeHTTPC(ctx, w, r)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		} else {
			c.logf("Handler: Actual request")
			c.handleActualRequest(w, r)
			h.ServeHTTPC(ctx, w, r)
		}
	})
}

func (c *Cors) HandlerFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		c.logf("HandlerFunc: Preflight request")
		c.handlePreflight(w, r)
	} else {
		c.logf("HandlerFunc: Actual request")
		c.handleActualRequest(w, r)
	}
}

func (c *Cors) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.Method == "OPTIONS" {
		c.logf("ServeHTTP: Preflight request")
		c.handlePreflight(w, r)

		if c.optionPassthrough {
			next(w, r)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	} else {
		c.logf("ServeHTTP: Actual request")
		c.handleActualRequest(w, r)
		next(w, r)
	}
}

func (c *Cors) handlePreflight(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()
	origin := r.Header.Get("Origin")

	if r.Method != "OPTIONS" {
		c.logf("  Preflight aborted: %s!=OPTIONS", r.Method)
		return
	}

	headers.Add("Vary", "Origin")
	headers.Add("Vary", "Access-Control-Request-Method")
	headers.Add("Vary", "Access-Control-Request-Headers")

	if origin == "" {
		c.logf("  Preflight aborted: empty origin")
		return
	}
	if !c.isOriginAllowed(origin) {
		c.logf("  Preflight aborted: origin '%s' not allowed", origin)
		return
	}

	reqMethod := r.Header.Get("Access-Control-Request-Method")
	if !c.isMethodAllowed(reqMethod) {
		c.logf("  Preflight aborted: method '%s' not allowed", reqMethod)
		return
	}
	reqHeaders := parseHeaderList(r.Header.Get("Access-Control-Request-Headers"))
	if !c.areHeadersAllowed(reqHeaders) {
		c.logf("  Preflight aborted: headers '%v' not allowed", reqHeaders)
		return
	}
	headers.Set("Access-Control-Allow-Origin", origin)

	headers.Set("Access-Control-Allow-Methods", strings.ToUpper(reqMethod))
	if len(reqHeaders) > 0 {

		headers.Set("Access-Control-Allow-Headers", strings.Join(reqHeaders, ", "))
	}
	if c.allowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
	if c.maxAge > 0 {
		headers.Set("Access-Control-Max-Age", strconv.Itoa(c.maxAge))
	}
	c.logf("  Preflight response headers: %v", headers)
}

func (c *Cors) handleActualRequest(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()
	origin := r.Header.Get("Origin")

	if r.Method == "OPTIONS" {
		c.logf("  Actual request no headers added: method == %s", r.Method)
		return
	}

	headers.Add("Vary", "Origin")
	if origin == "" {
		c.logf("  Actual request no headers added: missing origin")
		return
	}
	if !c.isOriginAllowed(origin) {
		c.logf("  Actual request no headers added: origin '%s' not allowed", origin)
		return
	}

	if !c.isMethodAllowed(r.Method) {
		c.logf("  Actual request no headers added: method '%s' not allowed", r.Method)

		return
	}
	headers.Set("Access-Control-Allow-Origin", origin)
	if len(c.exposedHeaders) > 0 {
		headers.Set("Access-Control-Expose-Headers", strings.Join(c.exposedHeaders, ", "))
	}
	if c.allowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
	c.logf("  Actual response added headers: %v", headers)
}

func (c *Cors) logf(format string, a ...interface{}) {
	if c.Log != nil {
		c.Log.Printf(format, a...)
	}
}

func (c *Cors) isOriginAllowed(origin string) bool {
	if c.allowOriginFunc != nil {
		return c.allowOriginFunc(origin)
	}
	if c.allowedOriginsAll {
		return true
	}
	origin = strings.ToLower(origin)
	for _, o := range c.allowedOrigins {
		if o == origin {
			return true
		}
	}
	for _, w := range c.allowedWOrigins {
		if w.match(origin) {
			return true
		}
	}
	return false
}

func (c *Cors) isMethodAllowed(method string) bool {
	if len(c.allowedMethods) == 0 {

		return false
	}
	method = strings.ToUpper(method)
	if method == "OPTIONS" {

		return true
	}
	for _, m := range c.allowedMethods {
		if m == method {
			return true
		}
	}
	return false
}

func (c *Cors) areHeadersAllowed(requestedHeaders []string) bool {
	if c.allowedHeadersAll || len(requestedHeaders) == 0 {
		return true
	}
	for _, header := range requestedHeaders {
		header = http.CanonicalHeaderKey(header)
		found := false
		for _, h := range c.allowedHeaders {
			if h == header {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
