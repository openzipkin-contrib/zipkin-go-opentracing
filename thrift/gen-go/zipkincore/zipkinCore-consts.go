// Autogenerated by Thrift Compiler (1.0.0-dev)
// DO NOT EDIT UNLESS YOU ARE SURE THAT YOU KNOW WHAT YOU ARE DOING

package zipkincore

import (
	"bytes"
	"context"
	"fmt"
	"github.com/apache/thrift/lib/go/thrift"
	"reflect"
)

// (needed to ensure safety because of naive import list construction.)
var _ = thrift.ZERO
var _ = fmt.Printf
var _ = context.Background
var _ = reflect.DeepEqual
var _ = bytes.Equal

const CLIENT_SEND = "cs"
const CLIENT_RECV = "cr"
const SERVER_SEND = "ss"
const SERVER_RECV = "sr"
const WIRE_SEND = "ws"
const WIRE_RECV = "wr"
const CLIENT_SEND_FRAGMENT = "csf"
const CLIENT_RECV_FRAGMENT = "crf"
const SERVER_SEND_FRAGMENT = "ssf"
const SERVER_RECV_FRAGMENT = "srf"
const HTTP_HOST = "http.host"
const HTTP_METHOD = "http.method"
const HTTP_PATH = "http.path"
const HTTP_URL = "http.url"
const HTTP_STATUS_CODE = "http.status_code"
const HTTP_REQUEST_SIZE = "http.request.size"
const HTTP_RESPONSE_SIZE = "http.response.size"
const LOCAL_COMPONENT = "lc"
const ERROR = "error"
const CLIENT_ADDR = "ca"
const SERVER_ADDR = "sa"

func init() {
}
