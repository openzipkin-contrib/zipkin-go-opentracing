package wire_test

import (
	"testing"

	"github.com/openzipkin/zipkin-go-opentracing"
	"github.com/openzipkin/zipkin-go-opentracing/wire"
)

func TestProtobufCarrier(t *testing.T) {
	var carrier zipkintracer.DelegatingCarrier = &wire.ProtobufCarrier{}

	var traceID, spanID, parentSpanID uint64 = 1, 2, 3
	sampled := true
	baggageKey, expVal := "key1", "val1"

	carrier.SetState(traceID, spanID, parentSpanID, sampled)
	carrier.SetBaggageItem(baggageKey, expVal)
	gotTraceID, gotSpanID, gotParentSpanId, gotSampled := carrier.State()
	if traceID != gotTraceID || spanID != gotSpanID || parentSpanID != gotParentSpanId || sampled != gotSampled {
		t.Errorf("Wanted state %d %d %d %t, got %d %d %d %t", spanID, traceID, parentSpanID, sampled, gotTraceID, gotSpanID, gotParentSpanId, gotSampled)
	}

	gotBaggage := map[string]string{}
	f := func(k, v string) {
		gotBaggage[k] = v
	}

	carrier.GetBaggage(f)
	value, ok := gotBaggage[baggageKey]
	if !ok {
		t.Errorf("Expected baggage item %s to exist", baggageKey)
	}
	if value != expVal {
		t.Errorf("Expected key %s to be %s, got %s", baggageKey, expVal, value)
	}
}
