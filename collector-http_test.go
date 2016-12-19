package zipkintracer

import (
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/apache/thrift/lib/go/thrift"

	"github.com/openzipkin/zipkin-go-opentracing/_thrift/gen-go/zipkincore"
)

func TestHttpCollector(t *testing.T) {
	server := newHTTPServer(t)
	c, err := NewHTTPCollector("http://localhost:10000/api/v1/spans")
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
	if err := c.Close(); err != nil {
		t.Fatalf("error during collection: %v", err)
	}
	if want, have := 1, len(server.spans()); want != have {
		t.Fatalf("never received a span")
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
	http.HandleFunc("/api/v1/spans", func(w http.ResponseWriter, r *http.Request) {
		contextType := r.Header.Get("Content-Type")
		if contextType != "application/x-thrift" {
			t.Fatalf(
				"except Content-Type should be application/x-thrift, but is %s",
				contextType)
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		buffer := thrift.NewTMemoryBuffer()
		if _, err = buffer.Write(body); err != nil {
			t.Error(err)
			return
		}
		transport := thrift.NewTBinaryProtocolTransport(buffer)
		_, size, err := transport.ReadListBegin()
		if err != nil {
			t.Error(err)
			return
		}
		var spans []*zipkincore.Span
		for i := 0; i < size; i++ {
			zs := &zipkincore.Span{}
			if err = zs.Read(transport); err != nil {
				t.Error(err)
				return
			}
			spans = append(spans, zs)
		}
		err = transport.ReadListEnd()
		if err != nil {
			t.Error(err)
			return
		}
		server.mutex.Lock()
		defer server.mutex.Unlock()
		server.zipkinSpans = append(server.zipkinSpans, spans...)
	})

	go func() {
		http.ListenAndServe(":10000", nil)
	}()

	return server
}
