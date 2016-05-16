package zipkintracer

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	otext "github.com/opentracing/opentracing-go/ext"

	"github.com/basvanbeek/zipkin-go-opentracing/_thrift/gen-go/zipkincore"
)

// A SpanRecorder handles all of the `RawSpan` data generated via an
// associated `Tracer` (see `NewStandardTracer`) instance. It also names
// the containing process and provides access to a straightforward tag map.
type SpanRecorder interface {
	// Implementations must determine whether and where to store `span`.
	RecordSpan(span RawSpan)
}

// Recorder implements the SpanRecorder interface.
type Recorder struct {
	collector Collector
	debug     bool
	endpoint  *zipkincore.Endpoint
}

// NewRecorder creates a new Zipkin Recorder backed by the provided Collector.
func NewRecorder(c Collector, debug bool, hostPort, serviceName string) SpanRecorder {
	return &Recorder{
		collector: c,
		debug:     debug,
		endpoint:  MakeEndpoint(hostPort, serviceName),
	}
}

// RecordSpan converts a RawSpan into the Zipkin representation of a span
// and records it to the underlying collector.
func (r *Recorder) RecordSpan(sp RawSpan) {
	if !sp.Sampled {
		return
	}

	var (
		parentSpanID = int64(sp.ParentSpanID)
		timestamp    = sp.Start.UnixNano() / 1e3
		duration     = sp.Duration.Nanoseconds() / 1e3
	)

	span := &zipkincore.Span{
		Name:      sp.Operation,
		ID:        int64(sp.SpanID),
		TraceID:   int64(sp.TraceID),
		ParentID:  &parentSpanID,
		Debug:     r.debug,
		Timestamp: &timestamp,
		Duration:  &duration,
	}

	if kind, ok := sp.Tags["span.kind"]; ok {
		switch kind {
		case otext.SpanKindRPCClient:
			Annotate(span, sp.Start, zipkincore.CLIENT_SEND, r.endpoint)
			Annotate(span, sp.Start.Add(sp.Duration), zipkincore.CLIENT_RECV, r.endpoint)
		case otext.SpanKindRPCServer:
			Annotate(span, sp.Start, zipkincore.SERVER_RECV, r.endpoint)
			Annotate(span, sp.Start.Add(sp.Duration), zipkincore.SERVER_SEND, r.endpoint)
		}
	}

	for key, value := range sp.Tags {
		AnnotateBinary(span, key, value, r.endpoint)
	}

	for _, spLog := range sp.Logs {
		if spLog.Timestamp.IsZero() {
			spLog.Timestamp = time.Now()
		}
		Annotate(span, spLog.Timestamp, spLog.Event, r.endpoint)
	}

	r.collector.Collect(span)
}

// Annotate annotates the span with the given value.
func Annotate(span *zipkincore.Span, timestamp time.Time, value string, host *zipkincore.Endpoint) {
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	span.Annotations = append(span.Annotations, &zipkincore.Annotation{
		Timestamp: timestamp.UnixNano() / 1e3,
		Value:     value,
		Host:      host,
	})
}

// AnnotateBinary annotates the span with a key and a value that will be []byte
// encoded.
func AnnotateBinary(span *zipkincore.Span, key string, value interface{}, host *zipkincore.Endpoint) {
	var a zipkincore.AnnotationType
	var b []byte
	// We are not using zipkincore.AnnotationType_I16 for types that could fit
	// as reporting on it seems to be broken on the zipkin web interface
	// (however, we can properly extract the number from zipkin storage
	// directly). int64 has issues with negative numbers but seems ok for
	// positive numbers needing more than 32 bit.
	switch v := value.(type) {
	case bool:
		a = zipkincore.AnnotationType_BOOL
		b = []byte("\x00")
		if v {
			b = []byte("\x01")
		}
	case []byte:
		a = zipkincore.AnnotationType_BYTES
		b = v
	case byte:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(v))
	case int8:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(v))
	case int16:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(v))
	case uint16:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(v))
	case int32:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(v))
	case uint32:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(v))
	case int64:
		a = zipkincore.AnnotationType_I64
		b = make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(v))
	case int:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 8)
		binary.BigEndian.PutUint32(b, uint32(v))
	case uint:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 8)
		binary.BigEndian.PutUint32(b, uint32(v))
	case uint64:
		a = zipkincore.AnnotationType_I64
		b = make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(v))
	case float32:
		a = zipkincore.AnnotationType_DOUBLE
		b = make([]byte, 8)
		bits := math.Float64bits(float64(v))
		binary.BigEndian.PutUint64(b, bits)
	case float64:
		a = zipkincore.AnnotationType_DOUBLE
		b = make([]byte, 8)
		bits := math.Float64bits(v)
		binary.BigEndian.PutUint64(b, bits)
	case string:
		a = zipkincore.AnnotationType_STRING
		b = []byte(v)
	default:
		// we have no handler for type's value, but let's get a string
		// representation of it.
		a = zipkincore.AnnotationType_STRING
		b = []byte(fmt.Sprintf("%+v", value))
	}
	span.BinaryAnnotations = append(span.BinaryAnnotations, &zipkincore.BinaryAnnotation{
		Key:            key,
		Value:          b,
		AnnotationType: a,
		Host:           host,
	})
}
