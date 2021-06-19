package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser 	// 通常是通过 TCP 或者 Unix 建立 socket 时得到的链接实例
	buf *bufio.Writer			// 防止阻塞而创建的带缓冲的buffer
	enc *gob.Encoder
	dec *gob.Decoder
}

//结构体是否实现了Codec这个接口
var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec  {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf: buf,
		enc: gob.NewEncoder(buf),
		dec: gob.NewDecoder(conn),
	}
}

/**
read 是解码过程
 */
func (g *GobCodec) ReadHeader(header *Header) error {
	return g.dec.Decode(header)
}

func (g *GobCodec) ReadBody(body interface{}) error {
	return g.dec.Decode(body)
}

/**
write 是编码过程
 */
func (g *GobCodec) Write(header *Header, body interface{}) (err error) {
	defer func() {
		_ = g.buf.Flush()
		if err != nil {
			_ = g.Close()
		}
	}()
	if err = g.enc.Encode(header); err != nil {
		log.Println("rpc encode: gob encoding header error:", err)
		return err
	}
	if err = g.enc.Encode(body); err != nil {
		log.Println("rpc encode: gob encoding body error:", err)
		return err
	}
	return nil
}
func (g *GobCodec) Close() error {
	return g.conn.Close()
}

