package jaeger

import (
	"context"
	jaegergin "github.com/Arrim/jaeger-gin"
	"github.com/opentracing/opentracing-go"

	"github.com/luraproject/lura/config"
	"github.com/luraproject/lura/proxy"
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

			span, ctx = opentracing.StartSpanFromContext(jaegergin.GetSpanFromContext(ctx), name)
			resp, err := next[0](ctx, req)

			if err != nil {
				if err.Error() != errCtxCanceledMsg {
					span.LogKV("error", err.Error())
				} else {
					span.LogKV("canceled", true)
				}
			}

			span.LogKV("complete", resp != nil && resp.IsComplete)

			span.Finish()

			return resp, err
		}
	}
}

func ProxyFactory(pf proxy.Factory) proxy.FactoryFunc {
	return func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		next, err := pf.New(cfg)
		if err != nil {
			return next, err
		}
		return Middleware("PIPE: " + cfg.Endpoint)(next), nil
	}
}

func BackendFactory(bf proxy.BackendFactory) proxy.BackendFactory {
	return func(cfg *config.Backend) proxy.Proxy {
		return Middleware("BACKEND: " + cfg.URLPattern)(bf(cfg))
	}
}
