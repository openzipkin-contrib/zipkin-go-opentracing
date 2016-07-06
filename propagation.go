package zipkintracer

import opentracing "github.com/opentracing/opentracing-go"

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
	spanContext opentracing.SpanContext,
	carrier interface{},
) error {
	ac, ok := carrier.(DelegatingCarrier)
	if !ok || ac == nil {
		return opentracing.ErrInvalidCarrier
	}
	sc, ok := spanContext.(*Context)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}
	if p.tracer.options.clientServerSameSpan {
		ac.SetState(sc.TraceID, sc.SpanID, sc.ParentSpanID, sc.Sampled)
	} else {
		ac.SetState(sc.TraceID, sc.ParentSpanID, 0, sc.Sampled)
	}
	for k, v := range sc.Baggage {
		ac.SetBaggageItem(k, v)
	}
	return nil
}

func (p *accessorPropagator) Extract(
	carrier interface{},
) (opentracing.SpanContext, error) {
	ac, ok := carrier.(DelegatingCarrier)
	if !ok || ac == nil {
		return nil, opentracing.ErrInvalidCarrier
	}

	traceID, spanID, parentSpanID, sampled := ac.State()
	sc := &Context{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		Sampled:      sampled,
	}
	ac.GetBaggage(func(k, v string) {
		if sc.Baggage == nil {
			sc.Baggage = map[string]string{}
		}
		sc.Baggage[k] = v
	})

	return sc, nil
}
