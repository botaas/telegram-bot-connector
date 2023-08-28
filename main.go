package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/time/rate"

	"github.com/botaas/telegram-bot-connector/bot"
	"github.com/botaas/telegram-bot-connector/broker/redis"
	"github.com/botaas/telegram-bot-connector/converter"
	"github.com/botaas/telegram-bot-connector/event"
	"github.com/botaas/telegram-bot-connector/models"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)

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

	var err error
	var concurrency int
	concurrencyStr, exist := os.LookupEnv("CONCURRENCY")
	if !exist {
		concurrency = 8
	} else {
		concurrency, err = strconv.Atoi(concurrencyStr)
		if err != nil {
			concurrency = 8
		}
	}

	var ratelimit int
	ratelimitStr, exist := os.LookupEnv("RATELIMIT")
	if !exist {
		ratelimit = 1000
	} else {
		ratelimit, err = strconv.Atoi(ratelimitStr)
		if err != nil {
			ratelimit = 1000
		}
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
	log.Infof("Started Telegram bot! Bot username: @%s.", bot.Self.UserName)

	eventManager := event.New()
	eventManager.RegisterHandler("message", &event.MessageHandler{
		Bot: bot,
	})
	eventManager.RegisterHandler("chat_action", &event.ChatActionHandler{
		Bot: bot,
	})

	var outboxChans = make([]chan *cloudevents.Event, concurrency)
	for i := 0; i < concurrency; i++ {
		outboxChans[i] = make(chan *cloudevents.Event, 1)
	}

	for i := 0; i < concurrency; i++ {
		ch := outboxChans[i]
		go func(index int) {
			for ev := range ch {
				err := eventManager.Process(ctx, ev)
				if err != nil {
					log.WithFields(
						log.Fields{
							"chain": index,
							"data":  string(ev.Data()),
						},
					).Error("Process outbox message error", err)
				}
			}
		}(i)
	}

	limiter := rate.NewLimiter(rate.Every(time.Minute), ratelimit)

	unsubscriber, err := broker.Subscribe(ctx, outbox, func(ev *cloudevents.Event) error {
		log.Printf("outbox event: %v\n", string(ev.Data()))

		err := limiter.Wait(ctx)
		if err != nil {
			return err
			// return errors.New(fmt.Sprintf("发送消息被限制: %v", err))
		}

		i := rand.Intn(concurrency)
		outboxChans[i] <- ev

		return nil
	})

	if err != nil {
		log.Fatal("Subscribe outbox error")
	}

	defer unsubscriber.Cancel()

	/* 切换为 webhook
	webHook := "https://api.telegram.org/bot%s/setWebhook?url=https://cargo-telegram-bot.herokuapp.com/%s"
	webhookConfig := tgbotapi.NewWebhook(fmt.Sprintf(webHook, core.Config.BotToken, core.Config.BotToken))
	_, _ = bot.SetWebhook(webhookConfig)
	updates = bot.ListenForWebhook("/" + bot.Token)
	*/

	var chans = make([]chan *tgbotapi.Update, concurrency)
	for i := 0; i < concurrency; i++ {
		chans[i] = make(chan *tgbotapi.Update, 1)
	}

	go func() {
		process_update := func(update *tgbotapi.Update) error {
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
					return errors.New(fmt.Sprintf("Error normalize Telegram Message: %v", err))
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
					return errors.New(fmt.Sprintf("publish to redis error: %v", err))
				}
			}

			if update.Message != nil {
				m, err := converter.NormalizeTelegramMessage(bot.API(), update.Message)
				if err != nil {
					return errors.New(fmt.Sprintf("Error normalize Telegram Message: %v", err))
				}

				event := cloudevents.NewEvent()
				event.SetType("message")
				event.SetData(cloudevents.ApplicationJSON, m)
				err = broker.Publish(ctx, inbox, &event)
				if err != nil {
					return errors.New(fmt.Sprintf("publish to redis error: %v", err))
				}
			}

			return nil
		}

		for i := 0; i < concurrency; i++ {
			ch := chans[i]
			go func(index int) {
				for update := range ch {
					b, _ := json.Marshal(update)
					log.Debugf("inbox, chan: %v: %v\n", index, string(b))

					err := process_update(update)
					if err != nil {
						log.WithFields(
							log.Fields{
								"chain":  index,
								"update": string(b),
							},
						).Error("Process update error", err)
					}
				}
			}(i)
		}
	}()

	for update := range bot.GetUpdatesChan() {
		i := 0
		if update.PreCheckoutQuery != nil {
			i = int(update.PreCheckoutQuery.From.ID % int64(concurrency))
		}

		if update.CallbackQuery != nil {
			i = int(update.CallbackQuery.From.ID % int64(concurrency))
		}

		if update.Message != nil {
			i = int(update.Message.Chat.ID % int64(concurrency))
		}

		chans[i] <- &update
	}
}
