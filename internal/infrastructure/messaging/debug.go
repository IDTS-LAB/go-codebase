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
	_ = json.NewEncoder(w).Encode(entries)
}
