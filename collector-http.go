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
const defaultHTTPTimeout = time.Second * 5

// HTTPCollector implements Collector by forwarding spans to a http server.
type HTTPCollector struct {
	logger          Logger
	url             string
	client          *http.Client
	asyncInputQueue chan *zipkincore.Span
	quit            chan struct{}
}

// HTTPOption sets a parameter for the HttpCollector
type HTTPOption func(c *HTTPCollector)

// HTTPLogger sets the logger used to report errors in the collection
// process. By default, a no-op logger is used, i.e. no errors are logged
// anywhere. It's important to set this option in a production service.
func HTTPLogger(logger Logger) HTTPOption {
	return func(c *HTTPCollector) { c.logger = logger }
}

// HTTPMaxAsyncQueueSize sets the maximum buffer size for the async queue.
func HTTPMaxAsyncQueueSize(size int) HTTPOption {
	return func(c *HTTPCollector) { c.asyncInputQueue = make(chan *zipkincore.Span, size) }
}

// HTTPTimeout sets maximum timeout for http request.
func HTTPTimeout(duration time.Duration) HTTPOption {
	return func(c *HTTPCollector) { c.client.Timeout = duration }
}

// NewHTTPCollector returns a new HTTP-backend Collector. url should be a http
// url for handle post request. timeout is passed to http client. queueSize control
// the maximum size of buffer of async queue. The logger is used to log errors,
// such as send failures;
func NewHTTPCollector(url string, options ...HTTPOption) (Collector, error) {
	c := &HTTPCollector{
		logger:          NewNopLogger(),
		url:             url,
		client:          &http.Client{Timeout: defaultHTTPTimeout},
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
func (c *HTTPCollector) Collect(s *zipkincore.Span) error {
	c.asyncInputQueue <- s
	return nil
}

// Close implements Collector.
func (c *HTTPCollector) Close() error {
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

func (c *HTTPCollector) loop() {
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
