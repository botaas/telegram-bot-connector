package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	unknownUser = "unknown"
	HTMLFormat  = "HTML"
	HTMLNick    = "htmlnick"
	MarkdownV2  = "MarkdownV2"
)

func TGGetParseMode(bot *tgbotapi.BotAPI, username string, text string) (textout string, parsemode string) {
	textout = username + text

	/*
		if bot.GetString("MessageFormat") == HTMLFormat {
			parsemode = tgbotapi.ModeHTML
		}

		if b.GetString("MessageFormat") == "Markdown" {
			parsemode = tgbotapi.ModeMarkdown
		}

		if b.GetString("MessageFormat") == MarkdownV2 {
			parsemode = MarkdownV2
		}

		if strings.ToLower(b.GetString("MessageFormat")) == HTMLNick {
			textout = username + html.EscapeString(text)
			parsemode = tgbotapi.ModeHTML
		}
	*/

	return textout, parsemode
}
