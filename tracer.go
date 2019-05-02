package zipkintracer

import (
	"errors"
	"fmt"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
)

type Propagator struct {
	Inject  func(model.SpanContext, interface{}) error
	Extract func(interface{}) (*model.SpanContext, error)
}

type tracerImpl struct {
	zipkinTracer zipkin.Tracer
	propagators  map[opentracing.BuiltinFormat]Propagator
}

func (t *tracerImpl) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	var startSpanOptions opentracing.StartSpanOptions
	for _, opt := range opts {
		opt.Apply(&startSpanOptions)
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

	newSpan := t.zipkinTracer.StartSpan(operationName, zopts...)

	for key, val := range startSpanOptions.Tags {
		newSpan.Tag(key, fmt.Sprint(val))
	}
	return &spanImpl{
		zipkinSpan: newSpan,
		tracer:     t,
	}
}

func (t *tracerImpl) Inject(sc opentracing.SpanContext, format interface{}, carrier interface{}) error {
	prpg, ok := t.propagators[format.(opentracing.BuiltinFormat)]
	if !ok {
		return opentracing.ErrUnsupportedFormat
	}

	zsc, ok := sc.(*spanContextImpl)
	if !ok {
		return errors.New("unexpected error")
	}

	return prpg.Inject(zsc, carrier)
}

func (t *tracerImpl) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	prpg, ok := t.propagators[format.(opentracing.BuiltinFormat)]
	if !ok {
		return nil, opentracing.ErrUnsupportedFormat
	}

	sc, err := prpg.Extract(carrier)
	if err != nil {
		return nil, err
	}

	return &spanContextImpl{zipkinContext: sc}, nil
}
