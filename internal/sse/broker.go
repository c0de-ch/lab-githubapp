package sse

import (
	"context"
	"sync"
)

type subscription struct {
	topic string
	ch    chan Event
}

type topicEvent struct {
	topic string
	event Event
}

type Broker struct {
	mu         sync.RWMutex
	clients    map[string]map[chan Event]struct{}
	register   chan subscription
	unregister chan subscription
	broadcast  chan topicEvent
}

func NewBroker() *Broker {
	return &Broker{
		clients:    make(map[string]map[chan Event]struct{}),
		register:   make(chan subscription),
		unregister: make(chan subscription),
		broadcast:  make(chan topicEvent, 256),
	}
}

func (b *Broker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case sub := <-b.register:
			b.mu.Lock()
			if b.clients[sub.topic] == nil {
				b.clients[sub.topic] = make(map[chan Event]struct{})
			}
			b.clients[sub.topic][sub.ch] = struct{}{}
			b.mu.Unlock()
		case sub := <-b.unregister:
			b.mu.Lock()
			if subs, ok := b.clients[sub.topic]; ok {
				delete(subs, sub.ch)
				close(sub.ch)
				if len(subs) == 0 {
					delete(b.clients, sub.topic)
				}
			}
			b.mu.Unlock()
		case te := <-b.broadcast:
			b.mu.RLock()
			if subs, ok := b.clients[te.topic]; ok {
				for ch := range subs {
					select {
					case ch <- te.event:
					default:
						// Client is slow, skip this event
					}
				}
			}
			b.mu.RUnlock()
		}
	}
}

func (b *Broker) Subscribe(topic string) (<-chan Event, func()) {
	ch := make(chan Event, 16)
	sub := subscription{topic: topic, ch: ch}
	b.register <- sub
	return ch, func() {
		b.unregister <- sub
	}
}

func (b *Broker) Publish(topic string, event Event) {
	b.broadcast <- topicEvent{topic: topic, event: event}
}
