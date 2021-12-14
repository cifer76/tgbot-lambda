package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/go-redis/redis/v8"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	dynsvc *dynamodb.Client
	bot    *tgbotapi.BotAPI
	rdb    *redis.Client
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

func handleUpdate(ctx context.Context, update tgbotapi.Update) {
	log.Printf("TG Update: %+v\n", update)

	var chatID int64
	if chatID = getChatIDInUpdate(&update); chatID == 0 {
		log.Println("not chatID found, unsupported update type")
		return
	}

	// If it's a command message
	if updateIsCommand(&update) {

		// 1. multi-stage command has a state machine
		// 2. any command interrupts ongoing multi-stage command's state machine

		// clear ongoing command's state
		clearState(ctx, chatID)

		content := ""
		switch update.Message.Command() {
		case "index":
			// /index is a multi-stage command, so it has a state machine
			state := &CommandState{
				ChatID:  chatID,
				Command: update.Message.Command(),
				Stage:   CommandReceived,
			}
			writeState(ctx, state)
			content = getLocalizedText(ctx, InputGroupLink)
		case "list", "recommend":
			content = "under development"
		default:
			content = getStartContent(ctx)
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, content)
		_, err := bot.Send(msg)
		if err != nil {
			log.Println(err)
		}
		return
	}

	// if not a command message

	// check ongoing operation
	var state *CommandState
	state, err := getState(ctx, chatID)
	if err != nil {
		log.Println(err)
		return
	}

	// ongoing operation exists
	if state != nil {
		h := stateHandler[state.Command]
		h(ctx, &update, state)
		return
	}

	// otherwise, take it a search scenario
	handleSearch(ctx, &update)

	return
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
		Debug:  true,
	}

	updates := bot.ListenForWebhook("/" + bot.Token)
	go http.ListenAndServe("0.0.0.0:8843", nil)

	for u := range updates {
		handleUpdate(context.Background(), u)
	}

}

func init() {
	// Initialize dynamodb client
	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v\n", err)
	}

	// Using the Config value, create the DynamoDB client
	dynsvc = dynamodb.NewFromConfig(cfg)

	redisAddr := os.Getenv("REDIS_ADDR")
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}
