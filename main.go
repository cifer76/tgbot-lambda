package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	bot    *tgbotapi.BotAPI
	client = lambda.New(session.New())
)

func callLambda() (string, error) {
	input := &lambda.GetAccountSettingsInput{}
	req, resp := client.GetAccountSettingsRequest(input)
	err := req.Send()
	output, _ := json.Marshal(resp.AccountUsage)
	return string(output), err
}

func HandleTGUpdates(ctx context.Context, event events.SQSEvent) (string, error) {

	// event
	eventJson, _ := json.MarshalIndent(event, "", "  ")
	log.Printf("EVENT: %s", eventJson)
	// environment variables
	log.Printf("REGION: %s", os.Getenv("AWS_REGION"))
	log.Println("ALL ENV VARS:")
	for _, element := range os.Environ() {
		log.Println(element)
	}
	// request context
	lc, _ := lambdacontext.FromContext(ctx)
	log.Printf("REQUEST ID: %s", lc.AwsRequestID)
	// global variable
	log.Printf("FUNCTION NAME: %s", lambdacontext.FunctionName)
	// context method
	deadline, _ := ctx.Deadline()
	log.Printf("DEADLINE: %s", deadline)
	// AWS SDK call
	usage, err := callLambda()
	log.Printf("RESPONSE: %s", usage)
	if err != nil {
		return "ERROR", err
	}
	return usage, nil

	// initialize tgbot
	/*
		bot, _ = tgbotapi.NewBotAPI("407954143:AAGDxLmxcr5DGVE3GY_Ih9pe8GIh-P0EhDI")
		bot.Debug = true

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			<-c
			os.Exit(0)
		}()

		log.Printf("Authorized on account %s", bot.Self.UserName)

		_, err := bot.SetWebhook(tgbotapi.NewWebhookWithCert("https://www.google.com:8443/"+bot.Token, "cert.pem"))
		if err != nil {
			log.Fatal(err)
			return
		}
		info, err := bot.GetWebhookInfo()
		if err != nil {
			log.Fatal(err)
			return
		}
		if info.LastErrorDate != 0 {
			log.Printf("Telegram callback failed: %s", info.LastErrorMessage)
		}

		updates := bot.ListenForWebhook("/" + bot.Token)
		go http.ListenAndServe("0.0.0.0:8443", nil)

		for update := range updates {
			if update.Message == nil { // ignore any non-Message Updates
				continue
			}

			log.Printf("%+v\n", update)
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID

			bot.Send(msg)
		}

	*/
}

func main() {
	runtime.Start(HandleTGUpdates)
}
