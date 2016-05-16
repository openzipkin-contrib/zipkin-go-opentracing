package zipkintracer

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// InMemoryRecorder is a simple thread-safe implementation of
// SpanRecorder that stores all reported spans in memory, accessible
// via reporter.GetSpans()
type InMemoryRecorder struct {
	spans []RawSpan
	lock  sync.Mutex
}

// NewInMemoryRecorder instantiates a new InMemoryRecorder for testing purposes.
func NewInMemoryRecorder() *InMemoryRecorder {
	return &InMemoryRecorder{
		spans: make([]RawSpan, 0),
	}
}

// RecordSpan implements RecordSpan() of SpanRecorder.
//
// The recorded spans can be retrieved via recorder.Spans slice.
func (recorder *InMemoryRecorder) RecordSpan(span RawSpan) {
	recorder.lock.Lock()
	defer recorder.lock.Unlock()
	recorder.spans = append(recorder.spans, span)
}

// GetSpans returns a snapshot of spans recorded so far.
func (recorder *InMemoryRecorder) GetSpans() []RawSpan {
	recorder.lock.Lock()
	defer recorder.lock.Unlock()
	spans := make([]RawSpan, len(recorder.spans))
	copy(spans, recorder.spans)
	return spans
}

func TestInMemoryRecorderSpans(t *testing.T) {
	recorder := NewInMemoryRecorder()
	var apiRecorder SpanRecorder = recorder
	span := RawSpan{
		Context:   Context{},
		Operation: "test-span",
		Start:     time.Now(),
		Duration:  -1,
	}
	apiRecorder.RecordSpan(span)
	if len(recorder.GetSpans()) != 1 {
		t.Fatal("No spans recorded")
	}
	if !reflect.DeepEqual(recorder.GetSpans()[0], span) {
		t.Fatal("Span not recorded")
	}
}

type CountingRecorder int32

func (c *CountingRecorder) RecordSpan(r RawSpan) {
	atomic.AddInt32((*int32)(c), 1)
}
