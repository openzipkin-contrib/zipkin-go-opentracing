package zipkintracer

import "github.com/opentracing/opentracing-go"

// A SpanEvent is emitted when a mutating command is called on a Span.
type SpanEvent interface{}

// EventCreate is emitted when a Span is created.
type EventCreate struct{ OperationName string }

// EventTag is received when SetTag is called.
type EventTag struct {
	Key   string
	Value interface{}
}

// EventLog is received when Log (or one of its derivatives) is called.
type EventLog opentracing.LogData

// EventFinish is received when Finish is called.
type EventFinish RawSpan

func (s *spanImpl) onCreate(opName string) {
	if s.event != nil {
		s.event(EventCreate{OperationName: opName})
	}
}
func (s *spanImpl) onTag(key string, value interface{}) {
	if s.event != nil {
		s.event(EventTag{Key: key, Value: value})
	}
}
func (s *spanImpl) onLog(ld opentracing.LogData) {
	if s.event != nil {
		s.event(EventLog(ld))
	}
}
func (s *spanImpl) onFinish(sp RawSpan) {
	if s.event != nil {
		s.event(EventFinish(sp))
	}
}
