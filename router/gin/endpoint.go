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

	c.Request, span = h.startTrace(c.Writer, c.Request)

	ctx := opentracing.ContextWithSpan(c, span)

	h.Handler(ctx.(*gin.Context))

	span.Finish()
}

func (h *handler) startTrace(_ gin.ResponseWriter, r *http.Request) (*http.Request, opentracing.Span) {
	ctx := r.Context()
	var span opentracing.Span
	so := startServerSpanOption(opentracing.HTTPHeadersCarrier(r.Header))

	span = opentracing.StartSpan(h.name, so)
	span.SetBaggageItem(PathAttribute, r.URL.Path)
	span.SetBaggageItem(HostAttribute, r.URL.Host)
	span.SetBaggageItem(MethodAttribute, r.Method)
	span.SetBaggageItem(UserAgentAttribute, r.UserAgent())

	return r.WithContext(ctx), span
}

func startServerSpanOption(headers opentracing.TextMapReader) opentracing.StartSpanOption {
	wireContext, _ := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, headers)

	return ext.RPCServerOption(wireContext)
}
