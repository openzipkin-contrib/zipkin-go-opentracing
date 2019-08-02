package zipkintracer

// Sampler functions return if a Zipkin span should be sampled, based on its
// traceID.
type Sampler func(id uint64) bool
