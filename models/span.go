package models


type Span struct {
	TraceID uint64 `json:"trace_id"`
	// unused field # 2
	Name string `thrift:"name,3" db:"name" json:"name"`
	ID uint64 `thrift:"id,4" db:"id" json:"id"`
	ParentID *uint64 `thrift:"parent_id,5" db:"parent_id" json:"parent_id,omitempty"`
	Debug bool `thrift:"debug,9" db:"debug" json:"debug,omitempty"`
	Timestamp int64 `thrift:"timestamp,10" db:"timestamp" json:"timestamp,omitempty"`
	Duration int64 `thrift:"duration,11" db:"duration" json:"duration,omitempty"`
	TraceIDHigh uint64 `thrift:"trace_id_high,12" db:"trace_id_high" json:"trace_id_high,omitempty"`
	BinaryAnnotations []*BinaryAnnotation `json:"binary_annotations"`
}

type BinaryAnnotation struct {
	Key string `thrift:"key,1" db:"key" json:"key"`
	Value []byte `thrift:"value,2" db:"value" json:"value"`
}