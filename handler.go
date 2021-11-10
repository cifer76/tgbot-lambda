package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Handler func(ctx context.Context, update *tgbotapi.Update, bot *tgbotapi.BotAPI)

func getHandler(ctx context.Context, update *tgbotapi.Update) Handler {
	if !update.Message.IsCommand() {
		return handleGroupLink
	}

	switch update.Message.Command() {
	case "start", "help":
		return handleStart
	default:
		return handleUnknownCommand
	}
}

func handleStart(ctx context.Context, update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Send your group link to request index it")
	_, err := bot.Send(msg)
	if err != nil {
		log.Println(err)
	}
}

func handleUnknownCommand(ctx context.Context, update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command")
	_, err := bot.Send(msg)
	if err != nil {
		log.Println(err)
	}
}

func handleGroupLink(ctx context.Context, update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	groupLink := update.Message.Text

	defer func() {
		_, err := bot.Send(msg)
		if err != nil {
			log.Println(err)
		}
	}()

	if !strings.HasPrefix(groupLink, "https://t.me/") || !strings.HasPrefix(groupLink, "t.me/") {
		msg.Text = "Invalid group link, the link must start with https://t.me/ or at least t.me/"
		return
	}

	// extract the group username
	groupUsername := strings.TrimSpace(groupLink[strings.Index(groupLink, "t.me/")+6:])

	// check username validity, telegram allows only letters, numbers and underscore characters in username
	pattern := regexp.MustCompile(`^[a-zA-Z]+[0-9_]+$`)
	if !pattern.MatchString(groupUsername) {
		msg.Text = "group username in the link invalid, must start with letters and contain only letters, numbers and underscore"
		return
	}

	// query group info
	chat, err := bot.GetChat(tgbotapi.ChatConfig{
		SuperGroupUsername: groupUsername,
	})
	if err != nil {
		log.Println(err)
		msg.Text = "index failed, please try again later"
		return
	}

	if chat.IsPrivate() {
		msg.Text = "index failed, private chats are not supported"
		return
	}

	log.Printf("Group info:\nname: %s\ntype: %s\nphoto: %s\ndescription: %s\n", chat.Title, chat.Type, chat.Photo, chat.Description)

	// index the group to persistent storage
	_, err = dynsvc.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("groups"),
		Item: map[string]types.AttributeValue{
			"username": &types.AttributeValueMemberS{Value: chat.UserName},
		},
	})
	if err != nil {
		log.Printf("index %s error: %v\n", chat.UserName, err)
		msg.Text = "index failed, please try again later"
		return
	}

	msg.Text = fmt.Sprintf("Group %s has been indexed", chat.Title)
}
