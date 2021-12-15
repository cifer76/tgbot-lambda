package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func getChatMessageFromUpdate(update *tgbotapi.Update) string {
	if update.Message != nil {
		return update.Message.Text
	} else if update.CallbackQuery != nil {
		return update.CallbackQuery.Data
	}
	return ""
}

func getChatIDFromUpdate(update *tgbotapi.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Chat.ID
	}
	return 0
}

func updateIsCommand(update *tgbotapi.Update) bool {
	return update.Message != nil && update.Message.IsCommand()
}

func formatMemberCount(count int) string {
	memberCount := ""
	if count < 1000 {
		memberCount = fmt.Sprintf("%d", count)
	} else if count < 1000000 {
		memberCount = fmt.Sprintf("%2f K", float64(count)/1000.0)
	} else {
		memberCount = fmt.Sprintf("%2f M", float64(count)/1000000.0)
	}
	return memberCount
}
