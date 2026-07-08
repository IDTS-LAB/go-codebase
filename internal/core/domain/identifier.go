package domain

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
)

type Identifier struct {
	value uuid.UUID
}

func NewIdentifier() Identifier {
	return Identifier{value: uuid.New()}
}

func IdentifierFromUUID(id uuid.UUID) Identifier {
	return Identifier{value: id}
}

func IdentifierFromString(s string) (Identifier, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return Identifier{}, fmt.Errorf("parse identifier: %w", err)
	}
	return Identifier{value: id}, nil
}

func (i Identifier) UUID() uuid.UUID {
	return i.value
}

func (i Identifier) String() string {
	return i.value.String()
}

func (i Identifier) IsZero() bool {
	return i.value == uuid.Nil
}

func (i Identifier) Equals(other Identifier) bool {
	return i.value == other.value
}

func (i Identifier) MarshalText() ([]byte, error) {
	return i.value.MarshalText()
}

func (i Identifier) UnmarshalText(data []byte) error {
	return i.value.UnmarshalText(data)
}

func (i Identifier) Value() (driver.Value, error) {
	return i.value.Value()
}

func (i Identifier) Scan(src interface{}) error {
	return i.value.Scan(src)
}
