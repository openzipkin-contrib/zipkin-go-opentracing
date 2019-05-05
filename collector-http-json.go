package zipkintracer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// JSONHTTPCollector implements Collector by forwarding spans to a http server.
type JSONHTTPCollector struct {
	logger        Logger
	url           string
	client        *http.Client
	batchInterval time.Duration
	batchSize     int
	maxBacklog    int
	spanc         chan *CoreSpan
	quit          chan struct{}
	shutdown      chan error
	reqCallback   RequestCallback
}

// JSONRequestCallback receives the initialized request from the Collector before
// sending it over the wire. This allows one to plug in additional headers or
// do other customization.
type JSONRequestCallback func(*http.Request)

// JSONHTTPOption sets a parameter for the HttpCollector
type JSONHTTPOption func(c *JSONHTTPCollector)

// JSONHTTPLogger sets the logger used to report errors in the collection
// process. By default, a no-op logger is used, i.e. no errors are logged
// anywhere. It's important to set this option in a production service.
func JSONHTTPLogger(logger Logger) JSONHTTPOption {
	return func(c *JSONHTTPCollector) { c.logger = logger }
}

// JSONHTTPTimeout sets maximum timeout for http request.
func JSONHTTPTimeout(duration time.Duration) JSONHTTPOption {
	return func(c *JSONHTTPCollector) { c.client.Timeout = duration }
}

// JSONHTTPBatchSize sets the maximum batch size, after which a collect will be
// triggered. The default batch size is 100 traces.
func JSONHTTPBatchSize(n int) JSONHTTPOption {
	return func(c *JSONHTTPCollector) { c.batchSize = n }
}

// JSONHTTPMaxBacklog sets the maximum backlog size,
// when batch size reaches this threshold, spans from the
// beginning of the batch will be disposed
func JSONHTTPMaxBacklog(n int) JSONHTTPOption {
	return func(c *JSONHTTPCollector) { c.maxBacklog = n }
}

// JSONHTTPBatchInterval sets the maximum duration we will buffer traces before
// emitting them to the collector. The default batch interval is 1 second.
func JSONHTTPBatchInterval(d time.Duration) JSONHTTPOption {
	return func(c *JSONHTTPCollector) { c.batchInterval = d }
}

// JSONHTTPClient sets a custom http client to use.
func JSONHTTPClient(client *http.Client) JSONHTTPOption {
	return func(c *JSONHTTPCollector) { c.client = client }
}

// JSONHTTPRequestCallback registers a callback function to adjust the collector
// *http.Request before it sends the request to Zipkin.
func JSONHTTPRequestCallback(rc RequestCallback) JSONHTTPOption {
	return func(c *JSONHTTPCollector) { c.reqCallback = rc }
}

// NewJSONHTTPCollector returns a new HTTP-backend Collector. url should be a http
// url for handle post request. timeout is passed to http client. queueSize control
// the maximum size of buffer of async queue. The logger is used to log errors,
// such as send failures;
func NewJSONHTTPCollector(url string, options ...JSONHTTPOption) (AgnosticCollector, error) {
	c := &JSONHTTPCollector{
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
	c.spanc = make(chan *CoreSpan, c.maxBacklog)

	go c.loop()
	return c, nil
}

// Collect implements Collector.
// attempts a non blocking send on the channel.
func (c *JSONHTTPCollector) Collect(s *CoreSpan) error {
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
func (c *JSONHTTPCollector) Close() error {
	close(c.quit)
	return <-c.shutdown
}

func (c *JSONHTTPCollector) loop() {
	var (
		nextSend = time.Now().Add(c.batchInterval)
		ticker   = time.NewTicker(c.batchInterval / 10)
		tickc    = ticker.C
	)
	defer ticker.Stop()

	// The following loop is single threaded
	// allocate enough space so we don't have to reallocate.
	batch := make([]*CoreSpan, 0, c.batchSize)

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

func (c *JSONHTTPCollector) send(sendBatch []*CoreSpan) error {

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
