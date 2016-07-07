package zipkintracer

import (
	"encoding/binary"
	"net"
	"strconv"

	"github.com/openzipkin/zipkin-go-opentracing/_thrift/gen-go/zipkincore"
)

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
