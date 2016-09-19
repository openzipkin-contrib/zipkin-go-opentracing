package zipkintracer

import (
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/stretchr/testify/assert"
)

func TestSpan_Baggage(t *testing.T) {
	recorder := NewInMemoryRecorder()
	tracer, err := NewTracer(
		recorder,
		WithSampler(func(_ uint64) bool { return true }),
	)
	if err != nil {
		t.Fatalf("Unable to create Tracer: %+v", err)
	}

	span := tracer.StartSpan("x")
	span.SetBaggageItem("x", "y")
	assert.Equal(t, "y", span.BaggageItem("x"))
	span.Finish()
	spans := recorder.GetSpans()
	assert.Equal(t, 1, len(spans))
	assert.Equal(t, map[string]string{"x": "y"}, spans[0].Context.Baggage)

	recorder.Reset()
	span = tracer.StartSpan("x")
	span.SetBaggageItem("x", "y")
	baggage := make(map[string]string)
	span.Context().ForeachBaggageItem(func(k, v string) bool {
		baggage[k] = v
		return true
	})
	assert.Equal(t, map[string]string{"x": "y"}, baggage)

	span.SetBaggageItem("a", "b")
	baggage = make(map[string]string)
	span.Context().ForeachBaggageItem(func(k, v string) bool {
		baggage[k] = v
		return false // exit early
	})
	assert.Equal(t, 1, len(baggage))
	span.Finish()
	spans = recorder.GetSpans()
	assert.Equal(t, 1, len(spans))
	assert.Equal(t, 2, len(spans[0].Context.Baggage))
}

func TestSpan_Sampling(t *testing.T) {
	recorder := NewInMemoryRecorder()
	tracer, err := NewTracer(
		recorder,
		WithSampler(func(_ uint64) bool { return true }),
	)
	if err != nil {
		t.Fatalf("Unable to create Tracer: %+v", err)
	}

	span := tracer.StartSpan("x")
	span.Finish()
	assert.Equal(t, 1, len(recorder.GetSampledSpans()), "by default span should be sampled")

	recorder.Reset()
	span = tracer.StartSpan("x")
	ext.SamplingPriority.Set(span, 0)
	span.Finish()
	assert.Equal(t, 0, len(recorder.GetSampledSpans()), "SamplingPriority=0 should turn off sampling")

	tracer, err = NewTracer(
		recorder,
		WithSampler(func(_ uint64) bool { return false }),
	)
	if err != nil {
		t.Fatalf("Unable to create Tracer: %+v", err)
	}

	recorder.Reset()
	span = tracer.StartSpan("x")
	span.Finish()
	assert.Equal(t, 0, len(recorder.GetSampledSpans()), "by default span should not be sampled")

	recorder.Reset()
	span = tracer.StartSpan("x")
	ext.SamplingPriority.Set(span, 1)
	span.Finish()
	assert.Equal(t, 1, len(recorder.GetSampledSpans()), "SamplingPriority=1 should turn on sampling")
}

func TestSpan_SingleLoggedTaggedSpan(t *testing.T) {
	recorder := NewInMemoryRecorder()
	tracer, err := NewTracer(
		recorder,
		WithSampler(func(_ uint64) bool { return true }),
	)
	if err != nil {
		t.Fatalf("Unable to create Tracer: %+v", err)
	}

	span := tracer.StartSpan("x")
	span.LogEventWithPayload("event", "payload")
	span.SetTag("tag", "value")
	span.Finish()
	spans := recorder.GetSpans()
	assert.Equal(t, 1, len(spans))
	assert.Equal(t, "x", spans[0].Operation)
	assert.Equal(t, 1, len(spans[0].Logs))
	// XXX: broken tests
	//   assert.Equal(t, "event", spans[0].Logs[0].Event)
	//   assert.Equal(t, "payload", spans[0].Logs[0].Payload)
	assert.Equal(t, opentracing.Tags{"tag": "value"}, spans[0].Tags)
}

func TestSpan_TrimUnsampledSpans(t *testing.T) {
	recorder := NewInMemoryRecorder()
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
	span.LogEventWithPayload("event", "payload")
	span.SetTag("tag", "value")
	span.Finish()
	spans := recorder.GetSpans()
	assert.Equal(t, 1, len(spans))
	assert.Equal(t, 1, len(spans[0].Logs))
	// XXX: broken tests
	//   assert.Equal(t, "event", spans[0].Logs[0].Event)
	//   assert.Equal(t, "payload", spans[0].Logs[0].Payload)
	assert.Equal(t, opentracing.Tags{"tag": "value"}, spans[0].Tags)

	recorder.Reset()
	// Tracer that trims only unsampled and never samples
	tracer, err = NewTracer(
		recorder,
		WithSampler(func(_ uint64) bool { return false }),
		TrimUnsampledSpans(true),
	)
	if err != nil {
		t.Fatalf("Unable to create Tracer: %+v", err)
	}

	span = tracer.StartSpan("x")
	span.LogEventWithPayload("event", "payload")
	span.SetTag("tag", "value")
	span.Finish()
	spans = recorder.GetSpans()
	assert.Equal(t, 1, len(spans))
	assert.Equal(t, 0, len(spans[0].Logs))
	assert.Equal(t, 0, len(spans[0].Tags))
}

func TestSpan_DropAllLogs(t *testing.T) {
	recorder := NewInMemoryRecorder()
	// Tracer that drops logs
	tracer, err := NewTracer(
		recorder,
		WithSampler(func(_ uint64) bool { return true }),
		DropAllLogs(true),
	)
	if err != nil {
		t.Fatalf("Unable to create Tracer: %+v", err)
	}

	span := tracer.StartSpan("x")
	span.LogEventWithPayload("event", "payload")
	span.SetTag("tag", "value")
	span.Finish()
	spans := recorder.GetSpans()
	assert.Equal(t, 1, len(spans))
	assert.Equal(t, "x", spans[0].Operation)
	assert.Equal(t, opentracing.Tags{"tag": "value"}, spans[0].Tags)
	// Only logs are dropped
	assert.Equal(t, 0, len(spans[0].Logs))
}
