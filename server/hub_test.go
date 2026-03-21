package server

import (
	"errors"
	"sync"
	"testing"
)

type fakeConn struct {
	mu       sync.Mutex
	messages [][]byte
	failSend bool
}

func (f *fakeConn) Send(msg []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failSend {
		return errors.New("connection closed")
	}
	f.messages = append(f.messages, msg)
	return nil
}

func (f *fakeConn) Messages() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]byte{}, f.messages...)
}

func TestHub_RegisterAndBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	c1 := &fakeConn{}
	c2 := &fakeConn{}
	hub.Register(c1)
	hub.Register(c2)

	done := make(chan struct{})
	hub.broadcast <- broadcastMsg{data: []byte(`{"event":"ticket_changed"}`), done: done}
	<-done

	if len(c1.Messages()) != 1 {
		t.Fatalf("expected 1 message for c1, got %d", len(c1.Messages()))
	}
	if len(c2.Messages()) != 1 {
		t.Fatalf("expected 1 message for c2, got %d", len(c2.Messages()))
	}
}

func TestHub_Unregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	c1 := &fakeConn{}
	hub.Register(c1)

	done := make(chan struct{})
	hub.broadcast <- broadcastMsg{data: []byte(`ping`), done: done}
	<-done

	hub.Unregister(c1)

	done2 := make(chan struct{})
	hub.broadcast <- broadcastMsg{data: []byte(`after-unregister`), done: done2}
	<-done2

	if len(c1.Messages()) != 1 {
		t.Fatalf("expected 1 message (only before unregister), got %d", len(c1.Messages()))
	}
}

func TestHub_RemovesDeadClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	good := &fakeConn{}
	dead := &fakeConn{failSend: true}
	hub.Register(good)
	hub.Register(dead)

	done := make(chan struct{})
	hub.broadcast <- broadcastMsg{data: []byte(`msg1`), done: done}
	<-done

	done2 := make(chan struct{})
	hub.broadcast <- broadcastMsg{data: []byte(`msg2`), done: done2}
	<-done2

	if len(good.Messages()) != 2 {
		t.Fatalf("expected 2 messages for good client, got %d", len(good.Messages()))
	}
}

func TestHub_Stop(t *testing.T) {
	hub := NewHub()
	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()
	hub.Stop()
	<-done
}
