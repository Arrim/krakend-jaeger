package gin

import (
	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/proxy"
	krakendgin "github.com/devopsfaith/krakend/router/gin"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"net/http"
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
	so := startServerSpanOption(opentracing.HTTPHeadersCarrier(r.Header))

	span, c := opentracing.StartSpanFromContext(ctx, h.name, so)
	span.LogKV(
		PathAttribute, r.URL.Path,
		HostAttribute, r.URL.Host,
		MethodAttribute, r.Method,
		UserAgentAttribute, r.UserAgent(),
	)

	ctx.Set("spanCtx", c)

	return r.WithContext(ctx), span
}

func startServerSpanOption(headers opentracing.TextMapReader) opentracing.StartSpanOption {
	wireContext, _ := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, headers)

	return ext.RPCServerOption(wireContext)
}
