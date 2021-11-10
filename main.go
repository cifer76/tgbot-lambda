package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func HandleTGUpdates(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	rsp := events.APIGatewayProxyResponse{
		StatusCode: 200,
	}

	// event
	eventJson, _ := json.Marshal(event)
	log.Printf("EVENT: %s", eventJson)

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Println("environment BOT_TOKEN empty!")
		return rsp, nil
	}

	// initialize tgbot, we don't use the NewBotAPI() method because it
	// always makes a getMe call for verification, since we are in the faas
	// environment, making a getMe call everytime the function get called is
	// resource wasting
	bot := &tgbotapi.BotAPI{
		Token:  botToken,
		Client: &http.Client{},
		Buffer: 100,
		Debug:  true,
	}

	update := tgbotapi.Update{}
	if err := json.Unmarshal([]byte(event.Body), &update); err != nil {
		log.Println("Malformed update message")
		return rsp, nil
	}

	if update.Message != nil { // ignore any non-Message Updates
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID
		_, err := bot.Send(msg)
		if err != nil {
			log.Println(err)
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil

	/*
		for update := range updates {
			if update.Message == nil { // ignore any non-Message Updates
				continue
			}

			log.Printf("%+v\n", update)
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		}

	*/
}

func main() {
	runtime.Start(HandleTGUpdates)
}
