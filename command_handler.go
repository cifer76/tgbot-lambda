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

type CommandHandler func(ctx context.Context, update *tgbotapi.Update, state *CommandState)

const (
	// state machine stages

	// general
	CommandReceived = "CommandReceived"
	Done            = "Done"

	// index command specific
	GroupLinkReceived  = "GroupLinkReceived"
	GroupTopicReceived = "GroupTopicReceived"
	GroupTagsReceived  = "GroupTagsReceived"
)

var (
	patternGroupUsername *regexp.Regexp // group username must be only letters, numbers and underscore
	patternGroupTag      *regexp.Regexp // group tag can be CJK characters and english letters
	patternGroupCategory *regexp.Regexp // group category can be CJK characters and english letters

	categoryKeyboardCN = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💻 编程", "Programming"),
			tgbotapi.NewInlineKeyboardButtonData("⚖️  政治", "Politics"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 经济金融", "Economics"),
			tgbotapi.NewInlineKeyboardButtonData("🖥 科技", "Technology"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("₿ 加密货币", "Cryptocurrencies"),
			tgbotapi.NewInlineKeyboardButtonData("⛓️ 区块链", "Blockchain"),
		),
	)

	categoryKeyboardEN = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💻 Programming", "Programming"),
			tgbotapi.NewInlineKeyboardButtonData("⚖️  Politics", "Politics"),
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

func indexCommandHandler(ctx context.Context, update *tgbotapi.Update, s *CommandState) {
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

		if s.Stage == Done {
			clearState(s.ChatID)
		} else {
			writeState(s)
		}
	}()

	switch s.Stage {
	case CommandReceived:
		s.Stage = GroupLinkReceived
		content = getLocalizedText(ctx, InputGroupLink)
	case GroupLinkReceived:
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
		s.Chat = chat
		s.Stage = GroupTagsReceived
		content = getLocalizedText(ctx, TagsInputting)

		log.Printf("Group info:\nID: %v\nname: %s\ntype: %s\ndescription: %s\n",
			chat.ID, chat.Title, chat.Type, chat.Description)
	case GroupTagsReceived:
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

		s.Tags = tags

		// write group info
		updatedOldValues, err := dynsvc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: aws.String("groups"),
			Key: map[string]types.AttributeValue{
				"username": &types.AttributeValueMemberS{Value: s.UserName},
			},
			ReturnValues:     types.ReturnValueUpdatedOld,
			UpdateExpression: aws.String("set title = :title, #t = :type, description = :desc, chat_id = :chat_id, category = :category, tags = :tags, update_at = :update_at, created_at = if_not_exists(created_at, :created_at)"),
			ExpressionAttributeNames: map[string]string{
				"#t": "type",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":title":      &types.AttributeValueMemberS{Value: s.Title},
				":type":       &types.AttributeValueMemberS{Value: s.Type},
				":desc":       &types.AttributeValueMemberS{Value: s.Description},
				":chat_id":    &types.AttributeValueMemberN{Value: strconv.FormatInt(s.ID, 10)},
				":created_at": &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
				":update_at":  &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
				":category":   &types.AttributeValueMemberS{Value: s.Category},
				":tags":       &types.AttributeValueMemberSS{Value: s.Tags},
			},
		})
		if err != nil {
			log.Printf("index %s error: %v\n", s.UserName, err)
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
		for _, t := range s.Tags {
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
						":group": &types.AttributeValueMemberSS{Value: []string{s.UserName}},
					},
				})
				if err != nil {
					log.Printf("add tag index %s:%s error: %v\n", tag, s.UserName, err)
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
						":group": &types.AttributeValueMemberSS{Value: []string{s.UserName}},
					},
				})
				if err != nil {
					log.Printf("delete tags index %s:%s error: %v\n", tag, s.UserName, err)
				}
			}(t)
		}
		wg.Wait()

		s.Stage = Done
		content = fmt.Sprintf(getLocalizedText(ctx, IndexSuccess), s.Title)
	default:
	}
}

func startCommandHandler(ctx context.Context, update *tgbotapi.Update, s *CommandState) {
	// get user data from
	var chatID int64
	var content string

	if update.Message != nil {
		chatID = update.Message.Chat.ID
	}

	content = getStartContent(ctx)
	msg := tgbotapi.NewMessage(chatID, content)
	_, err := bot.Send(msg)
	if err != nil {
		log.Println(err)
	}

	clearState(s.ChatID)

}

func getCommandHandler(command string) CommandHandler {

	switch command {
	case "index":
		return indexCommandHandler
	default:
		return startCommandHandler
	}
}

func init() {
	patternGroupUsername = regexp.MustCompile("^[a-zA-Z]+[0-9_a-zA-Z]+$")
	patternGroupTag = regexp.MustCompile("^[\u4e00-\u9fa5a-zA-Z0-9]+$")
	patternGroupCategory = regexp.MustCompile("^[\u4e00-\u9fa5a-zA-Z0-9]+$")
}
