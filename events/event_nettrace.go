package events

import (
	"github.com/opentracing/basictracer-go"
	"golang.org/x/net/trace"
)

// NetTraceIntegrator can be passed into a basictracer as NewSpanEventListener
// and causes all traces to be registered with the net/trace endpoint.
var NetTraceIntegrator = func() func(basictracer.SpanEvent) {
	var tr trace.Trace
	return func(e basictracer.SpanEvent) {
		switch t := e.(type) {
		case basictracer.EventCreate:
			tr = trace.New("tracing", t.OperationName)
		case basictracer.EventFinish:
			tr.Finish()
		case basictracer.EventLog:
			if t.Payload != nil {
				tr.LazyPrintf("%s (payload %v)", t.Event, t.Payload)
			} else {
				tr.LazyPrintf("%s", t.Event)
			}
		}
	}
}
