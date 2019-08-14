package zipkintracer

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	b3http "github.com/openzipkin-contrib/zipkin-go-opentracing/propagation/http"

	otobserver "github.com/opentracing-contrib/go-observer"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
)

type propagator interface {
	Inject(model.SpanContext, interface{}) error
	Extract(interface{}) (*model.SpanContext, error)
}

type tracerImpl struct {
	zipkinTracer *zipkin.Tracer
	propagators  map[opentracing.BuiltinFormat]propagator
	observer     otobserver.Observer
}

// Wrap receives a zipkin tracer and returns an opentracing
// tracer
func Wrap(tr *zipkin.Tracer, opts ...TracerOption) opentracing.Tracer {
	to := &TracerOptions{}
	for _, o := range opts {
		o(to)
	}

	return &tracerImpl{
		zipkinTracer: tr,
		propagators: map[opentracing.BuiltinFormat]propagator{
			opentracing.HTTPHeaders: b3http.Propagator,
			opentracing.TextMap:     b3http.Propagator,
		},
		observer: to.observer,
	}
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

	zopts = append(zopts, parseTagsAsZipkinOptions(startSpanOptions.Tags)...)

	newSpan := t.zipkinTracer.StartSpan(operationName, zopts...)

	sp := &spanImpl{
		zipkinSpan: newSpan,
		tracer:     t,
		startTime:  startTime,
	}
	if t.observer != nil {
		observer, _ := t.observer.OnStartSpan(sp, operationName, startSpanOptions)
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

		tags[translateTagKey(key)] = fmt.Sprint(val)
	}

	if len(tags) > 0 {
		zopts = append(zopts, zipkin.Tags(tags))
	}

	if !remoteEndpoint.Empty() {
		zopts = append(zopts, zipkin.RemoteEndpoint(remoteEndpoint))
	}

	return zopts
}

var tagsTranslation = map[string]string{
	"db.statement": "sql.query",
}

func translateTagKey(key string) string {
	if tKey, ok := tagsTranslation[key]; ok {
		return tKey
	}

	return key
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
