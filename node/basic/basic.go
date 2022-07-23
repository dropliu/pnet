package basic

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/dropliu/pnet/network"
	"github.com/dropliu/pnet/node"
	"github.com/dropliu/pnet/peer"
)

var (
	_ node.Node = (*basicNode)(nil)
)

type (
	basicNode struct {
		id string

		peerStore peer.PeerStore

		listeners listener
		handlers  streamHandler
		transport network.Transport
	}

	listener struct {
		h map[network.Listener]struct{}
		m sync.RWMutex
	}

	streamHandler struct {
		h map[string]node.StreamHandler
		m sync.RWMutex
	}

	Options struct {
		ID        string
		Transport network.Transport
	}
)

func New(opts Options) *basicNode {
	node := &basicNode{
		id: opts.ID,
		listeners: listener{
			h: make(map[network.Listener]struct{}),
		},
		handlers: streamHandler{
			h: make(map[string]node.StreamHandler),
		},
	}

	if len(node.id) == 0 {
		var buf [16]byte
		rand.Read(buf[:])
		node.id = hex.EncodeToString(buf[:])
	}

	return node
}

func (n *basicNode) ID() string {
	return n.id
}

func (n *basicNode) Listen(addr string) error {
	// TODO: 检查transport是否为空，缺省使用全局transport
	trans := n.transport
	ln, err := trans.Listen(addr)
	if err != nil {
		return err
	}

	n.listeners.m.Lock()
	n.listeners.h[ln] = struct{}{}
	n.listeners.m.Unlock()

	go n.serve(ln)

	return nil
}

func (n *basicNode) serve(ln network.Listener) {
	defer func() {
		n.listeners.m.Lock()
		delete(n.listeners.h, ln)
		n.listeners.m.Unlock()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}

		go n.addConn(conn)
	}
}

// Metedata 相关

type MetadataKey string

const (
	MetadataID     = "id"
	MetadatChannel = "channel"
)

func (n *basicNode) addConn(c network.Conn) {
	var identity string
	for {
		stream, err := c.AcceptStream(context.Background())
		if err != nil {
			return
		}
		metadata, err := n.identifyStream(stream)
		if err != nil {
			// 验证未通过
			log.Println("identify stream failed: %s", err.Error())
			return
		}

		// 未验证过
		if len(identity) == 0 {
			if id, ok := metadata[MetadataID]; !ok {
				log.Println("identify stream: no id")
				return
			} else {
				identity = id
			}
		}

		channel, ok := metadata[MetadatChannel]
		if !ok {
			stream.Close() // 关闭流
			log.Println("identify stream: no channel")
			continue
		}

		// TODO: 添加到peerstore

		// 处理流
		go n.handleStream(channel, stream)
	}
}

type Metadata struct {
	ID      string `json:"id"`
	Channel string `json:"channel"`
}

// identityStream 校验流
// FIXME: 校验协议过于粗糙，需要改进
// TODO: 上层计划实现一个收发消息的node，可考虑整合在一起
func (n *basicNode) identifyStream(strm network.Stream) (map[string]string, error) {
	strm.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	// 读取数据长度
	var b [4]byte
	if _, err := io.ReadFull(strm, b[:]); err != nil {
		return nil, err
	}
	var len = binary.BigEndian.Uint32(b[:])
	// 读取数据
	var buf = make([]byte, len)
	_, err := io.ReadFull(strm, buf)
	if err != nil {
		return nil, err
	}
	// 解析metadata
	var h map[string]string
	if err := json.Unmarshal(buf, &h); err != nil {
		return nil, err
	}

	return h, nil
}

func (n *basicNode) handleStream(channel string, stream network.Stream) {
	n.handlers.m.RLock()
	fn, ok := n.handlers.h[channel]
	n.handlers.m.RUnlock()

	defer stream.Close()

	// 当对应的channl不存在时，要返回异常给对面
	if !ok {
		// TODO: 响应异常
		log.Printf("handleStream: channle '%s' not exists", channel)
		return
	}

	// 响应metedata
	var h = map[string]string{
		MetadataID: n.id,
	}

	metadata, _ := json.Marshal(h)
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(metadata)))

	if _, err := stream.Write(length[:]); err != nil {
		return
	}
	if _, err := stream.Write(metadata); err != nil {
		return
	}

	// handle stream
	fn(stream)
}

func (n *basicNode) Addrs() []string {
	n.listeners.m.RLock()
	defer n.listeners.m.RUnlock()

	addrs := make([]string, len(n.listeners.h))
	for ln := range n.listeners.h {
		addrs = append(addrs, ln.Addr())
	}

	return addrs
}

// SetStreamHandler 设置流处理器
// TODO: 考虑用前缀树？
// TODO: 直接字符串匹配有点过于简单粗暴，考虑用规则匹配？
func (n *basicNode) SetStreamHandler(channel string, fn node.StreamHandler) error {
	n.handlers.m.Lock()
	defer n.handlers.m.Unlock()

	if _, ok := n.handlers.h[channel]; ok {
		return fmt.Errorf("channel '%s' already has a handler", channel)
	}
	n.handlers.h[channel] = fn
	return nil
}

func (n *basicNode) RemoveStreamHandler(channel string) {
	n.handlers.m.Lock()
	defer n.handlers.m.Unlock()

	delete(n.handlers.h, channel)
}

const IdentifyChannel = "identify"

// Connect 连接到目标节点（阻塞式）
func (n *basicNode) Connect(addr string) (id string, err error) {
	conn, err := n.transport.Dial(addr)
	if err != nil {
		return
	}

	stream, err := conn.NewStream(context.Background())
	if err != nil {
		return "", nil
	}
	defer stream.Close()

	var metedata = map[string]string{
		MetadataID:     n.id,
		MetadatChannel: IdentifyChannel,
	}
	msg, _ := json.Marshal(metedata)

	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(msg)))
	if _, err := stream.Write(msg); err != nil {
		return "", err
	}

	if _, err := stream.Write(msg); err != nil {
		return "", err
	}

	if _, err = io.ReadFull(stream, length[:]); err != nil {
		return "", err
	}

	var buf = make([]byte, binary.BigEndian.Uint32(length[:]))
	if _, err = io.ReadFull(stream, buf); err != nil {
		return "", err
	}

	var h map[string]string
	if err := json.Unmarshal(buf, &h); err != nil {
		return "", err
	}

	id, ok := h[MetadataID]
	if !ok {
		return "", fmt.Errorf("identify stream: no channel")
	}

	return id, nil
}

func (n *basicNode) PeerStore() peer.PeerStore {
	return n.peerStore
}

func (n *basicNode) NewStream(ctx context.Context, ID, Channel string) (network.Stream, error) {
	return nil, nil
}

func (*basicNode) Shutdown() error {
	return nil
}
