package events

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/nats-io/nats.go"
)

type jetStreamMsg interface {
	Ack() error
	Nak() error
	Data() []byte
}

type jetStreamer interface {
	Publish(subject string, data []byte) error
	Subscribe(subject string, cb func(msg jetStreamMsg)) error
}

type jsMsgAdapter struct {
	msg *nats.Msg
}

func (a *jsMsgAdapter) Ack() error   { return a.msg.Ack() }
func (a *jsMsgAdapter) Nak() error   { return a.msg.Nak() }
func (a *jsMsgAdapter) Data() []byte { return a.msg.Data }

type jsContextAdapter struct {
	js nats.JetStreamContext
}

func (a *jsContextAdapter) Publish(subject string, data []byte) error {
	_, err := a.js.Publish(subject, data)
	return err
}

func (a *jsContextAdapter) Subscribe(subject string, cb func(msg jetStreamMsg)) error {
	_, err := a.js.QueueSubscribe(subject, "event-bus", func(msg *nats.Msg) {
		cb(&jsMsgAdapter{msg: msg})
	}, nats.Durable("event-bus"), nats.MaxDeliver(-1), nats.AckWait(30*1e9), nats.ManualAck())
	return err
}

type NATSEventBus struct {
	js       jetStreamer
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewNATSEventBus(js jetStreamer) *NATSEventBus {
	return &NATSEventBus{
		js:       js,
		handlers: make(map[string][]Handler),
	}
}

func (b *NATSEventBus) Publish(ctx context.Context, event Event) error {
	data, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}
	subject := "events." + event.Type
	return b.js.Publish(subject, data)
}

func (b *NATSEventBus) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.handlers[eventType]) == 0 {
		b.startSubscription(eventType)
	}
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *NATSEventBus) startSubscription(eventType string) {
	subject := "events." + eventType
	_ = b.js.Subscribe(subject, func(msg jetStreamMsg) {
		payload := CreatePayload(eventType)
		if payload == nil {
			msg.Ack()
			return
		}
		if err := json.Unmarshal(msg.Data(), payload); err != nil {
			msg.Ack()
			return
		}
		event := Event{Type: eventType, Payload: payload}

		b.mu.RLock()
		handlers := b.handlers[eventType]
		b.mu.RUnlock()

		for _, h := range handlers {
			if err := h(context.Background(), event); err != nil {
				msg.Nak()
				return
			}
		}
		msg.Ack()
	})
}
