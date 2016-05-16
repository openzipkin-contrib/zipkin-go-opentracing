package wire

//go:generate protoc --gogofaster_out=$GOPATH/src/github.com/basvanbeek/zipkin-go-opentracing/wire wire.proto

// Run `go get github.com/gogo/protobuf/protoc-gen-gogofaster` to install the
// gogofaster generator binary.
