package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	CommandReceived int = iota
	GroupLinkReceived
	CategoryReceived
	TagsReceived
	Done
)

type CommandStateHandler func(ctx context.Context, state *CommandState, update *tgbotapi.Update) (string, error)

var (
	commandStateHandler = map[string]CommandStateHandler{
		"r": commandRequestIndexHandler,
	}
)

func commandRequestIndexHandler(ctx context.Context, state *CommandState, update *tgbotapi.Update) (string, error) {

	switch state.RequestIndexState.Next {
	case GroupLinkReceived:
		groupLink := update.Message.Text
		if !strings.HasPrefix(groupLink, "https://t.me/") && !strings.HasPrefix(groupLink, "t.me/") {
			return "", fmt.Errorf("Invalid group link, the link must start with https://t.me/ or at least t.me/")
		}
		// extract the group username
		groupUsername := strings.TrimSpace(groupLink[strings.Index(groupLink, "t.me/")+5:])
		// check username validity, telegram allows only letters, numbers and underscore characters in username
		pattern := regexp.MustCompile(`^[a-zA-Z]+[0-9_a-zA-Z]+$`)
		if !pattern.MatchString(groupUsername) {
			return "", fmt.Errorf("group username in the link invalid, must start with letters and contain only letters, numbers and underscore")
		}
		// query group info
		chat, err := bot.GetChat(tgbotapi.ChatConfig{
			SuperGroupUsername: "@" + groupUsername, // must be proceeded with @, refer to: https://core.telegram.org/bots/api#getchat
		})
		if err != nil {
			log.Printf("getChat for %s error: %v\n", groupUsername, err)
			return "", fmt.Errorf("can't find the group you provided, please check your group username")
		}
		state.RequestIndexState.Group.Chat = chat
		state.RequestIndexState.Next = CategoryReceived

		log.Printf("Group info:\nID: %v\nname: %s\ntype: %s\ndescription: %s\n",
			chat.ID, chat.Title, chat.Type, chat.Description)
		return "please input your group category", nil
	case CategoryReceived:
		// TODO do some validation of the category
		// TODO provide a virtual keyword to let the user choose the category
		category := update.Message.Text
		state.RequestIndexState.Group.Category = category
		state.RequestIndexState.Next = TagsReceived
		return "please input your group tags, separated by space", nil
	case TagsReceived:
		// TODO do some validation of the category
		// TODO write the group into dynamoDB
		// TODO clear the state machine
		tags := strings.Fields(update.Message.Text)
		state.RequestIndexState.Group.Tags = tags

		group := state.RequestIndexState.Group
		chat := group.Chat

		_, err := dynsvc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: aws.String("groups"),
			Key: map[string]types.AttributeValue{
				"username": &types.AttributeValueMemberS{Value: chat.UserName},
			},
			UpdateExpression: aws.String("set title = :title, description = :desc, chat_id = :chat_id, update_at = :update_at, created_at = if_not_exists(created_at, :created_at)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":title":      &types.AttributeValueMemberS{Value: chat.Title},
				":desc":       &types.AttributeValueMemberS{Value: chat.Description},
				":chat_id":    &types.AttributeValueMemberN{Value: strconv.FormatInt(chat.ID, 10)},
				":created_at": &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
				":update_at":  &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
			},
		})

		if err != nil {
			log.Printf("index %s error: %v\n", chat.UserName, err)
			return "index failed, please try again later", nil
		}

		clearState(ctx, state.ChatID)

		return fmt.Sprintf("Group %s has been indexed", state.RequestIndexState.Group.Chat.Title), nil

	default:
		return "", nil
	}
}
