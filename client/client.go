package client

import (
	gee_rpc "catwang.com/gee-rpc"
	"catwang.com/gee-rpc/codec"
	"errors"
	"fmt"
	"io"
	"sync"
)

/**
承载一次RPC调用所需要的信息
 */
type Call struct {
	Seq uint64					// 请求的序列号
	ServiceMethod string		// 请求方法
	Args interface{}			// 	请求参数
	Reply interface{}			// 响应结构体
	Error error
	Done chan *Call				// 为了支持异步调用，当call完成时会被赋值，
}

//当调用结束时，会调用 call.done() 通知调用方?
func (call *Call) done() {
	call.Done <- call
}

/**
客户端
 */
type Client struct {
	cc codec.Codec				// 编解码器，用来序列化和反序列化消息
	opt *gee_rpc.Option			//
	seq uint64
	header codec.Header			//请求的消息头，header 只有在请求发送时才需要，而请求发送是互斥的，因此每个客户端只需要一个，声明在 Client 结构体中可以复用
	pending map[uint64] *Call		// 还未处理完的请求
	closing bool				// 是否关闭,用户主动关闭
	shutdown bool 				// 是否关闭, shutdown 置为 true 一般是有错误发生
	sending sync.Mutex			// 发送的锁, 保证请求的发送有序
	mu sync.Mutex				// 更新状态的锁
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shut down")


func (client *Client) Close() error{
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShutdown
	}
	client.closing = true
	return client.cc.Close()
}

func (client *Client) IsAvailable() bool{
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.closing && !client.shutdown
}


func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

func (client *Client) terminateCalls (err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()

	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}
func (client *Client) receive() {
	var err error
	for err == nil {
		header := codec.Header{}
		if err = client.cc.ReadHeader(&header); err != nil {
			break
		}
		call := client.removeCall(header.Seq)

		switch {
		case call == nil:
			err = client.cc.ReadBody(nil)

		case header.Error != "":
			call.Error = fmt.Errorf(header.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	// error occurs, so terminateCalls pending calls
	client.terminateCalls(err)
}