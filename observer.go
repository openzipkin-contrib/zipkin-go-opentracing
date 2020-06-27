package zipkintracer

import (
	"time"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
)

// ZipkinStartSpanOptions allows ZipkinObserver.OnStartSpan() to inspect
// options used during zipkin.Span creation
type ZipkinStartSpanOptions struct {
	// Parent span context reference, if any
	Parent *model.SpanContext

	// Span's start time
	StartTime time.Time

	// Kind clarifies context of timestamp, duration and remoteEndpoint in a span.
	Kind model.Kind

	// Tags used during span creation
	Tags map[string]string

	// RemoteEndpoint used during span creation
	RemoteEndpoint *model.Endpoint
}

// ZipkinObserver may be registered with a Tracer to receive notifications about new Spans
type ZipkinObserver interface {
	// OnStartSpan is called when new Span is created. Creates and returns span observer.
	// If the observer is not interested in the given span, it must return nil.
	OnStartSpan(sp zipkin.Span, operationName string, options *ZipkinStartSpanOptions) ZipkinSpanObserver
}

// ZipkinSpanObserver is created by the ZipkinObserver and receives notifications about
// other Span events.
type ZipkinSpanObserver interface {
	// Callback called from zipkin.Span.SetName()
	OnSetName(operationName string)

	// Callback called from zipkin.Span.SetTag()
	OnSetTag(key, value string)

	// Callback called from zipkin.Span.SetRemoteEndpoint()
	OnSetRemoteEndpoint(remote *model.Endpoint)

	// Callback called from zipkin.Span.Annotate()
	OnAnnotate(t time.Time, annotation string)

	// Callback called from zipkin.Span.Finish()
	OnFinish()

	// Callback called from zipkin.Span.FinishedWithDuration()
	OnFinishedWithDuration(dur time.Duration)
}
