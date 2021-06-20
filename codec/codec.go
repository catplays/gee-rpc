package codec

import (
	"io"
)

/**
rpc 通信过程中请求和响应都需要用的

 */
type  Header struct {
	ServiceMethod string 	// 请求服务和方法，如service.method
	Seq uint64 				// 一次rpc请求的序列号
	Error string			// 异常信息
}

/**
编解码的抽象接口
 */
type Codec interface {
	io.Closer
	ReadHeader(header *Header) error
	ReadBody(interface{}) error
	Write(header *Header, body interface{}) error
}

type NewCodecFunc func(closer io.ReadWriteCloser) Codec

type Type string

const (
	GobType Type = "encoding/gob"
)

var NewCodecFuncMap  map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}

