package zipkintracer

import (
	"sync/atomic"

	"github.com/openzipkin/zipkin-go/model"
)

type CountingRecorder int32

func (c *CountingRecorder) Send(span model.SpanModel) {
	atomic.AddInt32((*int32)(c), 1)
}

func (c *CountingRecorder) Close() error {
	return nil
}
