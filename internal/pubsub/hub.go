package pubsub

import (
	"context"
	"log"

	"github.com/coder/websocket"
)

type Message struct {
	PollID string
	Data   []byte
}

// one client conenected via websocket
type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
	PollID string
}

type Hub struct {
	Clients    map[string]map[*Client]bool
	Broadcast  chan *Message
	Register   chan *Client
	Unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[string]map[*Client]bool),
		Broadcast:  make(chan *Message),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			conn := h.Clients[client.PollID]
			if conn == nil {
				conn = make(map[*Client]bool)
				h.Clients[client.PollID] = conn
			}
			conn[client] = true

		case client := <-h.Unregister:
			conn := h.Clients[client.PollID]
			if conn != nil {
				if _, ok := conn[client]; ok {
					delete(conn, client)
					close(client.Send)
					if len(conn) == 0 {
						delete(h.Clients, client.PollID)
					}
				}
			}

		case message := <-h.Broadcast:
			conn := h.Clients[message.PollID]
			for c := range conn {
				select {
				case c.Send <- message.Data:

				default:
					close(c.Send)
					delete(conn, c)
				}
			}
		}
	}
}

// WritePump sends messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	for m := range c.Send {
		err := c.Conn.Write(context.Background(), websocket.MessageText, m)
		if err != nil {
			log.Printf("Error writing to client %s: %v", c.PollID, err)
			break
		}
	}
}

// ReadPump listens for messages from the WebSocket connection
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		_, _, err := c.Conn.Read(context.Background())
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				log.Printf("Client %s disconnected normally.", c.PollID)
			} else {
				log.Printf("Error reading from client %s: %v", c.PollID, err)
			}
			break
		}
	}
}
