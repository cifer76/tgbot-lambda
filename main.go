package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/go-redis/redis/v8"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	startContent = `Welcome!

Input any keyword to search for the related groups.

or choose a command following suit your needs:

/start     - show this information
/index     - index/re-index a group
/list      - list groups by categories
/recommend - recommend some groups
`
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

func HandleTGUpdates(ctx context.Context, event events.APIGatewayProxyRequest) (rsp events.APIGatewayProxyResponse, err error) {
	rsp = events.APIGatewayProxyResponse{
		StatusCode: 200,
	}

	// event
	eventJson, _ := json.Marshal(event)
	log.Printf("EVENT: %s", eventJson)

	update := &tgbotapi.Update{}
	if err := json.Unmarshal([]byte(event.Body), update); err != nil {
		log.Println("Malformed update message")
		return rsp, nil
	}

	var chatID int64
	if chatID = getChatIDInUpdate(update); chatID == 0 {
		log.Println("not chatID found, unsupported update type")
		return
	}

	// Message handling logic
	//
	// 1. multi-stage command have a state machine
	// 2. any command received will interrupt ongoing multi-stage command's state machine
	if updateIsCommand(update) {
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
			content = "please input your group link"
		case "list", "recommend":
			content = "under development"
		default:
			content = startContent
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, content)
		_, err = bot.Send(msg)
		if err != nil {
			log.Println(err)
		}

		return
	}

	// check ongoing operation
	state, err := getState(ctx, chatID)
	if err != nil {
		log.Println(err)
		return
	}

	// ongoing operation exists
	if state != nil {
		h, ok := stateHandler[state.Command]
		if !ok {
			fmt.Println("wrong state, unknown command found", state)
			return
		}

		h(ctx, update, state)
		return
	}

	// otherwise, take it a search scenario
	handleSearch(ctx, update)
	return
}

func main() {
	runtime.Start(HandleTGUpdates)
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

	redisAddr := os.Getenv("REDIS_ADDR")
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}
