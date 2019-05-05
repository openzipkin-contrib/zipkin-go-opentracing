package models


type Span struct {
	TraceID string `json:"trace_id"`
	// unused field # 2
	Name string `json:"name"`
	ID string `json:"id"`
	ParentID string `json:"parent_id,omitempty"`
	Debug bool `json:"debug,omitempty"`
	Timestamp int64 `json:"timestamp,omitempty"`
	Duration int64 `json:"duration,omitempty"`
	TraceIDHigh string `json:"trace_id_high,omitempty"`
	BinaryAnnotations []*BinaryAnnotation `json:"binary_annotations"`
}

type BinaryAnnotation struct {
	Key string `json:"key"`
	Value string `json:"value"`
}