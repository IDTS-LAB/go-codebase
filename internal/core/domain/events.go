package domain

type DomainEvent interface { //nolint:revive
	EventType() string
	OccurredAt() interface{}
}

type BaseEvent struct {
	Type      string
	Timestamp interface{}
}

func (e BaseEvent) EventType() string {
	return e.Type
}

func (e BaseEvent) OccurredAt() interface{} {
	return e.Timestamp
}
