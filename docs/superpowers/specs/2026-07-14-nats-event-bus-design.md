# NATS Event Bus Design

## Overview

Make the EventBus switchable between the existing in-memory implementation and a new NATS-backed implementation, controlled by a config flag. The NATSEventBus uses the existing NATSMessenger for transport, a type registry for JSON deserialization, and is transparent to all existing handlers.

## Architecture

```
events.driver: "memory" → InMemoryEventBus (existing, unchanged)
events.driver: "nats"   → NATSEventBus
                              │
                              ├── Publish → JetStream.Publish("events.{type}", msg)
                              │
                              └── Subscribe → JetStream consumer "event-bus.{type}"
                                   → push consumer, manual ACK
                                   → only one worker receives each message
                                   → deserialize via type registry
                                   → dispatch to local handlers
                                   → on success: Ack()
                                   → on error: Nak() (retries indefinitely)
```

The `events.Module` selects the implementation at startup based on config. All command handlers and event handlers remain unchanged.

## Delivery Guarantees

When multiple API worker instances are running:

| Property | Core NATS (memory driver) | JetStream (nats driver) |
|----------|--------------------------|------------------------|
| Dispatch | In-process, synchronous | Distributed via NATS |
| Cross-instance | N/A | Queue consumer — one worker per message |
| Persistence | None | File-backed stream |
| Delivery | At-most-once | At-least-once |
| Failure | Error returned to publisher | Nak() → retry indefinitely |
| Crash mid-handler | Message lost | Message redelivered to another worker |

## Components

### 1. NATSMessenger JetStream Support (`internal/infrastructure/messaging/nats.go`)

The `NATSMessenger` gains JetStream context access:

```go
type NATSMessenger struct {
	conn     *nats.Conn
	js       nats.JetStreamContext
	debugBuf *debugBuffer
}
```

On connect, `NewNATSMessenger` creates a JetStream context:
```go
if conn != nil {
    js, err := conn.JetStream()
    if err != nil {
        return nil, fmt.Errorf("jetstream: %w", err)
    }
    m.js = js
}
```

New methods:
- `JetStream() nats.JetStreamContext` — exposes the JetStream context for use by `NATSEventBus`
- `JetStream() nats.JetStreamContext` — exposes the JetStream context for use by `NATSEventBus`

### 2. Type Registry (`internal/shared/events/registry.go`)

New file. Maps event type strings to factory functions for JSON deserialization:

```go
var registry = map[string]func() interface{}{}

func Register(eventType string, factory func() interface{})
func CreatePayload(eventType string) interface{}
```

Event types register themselves via `fx.Invoke` blocks in each module's Fx setup:

| Event Type | Struct | Package |
|-----------|--------|---------|
| `auth.user.registered` | `event.UserRegistered` | `authentication/domain/event` |
| `auth.user.email_verified` | `event.EmailVerified` | `authentication/domain/event` |
| `auth.user.password_reset_requested` | `event.PasswordResetRequested` | `authentication/domain/event` |
| `todo.created` | `event.TodoCreated` | `todo/domain/event` |
| `todo.updated` | `event.TodoUpdated` | `todo/domain/event` |
| `todo.completed` | `event.TodoCompleted` | `todo/domain/event` |
| `todo.deleted` | `event.TodoDeleted` | `todo/domain/event` |

### 3. JetStream Integration

**Stream** — auto-created at startup (or configured externally):
```yaml
nats:
  stream:
    name: events
    subjects: ["events.>"]
    storage: file
    retention: interest    # keeps messages until all consumers ACK
```

**Consumer** — push-based, one per service instance:
```yaml
nats:
  consumer:
    durable_name: event-bus
    deliver_group: event-bus    # queue group across instances
    ack_policy: explicit
    max_deliver: -1             # infinite retry on Nak
    ack_wait: 30s
```

The stream is created eagerly at startup via `ensureStream()` called from `provideEventBus()`. The push consumer is created implicitly by `QueueSubscribe` with a `Durable` name on first subscription to each subject.

### 4. NATSEventBus (`internal/shared/events/nats_event_bus.go`)

New file. Implements `events.EventBus` using JetStream via a thin `jetStreamer` abstraction:

```go
type jetStreamMsg interface {
    Ack() error
    Nak() error
    Data() []byte
}

type jetStreamer interface {
    Publish(subject string, data []byte) error
    Subscribe(subject string, cb func(msg jetStreamMsg)) error
}
```

Two adapters bridge real NATS types to these interfaces:
- `jsMsgAdapter` wraps `*nats.Msg` → implements `jetStreamMsg`
- `jsContextAdapter` wraps `nats.JetStreamContext` → implements `jetStreamer` (calls `QueueSubscribe` with queue group `"event-bus"`, `Durable("event-bus")`, `MaxDeliver(-1)`, `AckWait(30s)`, `ManualAck()`)

**Publish**: serializes `event.Payload` to JSON, publishes via `JetStream.Publish()` on subject `events.{event.Type}`

**Subscribe**: stores handler locally; on first subscribe to an event type, calls `js.Subscribe()` which registers the JetStream push consumer. The NATS callback:
  1. Creates payload via `CreatePayload(eventType)` from the type registry
  2. Unmarshals JSON into the payload struct
  3. Dispatches to all local handlers for that event type
  4. If all handlers succeed → `Ack()`
  5. If any handler fails → `Nak()` (redelivers, retries indefinitely)

Because all instances share queue group `"event-bus"`, NATS delivers each message to exactly one worker.

### 5. Config Changes

New `EventsConfig` struct:
```go
type EventsConfig struct {
    Driver string `koanf:"driver"` // "memory" or "nats"
}
```

Extended `NATSConfig` with JetStream config:
```go
type NATSConfig struct {
    URL           string         `koanf:"url"`
    DebugEndpoint bool           `koanf:"debug_endpoint"`
    Stream        StreamConfig   `koanf:"stream"`
    Consumer      ConsumerConfig `koanf:"consumer"`
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

Add to `configs/config.yaml`:
```yaml
events:
  driver: memory
```

Env overrides: `EVENTS_DRIVER=nats`

### 6. Module Wiring (`internal/shared/events/module.go`)

Modified to switch on config via a single `provideEventBus` function:

```go
func provideEventBus(cfg *config.Config, js nats.JetStreamContext, log domain.Logger) EventBus {
    var bus EventBus
    if cfg.Events.Driver == "nats" {
        ensureStream(js, cfg.NATS.Stream)
        bus = NewNATSEventBus(&jsContextAdapter{js: js})
    } else {
        bus = NewInMemoryEventBus()
    }
    return NewLoggingEventBus(bus, log)
}
```

`NewLoggingEventBus` accepts `EventBus` interface (not `*InMemoryEventBus`), so it wraps either implementation transparently.

`ensureStream` creates the JetStream stream on startup (idempotent — no-op if already exists):

```go
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
```

### 7. Auth Event JSON Tags

`authentication/domain/event/auth_events.go` needs JSON tags added to struct fields (for proper serialization via NATS).

## Config

```yaml
events:
  driver: memory   # "memory" or "nats"
```

## NATS Server

Enable JetStream in `docker-compose.yml`:

```yaml
nats:
  image: nats:2-alpine
  command: ["--http_port", "8222", "-js"]
```

The `-js` flag enables JetStream on the NATS server (no additional config needed for dev).

## Files Changed

| File | Change |
|------|--------|
| `docker-compose.yml` | Add `-js` flag to NATS server command |
| `docker-compose.dev.yml` | Add `-js` flag to NATS server command |
| `internal/infrastructure/messaging/nats.go` | Add JetStream context, expose via `JetStream()` method, provide from module |
| `internal/shared/events/registry.go` | New — type registry |
| `internal/shared/events/registry_test.go` | New — type registry tests |
| `internal/shared/events/nats_event_bus.go` | New — NATS-backed EventBus using JetStream + Ack/Nak with `jetStreamer` abstraction |
| `internal/shared/events/nats_event_bus_test.go` | New — NATSEventBus unit tests with mock JetStream |
| `internal/shared/events/logging_event_bus.go` | Change `NewLoggingEventBus` to accept `EventBus` interface |
| `internal/shared/events/module.go` | Config-driven `provideEventBus` with `ensureStream` + LoggingEventBus wrapper |
| `internal/shared/config/config.go` | Add `EventsConfig`, extend `NATSConfig` with `Stream`/`Consumer` |
| `configs/config.yaml` | Add `events.driver`, `nats.stream`, `nats.consumer` |
| `internal/authentication/domain/event/auth_events.go` | Add JSON tags |
| `internal/todo/module.go` | Register todo events in type registry (via `fx.Invoke`) |
| `internal/authentication/module.go` | Register auth events in type registry (via `fx.Invoke`) |

## Testing

- ✅ Unit tests for NATSEventBus Publish/Subscribe with mock `jetStreamer` (3 tests: publish, ack on success, nak on error)
- ✅ Unit tests for type registry (2 tests: register+create, unknown type returns nil)
- Integration test: start NATS with JetStream, publish event, verify handler receives deserialized payload (TBD)
- Config switching test: verify EventBus resolves to correct implementation (covered by module wiring — compile-time check)
