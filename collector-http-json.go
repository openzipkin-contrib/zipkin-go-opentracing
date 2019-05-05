package zipkintracer

import (
	"bytes"
	"encoding/json"
	"github.com/openzipkin-contrib/zipkin-go-opentracing/models"
	"net/http"
	"time"
)

// HTTPCollector implements Collector by forwarding spans to a http server.
type JsonHTTPCollector struct {
	logger        Logger
	url           string
	client        *http.Client
	batchInterval time.Duration
	batchSize     int
	maxBacklog    int
	spanc         chan *models.Span
	quit          chan struct{}
	shutdown      chan error
	reqCallback   RequestCallback
}

// RequestCallback receives the initialized request from the Collector before
// sending it over the wire. This allows one to plug in additional headers or
// do other customization.
type JsonRequestCallback func(*http.Request)

// HTTPOption sets a parameter for the HttpCollector
type JsonHTTPOption func(c *JsonHTTPCollector)

// HTTPLogger sets the logger used to report errors in the collection
// process. By default, a no-op logger is used, i.e. no errors are logged
// anywhere. It's important to set this option in a production service.
func JsonHTTPLogger(logger Logger) JsonHTTPOption {
	return func(c *JsonHTTPCollector) { c.logger = logger }
}

// HTTPTimeout sets maximum timeout for http request.
func JsonHTTPTimeout(duration time.Duration) JsonHTTPOption {
	return func(c *JsonHTTPCollector) { c.client.Timeout = duration }
}

// HTTPBatchSize sets the maximum batch size, after which a collect will be
// triggered. The default batch size is 100 traces.
func JsonHTTPBatchSize(n int) JsonHTTPOption {
	return func(c *JsonHTTPCollector) { c.batchSize = n }
}

// HTTPMaxBacklog sets the maximum backlog size,
// when batch size reaches this threshold, spans from the
// beginning of the batch will be disposed
func JsonHTTPMaxBacklog(n int) JsonHTTPOption {
	return func(c *JsonHTTPCollector) { c.maxBacklog = n }
}

// HTTPBatchInterval sets the maximum duration we will buffer traces before
// emitting them to the collector. The default batch interval is 1 second.
func JsonHTTPBatchInterval(d time.Duration) JsonHTTPOption {
	return func(c *JsonHTTPCollector) { c.batchInterval = d }
}

// HTTPClient sets a custom http client to use.
func JsonHTTPClient(client *http.Client) JsonHTTPOption {
	return func(c *JsonHTTPCollector) { c.client = client }
}

// HTTPRequestCallback registers a callback function to adjust the collector
// *http.Request before it sends the request to Zipkin.
func JsonHTTPRequestCallback(rc RequestCallback) JsonHTTPOption {
	return func(c *JsonHTTPCollector) { c.reqCallback = rc }
}

// NewHTTPCollector returns a new HTTP-backend Collector. url should be a http
// url for handle post request. timeout is passed to http client. queueSize control
// the maximum size of buffer of async queue. The logger is used to log errors,
// such as send failures;
func NewJsonHTTPCollector(url string, options ...JsonHTTPOption) (CollectorAgnostic, error) {
	c := &JsonHTTPCollector{
		logger:        NewNopLogger(),
		url:           url,
		client:        &http.Client{Timeout: defaultHTTPTimeout},
		batchInterval: defaultHTTPBatchInterval * time.Second,
		batchSize:     defaultHTTPBatchSize,
		maxBacklog:    defaultHTTPMaxBacklog,
		quit:          make(chan struct{}, 1),
		shutdown:      make(chan error, 1),
	}

	for _, option := range options {
		option(c)
	}

	// spanc can immediately accept maxBacklog spans and everything else is dropped.
	c.spanc = make(chan *models.Span, c.maxBacklog)

	go c.loop()
	return c, nil
}

// Collect implements Collector.
// attempts a non blocking send on the channel.
func (c *JsonHTTPCollector) Collect(s *models.Span) error {
	select {
	case c.spanc <- s:
		// Accepted.
	case <-c.quit:
		// Collector concurrently closed.
	default:
		c.logger.Log("msg", "queue full, disposing spans.", "size", len(c.spanc))
	}
	return nil
}

// Close implements Collector.
func (c *JsonHTTPCollector) Close() error {
	close(c.quit)
	return <-c.shutdown
}

func (c *JsonHTTPCollector) loop() {
	var (
		nextSend = time.Now().Add(c.batchInterval)
		ticker   = time.NewTicker(c.batchInterval / 10)
		tickc    = ticker.C
	)
	defer ticker.Stop()

	// The following loop is single threaded
	// allocate enough space so we don't have to reallocate.
	batch := make([]*models.Span, 0, c.batchSize)

	for {
		select {
		case span := <-c.spanc:
			batch = append(batch, span)
			if len(batch) == c.batchSize {
				c.send(batch)
				batch = batch[0:0]
				nextSend = time.Now().Add(c.batchInterval)
			}
		case <-tickc:
			if time.Now().After(nextSend) {
				if len(batch) > 0 {
					c.send(batch)
					batch = batch[0:0]
				}
				nextSend = time.Now().Add(c.batchInterval)
			}
		case <-c.quit:
			c.shutdown <- c.send(batch)
			return
		}
	}
}

func (c *JsonHTTPCollector) send(sendBatch []*models.Span) error {

	payload, err := json.Marshal(sendBatch)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"POST",
		c.url,
		bytes.NewBuffer(payload))
	if err != nil {
		c.logger.Log("err", err.Error())
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.reqCallback != nil {
		c.reqCallback(req)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Log("err", err.Error())
		return err
	}
	resp.Body.Close()
	// non 2xx code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.logger.Log("err", "HTTP POST span failed", "code", resp.Status)
	}
	return nil
}
