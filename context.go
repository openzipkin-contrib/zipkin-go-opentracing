package zipkintracer

import (
	"sync"

	"github.com/opentracing/opentracing-go"
)

// Context holds the BasicSpan metadata that propagates across process
// boundaries and satisfies the opentracing.SpanContext interface.
type Context struct {
	sync.Mutex

	// A probabilistically unique identifier for a [multi-span] trace.
	TraceID uint64

	// A probabilistically unique identifier for a span.
	SpanID uint64

	// The SpanID of this Context's parent, or 0 if there is no parent.
	ParentSpanID uint64

	// Whether the trace is sampled.
	Sampled bool

	// The baggage that propagates along with the Trace.
	Baggage map[string]string // initialized on first use
}

// SetBaggageItem is part of the opentracing.SpanContext interface
func (c *Context) SetBaggageItem(key, val string) opentracing.SpanContext {
	c.Lock()
	defer c.Unlock()

	if c.Baggage == nil {
		c.Baggage = make(map[string]string)
	}
	c.Baggage[key] = val
	return c
}

// BaggageItem is part of the opentracing.SpanContext interface
func (c *Context) BaggageItem(key string) string {
	c.Lock()
	defer c.Unlock()

	return c.Baggage[key]
}

// ForeachBaggageItem is part of the opentracing.SpanContext interface
func (c *Context) ForeachBaggageItem(handler func(k, v string) bool) {
	c.Lock()
	defer c.Unlock()
	for k, v := range c.Baggage {
		if !handler(k, v) {
			break
		}
	}
}
