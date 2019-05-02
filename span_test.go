package zipkintracer

import (
	"testing"

	"github.com/openzipkin/zipkin-go"

	"github.com/opentracing/opentracing-go/ext"
	"github.com/openzipkin/zipkin-go/reporter/recorder"
	"github.com/stretchr/testify/assert"
)

func TestSpan_Sampling(t *testing.T) {
	recorder := recorder.NewReporter()
	tracer, err := NewTracer(
		recorder,
		zipkin.WithSampler(func(_ uint64) bool { return true }),
		zipkin.WithNoopSpan(true),
	)
	if err != nil {
		t.Fatalf("Unable to create Tracer: %+v", err)
	}

	span := tracer.StartSpan("x")
	span.Finish()
	assert.Equal(t, 1, len(recorder.Flush()), "by default span should be sampled")

	span = tracer.StartSpan("x")
	ext.SamplingPriority.Set(span, 0)
	span.Finish()
	assert.Equal(t, 0, len(recorder.Flush()), "SamplingPriority=0 should turn off sampling")

	tracer, err = NewTracer(
		recorder,
		zipkin.WithSampler(func(_ uint64) bool { return false }),
	)
	if err != nil {
		t.Fatalf("Unable to create Tracer: %+v", err)
	}

	recorder.Flush()
	span = tracer.StartSpan("x")
	span.Finish()
	assert.Equal(t, 0, len(recorder.Flush()), "by default span should not be sampled")

	span = tracer.StartSpan("x")
	ext.SamplingPriority.Set(span, 1)
	span.Finish()
	assert.Equal(t, 1, len(recorder.Flush()), "SamplingPriority=1 should turn on sampling")
}
