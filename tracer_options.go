package zipkintracer

import (
	"net"

	"github.com/openzipkin/zipkin-go/model"

	otobserver "github.com/opentracing-contrib/go-observer"
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
	observer otobserver.Observer
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
