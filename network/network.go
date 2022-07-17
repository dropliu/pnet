package network

import "context"

type (

	// Transport 传输层
	Transport interface {
		// Listen 监听目标地址
		Listen(addr string) (Listener, error)
		// Dial 连接远程
		Dial(addr string) (Conn, error)
	}

	// Listener 监听器
	Listener interface {
		// Accept 监听新的连接
		Accept() (Conn, error)
	}

	// Conn 连接抽象，主要作用就是生成流
	Conn interface {
		// NewStream 在连接上创建一条流
		NewStream(context.Context) (Stream, error)
		// 改变该连接
		Close() error
	}

	// Stream
	Stream interface {
		// Conn 返回Conn所属的链接
		Conn() Conn
		// Read 读数据
		Read(p []byte) (n int, err error)
		// Write 写数据
		Write(p []byte) (n int, err error)
		// 关闭流
		Close() error
	}
)
