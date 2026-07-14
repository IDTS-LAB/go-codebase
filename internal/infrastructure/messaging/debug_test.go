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
