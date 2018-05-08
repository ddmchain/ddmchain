package xhandler

import (
	"net/http"
	"time"

	"golang.org/x/net/context"
)

func CloseHandler(next HandlerC) HandlerC {
	return HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {

		if wcn, ok := w.(http.CloseNotifier); ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			defer cancel()

			notify := wcn.CloseNotify()
			go func() {
				select {
				case <-notify:
					cancel()
				case <-ctx.Done():
				}
			}()
		}

		next.ServeHTTPC(ctx, w, r)
	})
}

func TimeoutHandler(timeout time.Duration) func(next HandlerC) HandlerC {
	return func(next HandlerC) HandlerC {
		return HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			ctx, _ = context.WithTimeout(ctx, timeout)
			next.ServeHTTPC(ctx, w, r)
		})
	}
}

func If(cond func(ctx context.Context, w http.ResponseWriter, r *http.Request) bool, condNext func(next HandlerC) HandlerC) func(next HandlerC) HandlerC {
	return func(next HandlerC) HandlerC {
		return HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			if cond(ctx, w, r) {
				condNext(next).ServeHTTPC(ctx, w, r)
			} else {
				next.ServeHTTPC(ctx, w, r)
			}
		})
	}
}
