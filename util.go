package main

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
		memberCount = fmt.Sprintf("%.1fk", float64(count)/1000.0)
	} else {
		memberCount = fmt.Sprintf("%.1fm", float64(count)/1000000.0)
	}
	return memberCount
}

const (
	UpdateType_TextMessage = iota
	UpdateType_CommandMessage
	UpdateType_UserBlockedBot   // user stop the bot
	UpdateType_UserUnblockedBot // user start or restart the bot
	UpdateType_GroupAddedBot
	UpdateType_GroupRemovedBot
)

func determineUpdateType(ctx context.Context, update *tgbotapi.Update) int {
	if update.MyChatMember != nil {
		status := update.MyChatMember.NewChatMember.Status
		if update.MyChatMember.Chat.ID > 0 { // private chats
			if status == "member" {
				return UpdateType_UserUnblockedBot
			} else if status == "kicked" {
				return UpdateType_UserBlockedBot
			}
		} else { // supergroup chats
			if status == "member" {
				return UpdateType_GroupAddedBot
			} else if status == "left" {
				return UpdateType_GroupRemovedBot
			}
		}
		return -1
	} else if update.Message != nil {
		if update.Message.Text != "" {
			if updateIsCommand(update) {
				return UpdateType_CommandMessage
			} else {
				return UpdateType_TextMessage
			}
		}

		if update.Message.NewChatMembers != nil {
			// TODO
			// update.Message.NewChatMembers[0] == "myself"?
			return -1
		}

	}
	return -1
}
