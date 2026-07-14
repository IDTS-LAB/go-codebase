# NATS Event Bus Design

## Overview

Make the EventBus switchable between the existing in-memory implementation and a new NATS-backed implementation, controlled by a config flag. The NATSEventBus uses the existing NATSMessenger for transport, a type registry for JSON deserialization, and is transparent to all existing handlers.

## Architecture

```
events.driver: "memory" → InMemoryEventBus (existing, unchanged)
events.driver: "nats"   → NATSEventBus
                              │
                              ▼
                         NATSMessenger
                         Publish("events.{type}", jsonBytes)
                              │
                              ▼
                      NATS queue group subscribe
                         → deserialize via type registry
                         → dispatch to local handlers
```

The `events.Module` selects the implementation at startup based on config. All command handlers and event handlers remain unchanged.

## Components

### 1. Type Registry (`internal/shared/events/registry.go`)

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

### 2. NATSEventBus (`internal/shared/events/nats_event_bus.go`)

New file. Implements `events.EventBus`:

- **Publish**: serializes `event.Payload` to JSON, publishes to NATS subject `events.{event.Type}` via `domain.Messenger`
- **Subscribe**: stores handler locally (same pattern as InMemoryEventBus); on first subscribe, starts a NATS subscription for that event type
- **NATS callback**: receives raw bytes, deserializes via type registry, reconstructs `events.Event`, dispatches to all registered local handlers
- Uses NATS queue group (`"event-bus"`) so multiple instances distribute events across the group

### 3. Config Changes

New `EventsConfig` struct:
```go
type EventsConfig struct {
    Driver string `koanf:"driver"` // "memory" or "nats"
}
```

Add to `configs/config.yaml`:
```yaml
events:
  driver: memory
```

Env override: `EVENTS_DRIVER=nats`

### 4. Module Wiring (`internal/shared/events/module.go`)

Modified to switch on config:

```
events.driver == "memory":
  NewInMemoryEventBus → LoggingEventBus → EventBus

events.driver == "nats":
  NewNATSEventBus(messenger, log) → EventBus
  (NATSEventBus already includes its own logging)
```

### 5. Auth Event JSON Tags

`authentication/domain/event/auth_events.go` needs JSON tags added to struct fields (for proper serialization via NATS).

## Config

```yaml
events:
  driver: memory   # "memory" or "nats"
```

## Files Changed

| File | Change |
|------|--------|
| `internal/shared/events/registry.go` | New — type registry |
| `internal/shared/events/nats_event_bus.go` | New — NATS-backed EventBus |
| `internal/shared/events/module.go` | Switch on config to select implementation |
| `internal/shared/config/config.go` | Add `EventsConfig` with `Driver` field |
| `configs/config.yaml` | Add `events.driver: memory` |
| `internal/authentication/domain/event/auth_events.go` | Add JSON tags |
| `internal/todo/module.go` | Register todo events in type registry |
| `internal/authentication/module.go` or `cmd/api/main.go` | Register auth events in type registry |

## Testing

- Unit tests for NATSEventBus Publish/Subscribe with a mock Messenger
- Unit tests for type registry (register + create payload)
- Integration test: publish event via NATSEventBus, verify handler receives deserialized payload
- Config switching test: verify EventBus resolves to correct implementation
