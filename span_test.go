// Copyright 2022 The OpenZipkin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
