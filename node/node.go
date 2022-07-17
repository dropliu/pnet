package node

import (
	"context"

	"github.com/dropliu/pnet/network"
	"github.com/dropliu/pnet/peer"
)

type (
	// Node 点对点网络中最基本的单元
	Node interface {
		// 节点ID == Network.ID
		ID() string

		// Connect 连接到目标节点
		Listen(addr string) error
		// Addrs 返回当前节点监听的地址
		Addrs() []string

		// SetStreamHandler 根据通道设置流处理器
		SetStreamHandler(channel string, fn StreamHandler) error
		// RemoveStreamHandler 移除流处理器
		RemoveStreamHandler(channel string)

		// Connect 根据地址连接到目标节点，会阻塞直到连接成功或失败
		Connect(addr string) (id string, err error)

		// PeerStore 返回对端存储，所有已连接对端的信息，会放到该存储中
		PeerStore() peer.PeerStore

		// NewStream  创建一条新的流到目标节点
		NewStream(ctx context.Context, ID, Channel string) (network.Stream, error)

		// Shutdown 关闭节点
		Shutdown() error
	}

	StreamHandler func(network.Stream)
)
