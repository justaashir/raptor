package server

type Conn interface {
	Send(msg []byte) error
}

type broadcastMsg struct {
	data []byte
	done chan struct{}
}

type Hub struct {
	register   chan Conn
	unregister chan Conn
	broadcast  chan broadcastMsg
	stop       chan struct{}
	clients    map[Conn]bool
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan Conn),
		unregister: make(chan Conn),
		broadcast:  make(chan broadcastMsg),
		stop:       make(chan struct{}),
		clients:    make(map[Conn]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.stop:
			return
		case c := <-h.register:
			h.clients[c] = true
		case c := <-h.unregister:
			delete(h.clients, c)
		case msg := <-h.broadcast:
			for c := range h.clients {
				if err := c.Send(msg.data); err != nil {
					delete(h.clients, c)
				}
			}
			if msg.done != nil {
				close(msg.done)
			}
		}
	}
}

func (h *Hub) Stop() {
	close(h.stop)
}

func (h *Hub) Register(c Conn)   { h.register <- c }
func (h *Hub) Unregister(c Conn) { h.unregister <- c }
func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- broadcastMsg{data: msg}
}
