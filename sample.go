package zipkintracer

import (
	"github.com/openzipkin/zipkin-go"
)

// Sampler functions return if a Zipkin span should be sampled, based on its
// traceID.
type Sampler func(id uint64) bool

var (
	NeverSample        = zipkin.NeverSample
	AlwaysSample       = zipkin.AlwaysSample
	NewModuloSampler   = zipkin.NewModuloSampler
	NewBoundarySampler = zipkin.NewBoundarySampler
	NewCountingSampler = zipkin.NewCountingSampler
)
