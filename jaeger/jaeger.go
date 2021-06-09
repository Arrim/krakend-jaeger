package jaeger

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"github.com/luraproject/lura/config"
	"github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

const (
	Namespace = "github_com/arrim/krakend-jaeger"
)

var (
	errNoExtraConfig                      = errors.New("no extra config defined for the jaeger module")
	errSingletonExporterFactoriesRegister = errors.New("expecting only one exporter factory registration per instance")
	registerOnce                          = new(sync.Once)
	closer                                io.Closer
)

type Config struct {
	ServiceName        string                 `json:"service_name"`
	SampleType         string                 `json:"sample_type"`
	SampleParam        float64                `json:"sample_param"`
	LogSpans           bool                   `json:"log_spans"`
	LocalAgentHostPort string                 `json:"local_agent_host_port"`
	Disabled           bool                   `json:"disabled"`
	Tags               map[string]interface{} `json:"tags"`
	CollectorEndpoint  string                 `json:"collector_endpoint"`
}

func (c *Config) GetTags() []opentracing.Tag {
	var tags []opentracing.Tag

	for k, v := range c.Tags {
		tags = append(tags, opentracing.Tag{
			Key:   k,
			Value: v,
		})
	}

	return tags
}

func Register(srvCfg config.ServiceConfig) error {
	cfg, err := parseCfg(srvCfg)
	if err != nil {
		return err
	}

	err = errSingletonExporterFactoriesRegister
	registerOnce.Do(func() {
		closer, err = InitJaeger(cfg)
		return
	})

	return err
}

func Close() error {
	if closer != nil {
		return closer.Close()
	}

	return nil
}

func InitJaeger(cfg *Config) (io.Closer, error) {
	jcfg := jaegercfg.Configuration{
		ServiceName: cfg.ServiceName,
		Disabled:    cfg.Disabled,
		Tags:        cfg.GetTags(),
		Sampler: &jaegercfg.SamplerConfig{
			Type:  cfg.SampleType,
			Param: cfg.SampleParam,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           cfg.LogSpans,
			CollectorEndpoint:  cfg.CollectorEndpoint,
			LocalAgentHostPort: cfg.LocalAgentHostPort,
		},
	}

	closer, err := jcfg.InitGlobalTracer(cfg.ServiceName)
	if err != nil {
		return nil, err
	}

	return closer, nil
}

func parseCfg(srvCfg config.ServiceConfig) (*Config, error) {
	cfg := new(Config)
	tmp, ok := srvCfg.ExtraConfig[Namespace]

	if !ok {
		return nil, errNoExtraConfig
	}

	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(tmp)

	if err := json.NewDecoder(buf).Decode(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
