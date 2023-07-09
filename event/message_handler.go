package event

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/botaas/telegram-bot-connector/bot"
	"github.com/botaas/telegram-bot-connector/models"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/mitchellh/mapstructure"
)

type MessageHandler struct {
	Bot *bot.Bot
}

func marshalInlineKeyboardMarkup(i *models.InlineKeyboardMarkup) (tgbotapi.InlineKeyboardMarkup, error) {
	var inlineKeyboardMarkup tgbotapi.InlineKeyboardMarkup

	err := mapstructure.Decode(i, &inlineKeyboardMarkup)
	return inlineKeyboardMarkup, err
}

func (h *MessageHandler) Handle(ctx context.Context, ev *cloudevents.Event) error {
	var payload models.Message
	err := json.Unmarshal(ev.Data(), &payload)
	if err != nil {
		return err
	}

	if len(payload.Text) > 0 {
		msg := tgbotapi.NewMessage(payload.Chat.ID, payload.Text)
		if payload.InlineKeyboardMarkup != nil {
			markup, err := marshalInlineKeyboardMarkup(payload.InlineKeyboardMarkup)
			if err == nil {
				msg.ReplyMarkup = markup
			}
		}
		msg.ReplyToMessageID = payload.ReplyToMessageID

		_, err = h.Bot.API().Send(msg)
		return err
	} else if payload.Photo != nil {
		for _, p := range payload.Photo {
			msg := tgbotapi.NewPhoto(payload.Chat.ID, tgbotapi.FileURL(p.Url))
			msg.ReplyToMessageID = payload.ReplyToMessageID
			msg.DisableNotification = payload.DisableNotification
			_, err = h.Bot.API().Send(msg)
		}
	} else if payload.Audio != nil {
		msg := tgbotapi.NewAudio(payload.Chat.ID, tgbotapi.FileURL(payload.Audio.Url))
		msg.ReplyToMessageID = payload.ReplyToMessageID
		_, err = h.Bot.API().Send(msg)
	} else if payload.Voice != nil {
		resp, err := http.Get(payload.Voice.Url)
		if err != nil {
			return err
		}
		file, err := os.CreateTemp("", "download-*")
		if err != nil {
			return err
		}
		defer os.Remove(file.Name())
		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return err
		}

		msg := tgbotapi.VoiceConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChatID:           payload.Chat.ID,
					ReplyToMessageID: payload.ReplyToMessageID,
				},
				File: tgbotapi.FilePath(file.Name()),
			},
			Duration: payload.Voice.Duration,
		}

		_, err = h.Bot.API().Send(msg)
	} else if payload.Video != nil {
		msg := tgbotapi.NewVideo(payload.Chat.ID, tgbotapi.FileURL(payload.Video.Url))
		msg.ReplyToMessageID = payload.ReplyToMessageID
		_, err = h.Bot.API().Send(msg)
	} else if payload.Invoice != nil {
		var prices []tgbotapi.LabeledPrice
		for _, p := range payload.Invoice.Prices {
			price := tgbotapi.LabeledPrice{
				Label:  p.Label,
				Amount: p.Amount,
			}

			prices = append(prices, price)
		}

		msg := tgbotapi.InvoiceConfig{
			BaseChat:            tgbotapi.BaseChat{ChatID: payload.Chat.ID},
			Title:               payload.Invoice.Title,
			Description:         payload.Invoice.Description,
			Payload:             payload.Invoice.Payload,
			ProviderToken:       payload.Invoice.ProviderToken,
			StartParameter:      payload.Invoice.StartParameter,
			Currency:            payload.Invoice.Currency,
			Prices:              prices,
			MaxTipAmount:        payload.Invoice.MaxTipAmount,
			SuggestedTipAmounts: payload.Invoice.SuggestedTipAmounts,
			PhotoURL:            payload.Invoice.PhotoURL,
			PhotoSize:           payload.Invoice.PhotoSize,
			PhotoWidth:          payload.Invoice.PhotoWidth,
			PhotoHeight:         payload.Invoice.PhotoHeight,
		}
		msg.ReplyToMessageID = payload.ReplyToMessageID
		_, err = h.Bot.API().Send(msg)
	}

	return err
}
