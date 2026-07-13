package cursor

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEncodeDecode(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	id := uuid.New()

	token := Encode(now, id)
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	c, err := Decode(token)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if !c.Timestamp.Equal(now) {
		t.Errorf("timestamp mismatch: got %v, want %v", c.Timestamp, now)
	}
	if c.ID != id {
		t.Errorf("id mismatch: got %v, want %v", c.ID, id)
	}
}

func TestDecodeInvalid(t *testing.T) {
	_, err := Decode("invalid-base64!")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}

	_, err = Decode("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}
