package server

import (
	"errors"
	"sync"
	"testing"
	"time"
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

func waitForMessages(f *fakeConn, count int, timeout time.Duration) [][]byte {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		msgs := f.Messages()
		if len(msgs) >= count {
			return msgs
		}
		time.Sleep(time.Millisecond)
	}
	return f.Messages()
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

	msgs1 := waitForMessages(c1, 1, 100*time.Millisecond)
	msgs2 := waitForMessages(c2, 1, 100*time.Millisecond)

	if len(msgs1) != 1 {
		t.Fatalf("expected 1 message for c1, got %d", len(msgs1))
	}
	if len(msgs2) != 1 {
		t.Fatalf("expected 1 message for c2, got %d", len(msgs2))
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

	// Wait for writePump to deliver the message before unregistering.
	waitForMessages(c1, 1, 100*time.Millisecond)

	hub.Unregister(c1)

	done2 := make(chan struct{})
	hub.broadcast <- broadcastMsg{data: []byte(`after-unregister`), done: done2}
	<-done2

	// Give writePump time to finish draining (it should already be done).
	time.Sleep(10 * time.Millisecond)

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

	msgs := waitForMessages(good, 2, 100*time.Millisecond)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages for good client, got %d", len(msgs))
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

type slowConn struct {
	mu       sync.Mutex
	messages [][]byte
}

func (s *slowConn) Send(msg []byte) error {
	time.Sleep(500 * time.Millisecond)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
	return nil
}

func (s *slowConn) Messages() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([][]byte{}, s.messages...)
}

func TestHub_SlowClientDoesNotBlockBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	slow := &slowConn{}
	fast := &fakeConn{}
	hub.Register(slow)
	hub.Register(fast)

	start := time.Now()
	done := make(chan struct{})
	hub.broadcast <- broadcastMsg{data: []byte(`hello`), done: done}
	<-done
	elapsed := time.Since(start)

	if elapsed >= 50*time.Millisecond {
		t.Fatalf("broadcast took %v, expected < 50ms (slow client blocked)", elapsed)
	}

	msgs := waitForMessages(fast, 1, 100*time.Millisecond)
	if len(msgs) != 1 {
		t.Fatalf("expected fast client to receive 1 message, got %d", len(msgs))
	}
}
