package events

import (
	"sync"
	"time"
)

// EventType identifies published event categories.
type EventType string

const (
	// EventCommandExecuted is emitted after a command completes.
	EventCommandExecuted EventType = "command_executed"
)

// Event captures domain happenings for observers.
type Event struct {
	Type      EventType
	Timestamp time.Time
	Command   string
	Raw       string
	File      string
	Metadata  map[string]string
}

// Listener consumes published events.
type Listener interface {
	Handle(Event)
}

// Bus is a simple observer dispatcher.
type Bus struct {
	mu        sync.RWMutex
	listeners []Listener
}

// NewBus creates an event bus.
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe registers a listener.
func (b *Bus) Subscribe(listener Listener) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.listeners = append(b.listeners, listener)
}

// Publish sends an event to listeners.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, l := range b.listeners {
		l.Handle(event)
	}
}
