package server

import (
	"log"
	"sync"
	"time"

	"github.com/husio/x/hermes/lru"
)

type Message struct {
	MessageID string
	Created   time.Time
	Content   string
}

type msghub struct {
	mu   sync.Mutex
	subs map[chan<- *Message]struct{}
	seen *lru.LRU
}

func newMsgHub() *msghub {
	return &msghub{
		subs: make(map[chan<- *Message]struct{}),
		seen: lru.NewLRU(2000),
	}
}

func (h *msghub) Subscribe(c chan<- *Message) {
	h.mu.Lock()
	h.subs[c] = struct{}{}
	h.mu.Unlock()
}

func (h *msghub) Unsubscribe(c chan<- *Message) {
	h.mu.Lock()
	delete(h.subs, c)
	h.mu.Unlock()
}

func (h *msghub) Publish(m *Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// ignore duplicated messages
	if _, ok := h.seen.Get(m.MessageID); ok {
		return
	}
	h.seen.Set(m.MessageID, struct{}{})

	for c := range h.subs {
		select {
		case c <- m:
		default:
			log.Printf("ignoring slow subscriber: %v", c)
		}
	}
}
