package peer

import "github.com/dropliu/pnet/network"

type (

	// Network 网络，准确的说是连接管理器
	PeerStore interface {
		PeerInfo(id string)

		// ConnsToPerr 连接到指定对端的连接
		ConnsToPeer(id string) []network.Conn
		// 关闭到对端的连接
		CloseConn(id string) error

		// Status
		Stats() Stats
		// Close 关闭所有的链接
		Close() error
	}

	// Stats 统计数据，上层应用需要统计当前网络状况
	// 1. 有多少个分区？ 连接了多少节点？每个节点采用什么协议？
	// 2. TODO: 待定
	Stats interface{}
)
