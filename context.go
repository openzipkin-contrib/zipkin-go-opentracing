package zipkintracer

import (
	"github.com/openzipkin/zipkin-go"
)

// SpanContext holds the basic Span metadata.
type spanContextImpl struct {
	zipkinContext zipkin.SpanContext
}

// ForeachBaggageItem belongs to the opentracing.SpanContext interface
func (c SpanContext) ForeachBaggageItem(handler func(k, v string) bool) {
}
