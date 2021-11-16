package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	dynsvc *dynamodb.Client
	bot    *tgbotapi.BotAPI
)

func HandleTGUpdates(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	rsp := events.APIGatewayProxyResponse{
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

	if update.Message == nil { // ignore any non-Message Updates
		return rsp, nil
	}

	h := getHandler(ctx, update)
	h(ctx, update, bot)

	return rsp, nil
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
}
