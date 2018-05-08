package xhandler

import (
	"net/http"

	"golang.org/x/net/context"
)

type Chain []func(next HandlerC) HandlerC

func (c *Chain) Add(f ...interface{}) {
	for _, h := range f {
		switch v := h.(type) {
		case func(http.Handler) http.Handler:
			c.Use(v)
		case func(HandlerC) HandlerC:
			c.UseC(v)
		default:
			panic("Adding invalid handler to the middleware chain")
		}
	}
}

func (c *Chain) With(f ...interface{}) *Chain {
	n := make(Chain, len(*c))
	copy(n, *c)
	n.Add(f...)
	return &n
}

func (c *Chain) UseC(f func(next HandlerC) HandlerC) {
	*c = append(*c, f)
}

func (c *Chain) Use(f func(next http.Handler) http.Handler) {
	xf := func(next HandlerC) HandlerC {
		return HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			n := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTPC(ctx, w, r)
			})
			f(n).ServeHTTP(w, r)
		})
	}
	*c = append(*c, xf)
}

func (c Chain) Handler(xh HandlerC) http.Handler {
	ctx := context.Background()
	return c.HandlerCtx(ctx, xh)
}

func (c Chain) HandlerFC(xhf HandlerFuncC) http.Handler {
	ctx := context.Background()
	return c.HandlerCtx(ctx, HandlerFuncC(xhf))
}

func (c Chain) HandlerH(h http.Handler) http.Handler {
	ctx := context.Background()
	return c.HandlerCtx(ctx, HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}))
}

func (c Chain) HandlerF(hf http.HandlerFunc) http.Handler {
	ctx := context.Background()
	return c.HandlerCtx(ctx, HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		hf(w, r)
	}))
}

func (c Chain) HandlerCtx(ctx context.Context, xh HandlerC) http.Handler {
	return New(ctx, c.HandlerC(xh))
}

func (c Chain) HandlerC(xh HandlerC) HandlerC {
	for i := len(c) - 1; i >= 0; i-- {
		xh = c[i](xh)
	}
	return xh
}

func (c Chain) HandlerCF(xhc HandlerFuncC) HandlerC {
	return c.HandlerC(HandlerFuncC(xhc))
}
