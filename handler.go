package main

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	commandRequestIndex = "r"
)

type CommandState struct {
	ChatID  int64  `json:"chatID"` // who is initiating the command?
	Command string `json:"command"`

	RequestIndexState *RequestIndexState `json:"requestIndexState"`
}

type GroupInfo struct {
	tgbotapi.Chat

	Category string   `json:"category"` // group category specified by the requestor
	Tags     []string `json:"tags"`     // group tags specified by the requestor
}

type RequestIndexState struct {
	Group *GroupInfo `json:"group"` // the basic info of the group being requested index

	Current int `json:"current"` // current stage
	Next    int `json:"next"`    // next stage
}

type Handler func(ctx context.Context, update *tgbotapi.Update)

func getHandler(ctx context.Context, update *tgbotapi.Update) Handler {
	if !update.Message.IsCommand() {
		return handleText
	}

	switch update.Message.Command() {
	case "start", "help", "h":
		return handleStart
	case "r":
		return handleRequestIndex
	default:
		return handleUnknownCommand
	}
}

func handleStart(ctx context.Context, update *tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Send your group link to request index it")
	_, err := bot.Send(msg)
	if err != nil {
		log.Println(err)
	}
}

func handleUnknownCommand(ctx context.Context, update *tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command")
	_, err := bot.Send(msg)
	if err != nil {
		log.Println(err)
	}
}

func handleRequestIndex(ctx context.Context, update *tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	state := &CommandState{
		ChatID:  update.Message.Chat.ID,
		Command: update.Message.Command(),
		RequestIndexState: &RequestIndexState{
			Group: &GroupInfo{},
			Next:  GroupLinkReceived,
		},
	}

	// write the state machine, overwrite the previous one if exists
	if err := writeState(ctx, state); err != nil {
		log.Println(err)
		return
	}

	msg.Text = "please input your group link"
	_, err := bot.Send(msg)
	if err != nil {
		log.Println(err)
		return
	}

	/*
		tokens := strings.Fields(update.Message.Text)
		tokens = tokens[1:] // index 0 is the command string

		for i, t := range tokens {
			switch state.Next {
			case GroupLinkReceived:
				if err := handleRIGroupLink(ctx, state, t); err != nil {
					msg.Text = err.Error()
					return
				}
				state.Next = CategoryReceived
			case CategoryReceived:
				if err := handleRIGroupCategory(ctx, state, t); err != nil {
					msg.Text = err.Error()
					return
				}
				state.Next = TagsReceived
			case TagsReceived:
				if err := handleRIGroupTags(ctx, state, tokens[i:]); err != nil {
					msg.Text = err.Error()
					return
				}
				state.Next = Done
				break
			}
		}

		if (len(tokens)) < 1 {
			state.Next = GroupLinkReceived
			msg.Text = "please input your group link"
			return
		}

		groupLink := tokens[1]
		if !strings.HasPrefix(groupLink, "https://t.me/") && !strings.HasPrefix(groupLink, "t.me/") {
			state.Next = GroupLinkReceived
			msg.Text = "Invalid group link, the link must start with https://t.me/ or t.me/, please re-enter"
			return
		}

		// extract the group username
		groupUsername := strings.TrimSpace(groupLink[strings.Index(groupLink, "t.me/")+5:])

		// check username validity, telegram allows only letters, numbers and underscore characters in username
		pattern := regexp.MustCompile(`^[a-zA-Z]+[0-9_a-zA-Z]+$`)
		if !pattern.MatchString(groupUsername) {
			state.Next = GroupLinkReceived
			msg.Text = "group username in the link invalid, must start with letters and contain only letters, numbers and underscore, please re-enter"
			return
		}

		// query group info
		chat, err := bot.GetChat(tgbotapi.ChatConfig{
			SuperGroupUsername: "@" + groupUsername, // must be proceeded with @, refer to: https://core.telegram.org/bots/api#getchat
		})
		if err != nil {
			log.Printf("getChat for %s error: %v\n", groupUsername, err)
			state.Next = GroupLinkReceived
			msg.Text = "can't find the group you provided, please check your group username and re-enter"
			return
		}

		state.Chat = chat
		state.Current = GroupLinkReceived

		if (len(tokens)) < 1 {
			state.Next = GroupLinkReceived
			msg.Text = "please input your group link"
			return
		}

		state.Next = CategoryReceived
	*/

}

func handleText(ctx context.Context, update *tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	msg := tgbotapi.NewMessage(chatID, "")

	state, err := getState(ctx, chatID)
	if err != nil { // redis error, log the error then do nothing, the user will receive no reply
		log.Println(err)
		return
	}

	// not in a command context
	if state == nil {
		// take this as search case

		// TODO handle search, search from dynamodb (or from other search engines)
		// construct the search results

		msg.Text = "search function is under development"
		_, err := bot.Send(msg)
		if err != nil {
			log.Println(err)
		}

		return
	}

	if h, ok := commandStateHandler[state.Command]; ok {
		reply, err := h(ctx, state, update)
		if err == nil {
			writeState(ctx, state)
			msg.Text = reply
		} else {
			msg.Text = err.Error()
		}
		_, err = bot.Send(msg)
		if err != nil {
			log.Println(err)
		}
	} else {
		fmt.Printf("wrong state: %v\n", state)
	}
}
