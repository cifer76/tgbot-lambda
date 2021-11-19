package main

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Handler func(ctx context.Context, update *tgbotapi.Update)

func getHandler(ctx context.Context, update *tgbotapi.Update) Handler {
	if !update.Message.IsCommand() {
		return handleText
	}

	switch update.Message.Command() {
	case "request":
		return handleRequestIndex
	default:
		return handleStart
	}
}

func handleStart(ctx context.Context, update *tgbotapi.Update) {
	content := `Welcome!

Input any keyword to search for the related groups.

or choose a command following suit your needs:

/start   - show this information
/request - request me to index/re-index a group
/list    - list the indexed groups, maybe by categories
`

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, content)
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
		Next:    GroupLinkReceived,
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
}

func handleText(ctx context.Context, update *tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	msg := tgbotapi.NewMessage(chatID, "")

	state, err := getState(ctx, chatID)
	if err != nil {
		// redis error, log the error then do nothing, the user will receive no reply
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

	// in a command's context
	if h, ok := stateHandler[state.Command]; ok {
		reply, err := h(ctx, update, state)
		msg.Text = reply
		if err != nil {
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
