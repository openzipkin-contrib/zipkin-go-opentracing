package zipkin

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strconv"
	"time"

	"github.com/basvanbeek/zipkin-go-opentracing/_thrift/gen-go/zipkincore"
)

// Span respresents a Zipkin Span
type Span struct {
	Span     zipkincore.Span
	Endpoint *zipkincore.Endpoint
	sampled  bool
}

// NewSpan creates a new Span
func NewSpan(hostPort, serviceName, methodName string, traceID, spanID, parentSpanID int64, sampled bool) *Span {
	endpoint := MakeEndpoint(hostPort, serviceName)
	if endpoint == nil {
		endpoint = zipkincore.Annotation_Host_DEFAULT
	}
	return &Span{
		Span: zipkincore.Span{
			Name:     methodName,
			TraceID:  traceID,
			ID:       spanID,
			ParentID: &parentSpanID,
		},
		sampled:  sampled,
		Endpoint: endpoint,
	}
}

// MakeEndpoint makes a Zipkin Endpoint
func MakeEndpoint(hostPort, serviceName string) *zipkincore.Endpoint {
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil
	}
	portInt, err := strconv.ParseInt(port, 10, 16)
	if err != nil {
		return nil
	}
	addrs, err := net.LookupIP(host)
	if err != nil {
		return nil
	}
	// we need the first IPv4 address.
	var addr net.IP
	for i := range addrs {
		addr = addrs[i].To4()
		if addr != nil {
			break
		}
	}
	if addr == nil {
		// none of the returned addresses is IPv4.
		return nil
	}
	endpoint := zipkincore.NewEndpoint()
	endpoint.Ipv4 = (int32)(binary.BigEndian.Uint32(addr))
	endpoint.Port = int16(portInt)
	endpoint.ServiceName = serviceName
	return endpoint
}

// Annotate annotates the span with the given value.
func (s *Span) Annotate(timestamp time.Time, value string, host *zipkincore.Endpoint) {
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	if host == nil {
		host = s.Endpoint
	}
	s.Span.Annotations = append(s.Span.Annotations, &zipkincore.Annotation{
		Timestamp: timestamp.UnixNano() / 1e3,
		Value:     value,
		Host:      host,
	})
}

// AnnotateBinary annotates the span with a key and a value that will be []byte
// encoded.
func (s *Span) AnnotateBinary(key string, value interface{}, host *zipkincore.Endpoint) {
	var a zipkincore.AnnotationType
	var b []byte
	// We are not using zipkincore.AnnotationType_I16 for types that could fit
	// as reporting on it seems to be broken on the zipkin web interface
	// (however, we can properly extract the number from zipkin storage
	// directly). int64 has issues with negative numbers but seems ok for
	// positive numbers needing more than 32 bit.
	switch v := value.(type) {
	case bool:
		a = zipkincore.AnnotationType_BOOL
		b = []byte("\x00")
		if v {
			b = []byte("\x01")
		}
	case []byte:
		a = zipkincore.AnnotationType_BYTES
		b = v
	case byte:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(v))
	case int8:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(v))
	case int16:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(v))
	case uint16:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(v))
	case int32:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(v))
	case uint32:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(v))
	case int64:
		a = zipkincore.AnnotationType_I64
		b = make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(v))
	case int:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 8)
		binary.BigEndian.PutUint32(b, uint32(v))
	case uint:
		a = zipkincore.AnnotationType_I32
		b = make([]byte, 8)
		binary.BigEndian.PutUint32(b, uint32(v))
	case uint64:
		a = zipkincore.AnnotationType_I64
		b = make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(v))
	case float32:
		a = zipkincore.AnnotationType_DOUBLE
		b = make([]byte, 8)
		bits := math.Float64bits(float64(v))
		binary.BigEndian.PutUint64(b, bits)
	case float64:
		a = zipkincore.AnnotationType_DOUBLE
		b = make([]byte, 8)
		bits := math.Float64bits(v)
		binary.BigEndian.PutUint64(b, bits)
	case string:
		a = zipkincore.AnnotationType_STRING
		b = []byte(v)
	default:
		// we have no handler for type's value, but let's get a string
		// representation of it.
		a = zipkincore.AnnotationType_STRING
		b = []byte(fmt.Sprintf("%+v", value))
	}
	if host == nil {
		host = s.Endpoint
	}
	s.Span.BinaryAnnotations = append(s.Span.BinaryAnnotations, &zipkincore.BinaryAnnotation{
		Key:            key,
		Value:          b,
		AnnotationType: a,
		Host:           host,
	})
}
