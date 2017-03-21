package zipkintracer

import (
	opentracing "github.com/opentracing/opentracing-go"
)

// Observer can be registered with the zipkin to recieve notifications
// about new Spans.
// The actual registration depends on the implementation, which might look
// like the below e.g :
// observer := myobserver.NewObserver()
// tracer := zipkin.NewTracer(..., zipkin.WithObserver(observer))
//
type Observer interface {
	// Create and return a span observer. Called when a span starts.
	// E.g :
	//     func StartSpan(opName string, opts ...opentracing.StartSpanOption) {
	//     var sp opentracing.Span
	//     sso := opentracing.StartSpanOptions{}
	//     var spObs opentracing.SpanObserver = observer.OnStartSpan(span, opName, sso)
	//     ...
	// }
	// OnStartSpan function needs to be defined for a package exporting
	// metrics as well.
	OnStartSpan(sp opentracing.Span, operationName string, options opentracing.StartSpanOptions) SpanObserver
}

// SpanObserver is created by the Observer and receives notifications about
// other Span events.
// zipkin should define these functions for each of the span operations
// which should call the registered (observer) callbacks.
type SpanObserver interface {
	// Callback called from opentracing.Span.SetOperationName()
	OnSetOperationName(operationName string)
	// Callback called from opentracing.Span.SetTag()
	OnSetTag(key string, value interface{})
	// Callback called from opentracing.Span.Finish()
	OnFinish(options opentracing.FinishOptions)
}

// observer is a dispatcher to other observers
type observer struct {
	observers []Observer
}

// spanObserver is a dispatcher to other span observers
type spanObserver struct {
	observers []SpanObserver
}

// noopSpanObserver is used when there are no observers registered on the
// Tracer or none of them returns span observers
var noopSpanObserver = spanObserver{}

func (o observer) OnStartSpan(sp opentracing.Span, operationName string, options opentracing.StartSpanOptions) SpanObserver {
	var spanObservers []SpanObserver
	for _, obs := range o.observers {
		spanObs := obs.OnStartSpan(sp, operationName, options)
		if spanObs != nil {
			if spanObservers == nil {
				spanObservers = make([]SpanObserver, 0, len(o.observers))
			}
			spanObservers = append(spanObservers, spanObs)
		}
	}
	if len(spanObservers) == 0 {
		return noopSpanObserver
	}

	return spanObserver{observers: spanObservers}
}

func (o spanObserver) OnSetOperationName(operationName string) {
	for _, obs := range o.observers {
		obs.OnSetOperationName(operationName)
	}
}

func (o spanObserver) OnSetTag(key string, value interface{}) {
	for _, obs := range o.observers {
		obs.OnSetTag(key, value)
	}
}

func (o spanObserver) OnFinish(options opentracing.FinishOptions) {
	for _, obs := range o.observers {
		obs.OnFinish(options)
	}
}
