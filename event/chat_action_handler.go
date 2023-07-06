package event

import (
	"context"
	"encoding/json"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/botaas/telegram-bot-connector/bot"
	"github.com/botaas/telegram-bot-connector/models"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type ChatActionHandler struct {
	Bot *bot.Bot
}

func (h *ChatActionHandler) Handle(ctx context.Context, ev *cloudevents.Event) error {
	var payload models.ChatAction
	err := json.Unmarshal(ev.Data(), &payload)
	if err != nil {
		return err
	}

	action := tgbotapi.NewChatAction(payload.ChatID, payload.Action)
	_, err = h.Bot.API().Request(action)

	return err
}
