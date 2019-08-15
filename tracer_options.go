package zipkintracer

import (
	otobserver "github.com/opentracing-contrib/go-observer"
)

// B3InjectOption type holds information on B3 injection style when using
// native OpenTracing HTTPHeadersCarrier.
type B3InjectOption int

// Available B3InjectOption values
const (
	B3InjectStandard B3InjectOption = iota
	B3InjectSingle
	B3InjectBoth
)

// TracerOptions allows creating a customized Tracer.
type TracerOptions struct {
	observer    otobserver.Observer
	b3InjectOpt B3InjectOption
}

// TracerOption allows for functional options.
// See: http://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type TracerOption func(opts *TracerOptions)

// WithObserver assigns an initialized observer to opts.observer
func WithObserver(observer otobserver.Observer) TracerOption {
	return func(opts *TracerOptions) {
		opts.observer = observer
	}
}

// WithB3InjectOption sets the B3 injection style if using the native OpenTracing HTTPHeadersCarrier
func WithB3InjectOption(b3InjectOption B3InjectOption) TracerOption {
	return func(opts *TracerOptions) {
		opts.b3InjectOpt = b3InjectOption
	}
}
