package zipkintracer

import (
	"fmt"
	otext "github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/openzipkin-contrib/zipkin-go-opentracing/models"

	"github.com/openzipkin-contrib/zipkin-go-opentracing/flag"
	"github.com/openzipkin-contrib/zipkin-go-opentracing/thrift/gen-go/zipkincore"
)

var (
	// SpanKindResource will be regarded as a SA annotation by Zipkin.
	JsonSpanKindResource = otext.SpanKindEnum("resource")
)

// Recorder implements the SpanRecorder interface.
type JsonRecorder struct {
	collector    CollectorAgnostic
	debug        bool
	endpoint     *zipkincore.Endpoint
	materializer func(logFields []log.Field) ([]byte, error)
}

// RecorderOption allows for functional options.
type JsonRecorderOption func(r *JsonRecorder)

// WithLogFmtMaterializer will convert OpenTracing Log fields to a LogFmt representation.
func JsonWithLogFmtMaterializer() JsonRecorderOption {
	return func(r *JsonRecorder) {
		r.materializer = MaterializeWithLogFmt
	}
}

// WithJSONMaterializer will convert OpenTracing Log fields to a JSON representation.
func JsonWithJSONMaterializer() JsonRecorderOption {
	return func(r *JsonRecorder) {
		r.materializer = MaterializeWithJSON
	}
}

// WithStrictMaterializer will only record event Log fields and discard the rest.
func JsonWithStrictMaterializer() JsonRecorderOption {
	return func(r *JsonRecorder) {
		r.materializer = StrictZipkinMaterializer
	}
}

// NewRecorder creates a new Zipkin Recorder backed by the provided Collector.
//
// hostPort and serviceName allow you to set the default Zipkin endpoint
// information which will be added to the application's standard core
// annotations. hostPort will be resolved into an IPv4 and/or IPv6 address and
// Port number, serviceName will be used as the application's service
// identifier.
//
// If application does not listen for incoming requests or an endpoint Context
// does not involve network address and/or port these cases can be solved like
// this:
//  # port is not applicable:
//  NewRecorder(c, debug, "192.168.1.12:0", "ServiceA")
//
//  # network address and port are not applicable:
//  NewRecorder(c, debug, "0.0.0.0:0", "ServiceB")
func NewJsonRecorder(c CollectorAgnostic, debug bool, hostPort, serviceName string, options ...JsonRecorderOption) SpanRecorder {
	r := &JsonRecorder{
		collector:    c,
		debug:        debug,
		endpoint:     makeEndpoint(hostPort, serviceName),
		materializer: MaterializeWithLogFmt,
	}
	for _, opts := range options {
		opts(r)
	}
	return r
}

// RecordSpan converts a RawSpan into the Zipkin representation of a span
// and records it to the underlying collector.
func (r *JsonRecorder) RecordSpan(sp RawSpan) {
	if !sp.Context.Sampled {
		return
	}
	span := &models.Span{
		Name:        sp.Operation,
		ID:          fmt.Sprintf("%d", sp.Context.SpanID),
		TraceID:     fmt.Sprintf("%d", sp.Context.TraceID.Low),
		TraceIDHigh: fmt.Sprintf("%d", sp.Context.TraceID.High),
		ParentID:    fmt.Sprintf("%d", sp.Context.ParentSpanID),
		Debug:       r.debug || (sp.Context.Flags&flag.Debug == flag.Debug),
	}
	// only send timestamp and duration if this process owns the current span.
	if sp.Context.Owner {
		timestamp := sp.Start.UnixNano() / 1e3
		duration := sp.Duration.Nanoseconds() / 1e3
		// since we always time our spans we will round up to 1 microsecond if the
		// span took less.
		if duration == 0 {
			duration = 1
		}
		span.Timestamp = timestamp
		span.Duration = duration
	}

	for key, value := range sp.Tags {
		annotateBinaryCore(span, key, value, r.endpoint)
	}

	_ = r.collector.Collect(span)
}

func annotateBinaryCore(span *models.Span, key string, value interface{}, host *zipkincore.Endpoint) {
	if b, ok := value.(bool); ok {
		if b {
			value = "true"
		} else {
			value = "false"
		}
	}
	span.BinaryAnnotations = append(span.BinaryAnnotations, &models.BinaryAnnotation{
		Key:      key,
		Value:    fmt.Sprintf("%+v", value),
		Endpoint: models.Endpoint{ServiceName: host.ServiceName, Port: host.Port, Ipv4: fmt.Sprintf("%d", host.Ipv4), Ipv6: string(host.Ipv6)},
	})
}
