package zipkintracer

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/apache/thrift/lib/go/thrift"

	"github.com/openzipkin/zipkin-go-opentracing/_thrift/gen-go/zipkincore"
)

// Default timeout for http request in seconds
const defaultHTTPTimeout = time.Second * 5

// defaultBatchInterval in seconds
const defaultHTTPBatchInterval = time.Second * 1

const defaultHTTPBatchSize = 100

const defaultHTTPMaxBacklog = 1000

// defaultLogErrorInterval, set to by default log every error
const defaultLogErrorInterval = time.Second * 0

// HTTPCollector implements Collector by forwarding spans to a http server.
type HTTPCollector struct {
	logger        Logger
	url           string
	client        *http.Client
	batchInterval time.Duration
	batchSize     int
	maxBacklog    int
	batch         []*zipkincore.Span
	spanc         chan *zipkincore.Span
	quit          chan struct{}
	shutdown      chan error
	sendMutex     *sync.Mutex
	batchMutex    *sync.Mutex
	stateLogger   *StateLogger
}

// HTTPOption sets a parameter for the HttpCollector
type HTTPOption func(c *HTTPCollector)

// HTTPLogger sets the logger used to report errors in the collection
// process. By default, a no-op logger is used, i.e. no errors are logged
// anywhere. It's important to set this option in a production service.
func HTTPLogger(logger Logger) HTTPOption {
	return func(c *HTTPCollector) { c.logger = logger }
}

// HTTPTimeout sets maximum timeout for http request.
func HTTPTimeout(duration time.Duration) HTTPOption {
	return func(c *HTTPCollector) { c.client.Timeout = duration }
}

// HTTPBatchSize sets the maximum batch size, after which a collect will be
// triggered. The default batch size is 100 traces.
func HTTPBatchSize(n int) HTTPOption {
	return func(c *HTTPCollector) { c.batchSize = n }
}

// HTTPMaxBacklog sets the maximum backlog size,
// when batch size reaches this threshold, spans from the
// beginning of the batch will be disposed
func HTTPMaxBacklog(n int) HTTPOption {
	return func(c *HTTPCollector) { c.maxBacklog = n }
}

// HTTPBatchInterval sets the maximum duration we will buffer traces before
// emitting them to the collector. The default batch interval is 1 second.
func HTTPBatchInterval(d time.Duration) HTTPOption {
	return func(c *HTTPCollector) { c.batchInterval = d }
}

// HTTPLogErrorInterval sets the maximal interval between logging
// of the same error into the log.
// Setting it to 0 will log every error into the log
func HTTPLogErrorInterval(d time.Duration) HTTPOption {
	return func(c *HTTPCollector) { c.stateLogger = NewStateLogger(c.logger, d) }
}

// NewHTTPCollector returns a new HTTP-backend Collector. url should be a http
// url for handle post request. timeout is passed to http client. queueSize control
// the maximum size of buffer of async queue. The logger is used to log errors,
// such as send failures;
func NewHTTPCollector(url string, options ...HTTPOption) (Collector, error) {
	logger := NewNopLogger()
	c := &HTTPCollector{
		logger:        logger,
		url:           url,
		client:        &http.Client{Timeout: defaultHTTPTimeout},
		batchInterval: defaultHTTPBatchInterval,
		batchSize:     defaultHTTPBatchSize,
		maxBacklog:    defaultHTTPMaxBacklog,
		batch:         []*zipkincore.Span{},
		spanc:         make(chan *zipkincore.Span),
		quit:          make(chan struct{}, 1),
		shutdown:      make(chan error, 1),
		sendMutex:     &sync.Mutex{},
		batchMutex:    &sync.Mutex{},
		stateLogger:   NewStateLogger(logger, defaultLogErrorInterval),
	}

	for _, option := range options {
		option(c)
	}

	go c.loop()
	return c, nil
}

// Collect implements Collector.
func (c *HTTPCollector) Collect(s *zipkincore.Span) error {
	c.spanc <- s
	return nil
}

// Close implements Collector.
func (c *HTTPCollector) Close() error {
	close(c.quit)
	return <-c.shutdown
}

func httpSerialize(spans []*zipkincore.Span) *bytes.Buffer {
	t := thrift.NewTMemoryBuffer()
	p := thrift.NewTBinaryProtocolTransport(t)
	if err := p.WriteListBegin(thrift.STRUCT, len(spans)); err != nil {
		panic(err)
	}
	for _, s := range spans {
		if err := s.Write(p); err != nil {
			panic(err)
		}
	}
	if err := p.WriteListEnd(); err != nil {
		panic(err)
	}
	return t.Buffer
}

func (c *HTTPCollector) loop() {
	var (
		nextSend = time.Now().Add(c.batchInterval)
		ticker   = time.NewTicker(c.batchInterval / 10)
		tickc    = ticker.C
	)
	defer ticker.Stop()

	for {
		select {
		case span := <-c.spanc:
			currentBatchSize := c.append(span)
			if currentBatchSize >= c.batchSize {
				nextSend = time.Now().Add(c.batchInterval)
				go c.send()
			}
		case <-tickc:
			if time.Now().After(nextSend) {
				nextSend = time.Now().Add(c.batchInterval)
				go c.send()
			}
		case <-c.quit:
			c.shutdown <- c.send()
			return
		}
	}
}

func (c *HTTPCollector) append(span *zipkincore.Span) (newBatchSize int) {
	c.batchMutex.Lock()
	defer c.batchMutex.Unlock()

	c.batch = append(c.batch, span)
	if len(c.batch) > c.maxBacklog {
		dispose := len(c.batch) - c.maxBacklog
		c.logger.Log("Backlog too long, disposing spans.", "count", dispose)
		c.batch = c.batch[dispose:]
	}
	newBatchSize = len(c.batch)
	return
}

func (c *HTTPCollector) send() error {
	// in order to prevent sending the same batch twice
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()

	// Select all current spans in the batch to be sent
	c.batchMutex.Lock()
	sendBatch := c.batch[:]
	c.batchMutex.Unlock()

	// Do not send an empty batch
	if len(sendBatch) == 0 {
		return nil
	}

	req, err := http.NewRequest(
		"POST",
		c.url,
		httpSerialize(sendBatch))
	if err != nil {
		c.logger.Log("err", err.Error())
		return err
	}
	req.Header.Set("Content-Type", "application/x-thrift")
	if _, err = c.client.Do(req); err != nil {
		c.stateLogger.LogError(err)
		return err
	}
	c.stateLogger.Fixed("Connection restored")

	// Remove sent spans from the batch
	c.batchMutex.Lock()
	c.batch = c.batch[len(sendBatch):]
	c.batchMutex.Unlock()

	return nil
}
