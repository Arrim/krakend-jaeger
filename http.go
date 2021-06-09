package jaeger

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"net/http"

	transport "github.com/luraproject/lura/transport/http/client"
)

var defaultClient = &http.Client{}

func NewHTTPClient(ctx context.Context) *http.Client {
	return defaultClient
}

func HTTPRequestExecutor(clientFactory transport.HTTPClientFactory) transport.HTTPRequestExecutor {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		client := clientFactory(ctx)
		r := req.WithContext(ctx)
		carrier := opentracing.HTTPHeadersCarrier(r.Header)
		c := opentracing.SpanFromContext(ctx)

		if err := opentracing.GlobalTracer().Inject(c.Context(), opentracing.HTTPHeaders, carrier); err != nil {
			return nil, err
		}

		return client.Do(r)
	}
}
