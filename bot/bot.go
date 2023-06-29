package bot

import (
	"os"
	"time"

	"github.com/botaas/telegram-bot-connector/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api          *tgbotapi.BotAPI
	editInterval time.Duration
	Self         *models.User
}

func New(token string, editInterval time.Duration) (*Bot, error) {
	var api *tgbotapi.BotAPI
	var err error
	apiEndpoint, exist := os.LookupEnv("TELEGRAM_API_ENDPOINT")
	if exist && apiEndpoint != "" {
		api, err = tgbotapi.NewBotAPIWithAPIEndpoint(token, apiEndpoint)
	} else {
		api, err = tgbotapi.NewBotAPI(token)
	}
	if err != nil {
		return nil, err
	}

	return &Bot{
		Self: &models.User{
			ID:                      api.Self.ID,
			UserName:                api.Self.UserName,
			LanguageCode:            api.Self.LanguageCode,
			IsBot:                   api.Self.IsBot,
			FirstName:               api.Self.FirstName,
			LastName:                api.Self.LastName,
			CanJoinGroups:           api.Self.CanJoinGroups,
			CanReadAllGroupMessages: api.Self.CanReadAllGroupMessages,
			SupportsInlineQueries:   api.Self.SupportsInlineQueries,
		},
		api:          api,
		editInterval: editInterval,
	}, nil
}

func (b *Bot) GetUpdatesChan() tgbotapi.UpdatesChannel {
	cfg := tgbotapi.NewUpdate(0)
	cfg.Timeout = 30
	return b.api.GetUpdatesChan(cfg)
}

func (b *Bot) Stop() {
	b.api.StopReceivingUpdates()
}

func (b *Bot) API() *tgbotapi.BotAPI {
	return b.api
}
