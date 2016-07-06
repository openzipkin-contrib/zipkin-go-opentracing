package zipkintracer

import (
	"encoding/binary"
	"io"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	opentracing "github.com/opentracing/opentracing-go"

	"github.com/openzipkin/zipkin-go-opentracing/wire"
)

type textMapPropagator struct {
	tracer *tracerImpl
}
type binaryPropagator struct {
	tracer *tracerImpl
}

const (
	prefixBaggage = "ot-baggage-"

	tracerStateFieldCount = 4

	zipkinTraceID      = "X-B3-TraceId"
	zipkinSpanID       = "X-B3-SpanId"
	zipkinParentSpanID = "X-B3-ParentSpanId"
	zipkinSampled      = "X-B3-Sampled"

	zipkinTraceIDLower      = "x-b3-traceid"
	zipkinSpanIDLower       = "x-b3-spanid"
	zipkinParentSpanIDLower = "x-b3-parentspanid"
	zipkinSampledLower      = "x-b3-sampled"
)

func (p *textMapPropagator) Inject(
	spanContext opentracing.SpanContext,
	opaqueCarrier interface{},
) error {
	sc, ok := spanContext.(*Context)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}
	carrier, ok := opaqueCarrier.(opentracing.TextMapWriter)
	if !ok {
		return opentracing.ErrInvalidCarrier
	}

	carrier.Set(zipkinTraceID, strconv.FormatUint(sc.TraceID, 16))
	carrier.Set(zipkinSpanID, strconv.FormatUint(sc.SpanID, 16))
	if !p.tracer.options.clientServerSameSpan {
		sc.ParentSpanID = 0
	}
	carrier.Set(zipkinParentSpanID, strconv.FormatUint(sc.ParentSpanID, 16))
	carrier.Set(zipkinSampled, strconv.FormatBool(sc.Sampled))

	sc.Lock()
	for k, v := range sc.Baggage {
		carrier.Set(prefixBaggage+k, v)
	}
	sc.Unlock()
	return nil
}

func (p *textMapPropagator) Extract(
	opaqueCarrier interface{},
) (opentracing.SpanContext, error) {
	carrier, ok := opaqueCarrier.(opentracing.TextMapReader)
	if !ok {
		return nil, opentracing.ErrInvalidCarrier
	}
	requiredFieldCount := 0
	var traceID, spanID, parentSpanID uint64
	var sampled bool
	var err error
	decodedBaggage := make(map[string]string)
	err = carrier.ForeachKey(func(k, v string) error {
		switch strings.ToLower(k) {
		case zipkinTraceIDLower:
			traceID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
		case zipkinSpanIDLower:
			spanID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
		case zipkinParentSpanIDLower:
			parentSpanID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
			if !p.tracer.options.clientServerSameSpan {
				parentSpanID = 0
			}
		case zipkinSampledLower:
			sampled, err = strconv.ParseBool(v)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
		default:
			lowercaseK := strings.ToLower(k)
			if strings.HasPrefix(lowercaseK, prefixBaggage) {
				decodedBaggage[strings.TrimPrefix(lowercaseK, prefixBaggage)] = v
			}
			// Balance off the requiredFieldCount++ just below...
			requiredFieldCount--
		}
		requiredFieldCount++
		return nil
	})
	if err != nil {
		return nil, err
	}
	if requiredFieldCount < tracerStateFieldCount {
		if requiredFieldCount == 0 {
			return nil, opentracing.ErrSpanContextNotFound
		}
		return nil, opentracing.ErrSpanContextCorrupted
	}

	return &Context{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		Sampled:      sampled,
		Baggage:      decodedBaggage,
	}, nil
}

func (p *binaryPropagator) Inject(
	spanContext opentracing.SpanContext,
	opaqueCarrier interface{},
) error {
	sc, ok := spanContext.(*Context)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}
	carrier, ok := opaqueCarrier.(io.Writer)
	if !ok {
		return opentracing.ErrInvalidCarrier
	}

	if !p.tracer.options.clientServerSameSpan {
		sc.ParentSpanID = 0
	}

	state := wire.TracerState{}
	state.TraceId = sc.TraceID
	state.SpanId = sc.SpanID
	state.ParentSpanId = sc.ParentSpanID
	state.Sampled = sc.Sampled
	state.BaggageItems = sc.Baggage

	b, err := proto.Marshal(&state)
	if err != nil {
		return err
	}

	// Write the length of the marshalled binary to the writer.
	length := uint32(len(b))
	if err = binary.Write(carrier, binary.BigEndian, &length); err != nil {
		return err
	}

	_, err = carrier.Write(b)
	return err
}

func (p *binaryPropagator) Extract(
	opaqueCarrier interface{},
) (opentracing.SpanContext, error) {
	carrier, ok := opaqueCarrier.(io.Reader)
	if !ok {
		return nil, opentracing.ErrInvalidCarrier
	}

	// Read the length of marshalled binary. io.ReadAll isn't that performant
	// since it keeps resizing the underlying buffer as it encounters more bytes
	// to read. By reading the length, we can allocate a fixed sized buf and read
	// the exact amount of bytes into it.
	var length uint32
	if err := binary.Read(carrier, binary.BigEndian, &length); err != nil {
		return nil, opentracing.ErrSpanContextCorrupted
	}
	buf := make([]byte, length)
	if n, err := carrier.Read(buf); err != nil {
		if n > 0 {
			return nil, opentracing.ErrSpanContextCorrupted
		}
		return nil, opentracing.ErrSpanContextNotFound
	}

	protoContext := wire.TracerState{}
	if err := proto.Unmarshal(buf, &protoContext); err != nil {
		return nil, opentracing.ErrSpanContextCorrupted
	}

	return &Context{
		TraceID:      protoContext.TraceId,
		Sampled:      protoContext.Sampled,
		SpanID:       protoContext.SpanId,
		ParentSpanID: protoContext.ParentSpanId,
		Baggage:      protoContext.BaggageItems,
	}, nil
}
