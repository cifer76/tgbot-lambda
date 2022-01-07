package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler func(ctx context.Context, update *tgbotapi.Update)

func handleSearch(ctx context.Context, update *tgbotapi.Update) {
	tokens := strings.Fields(update.Message.Text)

	var rsp string
	defer func() {
		if rsp == "" {
			return
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, rsp)
		msg.ParseMode = tgbotapi.ModeHTML
		msg.DisableWebPagePreview = true
		_, err := bot.Send(msg)
		if err != nil {
			log.Println(err)
		}
	}()

	keywords := []string{}
	for _, t := range tokens {
		if patternGroupTag.MatchString(t) {
			keywords = append(keywords, t)
		}
	}

	groups := opensearchSearchGroup(ctx, keywords)

	rsp = `
找到如下结果:

`

	for i, g := range groups {
		icon := "👥"
		if g.Type == "channel" {
			icon = "📢"
		}
		line := fmt.Sprintf("%d. %s %s - <a href=\"https://t.me/%s\">%s</a>\n", i+1, icon, formatMemberCount(g.MemberCount), g.Username, g.Title)
		rsp += line
	}
	return
}

func handleNewUserChat(ctx context.Context, update *tgbotapi.Update) {
	tguser := update.MyChatMember.From
	userRecord := UserRecord{
		ID:        tguser.ID,
		Username:  tguser.UserName,
		FirstName: tguser.FirstName,
		LastName:  tguser.LastName,
	}
	ddbWriteUser(ctx, userRecord)
}

func handleNewGroupChat(ctx context.Context, update *tgbotapi.Update) {
	groupChat := update.MyChatMember.Chat

	fmt.Printf("bot was added to a new group, groupID: %v, groupTitle: %v, groupUsername: %v, groupType: %v\n", groupChat.ID, groupChat.Title, groupChat.UserName, groupChat.Type)

	if groupChat.UserName == "" {
		fmt.Printf("group username is empty, not support indexing private group\n")
		return
	}

	groupChat, memberCount, err := getGroupInfo(ctx, groupChat.UserName)
	if err != nil {
		fmt.Printf("get group info failed, error: %v", err)
		return
	}
	log.Printf("group description: %s\n", groupChat.Description)

	//tags := getGroupTags(ctx, groupChat.Title, groupChat.Description)
	s := GroupInfo{
		Chat:        groupChat,
		MemberCount: memberCount,
	}

	opensearchWriteGroup(ctx, GroupRecord{
		Username:    s.UserName,
		ChatID:      s.ID,
		Title:       s.Title,
		Type:        s.Type,
		Description: s.Description,
		MemberCount: s.MemberCount,
	})
}

func handleUpdate(ctx context.Context, update tgbotapi.Update) {
	log.Printf("TG Update: %+v\n", update)

	// new user started with the bot
	if determineUpdateType(ctx, &update) == UpdateType_UserUnblockedBot {
		handleNewUserChat(ctx, &update)
		return
	}

	// the bot is added into a new group
	if determineUpdateType(ctx, &update) == UpdateType_GroupAddedBot {
		handleNewGroupChat(ctx, &update)
		return
	}

	var chatID int64
	if chatID = getChatIDFromUpdate(&update); chatID == 0 {
		log.Println("not chatID found, unsupported update type")
		return
	}

	s := getState(ctx, chatID)
	if updateIsCommand(&update) {
		// 1. in reality, it isn't necessary for every command to have a state machine
		//    but we take it as so, it makes our code simple and consistent
		// 2. any command interrupts an another command's state machine

		s = &CommandState{
			ChatID:  chatID,
			Command: update.Message.Command(),
			Stage:   CommandReceived,
		}
		writeState(s)
	}

	if s != nil {
		h := getCommandHandler(s.Command)
		h(ctx, &update, s)
		return
	}

	// not command and no state, then it's the simplest case: keyword search
	if update.Message != nil && update.Message.Text != "" {
		handleSearch(ctx, &update)
	} else {
		fmt.Println("unsupported update")
	}
}
