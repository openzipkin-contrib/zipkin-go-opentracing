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
	"fmt"
	"net"
	"time"

	otobserver "github.com/opentracing-contrib/go-observer"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/openzipkin/zipkin-go"
)

type spanImpl struct {
	tracer         *tracerImpl
	zipkinSpan     zipkin.Span
	observer       otobserver.SpanObserver
	zipkinObserver ZipkinSpanObserver
	options        ZipkinStartSpanOptions
}

func (s *spanImpl) SetOperationName(operationName string) opentracing.Span {
	if s.observer != nil {
		s.observer.OnSetOperationName(operationName)
	}

	if s.zipkinObserver != nil {
		s.zipkinObserver.OnSetName(operationName)
	}

	s.zipkinSpan.SetName(operationName)
	return s
}

func (s *spanImpl) SetTag(key string, value interface{}) opentracing.Span {
	if s.observer != nil {
		s.observer.OnSetTag(key, value)
	}

	endpointChanged := false

	switch key {
	case string(ext.SamplingPriority):
		// there are no means for now to change the sampling decision
		// but when finishedSpanHandler is in place we could change this.
		return s
	case string(ext.SpanKind):
		// this tag is translated into kind which can
		// only be set on span creation
		return s
	case string(ext.PeerService):
		serviceName, _ := value.(string)
		s.options.RemoteEndpoint.ServiceName = serviceName
		endpointChanged = true
	case string(ext.PeerHostIPv4):
		ipv4, _ := value.(string)
		s.options.RemoteEndpoint.IPv4 = net.ParseIP(ipv4)
		endpointChanged = true
	case string(ext.PeerHostIPv6):
		ipv6, _ := value.(string)
		s.options.RemoteEndpoint.IPv6 = net.ParseIP(ipv6)
		endpointChanged = true
	case string(ext.PeerPort):
		port, _ := value.(uint16)
		s.options.RemoteEndpoint.Port = port
		endpointChanged = true
	}

	if endpointChanged {
		s.zipkinSpan.SetRemoteEndpoint(s.options.RemoteEndpoint)

		if s.zipkinObserver != nil {
			s.zipkinObserver.OnSetRemoteEndpoint(s.options.RemoteEndpoint)
		}

		return s
	}

	strValue := fmt.Sprint(value)

	if s.zipkinObserver != nil {
		s.zipkinObserver.OnSetTag(key, strValue)
	}

	s.zipkinSpan.Tag(key, strValue)
	return s
}

func (s *spanImpl) LogKV(keyValues ...interface{}) {
	fields, err := log.InterleavedKVToFields(keyValues...)
	if err != nil {
		return
	}

	for _, field := range fields {
		t := time.Now()
		fieldValue := field.String()

		if s.zipkinObserver != nil {
			s.zipkinObserver.OnAnnotate(t, fieldValue)
		}

		s.zipkinSpan.Annotate(t, fieldValue)
	}
}

func (s *spanImpl) LogFields(fields ...log.Field) {
	s.logFields(time.Now(), fields...)
}

func (s *spanImpl) logFields(t time.Time, fields ...log.Field) {
	for _, field := range fields {
		annotation := field.String()

		if s.zipkinObserver != nil {
			s.zipkinObserver.OnAnnotate(t, annotation)
		}

		s.zipkinSpan.Annotate(t, annotation)
	}
}

func (s *spanImpl) LogEvent(event string) {
	s.Log(opentracing.LogData{
		Event: event,
	})
}

func (s *spanImpl) LogEventWithPayload(event string, payload interface{}) {
	s.Log(opentracing.LogData{
		Event:   event,
		Payload: payload,
	})
}

func (s *spanImpl) Log(ld opentracing.LogData) {
	if ld.Timestamp.IsZero() {
		ld.Timestamp = time.Now()
	}

	annotation := fmt.Sprintf("%s:%s", ld.Event, ld.Payload)

	if s.zipkinObserver != nil {
		s.zipkinObserver.OnAnnotate(ld.Timestamp, annotation)
	}

	s.zipkinSpan.Annotate(ld.Timestamp, annotation)
}

func (s *spanImpl) Finish() {
	if s.observer != nil {
		s.observer.OnFinish(opentracing.FinishOptions{})
	}

	if s.zipkinObserver != nil {
		s.zipkinObserver.OnFinish()
	}

	s.zipkinSpan.Finish()
}

func (s *spanImpl) FinishWithOptions(opts opentracing.FinishOptions) {
	if s.observer != nil {
		s.observer.OnFinish(opts)
	}

	for _, lr := range opts.LogRecords {
		s.logFields(lr.Timestamp, lr.Fields...)
	}

	if !opts.FinishTime.IsZero() {
		dur := opts.FinishTime.Sub(s.options.StartTime)

		if s.zipkinObserver != nil {
			s.zipkinObserver.OnFinishedWithDuration(dur)
		}

		s.zipkinSpan.FinishedWithDuration(dur)
		return
	}

	if s.zipkinObserver != nil {
		s.zipkinObserver.OnFinish()
	}

	s.zipkinSpan.Finish()
}

func (s *spanImpl) Tracer() opentracing.Tracer {
	return s.tracer
}

func (s *spanImpl) Context() opentracing.SpanContext {
	return SpanContext(s.zipkinSpan.Context())
}

func (s *spanImpl) SetBaggageItem(key, val string) opentracing.Span {
	return s
}

func (s *spanImpl) BaggageItem(key string) string {
	return ""
}
