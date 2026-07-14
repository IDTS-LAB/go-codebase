package events

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type mockJetStream struct {
	mu        sync.Mutex
	published []struct {
		subject string
		data    []byte
	}
	subscribed map[string]func(msg jetStreamMsg)
	lastMsg    *testMsg
}

func (m *mockJetStream) Publish(subject string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, struct {
		subject string
		data    []byte
	}{subject, data})
	return nil
}

func (m *mockJetStream) Subscribe(subject string, cb func(msg jetStreamMsg)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.subscribed == nil {
		m.subscribed = make(map[string]func(msg jetStreamMsg))
	}
	m.subscribed[subject] = cb
	return nil
}

func (m *mockJetStream) deliver(subject string, data []byte) {
	m.mu.Lock()
	cb := m.subscribed[subject]
	m.mu.Unlock()
	if cb != nil {
		msg := &testMsg{subj: subject, dat: data}
		m.lastMsg = msg
		cb(msg)
	}
}

type testMsg struct {
	subj  string
	dat   []byte
	acked bool
	naked bool
}

func (m *testMsg) Ack() error   { m.acked = true; return nil }
func (m *testMsg) Nak() error   { m.naked = true; return nil }
func (m *testMsg) Data() []byte { return m.dat }

func TestNATSEventBus_Publish(t *testing.T) {
	mock := &mockJetStream{}
	bus := NewNATSEventBus(mock)

	ctx := context.Background()
	err := bus.Publish(ctx, Event{
		Type:    "nats.test.event",
		Payload: &testPayload{Message: "hello"},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	mock.mu.Lock()
	published := len(mock.published)
	mock.mu.Unlock()

	if published != 1 {
		t.Fatalf("expected 1 publish, got %d", published)
	}
}

func TestNATSEventBus_SubscribeAcksOnSuccess(t *testing.T) {
	Register("nats.test.event2", func() interface{} { return &testPayload{} })

	mock := &mockJetStream{}
	bus := NewNATSEventBus(mock)

	bus.Subscribe("nats.test.event2", func(_ context.Context, event Event) error {
		return nil
	})

	mock.deliver("events.nats.test.event2", []byte(`{"message":"world"}`))

	if !mock.lastMsg.acked {
		t.Fatal("expected Ack on successful handler")
	}
	if mock.lastMsg.naked {
		t.Fatal("expected no Nak on successful handler")
	}
}

func TestNATSEventBus_SubscribeNaksOnError(t *testing.T) {
	Register("nats.test.event3", func() interface{} { return &testPayload{} })

	mock := &mockJetStream{}
	bus := NewNATSEventBus(mock)

	bus.Subscribe("nats.test.event3", func(_ context.Context, event Event) error {
		return errors.New("handler error")
	})

	mock.deliver("events.nats.test.event3", []byte(`{"message":"fail"}`))

	if !mock.lastMsg.naked {
		t.Fatal("expected Nak on handler error")
	}
	if mock.lastMsg.acked {
		t.Fatal("expected no Ack on handler error")
	}
}
