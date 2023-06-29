package converter

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"

	"github.com/botaas/telegram-bot-connector/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TranscriptionResponse struct {
	Text string `json:"text"`
}

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    string `json:"code"`
	} `json:"error"`
}

func voiceToText(bot *tgbotapi.BotAPI, fileID string) (string, error) {
	url, err := bot.GetFileDirectURL(fileID)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "mp3", "pipe:1")
	cmd.Stdin = resp.Body

	reader, writer := io.Pipe()
	cmd.Stdout = writer

	go func() {
		defer writer.Close()
		if err := cmd.Run(); err != nil {
			log.Print(err)
		}
	}()

	var reqeustBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&reqeustBody)
	fieldWriter, err := multipartWriter.CreateFormField("model")
	if err != nil {
		return "", err
	}

	fieldWriter.Write([]byte("whisper-1"))
	fileWrite, err := multipartWriter.CreateFormFile("file", fileID+".mp3")
	if err != nil {
		log.Println(err)
		return "", err
	}
	if _, err = io.Copy(fileWrite, reader); err != nil {
		log.Println(err)
		return "", err
	}
	if err := multipartWriter.Close(); err != nil {
		log.Fatal(err)
	}
	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", &reqeustBody)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	client := &http.Client{}
	resp, err = client.Do(req)
	if resp.StatusCode != 200 {
		var errorResponse ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		log.Println(errorResponse.Error.Message)
		return "", errors.New(errorResponse.Error.Message)
	}
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return "", err
	}
	var transcriptionResponse TranscriptionResponse
	err = json.Unmarshal(response, &transcriptionResponse)
	if err != nil {
		log.Println(err)
		return "", nil
	}
	log.Println(transcriptionResponse.Text)
	return transcriptionResponse.Text, nil
}

func NormalizeTelegramMessage(bot *tgbotapi.BotAPI, m *tgbotapi.Message) (*models.Message, error) {
	o := &models.Message{
		ID: m.MessageID,
		Chat: &models.Chat{
			ID: m.Chat.ID,
		},
		Text: m.Text,
		From: &models.User{
			ID:                      m.From.ID,
			UserName:                m.From.UserName,
			LanguageCode:            m.From.LanguageCode,
			IsBot:                   m.From.IsBot,
			FirstName:               m.From.FirstName,
			LastName:                m.From.LastName,
			CanJoinGroups:           m.From.CanJoinGroups,
			CanReadAllGroupMessages: m.From.CanReadAllGroupMessages,
			SupportsInlineQueries:   m.From.SupportsInlineQueries,
		},
		To: &models.User{
			ID:                      bot.Self.ID,
			UserName:                bot.Self.UserName,
			LanguageCode:            bot.Self.LanguageCode,
			IsBot:                   bot.Self.IsBot,
			FirstName:               bot.Self.FirstName,
			LastName:                bot.Self.LastName,
			CanJoinGroups:           bot.Self.CanJoinGroups,
			CanReadAllGroupMessages: bot.Self.CanReadAllGroupMessages,
			SupportsInlineQueries:   bot.Self.SupportsInlineQueries,
		},
	}

	if len(m.Photo) > 0 {
		var photos []*models.Photo
		for _, photo := range m.Photo {
			file, err := bot.GetFileDirectURL(photo.FileID)
			if err != nil {
				continue
			}

			photo := &models.Photo{
				Url:      file,
				Width:    photo.Width,
				Height:   photo.Height,
				FileID:   photo.FileID,
				FileSize: photo.FileSize,
			}

			photos = append(photos, photo)
		}

		o.Photo = photos
	}

	if m.Audio != nil {
		file, err := bot.GetFileDirectURL(m.Audio.FileID)
		if err != nil {
			return nil, err
		}

		audio := &models.Audio{
			Url:      file,
			FileID:   m.Audio.FileID,
			Duration: m.Audio.Duration,
			MimeType: m.Audio.MimeType,
			FileSize: m.Audio.FileSize,
		}
		o.Audio = audio
	}

	if m.Voice != nil {
		url, err := bot.GetFileDirectURL(m.Voice.FileID)
		if err != nil {
			return nil, err
		}

		text, err := voiceToText(bot, m.Voice.FileID)
		if err == nil {
			o.Text = text
		}

		voice := &models.Voice{
			Url:      url,
			FileID:   m.Voice.FileID,
			Duration: m.Voice.Duration,
			MimeType: m.Voice.MimeType,
			FileSize: m.Voice.FileSize,
		}
		o.Voice = voice
	}

	if m.SuccessfulPayment != nil {
		var orderInfo *models.OrderInfo
		if m.SuccessfulPayment.OrderInfo != nil {
			var shippingAddress *models.ShippingAddress
			if m.SuccessfulPayment.OrderInfo.ShippingAddress != nil {
				shippingAddress = &models.ShippingAddress{
					CountryCode: m.SuccessfulPayment.OrderInfo.ShippingAddress.CountryCode,
					State:       m.SuccessfulPayment.OrderInfo.ShippingAddress.State,
					City:        m.SuccessfulPayment.OrderInfo.ShippingAddress.City,
					StreetLine1: m.SuccessfulPayment.OrderInfo.ShippingAddress.StreetLine1,
					StreetLine2: m.SuccessfulPayment.OrderInfo.ShippingAddress.StreetLine2,
					PostCode:    m.SuccessfulPayment.OrderInfo.ShippingAddress.PostCode,
				}
			}

			orderInfo = &models.OrderInfo{
				Name:            m.SuccessfulPayment.OrderInfo.Name,
				PhoneNumber:     m.SuccessfulPayment.OrderInfo.PhoneNumber,
				Email:           m.SuccessfulPayment.OrderInfo.Email,
				ShippingAddress: shippingAddress,
			}
		}

		successfulPayment := &models.SuccessfulPayment{
			Currency:                m.SuccessfulPayment.Currency,
			TotalAmount:             m.SuccessfulPayment.TotalAmount,
			InvoicePayload:          m.SuccessfulPayment.InvoicePayload,
			ShippingOptionID:        m.SuccessfulPayment.ShippingOptionID,
			OrderInfo:               orderInfo,
			TelegramPaymentChargeID: m.SuccessfulPayment.TelegramPaymentChargeID,
			ProviderPaymentChargeID: m.SuccessfulPayment.ProviderPaymentChargeID,
		}
		o.SuccessfulPayment = successfulPayment
	}

	return o, nil
}

func NormalizeTelegramUser(user *tgbotapi.User) *models.User {
	return &models.User{
		ID:                      user.ID,
		UserName:                user.UserName,
		LanguageCode:            user.LanguageCode,
		IsBot:                   user.IsBot,
		FirstName:               user.FirstName,
		LastName:                user.LastName,
		CanJoinGroups:           user.CanJoinGroups,
		CanReadAllGroupMessages: user.CanReadAllGroupMessages,
		SupportsInlineQueries:   user.SupportsInlineQueries}
}
