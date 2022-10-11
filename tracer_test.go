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

	"github.com/openzipkin/zipkin-go/model"
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

func TestOTKindTagIsParsedSuccessfuly(t *testing.T) {
	tagCases := []map[string]interface{}{
		{string(ext.SpanKind): "server"},
		{"span.kind": "server"},
		{"span.kind": ext.SpanKindRPCServerEnum},
	}
	for _, tags := range tagCases {
		opts := parseTagsAsZipkinOptions(tags)

		rec := recorder.NewReporter()
		tr, _ := zipkin.NewTracer(rec)
		sp := tr.StartSpan("test", opts...)
		sp.Finish()
		spans := rec.Flush()
		if want, have := 1, len(spans); want != have {
			t.Fatalf("unexpected number of spans, want %d, have %d", want, have)
		}

		if want, have := model.Server, spans[0].Kind; want != have {
			t.Errorf("unexpected kind value, want %s, have %s", want, have)
		}
	}
}

func TestOTKindTagIsCantBeParsed(t *testing.T) {
	tags := map[string]interface{}{"span.kind": "banana"}
	opts := parseTagsAsZipkinOptions(tags)

	rec := recorder.NewReporter()
	tr, _ := zipkin.NewTracer(rec)
	sp := tr.StartSpan("test", opts...)
	sp.Finish()
	spans := rec.Flush()
	if want, have := 1, len(spans); want != have {
		t.Fatalf("unexpected number of spans, want %d, have %d", want, have)
	}

	if want, have := model.Undetermined, spans[0].Kind; want != have {
		t.Errorf("unexpected kind value, want %s, have %s", want, have)
	}

	if want, have := "banana", spans[0].Tags["span.kind"]; want != have {
		t.Errorf("unexpected tag value, want %s, have %s", want, have)
	}
}

func TestOptionsFromOTTags(t *testing.T) {
	tags := map[string]interface{}{}
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

	if want, have := "service_a", spans[0].RemoteEndpoint.ServiceName; want != have {
		t.Errorf("unexpected remote service name, want %s, have %s", want, have)
	}

	if want, have := "value", spans[0].Tags["key"]; want != have {
		t.Errorf("unexpected tag value, want %s, have %s", want, have)
	}
}
