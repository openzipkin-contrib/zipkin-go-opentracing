// +build go1.7

package main

import (
	"fmt"
	"net/http"
	"os"

	opentracing "github.com/opentracing/opentracing-go"

	zipkinopentracing "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"github.com/openzipkin-contrib/zipkin-go-opentracing/examples/cli_with_2_services/svc2"
	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter/http"
)

const (
	// Our service name.
	serviceName = "svc2"

	// Host + port of our service.
	hostPort = "127.0.0.1:61002"

	// Endpoint to send Zipkin spans to.
	zipkinHTTPEndpoint = "http://localhost:9411/api/v1/spans"

	// Debug mode.
	debug = false

	// same span can be set to true for RPC style spans (Zipkin V1) vs Node style (OpenTracing)
	sameSpan = true

	// make Tracer generate 128 bit traceID's for root spans.
	traceID128Bit = true
)

//svc2
func main() {
	// Create our HTTP collector.
	reporter := zipkinreporter.NewReporter(zipkinHTTPEndpoint)
	defer reporter.Close()

	// Create our tracer.
	tracer, err := zipkinopentracing.NewTracer(
		reporter,
		zipkin.WithLocalEndpoint(&model.Endpoint{ServiceName: serviceName}),
		zipkin.WithSharedSpans(sameSpan),
		zipkin.WithTraceID128Bit(traceID128Bit),
	)
	if err != nil {
		fmt.Printf("unable to create Zipkin tracer: %+v\n", err)
		os.Exit(-1)
	}

	// explicitly set our tracer to be the default tracer.
	opentracing.InitGlobalTracer(tracer)

	// create the service implementation
	service := svc2.NewService()

	// create the HTTP Server Handler for the service
	handler := svc2.NewHTTPHandler(tracer, service)

	// start the service
	fmt.Printf("Starting %s on %s\n", serviceName, hostPort)
	http.ListenAndServe(hostPort, handler)
}
