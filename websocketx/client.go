package websocketx

import (
	"bytes"
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 允许向对等端写入消息的时间
	writeWait = 10 * time.Second

	// 允许读取下一个pong消息的时间
	pongWait = 60 * time.Second

	// 每隔该时期向对等端发送ping消息。必须小于pongWait。
	pingPeriod = (pongWait * 9) / 10

	// 对等端允许的最大消息大小
	maxMessageSize = 512

	// 发送缓冲大小
	bufSize = 256
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client Client是websocket连接和hub之间的中介。
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// 一个 出站消息 的有缓冲的 channel。
	send chan []byte
}

// readPump  readPump从websocket连接中抽取消息并传给hub。
// 应用程序会为每个连接用goroutine运行readPump。应用程序保证每个连接最多只有一个 reader,
// 通过由这个goroutine执行所有的读取操作来确保这一点
func (c *Client) readPump() {
	//defer func() {
	//	c.hub.unregister <- c
	//	c.conn.Close()
	//}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logx.Errorf("❌ websocket 已关闭：%v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.hub.broadcast <- message
	}
}

// writePump  writePump将消息从hub传输至websocket连接。
// 为每个连接都将启动一个运行writePump的goroutine。
// 应用程序通过只允许这个goroutine执行所有写操作,来确保每个连接最多只有一个writer。
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		//c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			//把排队中的聊天消息添加到当前的websocket消息中。
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs ServeWs处理来自对端的websocket请求。
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logx.Errorf("❌ws 升级http 错误：%v", err)
		return
	}

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, bufSize),
	}
	client.hub.register <- client

	// 通过在新的goroutine中进行所有工作,允许调用者引用的内存被垃圾回收。.
	go client.writePump()
	go client.readPump()
}
