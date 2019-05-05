package models


type Span struct {
	TraceID string `json:"traceId"`
	// unused field # 2
	Name string `json:"name"`
	ID string `json:"id"`
	ParentID string `json:"parentId,omitempty"`
	Debug bool `json:"debug,omitempty"`
	Timestamp int64 `json:"timestamp,omitempty"`
	Duration int64 `json:"duration,omitempty"`
	TraceIDHigh string `json:"traceIdHigh,omitempty"`
	BinaryAnnotations []*BinaryAnnotation `json:"binaryAnnotations"`
}

type BinaryAnnotation struct {
	Key string `json:"key"`
	Value string `json:"value"`
	Endpoint Endpoint `json:"endpoint"`
}

type Endpoint struct {
	Ipv4 string `json:"ipv4"`
	Port int64  `json:"port"`
	ServiceName string `json:"serviceName"`
	Ipv6 string `json:"ipv6,omitempty"`
}