package main

import (
	"context"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var bot *tgbotapi.BotAPI

func main() {
	// initialize tgbot
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatalln("environment BOT_TOKEN empty!")
	}
	botDebug := os.Getenv("BOT_DEBUG")

	var err error
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = botDebug == "true"

	u := tgbotapi.NewUpdate(-1)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for u := range updates {
		handleUpdate(context.Background(), u)
	}
}
