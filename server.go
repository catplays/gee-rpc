package gee_rpc

import (
	"catwang.com/gee-rpc/codec"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

/**

GeeRPC 客户端固定采用 JSON 编码 Option，后续的 header 和 body 的编码方式
由 Option 中的 CodeType 指定，服务端首先使用 JSON 解码 Option，
然后通过 Option 得 CodeType 解码剩余的内容

 */
const MagicNumber = 0x3bef5c
// 消息协议
type Option struct {
	MagicNumber int        // 表示这是gee-prc
	CodecType   codec.Type // rpc 的编解码方式

}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

type Server struct {

}

type Request struct {
	h *codec.Header
	argv, replyV reflect.Value		// 请求的参数和响应值
}
var DefaultServer = NewServer()

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Accept (listener net.Listener)  {
	for  {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		go s.ServerConn(conn)
	}
}

func (s *Server) ServerConn(conn io.ReadWriteCloser) {
	defer func() {
		conn.Close()
	}()
	var option = &Option{}
	if err := json.NewDecoder(conn).Decode(option); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if option.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", option.MagicNumber)
		return
	}
	codecFunc, ok := codec.NewCodecFuncMap[option.CodecType]
	if !ok {
		log.Printf("rpc server: CodecTyper %x", option.CodecType)
		return
	}
	s.ServerCodec(codecFunc(conn))
}
var invalidRequest = struct{}{}

func (s *Server) ServerCodec(cc codec.Codec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	/**
	在一次连接中，允许接收多个请求，即多个 request header 和 request body，因此这里使用了 for 无限制地等待请求的到来，直到发生错误
	 */
	for  {
		req , err := s.readRequest(cc)
		if err != nil {
			if req == nil {
				break // it's not possible to recover, so close the connection
			}
			req.h.Error = err.Error()
			s.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go s.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	cc.Close()
}

func (s *Server) readRequest(cc codec.Codec) (*Request, error) {
	header, err := s.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &Request{
		h: header,
	}
	req.argv = reflect.New(reflect.TypeOf(""))
	if err := cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}

func (s *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	header := &codec.Header{}
	if err := cc.ReadHeader(header); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Printf("readHeader error :%v", err)
		}
		return nil, err
	}
	return header, nil
}


func (s *Server) handleRequest(cc codec.Codec, req *Request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.replyV =  reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.h.Seq))
	s.sendResponse(cc, req.h, req.replyV.Interface(), sending)
}

func (s *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}


func Accept(listener net.Listener)  {
	DefaultServer.Accept(listener)
}

