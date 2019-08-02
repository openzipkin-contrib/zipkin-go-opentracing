package zipkintracer

import (
	"testing"

	"github.com/openzipkin/zipkin-go"

	"github.com/openzipkin/zipkin-go/reporter/recorder"

	"github.com/opentracing/opentracing-go/log"
	"github.com/stretchr/testify/assert"
)

func TestSpan_SingleLoggedTaggedSpan(t *testing.T) {
	recorder := recorder.NewReporter()
	tracer, err := NewTracer(
		recorder,
		WithSampler(zipkin.AlwaysSample),
	)
	if err != nil {
		t.Fatalf("Unable to create Tracer: %+v", err)
	}

	span := tracer.StartSpan("x")
	span.LogEventWithPayload("event", "payload")
	span.LogFields(log.String("key_str", "value"), log.Uint32("32bit", 4294967295))
	span.SetTag("tag", "value")
	span.Finish()
	spans := recorder.Flush()
	assert.Equal(t, 1, len(spans))
	assert.Equal(t, "x", spans[0].Name)
	assert.Equal(t, 3, len(spans[0].Annotations))
	assert.Equal(t, map[string]string{"tag": "value"}, spans[0].Tags)
	assert.Equal(t, spans[0].Annotations[0].Value, "event:payload")
	assert.Equal(t, spans[0].Annotations[1].Value, "key_str:value")
	assert.Equal(t, spans[0].Annotations[2].Value, "32bit:4294967295")
}

func TestSpan_TrimUnsampledSpans(t *testing.T) {
	recorder := recorder.NewReporter()
	// Tracer that trims only unsampled but always samples
	tracer, err := NewTracer(
		recorder,
		WithSampler(func(_ uint64) bool { return true }),
		TrimUnsampledSpans(true),
	)
	if err != nil {
		t.Fatalf("Unable to create Tracer: %+v", err)
	}

	span := tracer.StartSpan("x")
	span.LogFields(log.String("key_str", "value"), log.Uint32("32bit", 4294967295))
	span.SetTag("tag", "value")
	span.Finish()
	spans := recorder.Flush()
	assert.Equal(t, 1, len(spans))
	assert.Equal(t, 2, len(spans[0].Annotations))
	assert.Equal(t, map[string]string{"tag": "value"}, spans[0].Tags)

	assert.Equal(t, spans[0].Annotations[0].Value, "key_str:value")
	assert.Equal(t, spans[0].Annotations[1].Value, "32bit:4294967295")

	// Tracer that trims only unsampled and never samples
	tracer, err = NewTracer(
		recorder,
		WithSampler(zipkin.NeverSample),
		TrimUnsampledSpans(true),
	)
	if err != nil {
		t.Fatalf("Unable to create Tracer: %+v", err)
	}

	span = tracer.StartSpan("x")
	span.LogFields(log.String("key_str", "value"), log.Uint32("32bit", 4294967295))
	span.SetTag("tag", "value")
	span.Finish()
	spans = recorder.Flush()
	assert.Equal(t, 0, len(spans))
}
