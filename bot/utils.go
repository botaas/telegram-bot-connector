package bot

import (
	"html"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TGGetParseMode(bot *tgbotapi.BotAPI, username string, text string) (textout string, parsemode string) {
	textout = username + text
	if bot.GetString("MessageFormat") == HTMLFormat {
		b.Log.Debug("Using mode HTML")
		parsemode = tgbotapi.ModeHTML
	}
	if b.GetString("MessageFormat") == "Markdown" {
		b.Log.Debug("Using mode markdown")
		parsemode = tgbotapi.ModeMarkdown
	}
	if b.GetString("MessageFormat") == MarkdownV2 {
		b.Log.Debug("Using mode MarkdownV2")
		parsemode = MarkdownV2
	}
	if strings.ToLower(b.GetString("MessageFormat")) == HTMLNick {
		b.Log.Debug("Using mode HTML - nick only")
		textout = username + html.EscapeString(text)
		parsemode = tgbotapi.ModeHTML
	}
	return textout, parsemode
}
