package zipkintracer

import (
	"time"

	opentracing "github.com/opentracing/opentracing-go"
)

type accessorPropagator struct {
	tracer *tracerImpl
}

// DelegatingCarrier is a flexible carrier interface which can be implemented
// by types which have a means of storing the trace metadata and already know
// how to serialize themselves (for example, protocol buffers).
type DelegatingCarrier interface {
	SetState(traceID, spanID, parentSpanID uint64, sampled bool)
	State() (traceID, spanID, parentSpanID uint64, sampled bool)
	SetBaggageItem(key, value string)
	GetBaggage(func(key, value string))
}

func (p *accessorPropagator) Inject(
	sp opentracing.Span,
	carrier interface{},
) error {
	ac, ok := carrier.(DelegatingCarrier)
	if !ok || ac == nil {
		return opentracing.ErrInvalidCarrier
	}
	si, ok := sp.(*spanImpl)
	if !ok {
		return opentracing.ErrInvalidSpan
	}
	meta := si.raw.Context
	if p.tracer.options.clientServerSameSpan {
		ac.SetState(meta.TraceID, meta.SpanID, meta.ParentSpanID, meta.Sampled)
	} else {
		ac.SetState(meta.TraceID, meta.ParentSpanID, 0, meta.Sampled)
	}
	for k, v := range si.raw.Baggage {
		ac.SetBaggageItem(k, v)
	}
	return nil
}

func (p *accessorPropagator) Join(
	operationName string,
	carrier interface{},
) (opentracing.Span, error) {
	ac, ok := carrier.(DelegatingCarrier)
	if !ok || ac == nil {
		return nil, opentracing.ErrInvalidCarrier
	}

	sp := p.tracer.getSpan()
	ac.GetBaggage(func(k, v string) {
		if sp.raw.Baggage == nil {
			sp.raw.Baggage = map[string]string{}
		}
		sp.raw.Baggage[k] = v
	})

	traceID, spanID, parentSpanID, sampled := ac.State()
	sp.raw.Context = Context{
		TraceID: traceID,
		Sampled: sampled,
	}
	if p.tracer.options.clientServerSameSpan {
		sp.raw.Context.SpanID = spanID
		sp.raw.Context.ParentSpanID = parentSpanID
	} else {
		sp.raw.Context.SpanID = randomID()
		sp.raw.Context.ParentSpanID = spanID
	}

	return p.tracer.startSpanInternal(
		sp,
		operationName,
		time.Now(),
		nil,
	), nil
}
