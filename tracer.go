package zipkintracer

import (
	"fmt"
	"net"
	"strings"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
)

type tracerImpl struct {
	zipkinTracer       *zipkin.Tracer
	textPropagator     *textMapPropagator
	accessorPropagator *accessorPropagator
	opts               *TracerOptions
}

// Wrap receives a zipkin tracer and returns an opentracing
// tracer
func Wrap(tr *zipkin.Tracer, opts ...TracerOption) opentracing.Tracer {
	t := &tracerImpl{
		zipkinTracer: tr,
		opts:         &TracerOptions{},
	}
	t.textPropagator = &textMapPropagator{t}
	t.accessorPropagator = &accessorPropagator{t}

	for _, o := range opts {
		o(t.opts)
	}

	return t
}

func (t *tracerImpl) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	var startSpanOptions opentracing.StartSpanOptions
	for _, opt := range opts {
		opt.Apply(&startSpanOptions)
	}

	zopts := make([]zipkin.SpanOption, 0)

	// Parent
	if len(startSpanOptions.References) > 0 {
		parent, ok := (startSpanOptions.References[0].ReferencedContext).(*SpanContext)
		if ok {
			zopts = append(zopts, zipkin.Parent(model.SpanContext(*parent)))
		}
	}

	startTime := time.Now()
	// Time
	if !startSpanOptions.StartTime.IsZero() {
		zopts = append(zopts, zipkin.StartTime(startSpanOptions.StartTime))
		startTime = startSpanOptions.StartTime
	}

	zopts = append(zopts, parseTagsAsZipkinOptions(startSpanOptions.Tags)...)

	newSpan := t.zipkinTracer.StartSpan(operationName, zopts...)

	sp := &spanImpl{
		zipkinSpan: newSpan,
		tracer:     t,
		startTime:  startTime,
	}
	if t.opts.observer != nil {
		observer, _ := t.opts.observer.OnStartSpan(sp, operationName, startSpanOptions)
		sp.observer = observer
	}

	return sp
}

func parseTagsAsZipkinOptions(t map[string]interface{}) []zipkin.SpanOption {
	zopts := make([]zipkin.SpanOption, 0)

	tags := map[string]string{}
	remoteEndpoint := &model.Endpoint{}

	if val, ok := t[string(ext.SpanKind)]; ok {
		kind, _ := val.(string)
		zopts = append(zopts, zipkin.Kind(model.Kind(strings.ToUpper(kind))))
	}

	if val, ok := t[string(ext.PeerService)]; ok {
		serviceName, _ := val.(string)
		remoteEndpoint.ServiceName = serviceName
	}

	if val, ok := t[string(ext.PeerHostIPv4)]; ok {
		ipv4, _ := val.(string)
		remoteEndpoint.IPv4 = net.ParseIP(ipv4)
	}

	if val, ok := t[string(ext.PeerHostIPv6)]; ok {
		ipv6, _ := val.(string)
		remoteEndpoint.IPv6 = net.ParseIP(ipv6)
	}

	if val, ok := t[string(ext.PeerPort)]; ok {
		port, _ := val.(uint16)
		remoteEndpoint.Port = port
	}

	for key, val := range t {
		if key == string(ext.SpanKind) ||
			key == string(ext.PeerService) ||
			key == string(ext.PeerHostIPv4) ||
			key == string(ext.PeerHostIPv6) ||
			key == string(ext.PeerPort) {
			continue
		}

		tags[key] = fmt.Sprint(val)
	}

	if len(tags) > 0 {
		zopts = append(zopts, zipkin.Tags(tags))
	}

	if !remoteEndpoint.Empty() {
		zopts = append(zopts, zipkin.RemoteEndpoint(remoteEndpoint))
	}

	return zopts
}

type delegatorType struct{}

// Delegator is the format to use for DelegatingCarrier.
var Delegator delegatorType

func (t *tracerImpl) Inject(sc opentracing.SpanContext, format interface{}, carrier interface{}) error {
	switch format {
	case opentracing.TextMap, opentracing.HTTPHeaders:
		return t.textPropagator.Inject(sc, carrier)
	case opentracing.Binary:
		// try with textMapPropagator
		return t.textPropagator.Inject(sc, carrier)
	}
	if _, ok := format.(delegatorType); ok {
		return t.accessorPropagator.Inject(sc, carrier)
	}
	return opentracing.ErrUnsupportedFormat
}

func (t *tracerImpl) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	switch format {
	case opentracing.TextMap, opentracing.HTTPHeaders:
		return t.textPropagator.Extract(carrier)
	case opentracing.Binary:
		// try with textMapPropagator
		return t.textPropagator.Extract(carrier)
	}
	if _, ok := format.(delegatorType); ok {
		return t.accessorPropagator.Extract(carrier)
	}
	return nil, opentracing.ErrUnsupportedFormat
}
