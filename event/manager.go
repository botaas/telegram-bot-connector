package event

import (
	"context"
	"errors"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type EventContext struct {
	context.Context
}

type EventHandler interface {
	Handle(ctx context.Context, ev *cloudevents.Event) error
}

type EventManager struct {
	handlers map[string]EventHandler
}

func New() *EventManager {
	return &EventManager{
		handlers: map[string]EventHandler{},
	}
}

func (em *EventManager) RegisterHandler(eventType string, handler EventHandler) {
	em.handlers[eventType] = handler
}

func (em *EventManager) Process(ctx context.Context, ev *cloudevents.Event) error {
	handler, exist := em.handlers[ev.Type()]
	if !exist {
		return errors.New(fmt.Sprintf("No EventHandler for type: %s\n", ev.Type()))
	}

	return handler.Handle(ctx, ev)
}
