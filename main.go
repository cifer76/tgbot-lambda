package main

import (
	"context"
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var bot *tgbotapi.BotAPI

func main() {
	// initialize tgbot
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatalln("environment BOT_TOKEN empty!")
	}
	// we don't use the NewBotAPI() method because it always makes a getMe
	// call for verification, we are sure that the bot token is correct so
	// we don't need this procedure
	bot = &tgbotapi.BotAPI{
		Token:  botToken,
		Client: &http.Client{},
		Buffer: 100,
	}

	u := tgbotapi.NewUpdate(-1)
	u.Timeout = 60
	updates, _ := bot.GetUpdatesChan(u)

	for u := range updates {
		handleUpdate(context.Background(), u)
	}
}
