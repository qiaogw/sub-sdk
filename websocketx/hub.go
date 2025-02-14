package websocketx

import "github.com/zeromicro/go-zero/core/logx"

// Hub Hub维护客户端的活动集合,并向客户端广播消息。
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			logx.Debugf("【register】:%+v", client)
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				logx.Debugf("【unregister】:%+v", client)
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
					logx.Debugf("【client.send <- message】:%+v", string(message))
				default:
					logx.Debugf("【broadcast delete】:%+v", string(message))
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
