package main

import "time"

type clientinfo struct {
	addr         string
	lastmesgsent time.Time
}

type Hub struct {
	clients map[*Client]bool
	// 广播消息缓冲区大小为256
	//另外设置定时器，每隔一段时间清空缓冲区
	//或者超出时清空缓冲区
	broadcast  chan []string
	buffer     []string
	register   chan *Client
	unregister chan *Client
	allclients []clientinfo
}

func newHub(bufferofbroadcast int64, maxconnections int64) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []string),
		buffer:     make([]string, 0, bufferofbroadcast),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		allclients: make([]clientinfo, 0, maxconnections),
	}
}

func (h *Hub) run() {
	//轮询检查不活跃的连接
	go func() {}()
	heartbeatTicker := time.NewTicker(15 * time.Second)
	for {
		select {
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
			//TODO:定时器定时触发websocket ping/pong检测
		case <-heartbeatTicker.C:
			for client := range h.clients {
				select {
				case client.send <- []byte("ping"):
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			/*
				case message := <-h.broadcast:
					//广播消息
					for client := range h.clients {
						select {
						case client.send <- message[0] + message[1] + message[2]:
						default:
							close(client.send)
							delete(h.clients, client)
						}
					}
					//缓冲区消息
					h.buffer = append(h.buffer, message[0]+message[1]+message[2])
					if int64(len(h.buffer)) > cap(h.buffer) {
						h.buffer = h.buffer[1:]
					}
			*/
		}
	}
}

func (h *Hub) checkClientHeartbeat() {
	// 使用互斥锁确保并发安全
	//	h.mutex.Lock()
	//	defer h.mutex.Unlock()

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
