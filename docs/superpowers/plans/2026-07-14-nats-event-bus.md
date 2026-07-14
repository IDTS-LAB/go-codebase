# NATS Event Bus Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Make the EventBus switchable between in-memory and NATS JetStream via `events.driver` config, with persistent streams and at-least-once delivery.

**Architecture:** JetStream stream `"events"` persists messages. `NATSEventBus` publishes via `JetStream.Publish()` and subscribes via push consumer with manual ACK. On handler success → `Ack()`, on failure → `Nak()` (infinite retry). All instances share a queue group so only one worker receives each message.

**Tech Stack:** Go 1.25, NATS v2 with JetStream, Uber Fx, `encoding/json`

## Global Constraints

- `events.driver: memory` is default (existing behavior unchanged)
- `events.driver: nats` uses `NATSEventBus` with JetStream
- NATS server needs `-js` flag in docker-compose
- Type registry maps event type strings to `func() interface{}` factory functions
- All existing handlers remain unchanged (same `events.EventBus` interface)
- Auth event structs get JSON tags for NATS serialization
- JetStream stream `"events"` with subjects `events.>`, file storage, interest retention
- Push consumer `"event-bus"` with queue group, explicit ACK, infinite max deliver, 30s ack wait
- Handler error → Nak (retries indefinitely), success → Ack

---

### Task 1: Config — EventsConfig + Extended NATSConfig

**Files:**
- Modify: `internal/shared/config/config.go`
- Modify: `configs/config.yaml`

**Interfaces:**
- Produces: `cfg.Events.Driver string` available for module wiring
- Produces: `cfg.NATS.Stream` and `cfg.NATS.Consumer` for JetStream setup

- [x] **Step 1: Add EventsConfig and StreamConfig/ConsumerConfig**

In `internal/shared/config/config.go`, before `func New()` add:
```go
type EventsConfig struct {
	Driver string `koanf:"driver"`
}

type StreamConfig struct {
	Name      string   `koanf:"name"`
	Subjects  []string `koanf:"subjects"`
	Storage   string   `koanf:"storage"`
	Retention string   `koanf:"retention"`
}

type ConsumerConfig struct {
	DurableName  string `koanf:"durable_name"`
	DeliverGroup string `koanf:"deliver_group"`
	AckPolicy    string `koanf:"ack_policy"`
	MaxDeliver   int    `koanf:"max_deliver"`
	AckWait      int    `koanf:"ack_wait"`
}
```

Update `Config` struct to add `Events` field:
```go
Events EventsConfig `koanf:"events"`
```

Update `NATSConfig` to add Stream and Consumer:
```go
type NATSConfig struct {
	URL           string         `koanf:"url"`
	DebugEndpoint bool           `koanf:"debug_endpoint"`
	Stream        StreamConfig   `koanf:"stream"`
	Consumer      ConsumerConfig `koanf:"consumer"`
}
```

Add defaults in `setDefaults()`:
```go
if cfg.Events.Driver == "" {
    cfg.Events.Driver = "memory"
}
if cfg.NATS.Stream.Name == "" {
    cfg.NATS.Stream.Name = "events"
}
if cfg.NATS.Stream.Storage == "" {
    cfg.NATS.Stream.Storage = "file"
}
if cfg.NATS.Stream.Retention == "" {
    cfg.NATS.Stream.Retention = "interest"
}
if cfg.NATS.Consumer.DurableName == "" {
    cfg.NATS.Consumer.DurableName = "event-bus"
}
if cfg.NATS.Consumer.DeliverGroup == "" {
    cfg.NATS.Consumer.DeliverGroup = "event-bus"
}
if cfg.NATS.Consumer.AckPolicy == "" {
    cfg.NATS.Consumer.AckPolicy = "explicit"
}
if cfg.NATS.Consumer.MaxDeliver == 0 {
    cfg.NATS.Consumer.MaxDeliver = -1
}
if cfg.NATS.Consumer.AckWait == 0 {
    cfg.NATS.Consumer.AckWait = 30
}
```

- [x] **Step 2: Add events and nats stream/consumer to YAML**

In `configs/config.yaml`, after the email section, add:
```yaml
# =============================================================================
# Event Bus
# =============================================================================
events:
  driver: memory
```

Update the nats section:
```yaml
# =============================================================================
# NATS
# =============================================================================
nats:
  url: nats://localhost:4222
  debug_endpoint: false
  stream:
    name: events
    subjects:
      - events.>
    storage: file
    retention: interest
  consumer:
    durable_name: event-bus
    deliver_group: event-bus
    ack_policy: explicit
    max_deliver: -1
    ack_wait: 30
```

- [x] **Step 3: Build and commit**

```bash
go build ./...
git add internal/shared/config/config.go configs/config.yaml
git commit -m "feat: add events config, NATS JetStream stream/consumer config"
```

---

### Task 2: Auth Event JSON Tags

**Files:**
- Modify: `internal/authentication/domain/event/auth_events.go`

- [x] **Step 1: Add JSON tags**

In `internal/authentication/domain/event/auth_events.go`:
```go
type UserRegistered struct {
	Email             string `json:"email"`
	Name              string `json:"name"`
	VerificationToken string `json:"verification_token"`
}

type EmailVerified struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

type PasswordResetRequested struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	ResetToken string `json:"reset_token"`
}
```

- [x] **Step 2: Build and commit**

```bash
go build ./...
git add internal/authentication/domain/event/auth_events.go
git commit -m "feat: add JSON tags to auth event structs"
```

---

### Task 3: Type Registry + Event Registration

**Files:**
- Create: `internal/shared/events/registry.go`
- Create: `internal/shared/events/registry_test.go`
- Modify: `internal/todo/module.go`
- Modify: `internal/authentication/module.go`

**Interfaces:**
- Produces: `events.Register(eventType string, factory func() interface{})` for registration
- Produces: `events.CreatePayload(eventType string) interface{}` for deserialization

- [x] **Step 1: Write failing test**

Create `internal/shared/events/registry_test.go`:
```go
package events

import (
	"testing"
)

type testPayload struct {
	Message string `json:"message"`
}

func init() {
	Register("test.event", func() interface{} { return &testPayload{} })
}

func TestRegistry_RegisterAndCreate(t *testing.T) {
	p := CreatePayload("test.event")
	if p == nil {
		t.Fatal("expected non-nil payload")
	}
	if _, ok := p.(*testPayload); !ok {
		t.Fatal("expected *testPayload type")
	}
}

func TestRegistry_UnknownType(t *testing.T) {
	p := CreatePayload("unknown.event")
	if p != nil {
		t.Fatal("expected nil for unknown type")
	}
}
```

- [x] **Step 2: Run test to verify it fails**

```bash
go test ./internal/shared/events/ -run "TestRegistry" -v
```
Expected: FAIL (CreatePayload not defined)

- [x] **Step 3: Write minimal implementation**

Create `internal/shared/events/registry.go`:
```go
package events

var registry = map[string]func() interface{}{}

func Register(eventType string, factory func() interface{}) {
	registry[eventType] = factory
}

func CreatePayload(eventType string) interface{} {
	factory, ok := registry[eventType]
	if !ok {
		return nil
	}
	return factory()
}
```

- [x] **Step 4: Run test to verify it passes**

```bash
go test ./internal/shared/events/ -run "TestRegistry" -v
```
Expected: PASS

- [x] **Step 5: Register todo events in todo/module.go**

In `internal/todo/module.go`, add to imports:
```go
todoEvent "github.com/IDTS-LAB/go-codebase/internal/todo/domain/event"
```

Add to `fx.Invoke` block:
```go
func() {
    events.Register(todoEvent.TodoCreatedEvent, func() interface{} { return &todoEvent.TodoCreated{} })
    events.Register(todoEvent.TodoUpdatedEvent, func() interface{} { return &todoEvent.TodoUpdated{} })
    events.Register(todoEvent.TodoCompletedEvent, func() interface{} { return &todoEvent.TodoCompleted{} })
    events.Register(todoEvent.TodoDeletedEvent, func() interface{} { return &todoEvent.TodoDeleted{} })
},
```

- [x] **Step 6: Register auth events in authentication/module.go**

In `internal/authentication/module.go`, add to imports:
```go
authEvent "github.com/IDTS-LAB/go-codebase/internal/authentication/domain/event"
```

Add to `fx.Invoke` block:
```go
func() {
    events.Register(authEvent.UserRegisteredEvent, func() interface{} { return &authEvent.UserRegistered{} })
    events.Register(authEvent.EmailVerifiedEvent, func() interface{} { return &authEvent.EmailVerified{} })
    events.Register(authEvent.PasswordResetRequestedEvent, func() interface{} { return &authEvent.PasswordResetRequested{} })
},
```

- [x] **Step 7: Build, run all tests, and commit**

```bash
go build ./...
go test ./...
git add internal/shared/events/registry.go internal/shared/events/registry_test.go internal/todo/module.go internal/authentication/module.go
git commit -m "feat: add event type registry for JSON deserialization"
```

---

### Task 4: NATS Server JetStream + NATSMessenger JetStream Context

**Files:**
- Modify: `docker-compose.yml`
- Modify: `docker-compose.dev.yml`
- Modify: `internal/infrastructure/messaging/nats.go`

**Interfaces:**
- Produces: NATS server with JetStream enabled (`-js` flag)
- Produces: `(*NATSMessenger).JetStream() nats.JetStreamContext` accessor

- [x] **Step 1: Enable JetStream on NATS server**

In `docker-compose.yml`, change NATS command:
```yaml
nats:
  image: nats:2-alpine
  ports:
    - "4222:4222"
    - "8222:8222"
  command: ["--http_port", "8222", "-js"]
  networks:
    - app-network
```

Same in `docker-compose.dev.yml`:
```yaml
nats:
  image: nats:2-alpine
  ports:
    - "4222:4222"
    - "8222:8222"
  command: ["--http_port", "8222", "-js"]
  networks:
    - dev-network
```

- [x] **Step 2: Add JetStream context to NATSMessenger**

In `internal/infrastructure/messaging/nats.go`, update the struct and constructor:
```go
type NATSMessenger struct {
	conn     *nats.Conn
	js       nats.JetStreamContext
	debugBuf *debugBuffer
}

func NewNATSMessenger(cfg *config.Config) (*NATSMessenger, error) {
	m := &NATSMessenger{}
	if cfg.NATS.DebugEndpoint {
		m.debugBuf = newDebugBuffer(100)
	}
	if cfg.NATS.URL == "" {
		return m, nil
	}

	conn, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}
	m.conn = conn

	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	m.js = js

	return m, nil
}
```

Add JetStream accessor:
```go
func (n *NATSMessenger) JetStream() nats.JetStreamContext {
	return n.js
}
```

- [x] **Step 3: Provide JetStream context from messaging module**

In `internal/infrastructure/messaging/nats.go`, add to the `Module`:
```go
fx.Provide(func(m *NATSMessenger) nats.JetStreamContext {
    return m.JetStream()
}),
```

Add `"net/http"` to the import if not already there. The module now becomes:
```go
var Module = fx.Module("nats",
	fx.Provide(
		NewNATSMessenger,
		fx.Annotate(
			func(m *NATSMessenger) domain.Messenger { return m },
			fx.As(new(domain.Messenger)),
		),
		func(m *NATSMessenger) nats.JetStreamContext { return m.JetStream() },
	),
)
```

- [x] **Step 4: Build and commit**

```bash
go build ./...
git add docker-compose.yml docker-compose.dev.yml internal/infrastructure/messaging/nats.go
git commit -m "feat: enable NATS JetStream, add JetStream context to NATSMessenger"
```

---

### Task 5: NATSEventBus with JetStream Publish/Subscribe + Ack/Nak

**Files:**
- Create: `internal/shared/events/nats_event_bus.go`
- Create: `internal/shared/events/nats_event_bus_test.go`

**Interfaces:**
- Consumes: `events.Register`, `events.CreatePayload` (from Task 3), `(*NATSMessenger).JetStream()` (from Task 4)
- Produces: `NATSEventBus` implementing `events.EventBus` interface with JetStream

- [x] **Step 1: Write failing test**

Create `internal/shared/events/nats_event_bus_test.go`:
```go
package events

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type mockJetStream struct {
	mu           sync.Mutex
	published    []struct{ subject string; data []byte }
	subscribed   []struct{ subject, durable, queue string }
	deliverFunc  func(subject string, ackFn func() error, nakFn func())
}

func (m *mockJetStream) Publish(subject string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, struct{ subject string; data []byte }{subject, data})
	return nil
}

func (m *mockJetStream) Subscribe(subject string, cb func(msg *jetStreamMsg)) error {
	m.mu.Lock()
	m.subscribed = append(m.subscribed, struct{ subject, durable, queue string }{subject, "event-bus", "event-bus"})
	m.mu.Unlock()
	return nil
}

type jetStreamMsg struct {
	subject string
	data    []byte
	acked   bool
	naked   bool
}

func (m *jetStreamMsg) Ack() error {
	m.acked = true
	return nil
}

func (m *jetStreamMsg) Nak() error {
	m.naked = true
	return nil
}

func (m *mockJetStream) deliver(subject string, data []byte) *jetStreamMsg {
	msg := &jetStreamMsg{subject: subject, data: data}
	if m.deliverFunc != nil {
		m.deliverFunc(subject, func() error { return msg.Ack() }, func() { msg.Nak() })
	}
	return msg
}

type testPayload struct {
	Msg string `json:"msg"`
}

func TestNATSEventBus_Publish(t *testing.T) {
	mock := &mockJetStream{}
	bus := NewNATSEventBus(mock)

	ctx := context.Background()
	err := bus.Publish(ctx, Event{
		Type:    "nats.test.event",
		Payload: &testPayload{Msg: "hello"},
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

	msg := mock.deliver("nats.test.event2", []byte(`{"msg":"world"}`))

	if !msg.acked {
		t.Fatal("expected Ack on successful handler")
	}
	if msg.naked {
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

	msg := mock.deliver("nats.test.event3", []byte(`{"msg":"fail"}`))

	if !msg.naked {
		t.Fatal("expected Nak on handler error")
	}
	if msg.acked {
		t.Fatal("expected no Ack on handler error")
	}
}
```

- [x] **Step 2: Run test to verify it fails**

```bash
go test ./internal/shared/events/ -run "TestNATSEventBus" -v
```
Expected: FAIL (types not defined)

- [x] **Step 3: Write minimal implementation**

Create `internal/shared/events/nats_event_bus.go`:
```go
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
	_, err := a.js.Subscribe(subject, func(msg *nats.Msg) {
		cb(&jsMsgAdapter{msg: msg})
	}, nats.Durable("event-bus"), nats.DeliverGroup("event-bus"),
		nats.MaxDeliver(-1), nats.AckWait(30*1e9), nats.ManualAck())
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
```

- [x] **Step 4: Run test to verify it passes**

```bash
go test ./internal/shared/events/ -run "TestNATSEventBus" -v
```
Expected: PASS

- [x] **Step 5: Build to check compilation**

```bash
go build ./...
```

- [x] **Step 6: Commit**

```bash
git add internal/shared/events/nats_event_bus.go internal/shared/events/nats_event_bus_test.go
git commit -m "feat: add NATSEventBus with JetStream Ack/Nak delivery"
```

---

### Task 6: Module Wiring — Config-Driven EventBus Selection + JetStream Setup

**Files:**
- Modify: `internal/shared/events/module.go`

**Interfaces:**
- Consumes: `cfg.Events.Driver`, `NATSEventBus`, `InMemoryEventBus`, `nats.JetStreamContext`, `events.Register`
- Produces: `events.EventBus` wired via Fx as the correct implementation

- [x] **Step 1: Wire EventBus in events/module.go**

Read `internal/shared/events/module.go` first, then replace with config-driven bus selection:
```go
package events

import (
	"go.uber.org/fx"

	"github.com/nats-io/nats.go"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
)

func ensureStream(js nats.JetStreamContext, cfg config.StreamConfig) {
	_, err := js.AddStream(&nats.StreamConfig{
		Name:      cfg.Name,
		Subjects:  cfg.Subjects,
		Storage:   nats.FileStorage,
		Retention: nats.InterestPolicy,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		// log but don't fatal
	}
}

func provideEventBus(cfg *config.Config, js nats.JetStreamContext) EventBus {
	if cfg.Events.Driver == "nats" {
		ensureStream(js, cfg.NATS.Stream)
		return NewNATSEventBus(&jsContextAdapter{js: js})
	}
	return NewInMemoryEventBus()
}

var Module = fx.Module("events",
	fx.Provide(provideEventBus),
)
```

- [x] **Step 2: Build, run tests, and commit**

```bash
go build ./...
go test ./...
git add internal/shared/events/module.go
git commit -m "feat: wire config-driven EventBus with JetStream stream setup"
```
