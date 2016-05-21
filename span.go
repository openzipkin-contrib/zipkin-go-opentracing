package zipkintracer

import (
	"fmt"
	"sync"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/openzipkin/zipkin-go-opentracing/_thrift/gen-go/zipkincore"
)

// Span provides access to the essential details of the span, for use
// by zipkintracer consumers.  These methods may only be called prior
// to (*opentracing.Span).Finish().
type Span interface {
	opentracing.Span

	// Context contains trace identifiers
	Context() Context

	// Operation names the work done by this span instance
	Operation() string

	// Start indicates when the span began
	Start() time.Time
}

// Implements the `Span` interface. Created via tracerImpl (see
// `zipkintracer.NewTracer()`).
type spanImpl struct {
	tracer     *tracerImpl
	event      func(SpanEvent)
	sync.Mutex // protects the fields below
	raw        RawSpan
	Endpoint   *zipkincore.Endpoint
	sampled    bool
}

var spanPool = &sync.Pool{New: func() interface{} {
	return &spanImpl{}
}}

func (s *spanImpl) reset() {
	s.tracer, s.event = nil, nil
	// Note: Would like to do the following, but then the consumer of RawSpan
	// (the recorder) needs to make sure that they're not holding on to the
	// baggage or logs when they return (i.e. they need to copy if they care):
	//
	// logs, baggage := s.raw.Logs[:0], s.raw.Baggage
	// for k := range baggage {
	// 	delete(baggage, k)
	// }
	// s.raw.Logs, s.raw.Baggage = logs, baggage
	//
	// That's likely too much to ask for. But there is some magic we should
	// be able to do with `runtime.SetFinalizer` to reclaim that memory into
	// a buffer pool when GC considers them unreachable, which should ease
	// some of the load. Hard to say how quickly that would be in practice
	// though.
	s.raw = RawSpan{}
}

func (s *spanImpl) SetOperationName(operationName string) opentracing.Span {
	s.Lock()
	defer s.Unlock()
	s.raw.Operation = operationName
	return s
}

func (s *spanImpl) trim() bool {
	return !s.raw.Sampled && s.tracer.options.trimUnsampledSpans
}

func (s *spanImpl) SetTag(key string, value interface{}) opentracing.Span {
	defer s.onTag(key, value)
	s.Lock()
	defer s.Unlock()
	if key == string(ext.SamplingPriority) {
		s.raw.Sampled = true
		return s
	}
	if s.trim() {
		return s
	}

	if s.raw.Tags == nil {
		s.raw.Tags = opentracing.Tags{}
	}
	s.raw.Tags[key] = value
	return s
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
	defer s.onLog(ld)
	s.Lock()
	defer s.Unlock()
	if s.trim() {
		return
	}

	if ld.Timestamp.IsZero() {
		ld.Timestamp = time.Now()
	}

	s.raw.Logs = append(s.raw.Logs, ld)
}

func (s *spanImpl) Finish() {
	s.FinishWithOptions(opentracing.FinishOptions{})
}

func (s *spanImpl) FinishWithOptions(opts opentracing.FinishOptions) {
	finishTime := opts.FinishTime
	if finishTime.IsZero() {
		finishTime = time.Now()
	}
	duration := finishTime.Sub(s.raw.Start)

	s.Lock()
	defer s.Unlock()
	if opts.BulkLogData != nil {
		s.raw.Logs = append(s.raw.Logs, opts.BulkLogData...)
	}
	s.raw.Duration = duration

	s.onFinish(s.raw)
	s.tracer.options.recorder.RecordSpan(s.raw)
	if s.tracer.options.debugAssertUseAfterFinish {
		// This makes it much more likely to catch a panic on any subsequent
		// operation since s.tracer is accessed on every call to `Lock`.
		s.reset()
	}
	spanPool.Put(s)
}

func (s *spanImpl) SetBaggageItem(restrictedKey, val string) opentracing.Span {
	canonicalKey, valid := opentracing.CanonicalizeBaggageKey(restrictedKey)
	if !valid {
		panic(fmt.Errorf("Invalid key: %q", restrictedKey))
	}

	s.Lock()
	defer s.Unlock()
	s.onBaggage(canonicalKey, val)
	if s.trim() {
		return s
	}

	if s.raw.Baggage == nil {
		s.raw.Baggage = make(map[string]string)
	}
	s.raw.Baggage[canonicalKey] = val
	return s
}

func (s *spanImpl) BaggageItem(restrictedKey string) string {
	canonicalKey, valid := opentracing.CanonicalizeBaggageKey(restrictedKey)
	if !valid {
		panic(fmt.Errorf("Invalid key: %q", restrictedKey))
	}

	s.Lock()
	defer s.Unlock()

	return s.raw.Baggage[canonicalKey]
}

func (s *spanImpl) Tracer() opentracing.Tracer {
	return s.tracer
}

func (s *spanImpl) Context() Context {
	return s.raw.Context
}

func (s *spanImpl) Operation() string {
	return s.raw.Operation
}

func (s *spanImpl) Start() time.Time {
	return s.raw.Start
}
