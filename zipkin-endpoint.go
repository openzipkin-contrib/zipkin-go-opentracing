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

	var addr4, addr16 net.IP
	for i := range addrs {
		if addr := addrs[i].To4(); addr == nil {
			if addr16 == nil {
				addr16 = addrs[i].To16() // IPv6 - 16 bytes
			}
		} else {
			if addr4 == nil {
				addr4 = addr // IPv4 - 4 bytes
			}
		}
		if addr16 != nil && addr4 != nil {
			break
		}
	}
	if addr4 == nil {
		if addr16 == nil {
			return nil
		}
		// we have an IPv6 but no IPv4, code IPv4 as 0 (none found)
		addr4 = []byte("\x00\x00\x00\x00")
	}

	endpoint := zipkincore.NewEndpoint()
	endpoint.Ipv4 = (int32)(binary.BigEndian.Uint32(addr4))
	endpoint.Ipv6 = []byte(addr16)
	endpoint.Port = int16(portInt)
	endpoint.ServiceName = serviceName

	return endpoint
}
