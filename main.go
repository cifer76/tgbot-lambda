package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	dynsvc *dynamodb.Client
	bot    *tgbotapi.BotAPI
)

func getChatIDInUpdate(update *tgbotapi.Update) int64 {
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

func init() {
	// Initialize dynamodb client
	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("ap-east-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v\n", err)
	}

	// Using the Config value, create the DynamoDB client
	dynsvc = dynamodb.NewFromConfig(cfg)
}
