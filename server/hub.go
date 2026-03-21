package server

type Conn interface {
	Send(msg []byte) error
}

type broadcastMsg struct {
	data []byte
	done chan struct{}
}

type client struct {
	conn Conn
	send chan []byte
}

func (cl *client) writePump() {
	for msg := range cl.send {
		if err := cl.conn.Send(msg); err != nil {
			return
		}
	}
}

type Hub struct {
	register   chan Conn
	unregister chan Conn
	broadcast  chan broadcastMsg
	stop       chan struct{}
	clients    map[Conn]*client
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan Conn),
		unregister: make(chan Conn),
		broadcast:  make(chan broadcastMsg),
		stop:       make(chan struct{}),
		clients:    make(map[Conn]*client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.stop:
			for conn, cl := range h.clients {
				close(cl.send)
				delete(h.clients, conn)
			}
			return
		case c := <-h.register:
			cl := &client{
				conn: c,
				send: make(chan []byte, 16),
			}
			h.clients[c] = cl
			go cl.writePump()
		case c := <-h.unregister:
			if cl, ok := h.clients[c]; ok {
				close(cl.send)
				delete(h.clients, c)
			}
		case msg := <-h.broadcast:
			for conn, cl := range h.clients {
				select {
				case cl.send <- msg.data:
				default:
					// Client channel full — drop slow client.
					close(cl.send)
					delete(h.clients, conn)
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
