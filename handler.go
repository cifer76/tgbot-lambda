package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
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

	if !strings.HasPrefix(groupLink, "https://t.me/") && !strings.HasPrefix(groupLink, "t.me/") {
		msg.Text = "Invalid group link, the link must start with https://t.me/ or at least t.me/"
		return
	}

	// extract the group username
	groupUsername := strings.TrimSpace(groupLink[strings.Index(groupLink, "t.me/")+5:])

	// check username validity, telegram allows only letters, numbers and underscore characters in username
	pattern := regexp.MustCompile(`^[a-zA-Z]+[0-9_a-zA-Z]+$`)
	if !pattern.MatchString(groupUsername) {
		msg.Text = "group username in the link invalid, must start with letters and contain only letters, numbers and underscore"
		return
	}

	// query group info
	chat, err := bot.GetChat(tgbotapi.ChatConfig{
		SuperGroupUsername: "@" + groupUsername, // must be proceeded with @, refer to: https://core.telegram.org/bots/api#getchat
	})
	if err != nil {
		log.Printf("getChat for %s error: %v\n", groupUsername, err)
		msg.Text = "can't find the group you provided, please check your group username"
		return
	}

	log.Printf("Group info:\nID: %v\nname: %s\ntype: %s\ndescription: %s\n",
		chat.ID, chat.Title, chat.Type, chat.Description)

	// index the group to persistent storage
	_, err = dynsvc.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("groups"),
		Item: map[string]types.AttributeValue{
			"username": &types.AttributeValueMemberS{Value: chat.UserName},
			"chat_id":  &types.AttributeValueMemberN{Value: strconv.FormatInt(chat.ID, 10)},
		},
	})
	if err != nil {
		log.Printf("index %s error: %v\n", chat.UserName, err)
		msg.Text = "index failed, please try again later"
		return
	}

	msg.Text = fmt.Sprintf("Group %s has been indexed", chat.Title)
}
