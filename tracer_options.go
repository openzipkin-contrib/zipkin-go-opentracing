package zipkintracer

import (
	"net"

	"github.com/openzipkin/zipkin-go/model"

	otobserver "github.com/opentracing-contrib/go-observer"
	"github.com/openzipkin/zipkin-go"
)

// Endpoint holds the network context of a node in the service graph.
type Endpoint struct {
	ServiceName string
	IPv4        net.IP
	IPv6        net.IP
	Port        uint16
}

func (e *Endpoint) toZipkin() *model.Endpoint {
	return &model.Endpoint{
		ServiceName: e.ServiceName,
		IPv4:        e.IPv4,
		IPv6:        e.IPv6,
		Port:        e.Port,
	}
}

// TracerOptions allows creating a customized Tracer.
type TracerOptions struct {
	localEndpoint *model.Endpoint

	// shouldSample is a function which is called when creating a new Span and
	// determines whether that Span is sampled. The randomized TraceID is supplied
	// to allow deterministic sampling decisions to be made across different nodes.
	sampler zipkin.Sampler

	// unsampledNoop turns potentially expensive operations on unsampled
	// Spans into no-ops. More precisely, tags and log events are silently
	// discarded. If NewSpanEventListener is set, the callbacks will still fire.
	unsampledNoop *bool

	// clientServerSameSpan allows for Zipkin V1 style span per RPC. This places
	// both client end and server end of a RPC call into the same span.
	sharedSpans *bool

	// traceID128Bit enables the generation of 128 bit traceIDs in case the tracer
	// needs to create a root span. By default regular 64 bit traceIDs are used.
	// Regardless of this setting, the library will propagate and support both
	// 64 and 128 bit incoming traces from upstream sources.
	traceID128Bit *bool

	noop *bool

	defaultTags map[string]string

	observer otobserver.Observer
}

// TracerOption allows for functional options.
// See: http://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type TracerOption func(opts *TracerOptions) error

// WithLocalEndpoint sets the local endpoint of the tracer.
func WithLocalEndpoint(e Endpoint) TracerOption {
	return func(opts *TracerOptions) error {
		opts.localEndpoint = e.toZipkin()
		return nil
	}
}

// WithSampler allows one to add a Sampler function
func WithSampler(sampler func(traceID uint64) bool) TracerOption {
	return func(opts *TracerOptions) error {
		opts.sampler = sampler
		return nil
	}
}

// TrimUnsampledSpans option, similar to WithNoopSpan
// in zipkin options
func TrimUnsampledSpans(trim bool) TracerOption {
	return func(opts *TracerOptions) error {
		opts.unsampledNoop = &trim
		return nil
	}
}

// TraceID128Bit option
func TraceID128Bit(val bool) TracerOption {
	return func(opts *TracerOptions) error {
		opts.traceID128Bit = &val
		return nil
	}
}

// ClientServerSameSpan allows to place client-side and server-side annotations
// for a RPC call in the same span (Zipkin V1 behavior) or different spans
// (more in line with other tracing solutions). By default this Tracer
// uses shared host spans (so client-side and server-side in the same span).
// If using separate spans you might run into trouble with Zipkin V1 as clock
// skew issues can't be remedied at Zipkin server side.
//
// similar to zipkin.WithSharedSpans
func ClientServerSameSpan(val bool) TracerOption {
	return func(opts *TracerOptions) error {
		opts.sharedSpans = &val
		return nil
	}
}

// WithTags allows one to set default tags to be added to each created span
func WithTags(tags map[string]string) TracerOption {
	return func(opts *TracerOptions) error {
		for k, v := range tags {
			opts.defaultTags[k] = v
		}
		return nil
	}
}

// WithNoopTracer allows one to start the Tracer as Noop implementation.
func WithNoopTracer(tracerNoop bool) TracerOption {
	return func(opts *TracerOptions) error {
		opts.noop = &tracerNoop
		return nil
	}
}

// WithObserver assigns an initialized observer to opts.observer
func WithObserver(observer otobserver.Observer) TracerOption {
	return func(opts *TracerOptions) error {
		opts.observer = observer
		return nil
	}
}

func (to TracerOptions) toZipkinTraceOptions() []zipkin.TracerOption {
	zto := []zipkin.TracerOption{}

	if to.localEndpoint != nil {
		zto = append(zto, zipkin.WithLocalEndpoint(to.localEndpoint))
	}

	if to.sampler != nil {
		zto = append(zto, zipkin.WithSampler(to.sampler))
	}

	if to.unsampledNoop != nil {
		zto = append(zto, zipkin.WithNoopSpan(*to.unsampledNoop))
	}

	if to.sharedSpans != nil {
		zto = append(zto, zipkin.WithSharedSpans(*to.sharedSpans))
	}

	if to.traceID128Bit != nil {
		zto = append(zto, zipkin.WithTraceID128Bit(*to.traceID128Bit))
	}

	if to.noop != nil {
		zto = append(zto, zipkin.WithNoopTracer(*to.noop))
	}

	if to.defaultTags != nil {
		zto = append(zto, zipkin.WithTags(to.defaultTags))
	}

	return zto
}
