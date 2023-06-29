package broker

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type Subscriber func(event *cloudevents.Event) error
type Unsubscriber interface {
	Cancel()
}

type Broker interface {
	Publish(ctx context.Context, channel string, event *cloudevents.Event) error
	Subscribe(ctx context.Context, channel string, fn Subscriber) (Unsubscriber, error)
}
