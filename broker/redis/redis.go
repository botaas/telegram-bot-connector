package redis

import (
	"context"
	"fmt"
	"log"

	"github.com/botaas/telegram-bot-connector/broker"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	goredis "github.com/redis/go-redis/v9"
)

type unsubscriber struct {
	pubsub *goredis.PubSub
}

func (s *unsubscriber) Cancel() {
	s.pubsub.Close()
}

type redis struct {
	rdb *goredis.Client
}

func New(opts ...Option) broker.Broker {
	o := &redisOptions{}
	for _, opt := range opts {
		opt(o)
	}

	rdb := goredis.NewClient(&goredis.Options{
		Addr:     o.addr,
		Password: o.password,
	})

	return &redis{
		rdb,
	}
}

func (r redis) Publish(ctx context.Context, channel string, event *cloudevents.Event) error {
	b, err := event.MarshalJSON()
	if err != nil {
		return err
	}

	return r.rdb.Publish(ctx, channel, string(b)).Err()
}

func (r *redis) Subscribe(ctx context.Context, channel string, fn broker.Subscriber) (broker.Unsubscriber, error) {
	// subscribe
	pubsub := r.rdb.Subscribe(ctx, channel)
	go func() {
		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				log.Printf("receive msg error: %v\n", err)
			}
			r.processOneMessage(ctx, msg, fn)
		}
	}()

	return &unsubscriber{
		pubsub: pubsub,
	}, nil
}

func (r *redis) processOneMessage(ctx context.Context, msg *goredis.Message, fn broker.Subscriber) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()

	ev := cloudevents.NewEvent()
	err := ev.UnmarshalJSON([]byte(msg.Payload))
	if err != nil {
		log.Printf("Unmarshal event error: %v\n", err)
	}

	err = fn(&ev)
	if err != nil {
		log.Printf("event process error: %v\n", err)
	}
}
