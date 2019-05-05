package zipkintracer

import (
	"encoding/json"
	"fmt"
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
		traceID      = uint64(17051370458307041793)
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
	if want, have := fmt.Sprintf("%08x", traceID), gotSpan.TraceID; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
	if want, have := fmt.Sprintf("%08x", spanID), gotSpan.ID; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
	if want, have := "", gotSpan.ParentID; want != have {
		t.Errorf("want %s, have %s", want, have)
	}

}

type jsonHttpServer struct {
	t            *testing.T
	zipkinSpans  []*CoreSpan
	zipkinHeader http.Header
	mutex        sync.RWMutex
}

func (s *jsonHttpServer) spans() []*CoreSpan {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.zipkinSpans
}

func newJsonHTTPServer(t *testing.T, port int) *jsonHttpServer {
	server := &jsonHttpServer{
		t:           t,
		zipkinSpans: make([]*CoreSpan, 0),
		mutex:       sync.RWMutex{},
	}

	handler := http.NewServeMux()

	handler.HandleFunc("/api/v1/spans", func(w http.ResponseWriter, r *http.Request) {
		contextType := r.Header.Get("Content-Type")
		if contextType != "application/json" {
			t.Fatalf("except Content-Type should be application/x-thrift, but is %s", contextType)
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
		var spans []*CoreSpan
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

func makeNewJsonSpan(hostPort, serviceName, methodName string, traceID, spanID, parentSpanID uint64, debug bool) *CoreSpan {
	timestamp := time.Now().UnixNano() / 1e3
	span := &CoreSpan{
		Name:      methodName,
		TraceID:   fmt.Sprintf("%08x", traceID),
		ID:        fmt.Sprintf("%08x", spanID),
		Debug:     debug,
		Timestamp: timestamp,
	}

	if parentSpanID > 0 {
		span.ParentID = fmt.Sprintf("%08x", parentSpanID)
	}

	return span
}
