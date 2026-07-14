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
- `QueueSubscribe` — removed in favor of JetStream push consumer

### 2. Type Registry (`internal/shared/events/registry.go`)

New file. Maps event type strings to factory functions for JSON deserialization:

```go
var registry = map[string]func() interface{}{}

func Register(eventType string, factory func() interface{})
func CreatePayload(eventType string) interface{}
```

Event types register themselves in `init()` functions:

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

The stream and consumer are created lazily on first `Subscribe()` if they don't exist.

### 4. NATSEventBus (`internal/shared/events/nats_event_bus.go`)

New file. Implements `events.EventBus` using JetStream:

- **Publish**: serializes `event.Payload` to JSON, publishes via `JetStream.Publish()` on subject `events.{event.Type}` (JetStream persists the message)
- **Subscribe**: stores handler locally; on first subscribe:
  1. Ensures stream `"events"` exists (creates if not)
  2. Ensures push consumer `"event-bus"` exists with queue group `"event-bus"` (creates if not)
  3. Calls `JetStream.Subscribe()` with the consumer config and manual ACK
- **NATS callback**: receives `*nats.Msg`, deserializes via type registry, reconstructs `events.Event`, dispatches to all registered local handlers
  - If all handlers succeed → `msg.Ack()` (marks processed, not redelivered)
  - If any handler fails → `msg.Nak()` (redelivers to this or another worker, retries indefinitely)
- Because all instances share queue group `"event-bus"`, NATS delivers each message to exactly one worker

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

Modified to switch on config:

```
events.driver == "memory":
  NewInMemoryEventBus → LoggingEventBus → EventBus

events.driver == "nats":
  NewNATSEventBus(messenger, log) → EventBus
  (NATSEventBus already includes its own logging)
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
| `internal/infrastructure/messaging/nats.go` | Add JetStream context, expose via `JetStream()` method |
| `internal/shared/events/registry.go` | New — type registry |
| `internal/shared/events/nats_event_bus.go` | New — NATS-backed EventBus using JetStream Publish/Subscribe + Ack/Nak |
| `internal/shared/events/module.go` | Switch on config to select implementation |
| `internal/shared/config/config.go` | Add `EventsConfig`, extend `NATSConfig` with `Stream`/`Consumer` |
| `configs/config.yaml` | Add `events.driver`, `nats.stream`, `nats.consumer` |
| `internal/authentication/domain/event/auth_events.go` | Add JSON tags |
| `internal/todo/module.go` | Register todo events in type registry |
| `internal/authentication/module.go` | Register auth events in type registry |

## Testing

- Unit tests for NATSEventBus Publish/Subscribe with a mock JetStream context
- Unit tests for type registry (register + create payload)
- Integration test: start NATS with JetStream, publish event, verify handler receives deserialized payload
- Config switching test: verify EventBus resolves to correct implementation
- Test Nak on handler error: publish event, handler returns error, verify message is redelivered
