package zipkin

import (
	"time"

	basictracer "github.com/opentracing/basictracer-go"
	otext "github.com/opentracing/opentracing-go/ext"

	"github.com/basvanbeek/zipkin-go-opentracing/_thrift/gen-go/zipkincore"
)

// Recorder implements the basictracer.Recorder interface.
type Recorder struct {
	collector Collector
	debug     bool
	Log       Logger
	endpoint  *zipkincore.Endpoint
}

// NewRecorder forwards basictracer.RawSpans to an appdash.Collector.
func NewRecorder(collector Collector, opts *config) *Recorder {
	if opts.Logger == nil {
		opts.Logger = NewNopLogger()
	}
	return &Recorder{
		collector: collector,
		Log:       opts.Logger,
		debug:     opts.Debug,
		endpoint:  opts.Endpoint,
	}
}

// RecordSpan converts a RawSpan into the Zipkin representation of a span
// and records it to the underlying collector.
func (r *Recorder) RecordSpan(sp basictracer.RawSpan) {
	if !sp.Sampled {
		return
	}

	var (
		parentSpanID = int64(sp.ParentSpanID)
		timestamp    = sp.Start.UnixNano() / 1e3
		duration     = sp.Duration.Nanoseconds() / 1e3
	)

	span := &Span{
		Span: zipkincore.Span{
			Name:      sp.Operation,
			ID:        int64(sp.SpanID),
			TraceID:   int64(sp.TraceID),
			ParentID:  &parentSpanID,
			Debug:     r.debug,
			Timestamp: &timestamp,
			Duration:  &duration,
		},
		sampled: sp.Sampled,
	}

	if kind, ok := sp.Tags["span.kind"]; ok {
		switch kind {
		case otext.SpanKindRPCClient:
			delete(sp.Tags, "span.kind")
			span.Annotate(sp.Start, zipkincore.CLIENT_SEND, r.endpoint)
			span.Annotate(sp.Start.Add(sp.Duration), zipkincore.CLIENT_RECV, r.endpoint)
		case otext.SpanKindRPCServer:
			delete(sp.Tags, "span.kind")
			span.Annotate(sp.Start, zipkincore.SERVER_RECV, r.endpoint)
			span.Annotate(sp.Start.Add(sp.Duration), zipkincore.SERVER_SEND, r.endpoint)
		}
	}

	for key, value := range sp.Tags {
		span.AnnotateBinary(key, value, r.endpoint)
	}

	for _, spLog := range sp.Logs {
		if spLog.Timestamp.IsZero() {
			spLog.Timestamp = time.Now()
		}
		span.Annotate(spLog.Timestamp, spLog.Event, r.endpoint)
	}

	r.collector.Collect(span)
}
