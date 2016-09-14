package zipkintracer

import (
	"github.com/apache/thrift/lib/go/thrift"
	"net/http"

	"github.com/openzipkin/zipkin-go-opentracing/_thrift/gen-go/zipkincore"
	"strings"
	"time"
)

const defaultMaxAsyncQueueSize = 64

// Default timeout for http request in seconds
const defaultHttpTimeout = time.Second * 5

// HttpCollector implements Collector by forwarding spans to a http server.
type HttpCollector struct {
	logger          Logger
	url             string
	client          *http.Client
	asyncInputQueue chan *zipkincore.Span
	quit            chan struct{}
}

// HttpOption sets a parameter for the HttpCollector
type HttpOption func(c *HttpCollector)

// HttpLogger sets the logger used to report errors in the collection
// process. By default, a no-op logger is used, i.e. no errors are logged
// anywhere. It's important to set this option in a production service.
func HttpLogger(logger Logger) HttpOption {
	return func(c *HttpCollector) { c.logger = logger }
}

// HttpMaxASyncQueueSize sets the maximum buffer size for the async queue.
func HttpMaxAsyncQueueSize(size int) HttpOption {
	return func(c *HttpCollector) { c.asyncInputQueue = make(chan *zipkincore.Span, size) }
}

// HttpTimeout sets maximum timeout for http request.
func HttpTimeout(duration time.Duration) HttpOption {
	return func(c *HttpCollector) { c.client.Timeout = duration }
}

// NewHttpCollector returns a new Http-backend Collector. url should be a http
// url for handle post request. timeout is passed to http client. queueSize control
// the maximum size of buffer of async queue. The logger is used to log errors,
// such as send failures;
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

// Collect implements Collector.
func (c *HttpCollector) Collect(s *zipkincore.Span) error {
	c.asyncInputQueue <- s
	return nil
}

// Close implements Collector.
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
