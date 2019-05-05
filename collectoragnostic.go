package zipkintracer

import (
	"github.com/openzipkin-contrib/zipkin-go-opentracing/thrift/gen-go/zipkincore"
)

// Collector represents a Zipkin trace collector, which is probably a set of
// remote endpoints.
type CollectorAgnostic interface {
	Collect(*CoreSpan) error
	Close() error
}

// NopCollector implements Collector but performs no work.
type NopAgnosticCollector struct{}

// Collect implements Collector.
func (NopAgnosticCollector) Collect(*zipkincore.Span) error { return nil }

// Close implements Collector.
func (NopAgnosticCollector) Close() error { return nil }
