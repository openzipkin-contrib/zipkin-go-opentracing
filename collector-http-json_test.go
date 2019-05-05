package zipkintracer

import (
	"encoding/json"
	"fmt"
	"github.com/openzipkin-contrib/zipkin-go-opentracing/models"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"testing"
	"time"
)


func TestJsonHttpCollector(t *testing.T) {
	t.Parallel()

	port := 10000
	server := newJsonHTTPServer(t, port)
	c, err := NewJsonHTTPCollector(fmt.Sprintf("http://localhost:%d/api/v1/spans", port),
		JsonHTTPBatchSize(1))
	if err != nil {
		t.Fatal(err)
	}

	var (
		serviceName  = "service"
		methodName   = "method"
		traceID      = uint64(123)
		spanID       = uint64(456)
		parentSpanID = uint64(0)
	)

	span := makeNewJsonSpan("1.2.3.4:1234", serviceName, methodName, traceID, spanID, parentSpanID, true)
	if err := c.Collect(span); err != nil {
		t.Errorf("error during collection: %v", err)
	}


	if err = eventually(func() bool { return len(server.spans()) == 1 }, 1*time.Second); err != nil {
		t.Fatalf("never received a span %v", server.spans())
	}

	gotSpan := server.spans()[0]
	if want, have := methodName, gotSpan.Name; want != have {
		t.Errorf("want %q, have %q", want, have)
	}
	if want, have := fmt.Sprintf("%d", traceID), gotSpan.TraceID; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
	if want, have := fmt.Sprintf("%d", spanID), gotSpan.ID; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
	if want, have := fmt.Sprintf("%d", parentSpanID), gotSpan.ParentID; want != have {
		t.Errorf("want %s, have %s", want, have)
	}

}

type jsonHttpServer struct {
	t            *testing.T
	zipkinSpans  []*models.Span
	zipkinHeader http.Header
	mutex        sync.RWMutex
}

func (s *jsonHttpServer) spans() []*models.Span {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.zipkinSpans
}

func newJsonHTTPServer(t *testing.T, port int) *jsonHttpServer {
	server := &jsonHttpServer{
		t:           t,
		zipkinSpans: make([]*models.Span, 0),
		mutex:       sync.RWMutex{},
	}

	handler := http.NewServeMux()

	handler.HandleFunc("/api/v1/spans", func(w http.ResponseWriter, r *http.Request) {
		contextType := r.Header.Get("Content-Type")
		if contextType != "application/json" {
			t.Fatalf(
				"except Content-Type should be application/x-thrift, but is %s",
				contextType)
		}

		// clone headers from request
		headers := make(http.Header, len(r.Header))
		for k, vv := range r.Header {
			vv2 := make([]string, len(vv))
			copy(vv2, vv)
			headers[k] = vv2
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var spans []*models.Span
		if err := json.Unmarshal(body, &spans); err != nil {
			log.Fatal(err.Error())
		}

		server.mutex.Lock()
		defer server.mutex.Unlock()
		server.zipkinSpans = append(server.zipkinSpans, spans...)
		server.zipkinHeader = headers
	})

	handler.HandleFunc("/api/v1/sleep", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(serverSleep)
	})

	go func() {
		http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
	}()

	return server
}

func makeNewJsonSpan(hostPort, serviceName, methodName string, traceID, spanID, parentSpanID uint64, debug bool) *models.Span {
	timestamp := time.Now().UnixNano() / 1e3
	return &models.Span{
		TraceID:   fmt.Sprintf("%d", traceID),
		Name:      methodName,
		ID:        fmt.Sprintf("%d", spanID),
		ParentID:  fmt.Sprintf("%d", parentSpanID),
		Debug:     debug,
		Timestamp: timestamp,
	}
}