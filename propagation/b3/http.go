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

package b3

import (
	"github.com/opentracing/opentracing-go"
	"github.com/openzipkin/zipkin-go/model"
	zb3 "github.com/openzipkin/zipkin-go/propagation/b3"
)

const (
	traceIDHeader      = "x-b3-traceid"
	spanIDHeader       = "x-b3-spanid"
	parentSpanIDHeader = "x-b3-parentspanid"
	sampledHeader      = "x-b3-sampled"
	flagsHeader        = "x-b3-flags"
)

func InjectHTTP(sc model.SpanContext, carrier interface{}) error {
	c, ok := carrier.(opentracing.TextMapWriter)
	if !ok {
		return opentracing.ErrInvalidCarrier
	}

	if (model.SpanContext{}) == sc {
		return zb3.ErrEmptyContext
	}

	if !sc.TraceID.Empty() && sc.ID > 0 {
		c.Set(traceIDHeader, sc.TraceID.String())
		c.Set(spanIDHeader, sc.ID.String())
		if sc.ParentID != nil {
			c.Set(parentSpanIDHeader, sc.ParentID.String())
		}
	}

	if sc.Debug {
		c.Set(flagsHeader, "1")
	} else if sc.Sampled != nil {
		if *sc.Sampled {
			c.Set(sampledHeader, "1")
		} else {
			c.Set(sampledHeader, "0")
		}
	}

	return nil
}

func ExtractHTTP(carrier interface{}) (*model.SpanContext, error) {
	c, ok := carrier.(opentracing.TextMapReader)
	if !ok {
		return nil, opentracing.ErrInvalidCarrier
	}

	var (
		traceID      string
		spanID       string
		parentSpanID string
		sampled      string
		flags        string
	)

	err := c.ForeachKey(func(key, val string) error {
		switch key {
		case traceIDHeader:
			traceID = val
		case spanIDHeader:
			spanID = val
		case parentSpanIDHeader:
			parentSpanID = val
		case sampledHeader:
			sampled = val
		case flagsHeader:
			flags = val
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return zb3.ParseHeaders(traceID, spanID, parentSpanID, sampled, flags)
}
