package events

import "testing"

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
