package domain

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type Timestamp struct {
	value time.Time
}

func NewTimestamp() Timestamp {
	return Timestamp{value: time.Now().UTC()}
}

func TimestampFrom(t time.Time) Timestamp {
	return Timestamp{value: t.UTC()}
}

func (t Timestamp) Time() time.Time {
	return t.value
}

func (t Timestamp) String() string {
	return t.value.Format(time.RFC3339)
}

func (t Timestamp) IsZero() bool {
	return t.value.IsZero()
}

func (t Timestamp) Equals(other Timestamp) bool {
	return t.value.Equal(other.value)
}

func (t Timestamp) Before(other Timestamp) bool {
	return t.value.Before(other.value)
}

func (t Timestamp) After(other Timestamp) bool {
	return t.value.After(other.value)
}

func (t Timestamp) Add(d time.Duration) Timestamp {
	return Timestamp{value: t.value.Add(d)}
}

func (t Timestamp) MarshalText() ([]byte, error) {
	return []byte(t.value.Format(time.RFC3339)), nil
}

func (t *Timestamp) UnmarshalText(data []byte) error {
	parsed, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return fmt.Errorf("parse timestamp: %w", err)
	}
	t.value = parsed
	return nil
}

func (t Timestamp) Value() (driver.Value, error) {
	return t.value, nil
}

func (t *Timestamp) Scan(src interface{}) error {
	switch v := src.(type) {
	case time.Time:
		t.value = v
		return nil
	case []byte:
		parsed, err := time.Parse(time.RFC3339, string(v))
		if err != nil {
			return fmt.Errorf("scan timestamp: %w", err)
		}
		t.value = parsed
		return nil
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return fmt.Errorf("scan timestamp: %w", err)
		}
		t.value = parsed
		return nil
	default:
		return fmt.Errorf("scan timestamp: unsupported type %T", src)
	}
}
