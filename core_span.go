package zipkintracer

type CoreSpan struct {
	TraceID           string                  `json:"traceId"`
	Name              string                  `json:"name"`
	ID                string                  `json:"id"`
	ParentID          string                  `json:"parentId,omitempty"`
	Debug             bool                    `json:"debug,omitempty"`
	Timestamp         int64                   `json:"timestamp,omitempty"`
	Duration          int64                   `json:"duration,omitempty"`
	TraceIDHigh       string                  `json:"traceIdHigh,omitempty"`
	BinaryAnnotations []*CoreBinaryAnnotation `json:"binaryAnnotations"`
}

type CoreBinaryAnnotation struct {
	Key      string       `json:"key"`
	Value    string       `json:"value"`
	Endpoint CoreEndpoint `json:"endpoint"`
}

type CoreEndpoint struct {
	Ipv4        string `json:"ipv4"`
	Port        int16  `json:"port"`
	ServiceName string `json:"serviceName"`
	Ipv6        string `json:"ipv6,omitempty"`
}
