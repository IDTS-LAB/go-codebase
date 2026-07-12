package cqrs

import (
	"context"
	"errors"
	"testing"
)

type testCommand struct {
	Value string
}

type testCommandHandler struct{}

func (h *testCommandHandler) Handle(ctx context.Context, cmd any) (any, error) {
	c := cmd.(testCommand)
	if c.Value == "error" {
		return nil, errors.New("test error")
	}
	return "handled:" + c.Value, nil
}

type testQuery struct {
	ID string
}

type testQueryHandler struct{}

func (h *testQueryHandler) Handle(ctx context.Context, query any) (any, error) {
	q := query.(testQuery)
	return "result:" + q.ID, nil
}

func TestCommandBus_Dispatch(t *testing.T) {
	bus := NewInMemoryCommandBus()
	bus.Register(testCommand{}, &testCommandHandler{})

	resp, err := bus.Dispatch(context.Background(), testCommand{Value: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.(string) != "handled:hello" {
		t.Fatalf("expected 'handled:hello', got '%s'", resp)
	}
}

func TestCommandBus_Unregistered(t *testing.T) {
	bus := NewInMemoryCommandBus()
	_, err := bus.Dispatch(context.Background(), testCommand{Value: "x"})
	if err == nil {
		t.Fatal("expected error for unregistered command")
	}
}

func TestCommandBus_HandlerError(t *testing.T) {
	bus := NewInMemoryCommandBus()
	bus.Register(testCommand{}, &testCommandHandler{})

	_, err := bus.Dispatch(context.Background(), testCommand{Value: "error"})
	if err == nil || err.Error() != "test error" {
		t.Fatalf("expected 'test error', got '%v'", err)
	}
}

func TestQueryBus_Ask(t *testing.T) {
	bus := NewInMemoryQueryBus()
	bus.Register(testQuery{}, &testQueryHandler{})

	resp, err := bus.Ask(context.Background(), testQuery{ID: "123"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.(string) != "result:123" {
		t.Fatalf("expected 'result:123', got '%s'", resp)
	}
}

func TestQueryBus_Unregistered(t *testing.T) {
	bus := NewInMemoryQueryBus()
	_, err := bus.Ask(context.Background(), testQuery{ID: "x"})
	if err == nil {
		t.Fatal("expected error for unregistered query")
	}
}
