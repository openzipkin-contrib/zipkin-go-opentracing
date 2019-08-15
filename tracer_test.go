package zipkintracer

import (
	"testing"

	"github.com/openzipkin/zipkin-go/reporter"
	"github.com/openzipkin/zipkin-go/reporter/recorder"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	zipkin "github.com/openzipkin/zipkin-go"
)

func newTracer(r reporter.Reporter, opts ...zipkin.TracerOption) opentracing.Tracer {
	tr, _ := zipkin.NewTracer(r, opts...)
	return Wrap(tr)
}

func TestOptionsFromOTTags(t *testing.T) {
	tags := map[string]interface{}{}
	tags[string(ext.SpanKind)] = "server"
	tags[string(ext.PeerService)] = "service_a"
	tags["key"] = "value"
	opts := parseTagsAsZipkinOptions(tags)

	rec := recorder.NewReporter()
	tr, _ := zipkin.NewTracer(rec)
	sp := tr.StartSpan("test", opts...)
	sp.Finish()
	spans := rec.Flush()
	if want, have := 1, len(spans); want != have {
		t.Fatalf("unexpected number of spans, want %d, have %d", want, have)
	}

	if want, have := "SERVER", string(spans[0].Kind); want != have {
		t.Errorf("unexpected span kind, want %s, have %s", want, have)
	}

	if want, have := "service_a", spans[0].RemoteEndpoint.ServiceName; want != have {
		t.Errorf("unexpected remote service name, want %s, have %s", want, have)
	}

	if want, have := "value", spans[0].Tags["key"]; want != have {
		t.Errorf("unexpected tag value, want %s, have %s", want, have)
	}
}
