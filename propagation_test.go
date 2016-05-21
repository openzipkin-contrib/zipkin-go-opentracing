package zipkintracer_test

import (
	"bytes"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	opentracing "github.com/opentracing/opentracing-go"

	zipkintracer "github.com/openzipkin/zipkin-go-opentracing"
)

type verbatimCarrier struct {
	zipkintracer.Context
	b map[string]string
}

var _ zipkintracer.DelegatingCarrier = &verbatimCarrier{}

func (vc *verbatimCarrier) SetBaggageItem(k, v string) {
	vc.b[k] = v
}

func (vc *verbatimCarrier) GetBaggage(f func(string, string)) {
	for k, v := range vc.b {
		f(k, v)
	}
}

func (vc *verbatimCarrier) SetState(tID, sID, pId uint64, sampled bool) {
	vc.Context = zipkintracer.Context{TraceID: tID, SpanID: sID, ParentSpanID: pId, Sampled: sampled}
}

func (vc *verbatimCarrier) State() (traceID, spanID, parentSpanID uint64, sampled bool) {
	return vc.Context.TraceID, vc.Context.SpanID, vc.Context.ParentSpanID, vc.Context.Sampled
}

func TestSpanPropagator(t *testing.T) {
	const op = "test"
	recorder := zipkintracer.NewInMemoryRecorder()
	tracer, err := zipkintracer.NewTracer(
		recorder,
		zipkintracer.ClientServerSameSpan(true),
	)
	if err != nil {
		t.Fatalf("Unable to create Tracer: %+v", err)
	}
	sp := tracer.StartSpan(op)
	sp.SetBaggageItem("foo", "bar")

	tmc := opentracing.HTTPHeaderTextMapCarrier(http.Header{})
	tests := []struct {
		typ, carrier interface{}
	}{
		{zipkintracer.Delegator, zipkintracer.DelegatingCarrier(&verbatimCarrier{b: map[string]string{}})},
		{opentracing.Binary, &bytes.Buffer{}},
		{opentracing.TextMap, tmc},
	}

	for i, test := range tests {
		if err := tracer.Inject(sp, test.typ, test.carrier); err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		child, err := tracer.Join(op, test.typ, test.carrier)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		child.Finish()
	}
	sp.Finish()

	spans := recorder.GetSpans()
	if a, e := len(spans), len(tests)+1; a != e {
		t.Fatalf("expected %d spans, got %d", e, a)
	}

	// The last span is the original one.
	exp, spans := spans[len(spans)-1], spans[:len(spans)-1]
	exp.Duration = time.Duration(123)
	exp.Start = time.Time{}.Add(1)

	for i, sp := range spans {
		if a, e := sp.ParentSpanID, exp.ParentSpanID; a != e {
			t.Fatalf("%d: ParentSpanID %d does not match expectation %d", i, a, e)
		} else {
			// Prepare for comparison.
			sp.SpanID, sp.ParentSpanID = exp.SpanID, 0
			sp.Duration, sp.Start = exp.Duration, exp.Start
		}
		if a, e := sp.TraceID, exp.TraceID; a != e {
			t.Fatalf("%d: TraceID changed from %d to %d", i, e, a)
		}
		if !reflect.DeepEqual(exp, sp) {
			t.Fatalf("%d: wanted %+v, got %+v", i, spew.Sdump(exp), spew.Sdump(sp))
		}
	}
}
