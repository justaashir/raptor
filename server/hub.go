package server

type Conn interface {
	Send(msg []byte) error
}

type Hub struct {
	register   chan Conn
	unregister chan Conn
	broadcast  chan []byte
	clients    map[Conn]bool
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan Conn),
		unregister: make(chan Conn),
		broadcast:  make(chan []byte),
		clients:    make(map[Conn]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
		case c := <-h.unregister:
			delete(h.clients, c)
		case msg := <-h.broadcast:
			for c := range h.clients {
				c.Send(msg)
			}
		}
	}
}

func (h *Hub) Register(c Conn)      { h.register <- c }
func (h *Hub) Unregister(c Conn)    { h.unregister <- c }
func (h *Hub) Broadcast(msg []byte) { h.broadcast <- msg }
