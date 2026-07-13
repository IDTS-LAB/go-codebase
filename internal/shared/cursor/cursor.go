package cursor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Cursor struct {
	Timestamp time.Time `json:"t"`
	ID        uuid.UUID `json:"i"`
}

func Encode(t time.Time, id uuid.UUID) string {
	c := Cursor{Timestamp: t.UTC(), ID: id}
	b, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(b)
}

func Decode(s string) (Cursor, error) {
	if s == "" {
		return Cursor{}, fmt.Errorf("empty cursor")
	}
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, fmt.Errorf("decode cursor: %w", err)
	}
	var c Cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return Cursor{}, fmt.Errorf("unmarshal cursor: %w", err)
	}
	return c, nil
}
