# NATS Grafana Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a Grafana dashboard tracking NATS server metrics, per-subject message throughput, and recent message payloads via a debug endpoint.

**Architecture:** Instrument the NATSMessenger with Prometheus counters (per-subject), add an in-memory ring buffer for recent messages exposed as `/debug/nats`, and provision a Grafana dashboard with three rows: server health, per-subject activity, and a payload viewer via Infinity datasource.

**Tech Stack:** Go 1.25, NATS v2, Prometheus, Grafana 10.4, Infinity plugin, Chi router, Uber Fx

## Global Constraints

- Follow existing code patterns (promauto for metrics, Chi for routing, Uber Fx for DI)
- Debug endpoint disabled by default (`NATS_DEBUG_ENDPOINT=false`)
- Metrics use `prometheus/client_golang` (already in go.mod)
- Dashboard JSON uses Grafana schema version 39 (matching existing dashboards)
- Infinity datasource plugin: `yesoreyeram-infinity-datasource`

---

### Task 1: Config — Add Debug Endpoint Toggle

**Files:**
- Modify: `internal/shared/config/config.go:72-74`
- Modify: `configs/config.yaml:43-44`
- Modify: `.env:39`

**Interfaces:**
- Consumes: existing `NATSConfig` struct
- Produces: `cfg.NATS.DebugEndpoint bool` available for use in route registration

- [ ] **Step 1: Add `DebugEndpoint` field to `NATSConfig`**

In `internal/shared/config/config.go`, change:
```go
type NATSConfig struct {
	URL string `koanf:"url"`
}
```
to:
```go
type NATSConfig struct {
	URL           string `koanf:"url"`
	DebugEndpoint bool   `koanf:"debug_endpoint"`
}
```

- [ ] **Step 2: Add env override for debug endpoint**

In `internal/shared/config/config.go`, after the NATS env block (after line 391), add:
```go
if v := os.Getenv("NATS_DEBUG_ENDPOINT"); v != "" {
	cfg.NATS.DebugEndpoint = v == "true" || v == "1"
}
```

- [ ] **Step 3: Add config YAML**

In `configs/config.yaml`, change:
```yaml
nats:
  url: nats://localhost:4222
```
to:
```yaml
nats:
  url: nats://localhost:4222
  debug_endpoint: false
```

- [ ] **Step 4: Add .env entry**

In `.env`, after `NATS_URL`, add:
```
NATS_DEBUG_ENDPOINT=false
```

- [ ] **Step 5: Run check and commit**

```bash
go build ./...
git add internal/shared/config/config.go configs/config.yaml .env
git commit -m "feat: add NATS debug endpoint config toggle"
```

---

### Task 2: Instrument NATSMessenger with Prometheus Counters

**Files:**
- Modify: `internal/infrastructure/messaging/nats.go`

**Interfaces:**
- Consumes: `domain.Messenger` interface (unchanged)
- Produces: Prometheus metrics `nats_published_total`, `nats_received_total`, `nats_publish_bytes_total`, `nats_received_bytes_total` available at `/metrics`

- [ ] **Step 1: Add Prometheus imports and metric vars**

Replace the content of `internal/infrastructure/messaging/nats.go` with:
```go
package messaging

import (
	"context"
	"fmt"

	"github.com/IDTS-LAB/go-codebase/internal/core/domain"
	"github.com/IDTS-LAB/go-codebase/internal/shared/config"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/fx"
)

var Module = fx.Module("nats",
	fx.Provide(
		NewNATSMessenger,
		fx.Annotate(
			func(m *NATSMessenger) domain.Messenger { return m },
			fx.As(new(domain.Messenger)),
		),
	),
)

var (
	natsPublishedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nats_published_total",
		Help: "Total number of NATS messages published",
	}, []string{"subject"})

	natsReceivedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nats_received_total",
		Help: "Total number of NATS messages received",
	}, []string{"subject"})

	natsPublishBytesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nats_publish_bytes_total",
		Help: "Total bytes published to NATS",
	}, []string{"subject"})

	natsReceivedBytesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nats_received_bytes_total",
		Help: "Total bytes received from NATS",
	}, []string{"subject"})
)

type NATSMessenger struct {
	conn *nats.Conn
}

func NewNATSMessenger(cfg *config.Config) (*NATSMessenger, error) {
	if cfg.NATS.URL == "" {
		return &NATSMessenger{}, nil
	}

	conn, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}

	return &NATSMessenger{conn: conn}, nil
}

func (n *NATSMessenger) Publish(ctx context.Context, subject string, data []byte) error {
	if n.conn == nil {
		return nil
	}
	natsPublishedTotal.WithLabelValues(subject).Inc()
	natsPublishBytesTotal.WithLabelValues(subject).Add(float64(len(data)))
	return n.conn.Publish(subject, data)
}

func (n *NATSMessenger) Subscribe(ctx context.Context, subject string, handler func(data []byte)) error {
	if n.conn == nil {
		return nil
	}
	_, err := n.conn.Subscribe(subject, func(msg *nats.Msg) {
		natsReceivedTotal.WithLabelValues(subject).Inc()
		natsReceivedBytesTotal.WithLabelValues(subject).Add(float64(len(msg.Data)))
		handler(msg.Data)
	})
	return err
}

func (n *NATSMessenger) Close() error {
	if n.conn != nil {
		return n.conn.Drain()
	}
	return nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/messaging/nats.go
git commit -m "feat: instrument NATSMessenger with Prometheus per-subject counters"
```

---

### Task 3: Create NATS Debug Endpoint

**Files:**
- Create: `internal/infrastructure/messaging/debug.go`
- Create: `internal/infrastructure/messaging/debug_test.go`

**Interfaces:**
- Consumes: `NATSMessenger` connection state, `cfg.NATS.DebugEndpoint` toggle
- Produces: `GET /debug/nats` HTTP handler returning JSON array of recent messages

- [ ] **Step 1: Write the failing test**

Create `internal/infrastructure/messaging/debug_test.go`:
```go
package messaging

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDebugBuffer_AppendAndRead(t *testing.T) {
	buf := newDebugBuffer(3)
	buf.append("foo", []byte("hello"))
	buf.append("bar", []byte("world"))
	entries := buf.read()

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Subject != "bar" {
		t.Fatalf("expected newest first: bar, got %s", entries[0].Subject)
	}
	if entries[1].Subject != "foo" {
		t.Fatalf("expected second: foo, got %s", entries[1].Subject)
	}
}

func TestDebugBuffer_Capacity(t *testing.T) {
	buf := newDebugBuffer(2)
	buf.append("a", []byte("1"))
	buf.append("b", []byte("2"))
	buf.append("c", []byte("3"))

	entries := buf.read()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Subject != "c" {
		t.Fatalf("expected newest: c, got %s", entries[0].Subject)
	}
}

func TestDebugHandler_Enabled(t *testing.T) {
	handler := &debugNATSHandler{buffer: newDebugBuffer(10)}
	handler.buffer.append("test.subj", []byte(`{"msg":"hello"}`))

	req := httptest.NewRequest("GET", "/debug/nats", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp []debugEntry
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp))
	}
	if resp[0].Subject != "test.subj" {
		t.Fatalf("expected test.subj, got %s", resp[0].Subject)
	}
}

func TestDebugHandler_NotEnabled(t *testing.T) {
	handler := &debugNATSHandler{buffer: nil}

	req := httptest.NewRequest("GET", "/debug/nats", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/infrastructure/messaging/ -run "TestDebug" -v
```
Expected: FAIL (types/functions not defined)

- [ ] **Step 3: Write minimal implementation**

Create `internal/infrastructure/messaging/debug.go`:
```go
package messaging

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type debugEntry struct {
	Subject   string `json:"subject"`
	Timestamp int64  `json:"timestamp"`
	Payload   string `json:"payload"`
}

type debugBuffer struct {
	mu   sync.RWMutex
	buf  []debugEntry
	cap  int
	next int
	full bool
}

func newDebugBuffer(capacity int) *debugBuffer {
	return &debugBuffer{
		buf: make([]debugEntry, capacity),
		cap: capacity,
	}
}

func (b *debugBuffer) append(subject string, data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	payload := string(data)
	if len(payload) > 1024 {
		payload = payload[:1024]
	}

	b.buf[b.next] = debugEntry{
		Subject:   subject,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}
	b.next = (b.next + 1) % b.cap
	if b.next == 0 {
		b.full = true
	}
}

func (b *debugBuffer) read() []debugEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var start, count int
	if b.full {
		count = b.cap
		start = b.next
	} else {
		count = b.next
		start = 0
	}

	entries := make([]debugEntry, count)
	for i := 0; i < count; i++ {
		idx := (start + i) % b.cap
		entries[i] = b.buf[idx]
	}

	// Reverse so newest is first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries
}

type debugNATSHandler struct {
	buffer *debugBuffer
}

func (h *debugNATSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.buffer == nil {
		http.NotFound(w, r)
		return
	}
	entries := h.buffer.read()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/infrastructure/messaging/ -run "TestDebug" -v
```
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/messaging/debug.go internal/infrastructure/messaging/debug_test.go
git commit -m "feat: add NATS debug endpoint with in-memory ring buffer"
```

---

### Task 4: Wire Debug Endpoint Into Router

**Files:**
- Modify: `internal/infrastructure/messaging/nats.go`
- Modify: `internal/shared/router/router.go`
- Modify: `cmd/api/main.go`

**Interfaces:**
- Consumes: `cfg.NATS.DebugEndpoint` (from Task 1), `NATSMessenger` (records messages into buffer)
- Produces: `/debug/nats` route registered on the API router

- [ ] **Step 1: Integrate debug buffer into NATSMessenger**

In `internal/infrastructure/messaging/nats.go`, modify `NATSMessenger` to hold a debug buffer. Change the struct and constructor:

```go
type NATSMessenger struct {
	conn      *nats.Conn
	debugBuf  *debugBuffer
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
	return m, nil
}
```

Add to `Publish()`, at the start:
```go
if n.debugBuf != nil {
	n.debugBuf.append(subject, data)
}
```

Add a method to expose the handler:
```go
func (n *NATSMessenger) DebugHandler() http.Handler {
	return &debugNATSHandler{buffer: n.debugBuf}
}
```

Add `"net/http"` to imports.

- [ ] **Step 2: Add DebugNATSHandler to router Handlers struct**

In `internal/shared/router/router.go`, add to the `Handlers` struct:
```go
type Handlers struct {
	Auth             *chi.Mux
	Todo             *chi.Mux
	Authz            *chi.Mux
	User             *chi.Mux
	Tenant           *chi.Mux
	MetricsHandler   http.Handler
	DebugNATSHandler http.Handler
}
```

Add registration after the metrics handler block (after line 45):
```go
if h.DebugNATSHandler != nil {
	r.Get("/debug/nats", h.DebugNATSHandler.ServeHTTP)
}
```

- [ ] **Step 3: Populate and pass the handler in main.go**

In `cmd/api/main.go`, add to the `var` block:
```go
var (
	...
	natsMessenger   *messaging.NATSMessenger
	debugNATSHandler http.Handler
)
```

Add to `fx.Populate`:
```go
fx.Populate(&natsMessenger),
```

After `fx.Populate` block, add:
```go
if natsMessenger != nil {
	debugNATSHandler = natsMessenger.DebugHandler()
}
```

Add `debugNATSHandler` to `router.Handlers`:
```go
root := router.NewRouter(router.Handlers{
	...
	DebugNATSHandler: debugNATSHandler,
}, mw, log, cfg, db)
```

- [ ] **Step 4: Build and verify**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/messaging/nats.go internal/shared/router/router.go cmd/api/main.go
git commit -m "feat: wire NATS debug endpoint into API router"
```

---

### Task 5: Create Grafana Dashboard JSON

**Files:**
- Create: `deployments/grafana/dashboards/nats.json`

- [ ] **Step 1: Create the dashboard JSON**

Create `deployments/grafana/dashboards/nats.json`:
```json
{
  "title": "NATS",
  "uid": "nats",
  "schemaVersion": 39,
  "version": 1,
  "timezone": "browser",
  "panels": [
    {
      "title": "Active Connections",
      "type": "stat",
      "gridPos": {"h": 4, "w": 4, "x": 0, "y": 0},
      "targets": [{"expr": "nats_connections", "legendFormat": ""}]
    },
    {
      "title": "Subscriptions",
      "type": "stat",
      "gridPos": {"h": 4, "w": 4, "x": 4, "y": 0},
      "targets": [{"expr": "nats_subscriptions", "legendFormat": ""}]
    },
    {
      "title": "Uptime",
      "type": "stat",
      "gridPos": {"h": 4, "w": 4, "x": 8, "y": 0},
      "targets": [{"expr": "nats_uptime_seconds", "legendFormat": ""}],
      "fieldConfig": {"defaults": {"unit": "s"}}
    },
    {
      "title": "Messages In/Out",
      "type": "graph",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 4},
      "targets": [
        {"expr": "rate(nats_messages_sent_total[5m])", "legendFormat": "sent"},
        {"expr": "rate(nats_messages_received_total[5m])", "legendFormat": "received"}
      ],
      "lines": true,
      "fill": 1
    },
    {
      "title": "Bytes In/Out",
      "type": "graph",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 4},
      "targets": [
        {"expr": "rate(nats_out_bytes_total[5m])", "legendFormat": "out"},
        {"expr": "rate(nats_in_bytes_total[5m])", "legendFormat": "in"}
      ],
      "lines": true,
      "fill": 1
    },
    {
      "title": "Publish Rate by Subject",
      "type": "bargauge",
      "gridPos": {"h": 8, "w": 8, "x": 0, "y": 12},
      "targets": [{"expr": "rate(nats_published_total[5m])", "legendFormat": "{{subject}}"}]
    },
    {
      "title": "Receive Rate by Subject",
      "type": "bargauge",
      "gridPos": {"h": 8, "w": 8, "x": 8, "y": 12},
      "targets": [{"expr": "rate(nats_received_total[5m])", "legendFormat": "{{subject}}"}]
    },
    {
      "title": "Data Volume by Subject",
      "type": "graph",
      "gridPos": {"h": 8, "w": 8, "x": 16, "y": 12},
      "targets": [{"expr": "rate(nats_publish_bytes_total[5m])", "legendFormat": "{{subject}}"}],
      "lines": true,
      "fill": 1
    },
    {
      "title": "Recent Messages",
      "type": "table",
      "gridPos": {"h": 8, "w": 24, "x": 0, "y": 20},
      "targets": [{
        "refId": "A",
        "type": "json",
        "url": "http://api:8080/debug/nats",
        "fields": ["timestamp", "subject", "payload"]
      }],
      "datasource": "Infinity"
    }
  ]
}
```

- [ ] **Step 2: Update dashboard provisioning config**

In `deployments/grafana/dashboards/dashboard.yml`, no change needed — the existing `path: /etc/grafana/provisioning/dashboards` directory already picks up all `.json` files in the directory.

- [ ] **Step 3: Commit**

```bash
git add deployments/grafana/dashboards/nats.json
git commit -m "feat: add NATS Grafana dashboard"
```

---

### Task 6: Add Infinity Datasource + Docker Compose Updates

**Files:**
- Modify: `deployments/grafana/datasources/datasources.yml`
- Modify: `docker-compose.yml`

- [ ] **Step 1: Add Infinity datasource**

In `deployments/grafana/datasources/datasources.yml`, add after the Jaeger block:
```yaml
  - name: Infinity
    type: yesoreyeram-infinity-datasource
    access: proxy
    url: http://api:8080
    isDefault: false
    editable: false
```

- [ ] **Step 2: Add GF_INSTALL_PLUGINS to Grafana service**

In `docker-compose.yml`, under `grafana.environment`, add:
```yaml
      - GF_INSTALL_PLUGINS=yesoreyeram-infinity-datasource
```

- [ ] **Step 3: Add NATS_DEBUG_ENDPOINT to API service**

In `docker-compose.yml`, under `api.environment`, add:
```yaml
      - NATS_DEBUG_ENDPOINT=true
```

- [ ] **Step 4: Commit**

```bash
git add deployments/grafana/datasources/datasources.yml docker-compose.yml
git commit -m "feat: add Infinity datasource and enable NATS debug endpoint"
```
