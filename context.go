package zipkintracer

import (
	"sync"

	"github.com/opentracing/opentracing-go"

	"github.com/openzipkin/zipkin-go-opentracing/flag"
)

// SpanContext holds the BasicSpan metadata that propagates across process
// boundaries and satisfies the opentracing.SpanContext interface.
type SpanContext struct {
	// A probabilistically unique identifier for a [multi-span] trace.
	TraceID uint64

	// A probabilistically unique identifier for a span.
	SpanID uint64

	// Whether the trace is sampled.
	Sampled bool

	// The span's associated baggage.
	baggageLock sync.Mutex
	Baggage     map[string]string // initialized on first use

	// The SpanID of this Context's parent, or nil if there is no parent.
	ParentSpanID *uint64

	// Flags provides the ability to create and communicate feature flags.
	Flags flag.Flags
}

// BaggageItem belongs to the opentracing.SpanContext interface
func (c *SpanContext) BaggageItem(key string) string {
	// TODO: if we want to support onBaggage, need a pointer to the bt.Span.
	//   s.onBaggage(canonicalKey, val)
	//   if s.trim() {
	//   	return s
	//   }

	c.baggageLock.Lock()
	defer c.baggageLock.Unlock()

	if c.Baggage == nil {
		return ""
	}
	return c.Baggage[key]
}

// SetBaggageItem belongs to the opentracing.SpanContext interface
func (c *SpanContext) SetBaggageItem(key, val string) opentracing.SpanContext {
	c.baggageLock.Lock()
	defer c.baggageLock.Unlock()
	if c.Baggage == nil {
		c.Baggage = make(map[string]string)
	}
	c.Baggage[key] = val
	return c
}

// ForeachBaggageItem belongs to the opentracing.SpanContext interface
func (c *SpanContext) ForeachBaggageItem(handler func(k, v string) bool) {
	c.baggageLock.Lock()
	defer c.baggageLock.Unlock()
	for k, v := range c.Baggage {
		if !handler(k, v) {
			break
		}
	}
}
