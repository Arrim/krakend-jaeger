package jaeger

import (
	"context"
	"github.com/opentracing/opentracing-go"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/proxy"
)

const errCtxCanceledMsg = "context canceled"

func Middleware(name string) proxy.Middleware {
	return func(next ...proxy.Proxy) proxy.Proxy {
		if len(next) > 1 {
			panic(proxy.ErrTooManyProxies)
		}
		if len(next) < 1 {
			panic(proxy.ErrNotEnoughProxies)
		}
		return func(ctx context.Context, req *proxy.Request) (*proxy.Response, error) {
			var span opentracing.Span
			span, ctx = opentracing.StartSpanFromContext(ctx, name, nil)
			resp, err := next[0](ctx, req)

			if err != nil {
				if err.Error() != errCtxCanceledMsg {
					span.SetBaggageItem("error", err.Error())
				} else {
					span.SetBaggageItem("canceled", "true")
				}
			}

			if resp != nil && resp.IsComplete {
				span.SetBaggageItem("complete", "true")
			} else {
				span.SetBaggageItem("complete", "false")
			}

			span.Finish()

			return resp, err
		}
	}
}

func BackendFactory(bf proxy.BackendFactory) proxy.BackendFactory {
	return func(cfg *config.Backend) proxy.Proxy {
		return Middleware("backend-" + cfg.URLPattern)(bf(cfg))
	}
}
