// Copyright 2019 The OpenZipkin Authors
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
		var zipkinStartSpanOptions ZipkinStartSpanOptions

		opts := parseTagsAsZipkinOptions(tags, &zipkinStartSpanOptions)

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

		if zipkinStartSpanOptions.Tags == nil {
			t.Errorf("unexpected start options tags value, want non-nil map, have %v", zipkinStartSpanOptions.Tags)
		}

		if len(zipkinStartSpanOptions.Tags) != 0 {
			t.Errorf("unexpected start options tags value, want empty map, have %v", zipkinStartSpanOptions.Tags)
		}

		if zipkinStartSpanOptions.RemoteEndpoint == nil {
			t.Errorf("unexpected start options remote endpoint value, want non-nil instance, have %v", zipkinStartSpanOptions.RemoteEndpoint)
		}

		if !zipkinStartSpanOptions.RemoteEndpoint.Empty() {
			t.Errorf("unexpected start options remote endpoint value, want empty instance, have %v", zipkinStartSpanOptions.RemoteEndpoint)
		}

		if want, have := model.Server, zipkinStartSpanOptions.Kind; want != have {
			t.Errorf("unexpected start options kind value, want %s, have %s", want, have)
		}
	}
}

func TestOTKindTagIsCantBeParsed(t *testing.T) {
	var zipkinStartSpanOptions ZipkinStartSpanOptions

	tags := map[string]interface{}{"span.kind": "banana"}
	opts := parseTagsAsZipkinOptions(tags, &zipkinStartSpanOptions)

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

	if zipkinStartSpanOptions.Tags == nil {
		t.Errorf("unexpected start options tags value, want non-nil map, have %v", zipkinStartSpanOptions.Tags)
	}

	if len(zipkinStartSpanOptions.Tags) == 0 {
		t.Errorf("unexpected start options tags value, want non-empty map, have %v", zipkinStartSpanOptions.Tags)
	}

	if want, have := "banana", zipkinStartSpanOptions.Tags["span.kind"]; want != have {
		t.Errorf("unexpected start options tags[span.kind] value, want %s, have %s", want, have)
	}
}

func TestOptionsFromOTTags(t *testing.T) {
	var zipkinStartSpanOptions ZipkinStartSpanOptions

	const (
		sServiceA = "service_a"
		sValue    = "value"
		sKey      = "key"
	)

	tags := map[string]interface{}{}
	tags[string(ext.PeerService)] = sServiceA
	tags[sKey] = sValue
	opts := parseTagsAsZipkinOptions(tags, &zipkinStartSpanOptions)

	rec := recorder.NewReporter()
	tr, _ := zipkin.NewTracer(rec)
	sp := tr.StartSpan("test", opts...)
	sp.Finish()
	spans := rec.Flush()
	if want, have := 1, len(spans); want != have {
		t.Fatalf("unexpected number of spans, want %d, have %d", want, have)
	}

	if want, have := sServiceA, spans[0].RemoteEndpoint.ServiceName; want != have {
		t.Errorf("unexpected remote service name, want %s, have %s", want, have)
	}

	if want, have := sValue, spans[0].Tags[sKey]; want != have {
		t.Errorf("unexpected tag value, want %s, have %s", want, have)
	}

	if zipkinStartSpanOptions.Tags == nil {
		t.Errorf("unexpected start options tags value, want non-nil map, have %s", zipkinStartSpanOptions.Tags)
	}

	if len(zipkinStartSpanOptions.Tags) == 0 {
		t.Errorf("unexpected start options tags value, want non-empty map, have %s", zipkinStartSpanOptions.Tags)
	}

	if want, have := sValue, zipkinStartSpanOptions.Tags[sKey]; want != have {
		t.Errorf("unexpected start options tags[key] value, want %s, have %s", want, have)
	}

	if zipkinStartSpanOptions.RemoteEndpoint == nil {
		t.Errorf("unexpected start options remote endpoint value, want non-nil instance, have %v", zipkinStartSpanOptions.RemoteEndpoint)
	}

	if zipkinStartSpanOptions.RemoteEndpoint.Empty() {
		t.Errorf("unexpected start options remote endpoint value, want non-empty instance, have %v", zipkinStartSpanOptions.RemoteEndpoint)
	}

	if want, have := sServiceA, zipkinStartSpanOptions.RemoteEndpoint.ServiceName; want != have {
		t.Errorf("unexpected start options remote service name, want %s, have %s", want, have)
	}
}
