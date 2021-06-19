package main

import (
	"catwang.com/gee-rpc"
	geerpc "catwang.com/gee-rpc/codec"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

func startServer(addr chan string) {
	l, err := net.Listen("tcp",":0")
	if err != nil {
		panic(err)
	}
	log.Println("startServer at:", l.Addr())
	addr <- l.Addr().String()
	gee_rpc.Accept(l)
}


func main() {

	addr := make(chan string)
	go startServer(addr)

	// simple geerpc client
	conn, _ := net.Dial("tcp", <-addr)
	defer func() {
		conn.Close()
	}()
	time.Sleep(time.Second)

	// send option
	json.NewEncoder(conn).Encode(gee_rpc.DefaultOption)

	cc := geerpc.NewGobCodec(conn)

	for i:=0; i<5; i++ {
		h := &geerpc.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           int64(i),
		}
		cc.Write(h, fmt.Sprintf("geerpc req:%d",h.Seq))
		cc.ReadHeader(h)
		var reply string
		cc.ReadBody(&reply)
		log.Println("reply:",reply)
	}
}

