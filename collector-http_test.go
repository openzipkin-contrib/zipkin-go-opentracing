package zipkintracer

import (
	"testing"
	"time"

	"io/ioutil"
	"net/http"

	"github.com/apache/thrift/lib/go/thrift"

	"encoding/base64"
	"encoding/json"
	"github.com/openzipkin/zipkin-go-opentracing/_thrift/gen-go/zipkincore"
	"sync"
)

func TestHttpCollector(t *testing.T) {
	server := newHTTPServer(t)
	c, err := NewHTTPCollector("http://localhost:10000/zipkin")
	if err != nil {
		t.Fatal(err)
	}

	var (
		serviceName  = "service"
		methodName   = "method"
		traceID      = int64(123)
		spanID       = int64(456)
		parentSpanID = int64(0)
		value        = "foo"
	)

	span := makeNewSpan("1.2.3.4:1234", serviceName, methodName, traceID, spanID, parentSpanID, true)
	annotate(span, time.Now(), "foo", nil)
	if err := c.Collect(span); err != nil {
		t.Errorf("error during collection: %v", err)
	}

	// Need to yield to the select loop to accept the send request, and then
	// yield again to the send operation to write to the socket. I think the
	// best way to do that is just give it some time.

	deadline := time.Now().Add(1 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("never received a span")
		}
		if want, have := 1, len(server.spans()); want != have {
			time.Sleep(time.Millisecond)
			continue
		}
		break
	}

	gotSpan := server.spans()[0]
	if want, have := methodName, gotSpan.GetName(); want != have {
		t.Errorf("want %q, have %q", want, have)
	}
	if want, have := traceID, gotSpan.TraceID; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
	if want, have := spanID, gotSpan.ID; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
	if want, have := parentSpanID, *gotSpan.ParentID; want != have {
		t.Errorf("want %d, have %d", want, have)
	}

	if want, have := 1, len(gotSpan.GetAnnotations()); want != have {
		t.Fatalf("want %d, have %d", want, have)
	}

	gotAnnotation := gotSpan.GetAnnotations()[0]
	if want, have := value, gotAnnotation.GetValue(); want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}

type httpServer struct {
	t           *testing.T
	zipkinSpans []*zipkincore.Span
	mutex       sync.RWMutex
}

func (s *httpServer) spans() []*zipkincore.Span {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.zipkinSpans
}

func newHTTPServer(t *testing.T) *httpServer {
	server := &httpServer{
		t:           t,
		zipkinSpans: make([]*zipkincore.Span, 0),
		mutex:       sync.RWMutex{},
	}
	http.HandleFunc("/zipkin", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		var encodeSpans []string
		err = json.Unmarshal(body, &encodeSpans)
		if err != nil {
			t.Error(err)
		}

		var spans []*zipkincore.Span
		for _, encodeSpan := range encodeSpans {
			buffer := thrift.NewTMemoryBuffer()
			bytes, err := base64.StdEncoding.DecodeString(encodeSpan)
			if err != nil {
				t.Error(err)
				return
			}
			if _, err := buffer.Write(bytes); err != nil {
				t.Error(err)
				return
			}
			transport := thrift.NewTBinaryProtocolTransport(buffer)
			zs := &zipkincore.Span{}
			if err := zs.Read(transport); err != nil {
				t.Error(err)
				return
			}
			spans = append(spans, zs)
		}

		server.mutex.Lock()
		defer server.mutex.Unlock()
		for _, span := range spans {
			server.zipkinSpans = append(server.zipkinSpans, span)
		}
	})

	go func() {
		http.ListenAndServe(":10000", nil)
	}()
	return server
}
