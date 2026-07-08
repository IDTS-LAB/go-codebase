package domain

import (
	"time"
)

type AggregateRoot struct {
	Entity
	events []DomainEvent
}

func NewAggregateRoot() AggregateRoot {
	return AggregateRoot{
		Entity: NewEntity(),
		events: make([]DomainEvent, 0),
	}
}

func (a *AggregateRoot) RecordEvent(event DomainEvent) {
	a.events = append(a.events, event)
	a.Touch()
}

func (a *AggregateRoot) PullEvents() []DomainEvent {
	events := a.events
	a.events = make([]DomainEvent, 0)
	return events
}

func (a *AggregateRoot) Events() []DomainEvent {
	return a.events
}

func (a *AggregateRoot) HasEvents() bool {
	return len(a.events) > 0
}

func (a *AggregateRoot) ClearEvents() {
	a.events = make([]DomainEvent, 0)
}

func (a *AggregateRoot) TouchWith(t time.Time) {
	a.UpdatedAt = t
}
