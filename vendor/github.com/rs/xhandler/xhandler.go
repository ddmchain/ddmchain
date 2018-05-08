
package xhandler 

import (
	"net/http"

	"golang.org/x/net/context"
)

type HandlerC interface {
	ServeHTTPC(context.Context, http.ResponseWriter, *http.Request)
}

type HandlerFuncC func(context.Context, http.ResponseWriter, *http.Request)

func (f HandlerFuncC) ServeHTTPC(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	f(ctx, w, r)
}

func New(ctx context.Context, h HandlerC) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTPC(ctx, w, r)
	})
}
