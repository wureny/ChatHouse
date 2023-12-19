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

const CAPOFBUFFER = 1024

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

	//轮询检查不活跃的连接
	go func() {
		for {
			h.checkClientHeartbeat()
			time.Sleep(pongWait)
		}
	}()
	//检查buffer是否大于CAPOFBUFFER，满了就批量发送消息给clients
	go func() {
		for {
			if len(h.buffer) >= CAPOFBUFFER {
				for client := range h.clients {
					select {
					case client.send <- []byte(h.buffer[0]):
						h.buffer = h.buffer[1:]
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			time.Sleep(3 * time.Second)
		}
	}()
	//检查每个用户的lastmsgsent是否在15分钟之内，否则断开连接
	go func() {
		for {
			for i, v := range h.allclients {
				if time.Now().Sub(v.lastmesgsent) > 15*time.Minute {
					for client := range h.clients {
						if client.conn.RemoteAddr().String() == v.addr {
							close(client.send)
							delete(h.clients, client)
							break
						}
					}
					h.allclients = append(h.allclients[:i], h.allclients[i+1:]...)
				}
			}
			time.Sleep(3 * time.Second)
		}
	}()
	//每隔两秒钟将buffer中的消息发送给所有的clients
	go func() {
		for {
			for client := range h.clients {
				select {
				case client.send <- []byte(h.buffer[0]):
					h.buffer = h.buffer[1:]
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			time.Sleep(2 * time.Second)
		}
	}()
	//heartbeatTicker := time.NewTicker(15 * time.Second)
	for {
		select {
		//TODO：处理最大连接数
		//TODO：处理重复连接
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
		/*case <-heartbeatTicker.C:
		for client := range h.clients {
			select {
			case client.send <- []byte("ping"):
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}*/
		//TODO：数据格式问题
		//TODO: 更新发送消息的client的lastmsgsent
		case message := <-h.broadcast:
			go func() {
				for i, v := range h.allclients {
					if v.addr == message[0] {
						h.allclients[i].lastmesgsent = time.Now()
						break
					}
				}
			}()
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
