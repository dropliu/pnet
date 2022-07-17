package main

import (
	"context"
	"fmt"

	"github.com/dropliu/pnet"
	"github.com/dropliu/pnet/network"
)

// handleStream 流处理器
func handleStream(strm network.Stream) {
	// 读取请求
	var buf [1024]byte
	n, err := strm.Read(buf[:])
	if err != nil {
		fmt.Println("handle stream: read", err)
		return
	}
	fmt.Println("request", string(buf[:n]))

	// 写入想想
	message := append(buf[:n], " world"...)
	_, err = strm.Write([]byte(message))
	if err != nil {
		fmt.Println("handle stream: write", err)
		return
	}
}

func main() {
	const (
		id1      = "id1"
		addr1    = "http2://:5566"
		channel1 = "ftp"
		id2      = "id2"
	)

	// 新建节点1
	node1 := pnet.New(pnet.WithID(id1))
	defer node1.Shutdown()
	// 设置对应路径的流处理器
	node1.SetStreamHandler(channel1, handleStream)
	// 节点1 监听本机端口 5566
	if err := node1.Listen(addr1); err != nil {
		fmt.Println("node1 listen", err)
		return
	}

	// 新建节点2
	node2 := pnet.New(pnet.WithID(id2))
	defer node2.Shutdown()
	// 节点2 连接到节点1
	id, err := node2.Connect(addr1)
	if err != nil {
		fmt.Println("node2 connect", err)
		return
	}
	if id != id1 {
		fmt.Println("mismatched node: ", id)
		return
	}

	// 节点2，新建一条流链接到节点1
	strm, err := node2.NewStream(context.TODO(), id1, channel1)
	if err != nil {
		fmt.Println("new stream", err)
		return
	}
	defer strm.Close()

	// 写入数据，发送消息
	message := []byte("hello")
	if _, err := strm.Write(message); err != nil {
		fmt.Println("write stream", err)
		return
	}

	// 读取数据，接收响应消息
	var buf [1024]byte
	n, err := strm.Read(buf[:])
	if err != nil {
		fmt.Println("read stream", err)
		return
	}

	// 输出响应
	fmt.Println("response", string(buf[:n]))
}
