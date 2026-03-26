package agentruntime

import "sync"

type Subscriber[T any] chan T

type Hub[T any] struct {
	mu          sync.RWMutex
	subscribers map[int64]map[Subscriber[T]]struct{}
}

func NewHub[T any]() *Hub[T] {
	return &Hub[T]{
		subscribers: make(map[int64]map[Subscriber[T]]struct{}),
	}
}

func (h *Hub[T]) Subscribe(id int64) (Subscriber[T], func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(Subscriber[T], 32)
	if _, ok := h.subscribers[id]; !ok {
		h.subscribers[id] = make(map[Subscriber[T]]struct{})
	}
	h.subscribers[id][ch] = struct{}{}

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if set, ok := h.subscribers[id]; ok {
			delete(set, ch)
			if len(set) == 0 {
				delete(h.subscribers, id)
			}
		}
		close(ch)
	}
}

func (h *Hub[T]) Publish(id int64, value T) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.subscribers[id] {
		select {
		case ch <- value:
		default:
		}
	}
}
