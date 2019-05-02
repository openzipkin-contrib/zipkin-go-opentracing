package zipkintracer

import (
	"github.com/openzipkin/zipkin-go/model"
)

// SpanContext holds the basic Span metadata.
type spanContextImpl struct {
	zipkinContext model.SpanContext
}

// ForeachBaggageItem belongs to the opentracing.SpanContext interface
func (c *spanContextImpl) ForeachBaggageItem(handler func(k, v string) bool) {
}
