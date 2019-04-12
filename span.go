package zipkintracer

import (
	"fmt"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/openzipkin/zipkin-go"
)

type FinisherAt interface {
	FinishedAt(t time.Time)
}

type spanImpl struct {
	tracer     *tracerImpl
	zipkinSpan zipkin.Span
}

func (s *spanImpl) SetOperationName(operationName string) opentracing.Span {
	s.zipkinSpan.SetName(operationName)
	return s
}

func (s *spanImpl) SetTag(key string, value interface{}) opentracing.Span {
	s.zipkinSpan.Tag(key, fmt.Sprint(value))
	return s
}

func (s *spanImpl) LogKV(keyValues ...interface{}) {
	fields, err := log.InterleavedKVToFields(keyValues...)
	if err != nil {
		return
	}

	for _, field := range fields {
		s.zipkinSpan.Annotate(time.Now(), field.String())
	}
}

func (s *spanImpl) LogFields(fields ...log.Field) {
	s.logFields(time.Now(), fields)
}

func (s *spanImpl) logFields(t time.Time, fields ...log.Field) {
	for _, field := range fields {
		s.zipkinSpan.Annotate(t, field.String())
	}
}

func (s *spanImpl) LogEvent(event string) {
	// Deprecated: do nothing
}

func (s *spanImpl) LogEventWithPayload(event string, payload interface{}) {
	// Deprecated: do nothing
}

func (s *spanImpl) Log(ld opentracing.LogData) {
	// Deprecated: do nothing
}

func (s *spanImpl) Finish() {
	s.zipkinSpan.Finish()
}

func (s *spanImpl) FinishWithOptions(opts opentracing.FinishOptions) {

	for _, lr := range opts.LogRecords {
		s.logFields(lr.Timestamp, lr.Fields)
	}

	if opts.FinishTime != nil {
		f, ok := s.zipkinSpan.(FinisherAt)
		if !ok {
			return
		}
		f.FinishedAt(opts.FinishTime)
		return
	}

	f.Finish()
}

func (s *spanImpl) Tracer() opentracing.Tracer {
	return s.tracerImpl
}

func (s *spanImpl) Context() opentracing.SpanContext {
	return &spanContextImpl{zipkinContext: s.zipkinSpan.Context()}

}

func (s *spanImpl) SetBaggageItem(key, val string) opentracing.Span {
	// Do nothing for now
	return s
}

func (s *spanImpl) BaggageItem(key string) string {
	return ""
}
