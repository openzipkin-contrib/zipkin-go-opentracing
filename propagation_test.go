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

package zipkintracer_test

import (
	stdHTTP "net/http"
	"testing"

	"github.com/opentracing/opentracing-go"
	zipkintracer "github.com/openzipkin-contrib/zipkin-go-opentracing"
	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	zb3 "github.com/openzipkin/zipkin-go/propagation/b3"
	"github.com/openzipkin/zipkin-go/reporter"
)

func TestHTTPExtractFlagsOnly(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}
	c.Set(zb3.Flags, "1")

	spanContext, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))
	if err != nil {
		t.Fatalf("Extract failed: %+v", err)
	}

	sc, ok := spanContext.(zipkintracer.SpanContext)
	if !ok {
		t.Fatal("Expected valid SpanContext")
	}

	if want, have := true, sc.Debug; want != have {
		t.Errorf("sc.Debug want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractSampledOnly(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}
	c.Set(zb3.Sampled, "0")

	spanContext, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))
	if err != nil {
		t.Fatalf("Extract failed: %+v", err)
	}

	sc, ok := spanContext.(zipkintracer.SpanContext)
	if !ok {
		t.Fatal("Expected valid SpanContext")
	}

	if sc.Sampled == nil {
		t.Fatalf("Sampled want %t, have nil", false)
	}

	if want, have := false, *sc.Sampled; want != have {
		t.Errorf("Sampled want %t, have %t", want, have)
	}

	c = stdHTTP.Header{}
	c.Set(zb3.Sampled, "1")

	spanContext, err = tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))
	if err != nil {
		t.Fatalf("Extract failed: %+v", err)
	}

	sc, ok = spanContext.(zipkintracer.SpanContext)
	if !ok {
		t.Fatal("Expected valid SpanContext")
	}

	if sc.Sampled == nil {
		t.Fatalf("Sampled want %t, have nil", true)
	}

	if want, have := true, *sc.Sampled; want != have {
		t.Errorf("Sampled want %t, have %t", want, have)
	}
}

func TestHTTPExtractFlagsAndSampledOnly(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}
	c.Set(zb3.Flags, "1")
	c.Set(zb3.Sampled, "1")

	spanContext, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))
	if err != nil {
		t.Fatalf("Extract failed: %+v", err)
	}

	sc, ok := spanContext.(zipkintracer.SpanContext)
	if !ok {
		t.Fatal("Expected valid SpanContext")
	}

	if want, have := true, sc.Debug; want != have {
		t.Errorf("Debug want %+v, have %+v", want, have)
	}

	// Sampled should not be set when sc.Debug is set.
	if sc.Sampled != nil {
		t.Errorf("Sampled want nil, have %+v", *sc.Sampled)
	}
}

func TestHTTPExtractSampledErrors(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}
	c.Set(zb3.Sampled, "2")

	spanContext, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))

	sc, ok := spanContext.(zipkintracer.SpanContext)
	if !ok {
		t.Fatal("Expected valid SpanContext")
	}

	if want, have := zb3.ErrInvalidSampledHeader, err; want != have {
		t.Errorf("SpanContext Error want %+v, have %+v", want, have)
	}

	if sc != (zipkintracer.SpanContext{}) {
		t.Errorf("SpanContext want empty, have: %+v", sc)
	}
}

func TestHTTPExtractFlagsErrors(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	values := map[string]bool{
		"1":    true,  // only acceptable Flags value, debug switches to true
		"true": false, // true is not a valid value for Flags
		"3":    false, // Flags is not a bitset
		"6":    false, // Flags is not a bitset
		"7":    false, // Flags is not a bitset
	}
	for value, debug := range values {
		c := stdHTTP.Header{}
		c.Set(zb3.Flags, value)
		spanContext, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))
		if err != nil {
			// Flags should not trigger failed extraction
			t.Fatalf("Extract failed: %+v", err)
		}

		sc, ok := spanContext.(zipkintracer.SpanContext)
		if !ok {
			t.Fatal("Expected valid SpanContext")
		}

		if want, have := debug, sc.Debug; want != have {
			t.Errorf("SpanContext Error want %t, have %t", want, have)
		}
	}
}

func newTracer(r reporter.Reporter, opts ...zipkin.TracerOption) opentracing.Tracer {
	tr, _ := zipkin.NewTracer(r, opts...)
	return zipkintracer.Wrap(tr)
}

func TestHTTPExtractTraceIDError(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}
	c.Set(zb3.TraceID, "invalid_data")

	_, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))

	if want, have := zb3.ErrInvalidTraceIDHeader, err; want != have {
		t.Errorf("Extract Error want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractSpanIDError(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}
	c.Set(zb3.SpanID, "invalid_data")

	_, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))

	if want, have := zb3.ErrInvalidSpanIDHeader, err; want != have {
		t.Errorf("Extract Error want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractTraceIDOnlyError(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}
	c.Set(zb3.TraceID, "1")

	_, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))

	if want, have := zb3.ErrInvalidScope, err; want != have {
		t.Errorf("Extract Error want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractSpanIDOnlyError(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}
	c.Set(zb3.SpanID, "1")

	_, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))

	if want, have := zb3.ErrInvalidScope, err; want != have {
		t.Errorf("Extract Error want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractParentIDOnlyError(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}
	c.Set(zb3.ParentSpanID, "1")

	_, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))

	if want, have := zb3.ErrInvalidScopeParent, err; want != have {
		t.Errorf("Extract Error want %+v, have %+v", want, have)
	}
}

func TestHTTPExtractInvalidParentIDError(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}
	c.Set(zb3.TraceID, "1")
	c.Set(zb3.SpanID, "2")
	c.Set(zb3.ParentSpanID, "invalid_data")

	_, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))

	if want, have := zb3.ErrInvalidParentSpanIDHeader, err; want != have {
		t.Errorf("Extract Error want %+v, have %+v", want, have)
	}

}

func TestHTTPInjectEmptyContextError(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	err := tracer.Inject(zipkintracer.SpanContext{}, opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier{})

	if want, have := zb3.ErrEmptyContext, err; want != have {
		t.Errorf("HTTPInject Error want %+v, have %+v", want, have)
	}
}

func TestHTTPInjectDebugOnly(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}
	sc := zipkintracer.SpanContext{
		Debug: true,
	}

	tracer.Inject(sc, opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))

	if want, have := "1", c.Get(zb3.Flags); want != have {
		t.Errorf("Flags want %s, have %s", want, have)
	}
}

func TestHTTPInjectSampledOnly(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}

	sampled := false
	sc := zipkintracer.SpanContext{
		Sampled: &sampled,
	}

	tracer.Inject(sc, opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))

	if want, have := "0", c.Get(zb3.Sampled); want != have {
		t.Errorf("Sampled want %s, have %s", want, have)
	}
}

func TestHTTPInjectUnsampledTrace(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}
	sampled := false
	sc := zipkintracer.SpanContext{
		TraceID: model.TraceID{Low: 1},
		ID:      model.ID(2),
		Sampled: &sampled,
	}

	tracer.Inject(sc, opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))

	if want, have := "0", c.Get(zb3.Sampled); want != have {
		t.Errorf("Sampled want %s, have %s", want, have)
	}
}

func TestHTTPInjectSampledAndDebugTrace(t *testing.T) {
	tracer := zipkintracer.Wrap(nil)
	c := stdHTTP.Header{}

	sampled := true
	sc := zipkintracer.SpanContext{
		TraceID: model.TraceID{Low: 1},
		ID:      model.ID(2),
		Debug:   true,
		Sampled: &sampled,
	}

	tracer.Inject(sc, opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c))

	if want, have := "", c.Get(zb3.Sampled); want != have {
		t.Errorf("Sampled want empty, have %s", have)
	}

	if want, have := "1", c.Get(zb3.Flags); want != have {
		t.Errorf("Debug want %s, have %s", want, have)
	}
}
