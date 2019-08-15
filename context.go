package zipkintracer

import (
	"github.com/openzipkin/zipkin-go/model"
)

// SpanContext holds the basic Span metadata.
type SpanContext model.SpanContext

// ForeachBaggageItem belongs to the opentracing.SpanContext interface
func (c SpanContext) ForeachBaggageItem(handler func(k, v string) bool) {}
