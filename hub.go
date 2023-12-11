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

}
