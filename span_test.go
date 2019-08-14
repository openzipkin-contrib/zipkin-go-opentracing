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
	tracer := newTracer(
		recorder,
		zipkin.WithSampler(zipkin.AlwaysSample),
	)

	span := tracer.StartSpan("x")
	span.LogEventWithPayload("key1", "{\"user\": 123}")
	span.LogFields(log.String("key2", "value2"), log.Uint32("32bit", 4294967295))
	span.SetTag("key3", "value3")
	span.Finish()
	spans := recorder.Flush()
	assert.Equal(t, 1, len(spans))
	assert.Equal(t, "x", spans[0].Name)
	assert.Equal(t, 3, len(spans[0].Annotations))
	assert.Equal(t, map[string]string{"key3": "value3"}, spans[0].Tags)
	assert.Equal(t, spans[0].Annotations[0].Value, "key1:{\"user\": 123}")
	assert.Equal(t, spans[0].Annotations[1].Value, "key2:value2")
	assert.Equal(t, spans[0].Annotations[2].Value, "32bit:4294967295")
}

func TestSpan_TrimUnsampledSpans(t *testing.T) {
	recorder := recorder.NewReporter()
	// Tracer that trims only unsampled but always samples
	tracer := newTracer(
		recorder,
		zipkin.WithSampler(func(_ uint64) bool { return true }),
		zipkin.WithNoopSpan(true),
	)

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
	tracer = newTracer(
		recorder,
		zipkin.WithSampler(zipkin.NeverSample),
		zipkin.WithNoopSpan(true),
	)

	span = tracer.StartSpan("x")
	span.LogFields(log.String("key_str", "value"), log.Uint32("32bit", 4294967295))
	span.SetTag("tag", "value")
	span.Finish()
	spans = recorder.Flush()
	assert.Equal(t, 0, len(spans))
}

func TestTagTranslation(t *testing.T) {
	recorder := recorder.NewReporter()
	// Tracer that trims only unsampled but always samples
	tracer := newTracer(recorder)

	span := tracer.StartSpan("x")
	span.SetTag("db.statement", "SELECT 1")
	span.SetTag("db.type", "mysql")
	span.Finish()
	spans := recorder.Flush()
	assert.Equal(t, 1, len(spans))
	assert.Equal(t, map[string]string{"sql.query": "SELECT 1", "db.type": "mysql"}, spans[0].Tags)
}
