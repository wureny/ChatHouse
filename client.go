package main

import (
	"github.com/gorilla/websocket"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	//消息格式为：发送者用户名+发送时间+消息内容
	//对消息长度应该有限制
	send chan []byte
}

func (c *Client) readPump() {}

func (c *Client) writePump() {}
