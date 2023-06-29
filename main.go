package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/botaas/telegram-bot-connector/bot"
	"github.com/botaas/telegram-bot-connector/broker/redis"
	"github.com/botaas/telegram-bot-connector/converter"
	"github.com/botaas/telegram-bot-connector/event"
	"github.com/botaas/telegram-bot-connector/models"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {

	outbox, exist := os.LookupEnv("OUTBOX")
	if !exist || outbox == "" {
		log.Fatal("outbox not provide")
	}
	log.Printf("outbox %s\n", outbox)

	inbox, exist := os.LookupEnv("INBOX")
	if !exist || inbox == "" {
		log.Fatal("inbox not provide")
	}
	log.Printf("inbox %s\n", inbox)

	addr, exist := os.LookupEnv("REDIS_ADDR")
	if !exist || addr == "" {
		log.Fatal("redis addr not provide")
	}

	redisPassword, exist := os.LookupEnv("REDIS_PASSWORD")

	broker := redis.New(
		redis.WithAddr(addr),
		redis.WithPassword(redisPassword),
	)

	token, exist := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	if !exist || token == "" {
		log.Fatal("token not provide")
	}

	bot, err := bot.New(token, time.Duration(3*int(time.Second)))
	if err != nil {
		log.Fatalf("Couldn't start Telegram bot: %v", err)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		bot.Stop()
		os.Exit(0)
	}()

	ctx := context.Background()
	log.Printf("Started Telegram bot! Message @%s to start.", bot.Self.UserName)

	eventManager := event.New()
	eventManager.RegisterHandler("message", &event.MessageHandler{
		Bot: bot,
	})

	unsubscriber, err := broker.Subscribe(ctx, outbox, func(ev *cloudevents.Event) error {
		log.Printf("outbox event: %v\n", string(ev.Data()))
		return eventManager.Process(ctx, ev)
	})

	if err != nil {
		log.Fatal("Subscribe outbox error")
	}

	defer unsubscriber.Cancel()

	for update := range bot.GetUpdatesChan() {
		b, _ := json.Marshal(update)
		fmt.Printf("inbox: %s\n", string(b))

		if update.PreCheckoutQuery != nil {
			event := cloudevents.NewEvent()
			event.SetType("pre_checkout_query")
			event.SetData(cloudevents.ApplicationJSON, update.PreCheckoutQuery)
			err = broker.Publish(ctx, inbox, &event)
			if err != nil {
				log.Printf("publish to redis error: %v", err)
			}

			pca := tgbotapi.PreCheckoutConfig{
				OK:                 true,
				PreCheckoutQueryID: update.PreCheckoutQuery.ID,
			}

			_, err := bot.API().Request(pca)
			if err != nil {
				log.Printf("send pre_checkout error: %v", err)
			}
		}

		if update.CallbackQuery != nil {
			m, err := converter.NormalizeTelegramMessage(bot.API(), update.CallbackQuery.Message)
			if err != nil {
				log.Printf("Error normalize Telegram Message: %v", string(b))
				continue
			}

			callbackQuery := &models.CallbackQuery{
				ID:              update.CallbackQuery.ID,
				From:            converter.NormalizeTelegramUser(update.CallbackQuery.From),
				Message:         m,
				InlineMessageID: update.CallbackQuery.InlineMessageID,
				ChatInstance:    update.CallbackQuery.ChatInstance,
				Data:            update.CallbackQuery.Data,
				GameShortName:   update.CallbackQuery.GameShortName,
			}

			event := cloudevents.NewEvent()
			event.SetType("callback_query")
			event.SetData(cloudevents.ApplicationJSON, callbackQuery)
			err = broker.Publish(ctx, inbox, &event)
			if err != nil {
				log.Printf("publish to redis error: %v", err)
			}
		}

		if update.Message != nil {
			m, err := converter.NormalizeTelegramMessage(bot.API(), update.Message)
			if err != nil {
				log.Printf("Error normalize Telegram Message: %v", string(b))
				continue
			}

			event := cloudevents.NewEvent()
			event.SetType("message")
			event.SetData(cloudevents.ApplicationJSON, m)
			err = broker.Publish(ctx, inbox, &event)
			if err != nil {
				log.Printf("publish to redis error: %v", err)
			}
		}
	}
}
