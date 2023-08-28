package event

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/botaas/telegram-bot-connector/bot"
	"github.com/botaas/telegram-bot-connector/models"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/mitchellh/mapstructure"
)

func IsURL(str string) bool {
	// 解析URL
	u, err := url.Parse(str)
	if err != nil {
		return false
	}

	return u.Scheme != ""
}

type MessageHandler struct {
	Bot *bot.Bot
}

func marshalInlineKeyboardMarkup(i *models.InlineKeyboardMarkup) (tgbotapi.InlineKeyboardMarkup, error) {
	var inlineKeyboardMarkup tgbotapi.InlineKeyboardMarkup

	err := mapstructure.Decode(i, &inlineKeyboardMarkup)
	return inlineKeyboardMarkup, err
}

func (h *MessageHandler) Handle(ctx context.Context, ev *cloudevents.Event) error {
	var cmsg models.Message
	err := json.Unmarshal(ev.Data(), &cmsg)
	if err != nil {
		return err
	}

	/*
		username := ""
		if cmsg.From != nil {
			username = cmsg.From.UserName
		}
	*/

	if len(cmsg.Text) > 0 {
		msg := tgbotapi.NewMessage(cmsg.Chat.ID, cmsg.Text)
		if cmsg.InlineKeyboardMarkup != nil {
			markup, err := marshalInlineKeyboardMarkup(cmsg.InlineKeyboardMarkup)
			if err == nil {
				msg.ReplyMarkup = markup
			}
		}

		msg.ReplyToMessageID = cmsg.ReplyToMessageID
		msg.DisableNotification = cmsg.DisableNotification
		msg.ProtectContent = cmsg.ProtectContent
		_, err = h.Bot.API().Send(msg)
		return err
	} else if cmsg.Photo != nil {
		for _, p := range cmsg.Photo {
			var file tgbotapi.RequestFileData
			if len(p.FileID) > 0 {
				file = tgbotapi.FileID(p.FileID)
			} else if len(p.Url) > 0 {
				file = tgbotapi.FileURL(p.Url)
			}

			msg := tgbotapi.NewPhoto(cmsg.Chat.ID, file)

			msg.ReplyToMessageID = cmsg.ReplyToMessageID
			msg.DisableNotification = cmsg.DisableNotification
			msg.ProtectContent = cmsg.ProtectContent

			_, err = h.Bot.API().Send(msg)
		}
	} else if cmsg.Audio != nil {
		var file tgbotapi.RequestFileData
		if len(cmsg.Audio.FileID) > 0 {
			file = tgbotapi.FileID(cmsg.Audio.FileID)
		} else if len(cmsg.Audio.Url) > 0 {
			file = tgbotapi.FileURL(cmsg.Audio.Url)
		}

		msg := tgbotapi.NewAudio(cmsg.Chat.ID, file)
		msg.ReplyToMessageID = cmsg.ReplyToMessageID
		msg.DisableNotification = cmsg.DisableNotification
		msg.ProtectContent = cmsg.ProtectContent

		_, err = h.Bot.API().Send(msg)
	} else if cmsg.Voice != nil {
		var file tgbotapi.RequestFileData
		if len(cmsg.Voice.FileID) > 0 {
			file = tgbotapi.FileID(cmsg.Voice.FileID)
		} else if len(cmsg.Voice.Url) > 0 {
			resp, err := http.Get(cmsg.Voice.Url)
			if err != nil {
				return err
			}
			f, err := os.CreateTemp("", "download-*")
			if err != nil {
				return err
			}
			defer os.Remove(f.Name())
			_, err = io.Copy(f, resp.Body)
			if err != nil {
				return err
			}

			file = tgbotapi.FilePath(f.Name())
		}

		msg := tgbotapi.NewVoice(cmsg.Chat.ID, file)
		msg.ReplyToMessageID = cmsg.ReplyToMessageID
		msg.DisableNotification = cmsg.DisableNotification
		msg.ProtectContent = cmsg.ProtectContent
		msg.Duration = cmsg.Voice.Duration

		_, err = h.Bot.API().Send(msg)
	} else if cmsg.Video != nil {
		var file tgbotapi.RequestFileData
		if len(cmsg.Video.FileID) > 0 {
			file = tgbotapi.FileID(cmsg.Video.FileID)
		} else if len(cmsg.Video.Url) > 0 {
			file = tgbotapi.FileURL(cmsg.Video.Url)
		}

		msg := tgbotapi.NewVideo(cmsg.Chat.ID, file)
		msg.ReplyToMessageID = cmsg.ReplyToMessageID
		msg.DisableNotification = cmsg.DisableNotification
		msg.ProtectContent = cmsg.ProtectContent
		_, err = h.Bot.API().Send(msg)
	} else if cmsg.Invoice != nil {
		var prices []tgbotapi.LabeledPrice
		for _, p := range cmsg.Invoice.Prices {
			price := tgbotapi.LabeledPrice{
				Label:  p.Label,
				Amount: p.Amount,
			}

			prices = append(prices, price)
		}

		msg := tgbotapi.InvoiceConfig{
			BaseChat:            tgbotapi.BaseChat{ChatID: cmsg.Chat.ID},
			Title:               cmsg.Invoice.Title,
			Description:         cmsg.Invoice.Description,
			Payload:             cmsg.Invoice.Payload,
			ProviderToken:       cmsg.Invoice.ProviderToken,
			StartParameter:      cmsg.Invoice.StartParameter,
			Currency:            cmsg.Invoice.Currency,
			Prices:              prices,
			MaxTipAmount:        cmsg.Invoice.MaxTipAmount,
			SuggestedTipAmounts: cmsg.Invoice.SuggestedTipAmounts,
			PhotoURL:            cmsg.Invoice.PhotoURL,
			PhotoSize:           cmsg.Invoice.PhotoSize,
			PhotoWidth:          cmsg.Invoice.PhotoWidth,
			PhotoHeight:         cmsg.Invoice.PhotoHeight,
		}
		msg.ReplyToMessageID = cmsg.ReplyToMessageID
		msg.ProtectContent = cmsg.ProtectContent
		_, err = h.Bot.API().Send(msg)
	} else if cmsg.MediaGroup != nil {
		files := []any{}

		for _, f := range cmsg.MediaGroup.Files {
			var media models.BaseInputMedia
			err := mapstructure.Decode(f, &media)
			if err != nil {
				continue
			}

			var requestFileData tgbotapi.RequestFileData
			if IsURL(media.Media) {
				requestFileData = tgbotapi.FileURL(media.Media)
			} else {
				requestFileData = tgbotapi.FileID(media.Media)
			}

			switch media.Type {
			case "photo":
				photo := tgbotapi.NewInputMediaPhoto(
					requestFileData,
				)
				files = append(files, photo)
			case "audio":
				audio := tgbotapi.NewInputMediaAudio(
					requestFileData,
				)
				files = append(files, audio)
			case "video":
				video := tgbotapi.NewInputMediaVideo(
					requestFileData,
				)

				files = append(files, video)
			case "animation":
				animation := tgbotapi.NewInputMediaAnimation(
					requestFileData,
				)
				files = append(files, animation)
			case "document":
				document := tgbotapi.NewInputMediaDocument(
					requestFileData,
				)

				/*
					document.Caption, document.ParseMode = bot.TGGetParseMode(h.Bot.API(), username, "")
					if len(document.Caption) == 0 && len(document.ParseMode) == 0 {
					}
				*/

				files = append(files, document)
			}
		}

		mediaGroup := tgbotapi.NewMediaGroup(
			cmsg.Chat.ID,
			files,
		)

		_, err = h.Bot.API().SendMediaGroup(mediaGroup)
	}

	return err
}
