package zipkintracer

import (
	"errors"
	"fmt"
	"time"

	b3http "github.com/openzipkin-contrib/zipkin-go-opentracing/propagation/http"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
)

type propagator interface {
	Inject(model.SpanContext, interface{}) error
	Extract(interface{}) (*model.SpanContext, error)
}

type tracerImpl struct {
	zipkinTracer *zipkin.Tracer
	propagators  map[opentracing.BuiltinFormat]propagator
}

// NewTracer returns an opentracing.Tracer interface wrapping zipkin tracer
func NewTracer(rep reporter.Reporter, opts ...zipkin.TracerOption) (opentracing.Tracer, error) {
	tr, err := zipkin.NewTracer(rep, opts...)
	if err != nil {
		return nil, err
	}

	return &tracerImpl{
		zipkinTracer: tr,
		propagators: map[opentracing.BuiltinFormat]propagator{
			opentracing.TextMap: b3http.Propagator,
		},
	}, nil
}

func (t *tracerImpl) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	var startSpanOptions opentracing.StartSpanOptions
	for _, opt := range opts {
		opt.Apply(&startSpanOptions)
	}

	zopts := make([]zipkin.SpanOption, 0)

	// Parent
	if len(startSpanOptions.References) > 0 {
		parent, ok := (startSpanOptions.References[0].ReferencedContext).(*spanContextImpl)
		if ok {
			zopts = append(zopts, zipkin.Parent(parent.zipkinContext))
		}
	}

	startTime := time.Now()
	// Time
	if !startSpanOptions.StartTime.IsZero() {
		zopts = append(zopts, zipkin.StartTime(startSpanOptions.StartTime))
		startTime = startSpanOptions.StartTime
	}

	newSpan := t.zipkinTracer.StartSpan(operationName, zopts...)

	for key, val := range startSpanOptions.Tags {
		newSpan.Tag(key, fmt.Sprint(val))
	}
	return &spanImpl{
		zipkinSpan: newSpan,
		tracer:     t,
		startTime:  startTime,
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

	return prpg.Inject(zsc.zipkinContext, carrier)
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

	return &spanContextImpl{zipkinContext: *sc}, nil
}
