package zipkintracer

import (
	"encoding/binary"
	"errors"
	"net"
	"strconv"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/openzipkin/zipkin-go-opentracing/_thrift/gen-go/zipkincore"
)

// ErrInvalidEndpoint will be thrown if hostPort parameter is corrupted or host
// can't be resolved
var ErrInvalidEndpoint = errors.New("Invalid Endpoint. Please check hostPort parameter")

// Tracer extends the opentracing.Tracer interface with methods to
// probe implementation state, for use by zipkintracer consumers.
type Tracer interface {
	opentracing.Tracer

	// Options gets the Options used in New() or NewWithOptions().
	Options() TracerOptions
}

// TracerOptions allows creating a customized Tracer.
type TracerOptions struct {
	shouldSample               func(traceID uint64) bool
	trimUnsampledSpans         bool
	recorder                   SpanRecorder
	newSpanEventListener       func() func(SpanEvent)
	logger                     Logger
	debugAssertSingleGoroutine bool
	debugAssertUseAfterFinish  bool
	clientServerSameSpan       bool
}

// TracerOption allows for functional options.
// See: http://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type TracerOption func(opts *TracerOptions) error

// WithSampler allows one to add a Sampler function
func WithSampler(sampler Sampler) TracerOption {
	return func(opts *TracerOptions) error {
		opts.shouldSample = sampler
		return nil
	}
}

// TrimUnsampledSpans option
func TrimUnsampledSpans(trim bool) TracerOption {
	return func(opts *TracerOptions) error {
		opts.trimUnsampledSpans = trim
		return nil
	}
}

// WithLogger option
func WithLogger(logger Logger) TracerOption {
	return func(opts *TracerOptions) error {
		opts.logger = logger
		return nil
	}
}

// DebugAssertSingleGoroutine option
func DebugAssertSingleGoroutine(val bool) TracerOption {
	return func(opts *TracerOptions) error {
		opts.debugAssertSingleGoroutine = val
		return nil
	}
}

// DebugAssertUseAfterFinish option
func DebugAssertUseAfterFinish(val bool) TracerOption {
	return func(opts *TracerOptions) error {
		opts.debugAssertUseAfterFinish = val
		return nil
	}
}

// ClientServerSameSpan allows to place client-side and server-side annotations
// for a RPC call in the same span (Zipkin V1 behavior). By default this Tracer
// uses single host spans (so client-side and server-side in separate spans).
func ClientServerSameSpan(val bool) TracerOption {
	return func(opts *TracerOptions) error {
		opts.clientServerSameSpan = val
		return nil
	}
}

// NewTracer creates a new OpenTracing compatible Zipkin Tracer.
func NewTracer(recorder SpanRecorder, options ...TracerOption) (opentracing.Tracer, error) {
	opts := &TracerOptions{
		recorder:             recorder,
		shouldSample:         alwaysSample,
		trimUnsampledSpans:   false,
		newSpanEventListener: func() func(SpanEvent) { return nil },
		logger:               &nopLogger{},
		debugAssertSingleGoroutine: false,
		debugAssertUseAfterFinish:  false,
		clientServerSameSpan:       false,
	}
	for _, o := range options {
		err := o(opts)
		if err != nil {
			return nil, err
		}
	}
	rval := &tracerImpl{options: *opts}
	rval.textPropagator = &textMapPropagator{rval}
	rval.binaryPropagator = &binaryPropagator{rval}
	rval.accessorPropagator = &accessorPropagator{rval}
	return rval, nil
}

// makeEndpoint takes the hostport and service name that represent this Zipkin
// service, and returns an endpoint that's embedded into the Zipkin core Span
// type. It will return a nil endpoint if the input parameters are malformed.
func makeEndpoint(hostport, serviceName string) *zipkincore.Endpoint {
	host, port, err := net.SplitHostPort(hostport)
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

// Implements the `Tracer` interface.
type tracerImpl struct {
	options            TracerOptions
	textPropagator     *textMapPropagator
	binaryPropagator   *binaryPropagator
	accessorPropagator *accessorPropagator
}

func (t *tracerImpl) StartSpan(
	operationName string,
	opts ...opentracing.StartSpanOption,
) opentracing.Span {
	sso := opentracing.StartSpanOptions{}
	for _, o := range opts {
		o.Apply(&sso)
	}
	return t.StartSpanWithOptions(operationName, sso)
}

func (t *tracerImpl) StartSpanWithOptions(
	operationName string,
	opts opentracing.StartSpanOptions,
) opentracing.Span {
	// Start time.
	startTime := opts.StartTime
	if startTime.IsZero() {
		startTime = time.Now()
	}

	// Tags.
	tags := opts.Tags

	// Build the new span. This is the only allocation: We'll return this as
	// an opentracing.Span.
	sp := &spanImpl{
		raw: RawSpan{
			Context: &Context{},
		},
	}
	// Look for a parent in the list of References.
	//
	// TODO: would be nice if basictracer did something with all
	// References, not just the first one.
ReferencesLoop:
	for _, ref := range opts.References {
		switch ref.Type {
		case opentracing.ChildOfRef,
			opentracing.FollowsFromRef:

			refSC := ref.Referee.(*Context)
			sp.raw.TraceID = refSC.TraceID
			sp.raw.ParentSpanID = refSC.SpanID
			sp.raw.Sampled = refSC.Sampled

			if tags[string(ext.SpanKind)] == ext.SpanKindRPCServer &&
				t.options.clientServerSameSpan {
				sp.raw.SpanID = refSC.SpanID
				sp.raw.ParentSpanID = refSC.ParentSpanID
			} else {
				sp.raw.SpanID = randomID()
				sp.raw.ParentSpanID = refSC.SpanID
			}

			refSC.Lock()
			if l := len(refSC.Baggage); l > 0 {
				sp.raw.Baggage = make(map[string]string, len(refSC.Baggage))
				for k, v := range refSC.Baggage {
					sp.raw.Baggage[k] = v
				}
			}
			refSC.Unlock()
			break ReferencesLoop
		}
	}

	if sp.raw.TraceID == 0 {
		// No parent Span found; allocate new trace and span ids and determine
		// the Sampled status.
		sp.raw.TraceID, sp.raw.SpanID = randomID2()
		sp.raw.Sampled = t.options.shouldSample(sp.raw.TraceID)
	}

	return t.startSpanInternal(
		sp,
		operationName,
		startTime,
		tags,
	)
}

func (t *tracerImpl) startSpanInternal(
	sp *spanImpl,
	operationName string,
	startTime time.Time,
	tags opentracing.Tags,
) opentracing.Span {
	sp.tracer = t
	sp.event = t.options.newSpanEventListener()
	sp.raw.Operation = operationName
	sp.raw.Start = startTime
	sp.raw.Duration = -1
	sp.raw.Tags = tags
	if t.options.debugAssertSingleGoroutine {
		sp.SetTag(debugGoroutineIDTag, curGoroutineID())
	}
	defer sp.onCreate(operationName)
	return sp
}

type delegatorType struct{}

// Delegator is the format to use for DelegatingCarrier.
var Delegator delegatorType

func (t *tracerImpl) Inject(sc opentracing.SpanContext, format interface{}, carrier interface{}) error {
	switch format {
	case opentracing.TextMap:
		return t.textPropagator.Inject(sc, carrier)
	case opentracing.Binary:
		return t.binaryPropagator.Inject(sc, carrier)
	}
	if _, ok := format.(delegatorType); ok {
		return t.accessorPropagator.Inject(sc, carrier)
	}
	return opentracing.ErrUnsupportedFormat
}

func (t *tracerImpl) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	switch format {
	case opentracing.TextMap:
		return t.textPropagator.Extract(carrier)
	case opentracing.Binary:
		return t.binaryPropagator.Extract(carrier)
	}
	if _, ok := format.(delegatorType); ok {
		return t.accessorPropagator.Extract(carrier)
	}
	return nil, opentracing.ErrUnsupportedFormat
}

func (t *tracerImpl) Options() TracerOptions {
	return t.options
}
