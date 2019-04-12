package zipkintracer

import (
	"fmt"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/openzipkin/zipkin-go"
)

type tracerImpl struct {
	zipkinTracer zipkin.Tracer
}

func (t *tracerImpl) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {

	var startSpanOptions opentracing.StartSpanOptions

	for _, opt := range opts {
		opt(startSpanOptions)
	}

	zopts := make([]zipkin.SpanOption, 0)

	// Parent
	if len(startSpanOptions.References) > 0 {
		parent, ok := startSpanOptions.References[0].(*spanContextImpl)
		if ok {
			zopts = append(zopts, zipkin.Parent(parent.zipkinContext))
		}
	}

	// Time
	if !startSpanOptions.StartTime.IsZero() {
		zopts = append(zopts, zipkin.StartTime(startSpanOptions.StartTime))
	}

	newSpan := t.zipkinTracer.StartSpan(operationName, zopts)

	for key, val := range startSpanOptions.Tags {
		newSpan.Tag(key, fmt.Sprint(val))
	}
	return &spanImpl{
		zipkinSpan: newSpan,
		tracer:     t,
	}
}

func (t *tracerImpl) Inject(sc opentracing.SpanContext, format interface{}, carrier interface{}) error {
	switch format {
	case opentracing.TextMap, opentracing.HTTPHeaders:
		return t.textPropagator.Inject(sc, carrier)
	case opentracing.Binary:
		return t.binaryPropagator.Inject(sc, carrier)
	}
	if _, ok := format.(delegatorType); ok {
		return t.accessorPropagator.Inject(sc, carrier)
	}
	return opentracing.ErrUnsupportedFormat
}

func (t *tracerImpl) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	switch format {
	case opentracing.TextMap, opentracing.HTTPHeaders:
		return t.textPropagator.Extract(carrier)
	case opentracing.Binary:
		return t.binaryPropagator.Extract(carrier)
	}
	if _, ok := format.(delegatorType); ok {
		return t.accessorPropagator.Extract(carrier)
	}
	return nil, opentracing.ErrUnsupportedFormat
}
