package zipkintracer

// CoreSpan represents the span to be sent to the zipkin server
type CoreSpan struct {
	TraceID           string                  `json:"traceId"`
	Name              string                  `json:"name"`
	ID                string                  `json:"id"`
	ParentID          string                  `json:"parentId,omitempty"`
	Debug             bool                    `json:"debug,omitempty"`
	Timestamp         int64                   `json:"timestamp,omitempty"`
	Duration          int64                   `json:"duration,omitempty"`
	TraceIDHigh       string                  `json:"traceIdHigh,omitempty"`
	Annotations       []*CoreAnnotation       `json:"annotations"`
	BinaryAnnotations []*CoreBinaryAnnotation `json:"binaryAnnotations"`
}

// CoreBinaryAnnotation represents the tags added in the span
type CoreBinaryAnnotation struct {
	Key      string       `json:"key"`
	Value    string       `json:"value"`
	Endpoint CoreEndpoint `json:"endpoint"`
}

// CoreAnnotation represents the tags that are inherited from parent spans
type CoreAnnotation struct {
	Timestamp int64         `json:"timestamp"`
	Value     string        `json:"value"`
	Host      *CoreEndpoint `json:"host,omitempty"`
}

// CoreEndpoint represents the tags that are applied to all spans from the given service.
type CoreEndpoint struct {
	Ipv4        string `json:"ipv4"`
	Port        int16  `json:"port"`
	ServiceName string `json:"serviceName"`
	Ipv6        string `json:"ipv6,omitempty"`
}
