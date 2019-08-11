package zipkintracer

import (
	"sync/atomic"

	"github.com/openzipkin/zipkin-go/model"
)

type CountingSender int32

func (c *CountingSender) Send(span model.SpanModel) {
	atomic.AddInt32((*int32)(c), 1)
}

func (c *CountingSender) Close() error {
	return nil
}
