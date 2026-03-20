package server

import (
	"testing"
	"time"
)

type fakeConn struct {
	messages [][]byte
}

func (f *fakeConn) Send(msg []byte) error {
	f.messages = append(f.messages, msg)
	return nil
}

func TestHub_RegisterAndBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := &fakeConn{}
	c2 := &fakeConn{}
	hub.Register(c1)
	hub.Register(c2)

	// Give goroutine time to process
	time.Sleep(10 * time.Millisecond)

	hub.Broadcast([]byte(`{"event":"ticket_changed"}`))
	time.Sleep(10 * time.Millisecond)

	if len(c1.messages) != 1 {
		t.Fatalf("expected 1 message for c1, got %d", len(c1.messages))
	}
	if len(c2.messages) != 1 {
		t.Fatalf("expected 1 message for c2, got %d", len(c2.messages))
	}
}

func TestHub_Unregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := &fakeConn{}
	hub.Register(c1)
	time.Sleep(10 * time.Millisecond)

	hub.Unregister(c1)
	time.Sleep(10 * time.Millisecond)

	hub.Broadcast([]byte(`{"event":"ticket_changed"}`))
	time.Sleep(10 * time.Millisecond)

	if len(c1.messages) != 0 {
		t.Fatalf("expected 0 messages after unregister, got %d", len(c1.messages))
	}
}
