package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type StateHandler func(ctx context.Context, update *tgbotapi.Update, state *CommandState)

const (
	// state machine stages

	// general
	CommandReceived = "CommandReceived"
	Done            = "Done"

	// index command specific
	GroupLinkReceived  = "GroupLinkReceived"
	GroupTopicReceived = "GroupTopicReceived"

	// TODO list group command specific
)

var (
	stateHandler = map[string]StateHandler{
		"index": indexStateHandler,
	}

	patternGroupUsername *regexp.Regexp // group username must be only letters, numbers and underscore
	patternGroupTag      *regexp.Regexp // group tag can be CJK characters and english letters
	patternGroupCategory *regexp.Regexp // group category can be CJK characters and english letters

	categoryKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💻 Programming", "Programming"),
			tgbotapi.NewInlineKeyboardButtonData("⚖️ Politics", "Politics"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Economics", "Economics"),
			tgbotapi.NewInlineKeyboardButtonData("🖥 Technology", "Technology"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("₿ Cryptocurrencies", "Cryptocurrencies"),
			tgbotapi.NewInlineKeyboardButtonData("⛓️ Blockchain", "Blockchain"),
		),
	)
)

func init() {
	patternGroupUsername = regexp.MustCompile("^[a-zA-Z]+[0-9_a-zA-Z]+$")
	patternGroupTag = regexp.MustCompile("^[\u4e00-\u9fa5a-zA-Z0-9]+$")
	patternGroupCategory = regexp.MustCompile("^[\u4e00-\u9fa5a-zA-Z0-9]+$")
}

func indexStateHandler(ctx context.Context, update *tgbotapi.Update, cs *CommandState) {
	// get user data from
	var userInput string
	var chatID int64
	var content string
	if update.Message != nil {
		userInput = update.Message.Text
		chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		userInput = update.CallbackQuery.Data
		chatID = update.CallbackQuery.Message.Chat.ID
	} else {
		content = "unsupported input"
		return
	}

	msg := tgbotapi.NewMessage(chatID, "")
	defer func() {
		msg.Text = content

		_, err := bot.Send(msg)
		if err != nil {
			log.Println(err)
		}

		if cs.Stage == Done {
			clearState(ctx, cs.ChatID)
		} else {
			writeState(ctx, cs)
		}
	}()

	switch cs.Stage {
	case CommandReceived:
		groupUsername := userInput
		if strings.HasPrefix(groupUsername, "https://t.me/") || strings.HasPrefix(groupUsername, "t.me/") {
			// extract the group username
			groupUsername = strings.TrimSpace(groupUsername[strings.Index(groupUsername, "t.me/")+5:])
			// check username validity, telegram allows only letters, numbers and underscore characters in username
			if !patternGroupUsername.MatchString(groupUsername) {
				content = getLocalizedText(ctx, UsernameInvalid)
				return
			}
		}
		// query group info
		chat, err := bot.GetChat(tgbotapi.ChatConfig{
			SuperGroupUsername: "@" + groupUsername, // must be proceeded with @, refer to: https://core.telegram.org/bots/api#getchat
		})
		if err != nil {
			log.Printf("getChat for %s error: %v\n", groupUsername, err)
			content = getLocalizedText(ctx, GroupNotFound)
			return
		}
		cs.Chat = chat
		cs.Stage = GroupLinkReceived

		log.Printf("Group info:\nID: %v\nname: %s\ntype: %s\ndescription: %s\n",
			chat.ID, chat.Title, chat.Type, chat.Description)

		content = getLocalizedText(ctx, TopicChoosing)
		msg.ReplyMarkup = categoryKeyboard

	case GroupLinkReceived:
		// do some validation of the category
		topic := userInput
		if !patternGroupCategory.MatchString(topic) {
			content = getLocalizedText(ctx, TopicInvalid)
			return
		}

		content = getLocalizedText(ctx, TagsInputting)
		cs.Category = topic
		cs.Stage = GroupTopicReceived
	case GroupTopicReceived:
		tags := strings.Fields(userInput)

		// do some validation of the tags
		filtered := []string{}
		for _, t := range tags {
			if patternGroupTag.MatchString(t) {
				filtered = append(filtered, t)
			}
		}
		tags = filtered

		// support up to 3 tags for each group
		if len(tags) > 3 {
			tags = tags[:3]
		}

		cs.Tags = tags

		// write group info
		updatedOldValues, err := dynsvc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: aws.String("groups"),
			Key: map[string]types.AttributeValue{
				"username": &types.AttributeValueMemberS{Value: cs.UserName},
			},
			ReturnValues:     types.ReturnValueUpdatedOld,
			UpdateExpression: aws.String("set title = :title, #t = :type, description = :desc, chat_id = :chat_id, category = :category, tags = :tags, update_at = :update_at, created_at = if_not_exists(created_at, :created_at)"),
			ExpressionAttributeNames: map[string]string{
				"#t": "type",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":title":      &types.AttributeValueMemberS{Value: cs.Title},
				":type":       &types.AttributeValueMemberS{Value: cs.Type},
				":desc":       &types.AttributeValueMemberS{Value: cs.Description},
				":chat_id":    &types.AttributeValueMemberN{Value: strconv.FormatInt(cs.ID, 10)},
				":created_at": &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
				":update_at":  &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
				":category":   &types.AttributeValueMemberS{Value: cs.Category},
				":tags":       &types.AttributeValueMemberSS{Value: cs.Tags},
			},
		})
		if err != nil {
			log.Printf("index %s error: %v\n", cs.UserName, err)
			content = getLocalizedText(ctx, IndexFailed)
			return
		}

		// Write tags(for search) info
		// In this step, we move the group from the old tags indexes to the new tags'
		oldTags := []string{}
		checkOld := map[string]bool{}
		toDelete := []string{}
		toAdd := []string{}
		_ = attributevalue.Unmarshal(updatedOldValues.Attributes["tags"], oldTags)
		for _, o := range oldTags {
			checkOld[o] = true
		}
		for _, t := range cs.Tags {
			if _, ok := checkOld[t]; !ok { // if the new tag doesn't overlap with old tags, we need to index the group use the new tag
				toAdd = append(toAdd, t)
			} else { // if overlapped, mark the old tag to false for reservation
				checkOld[t] = false
			}
		}
		for t, v := range checkOld {
			if v == false { // we have marked the overlapped tags to false
				continue
			}
			toDelete = append(toDelete, t)
		}

		var wg sync.WaitGroup
		for _, t := range toAdd {
			wg.Add(1)
			go func(tag string) {
				defer wg.Done()
				_, err := dynsvc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
					TableName: aws.String("tags"),
					Key: map[string]types.AttributeValue{
						"tag": &types.AttributeValueMemberS{Value: tag},
					},
					UpdateExpression: aws.String("add groups :group"),
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":group": &types.AttributeValueMemberSS{Value: []string{cs.UserName}},
					},
				})
				if err != nil {
					log.Printf("add tag index %s:%s error: %v\n", tag, cs.UserName, err)
				}
			}(t)
		}
		for _, t := range toDelete {
			wg.Add(1)
			go func(tag string) {
				defer wg.Done()
				_, err := dynsvc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
					TableName: aws.String("tags"),
					Key: map[string]types.AttributeValue{
						"tag": &types.AttributeValueMemberS{Value: tag},
					},
					UpdateExpression: aws.String("delete groups :group"),
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":group": &types.AttributeValueMemberSS{Value: []string{cs.UserName}},
					},
				})
				if err != nil {
					log.Printf("delete tags index %s:%s error: %v\n", tag, cs.UserName, err)
				}
			}(t)
		}
		wg.Wait()

		content = fmt.Sprintf(getLocalizedText(ctx, IndexSuccess), cs.Title)

		cs.Stage = Done

	default:
	}
}
