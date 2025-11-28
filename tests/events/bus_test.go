package events_test

import (
	"testing"

	"softwaredesign/src/events"
)

type mockObserver struct {
	received []events.Event
}

func (m *mockObserver) Handle(e events.Event) {
	m.received = append(m.received, e)
}

func TestBusSubscribePublish(t *testing.T) {
	bus := events.NewBus()
	obs := &mockObserver{}
	bus.Subscribe(obs)

	event := events.Event{
		Type:      events.EventCommandExecuted,
		Command:   "test",
		Raw:       "test command",
	}
	bus.Publish(event)

	if len(obs.received) != 1 {
		t.Fatalf("observer should receive one event, got %d", len(obs.received))
	}
	if obs.received[0].Command != "test" {
		t.Fatalf("event command mismatch")
	}
}

func TestBusMultipleObservers(t *testing.T) {
	bus := events.NewBus()
	obs1 := &mockObserver{}
	obs2 := &mockObserver{}
	bus.Subscribe(obs1)
	bus.Subscribe(obs2)

	event := events.Event{
		Type:    events.EventCommandExecuted,
		Command: "test",
	}
	bus.Publish(event)

	if len(obs1.received) != 1 || len(obs2.received) != 1 {
		t.Fatalf("both observers should receive event")
	}
}

