package zipkin

import (
	"errors"

	basictracer "github.com/opentracing/basictracer-go"
	opentracing "github.com/opentracing/opentracing-go"

	"github.com/basvanbeek/zipkin-go-opentracing/_thrift/gen-go/zipkincore"
)

var _ opentracing.Tracer = NewTracer(nil) // Compile time check.

// ErrInvalidEndpoint will be thrown if hostPort parameter is corrupted or host
// can't be resolved
var ErrInvalidEndpoint = errors.New("Invalid Endpoint. Please check hostPort parameter")

// config defines options for a Tracer.
type config struct {
	ShouldSample       func(traceID uint64) bool
	TrimUnsampledSpans bool
	Logger             Logger
	Debug              bool
	Endpoint           *zipkincore.Endpoint
}

// Option allows for functional options.
// See: http://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type Option func(c *config) error

// NewTracer creates a new opentracing.Tracer that records spans to the given
// Zipkin Collector.
func NewTracer(c Collector) opentracing.Tracer {
	tracer, err := NewTracerWithOptions(c)
	if err != nil {
		panic(err)
	}
	return tracer
}

// NewTracerWithOptions creates a new opentracing.Tracer for Zipkin with custom
// options set.
func NewTracerWithOptions(c Collector, options ...Option) (opentracing.Tracer, error) {
	cfg := &config{
		ShouldSample:       alwaysSample,
		TrimUnsampledSpans: false,
		Logger:             NewNopLogger(),
		Debug:              false,
		Endpoint:           nil,
	}
	for _, option := range options {
		if err := option(cfg); err != nil {
			return nil, err
		}
	}
	btOptions := basictracer.DefaultOptions()
	btOptions.ShouldSample = cfg.ShouldSample
	btOptions.TrimUnsampledSpans = cfg.TrimUnsampledSpans
	btOptions.Recorder = NewRecorder(c, cfg)
	return basictracer.NewWithOptions(btOptions), nil
}

// SetSampler allows one to add a Sampler function
func SetSampler(sampler Sampler) Option {
	return func(c *config) error {
		c.ShouldSample = sampler
		return nil
	}
}

// TrimUnsampledSpans option
func TrimUnsampledSpans(trim bool) Option {
	return func(c *config) error {
		c.TrimUnsampledSpans = trim
		return nil
	}
}

// SetLogger option
func SetLogger(logger Logger) Option {
	return func(c *config) error {
		c.Logger = logger
		return nil
	}
}

// Debug option
func Debug(debug bool) Option {
	return func(c *config) error {
		c.Debug = debug
		return nil
	}
}

// Endpoint sets the default Zipkin Endpoint to use if not explicitely provided
// when Annotating.
func Endpoint(hostPort, serviceName string) Option {
	return func(c *config) error {
		endpoint := MakeEndpoint(hostPort, serviceName)
		if endpoint == nil {
			return ErrInvalidEndpoint
		}
		c.Endpoint = endpoint
		return nil
	}
}
