package gin

import (
	"net/http"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/proxy"
	krakendgin "github.com/devopsfaith/krakend/router/gin"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	jaegergin "github.com/rekamarket/jaeger-gin"
)

const (
	HostAttribute      = "http.host"
	MethodAttribute    = "http.method"
	PathAttribute      = "http.path"
	UserAgentAttribute = "http.user_agent"
)

type handler struct {
	name             string
	Handler          gin.HandlerFunc
	IsPublicEndpoint bool
}

func New(hf krakendgin.HandlerFactory) krakendgin.HandlerFactory {
	return func(cfg *config.EndpointConfig, p proxy.Proxy) gin.HandlerFunc {
		return HandlerFunc(cfg, hf(cfg, p))
	}
}

func HandlerFunc(cfg *config.EndpointConfig, next gin.HandlerFunc) gin.HandlerFunc {
	h := &handler{
		name:    cfg.Endpoint,
		Handler: next,
	}

	return h.HandlerFunc
}

func (h *handler) HandlerFunc(c *gin.Context) {
	var span opentracing.Span

	c.Request, span = h.startTrace(c)
	defer span.Finish()

	h.Handler(c)
}

func (h *handler) startTrace(ctx *gin.Context) (*http.Request, opentracing.Span) {
	r := ctx.Request
	var span opentracing.Span

	span, c := opentracing.StartSpanFromContext(jaegergin.GetSpanFromContext(ctx), h.name)
	span.LogKV(
		PathAttribute, r.URL.Path,
		HostAttribute, r.URL.Host,
		MethodAttribute, r.Method,
		UserAgentAttribute, r.UserAgent(),
	)

	jaegergin.InjectSpanInGinContext(c, ctx)

	return r.WithContext(ctx), span
}
