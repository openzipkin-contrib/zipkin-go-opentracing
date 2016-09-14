package zipkintracer

import (
	"github.com/apache/thrift/lib/go/thrift"
	"net/http"

	"github.com/openzipkin/zipkin-go-opentracing/_thrift/gen-go/zipkincore"
	"strings"
	"time"
)

const defaultMaxAsyncQueueSize = 64
const defaultHttpTimeout = time.Second * 5

type HttpCollector struct {
	logger          Logger
	url             string
	client          *http.Client
	asyncInputQueue chan *zipkincore.Span
	quit            chan struct{}
}

// HttpOption sets a parameter for the HttpCollector
type HttpOption func(c *HttpCollector)

func HttpLogger(logger Logger) HttpOption {
	return func(c *HttpCollector) { c.logger = logger }
}

func HttpMaxAsyncQueueSize(size int) HttpOption {
	return func(c *HttpCollector) { c.asyncInputQueue = make(chan *zipkincore.Span, size) }
}

func HttpTimeout(duration time.Duration) HttpOption {
	return func(c *HttpCollector) { c.client.Timeout = duration }
}

func NewHttpCollector(url string, options ...HttpOption) (Collector, error) {
	c := &HttpCollector{
		logger:          NewNopLogger(),
		url:             url,
		client:          &http.Client{Timeout: defaultHttpTimeout},
		asyncInputQueue: make(chan *zipkincore.Span, defaultMaxAsyncQueueSize),
		quit:            make(chan struct{}, 1),
	}

	for _, option := range options {
		option(c)
	}
	go c.loop()
	return c, nil
}

func (c *HttpCollector) Collect(s *zipkincore.Span) error {
	c.asyncInputQueue <- s
	return nil
}

func (c *HttpCollector) Close() error {
	c.quit <- struct{}{}
	return nil
}

func httpSerialize(s *zipkincore.Span) []byte {
	t := thrift.NewTMemoryBuffer()
	p := thrift.NewTBinaryProtocolTransport(t)
	if err := s.Write(p); err != nil {
		panic(err)
	}
	return t.Buffer.Bytes()
}

func (c *HttpCollector) loop() {
	for {
		select {
		case span := <-c.asyncInputQueue:
			req, err := http.NewRequest(
				"POST",
				c.url,
				strings.NewReader(string(httpSerialize(span))))
			if err != nil {
				_ = c.logger.Log("err", err.Error())
			}

			go func() {
				_, err := c.client.Do(req)
				if err != nil {
					_ = c.logger.Log("err", err.Error())
				}
			}()
		case <-c.quit:
			return
		}
	}
}
