package main

import (
	"sync"
	"time"
)

type clientinfo struct {
	addr         string
	lastmesgsent time.Time
}

type Hub struct {
	clientMutex sync.Mutex
	clients     map[*Client]bool
	// 广播消息缓冲区大小为256
	//另外设置定时器，每隔一段时间清空缓冲区
	//或者超出时清空缓冲区
	broadcast chan []string

	bufferMutex sync.Mutex
	buffer      []string
	register    chan *Client
	unregister  chan *Client

	allclients   []clientinfo
	clientsMutex sync.Mutex
}

func newHub(bufferofbroadcast int64, maxconnections int64) *Hub {
	return &Hub{
		clientMutex: sync.Mutex{},
		clients:     make(map[*Client]bool),

		broadcast: make(chan []string),

		bufferMutex: sync.Mutex{},
		buffer:      make([]string, 0, bufferofbroadcast),

		register:   make(chan *Client),
		unregister: make(chan *Client),

		clientsMutex: sync.Mutex{},
		allclients:   make([]clientinfo, 0, maxconnections),
	}
}

func (h *Hub) run() {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	h.bufferMutex.Lock()
	defer h.bufferMutex.Unlock()
	h.clientMutex.Lock()
	defer h.clientMutex.Unlock()

	//TODO：轮询检查不活跃的连接
	go func() {}()
	//TODO：检查buffer是否已满，满了就批量发送消息给clients
	go func() {}()
	heartbeatTicker := time.NewTicker(15 * time.Second)
	for {
		select {
		//TODO：处理最大连接数
		case client := <-h.register:
			h.clients[client] = true
			h.allclients = append(h.allclients, clientinfo{client.conn.RemoteAddr().String(), time.Now()})
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				for i, v := range h.allclients {
					if v.addr == client.conn.RemoteAddr().String() {
						h.allclients = append(h.allclients[:i], h.allclients[i+1:]...)
						break
					}
				}
				close(client.send)
			}
			//定时器定时触发websocket ping/pong检测
		case <-heartbeatTicker.C:
			for client := range h.clients {
				select {
				case client.send <- []byte("ping"):
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			//TODO：数据格式问题
		case message := <-h.broadcast:
			//检查buffer是否已满，没满便加入buffer缓存
			if len(h.buffer)+len(message[0]+message[1]+message[2]) <= cap(h.buffer) {
				h.buffer = append(h.buffer, message[0]+message[1]+message[2])
			} else {
				time.Sleep(3 * time.Second)
			}
			if len(h.buffer)+len(message[0]+message[1]+message[2]) <= cap(h.buffer) {
				h.buffer = append(h.buffer, message[0]+message[1]+message[2])
			}
		}
	}
}

func (h *Hub) checkClientHeartbeat() {
	// 使用互斥锁确保并发安全
	h.clientMutex.Lock()
	defer h.clientMutex.Unlock()

	// 遍历客户端，发送 Ping 消息
	for client := range h.clients {
		select {
		case client.send <- []byte("ping"):
			// Ping 消息成功发送
		default:
			// 发送失败，可能是通道已关闭，需要处理失活的客户端
			close(client.send)
			delete(h.clients, client)
		}
	}
}
