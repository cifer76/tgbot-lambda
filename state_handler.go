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

type StateHandler func(ctx context.Context, update *tgbotapi.Update, state *CommandState) (string, error)

const (
	CommandReceived int = iota
	GroupLinkReceived
	CategoryReceived
	TagsReceived
	Done
)

var (
	stateHandler = map[string]StateHandler{
		"request": requestIndexStateHandler,
	}

	patternGroupUsername *regexp.Regexp // group username must be only letters, numbers and underscore
	patternGroupTag      *regexp.Regexp // group tag can be CJK characters and english letters
	patternGroupCategory *regexp.Regexp // group category can be CJK characters and english letters
)

func init() {
	patternGroupUsername = regexp.MustCompile("^[a-zA-Z]+[0-9_a-zA-Z]+$")
	patternGroupTag = regexp.MustCompile("^[\u4e00-\u9fa5a-zA-Z0-9]+$")
	patternGroupCategory = regexp.MustCompile("^[\u4e00-\u9fa5a-zA-Z0-9]+$")
}

func requestIndexStateHandler(ctx context.Context, update *tgbotapi.Update, cs *CommandState) (string, error) {
	defer func() {
		if cs.Next == Done {
			clearState(ctx, cs.ChatID)
		} else {
			writeState(ctx, cs)
		}
	}()

	switch cs.Next {
	case GroupLinkReceived:
		groupLink := update.Message.Text
		if !strings.HasPrefix(groupLink, "https://t.me/") && !strings.HasPrefix(groupLink, "t.me/") {
			return "", fmt.Errorf("Invalid group link, the link must start with https://t.me/ or at least t.me/")
		}
		// extract the group username
		groupUsername := strings.TrimSpace(groupLink[strings.Index(groupLink, "t.me/")+5:])
		// check username validity, telegram allows only letters, numbers and underscore characters in username
		if !patternGroupUsername.MatchString(groupUsername) {
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
		cs.Chat = chat
		cs.Next = CategoryReceived

		log.Printf("Group info:\nID: %v\nname: %s\ntype: %s\ndescription: %s\n",
			chat.ID, chat.Title, chat.Type, chat.Description)
		return "please input your group category", nil
	case CategoryReceived:
		// TODO provide a virtual keyword to let the user choose the category

		// do some validation of the category
		category := update.Message.Text
		if !patternGroupCategory.MatchString(category) {
			return "category invalid, please re-input", nil
		}

		cs.Category = category
		cs.Next = TagsReceived
		return "please input your group tags, separated by space", nil
	case TagsReceived:
		tags := strings.Fields(update.Message.Text)

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
			UpdateExpression: aws.String("set title = :title, type = :type, description = :desc, chat_id = :chat_id, category = :category, tags = :tags, update_at = :update_at, created_at = if_not_exists(created_at, :created_at)"),
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
			return "index failed, please try again later", nil
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
		cs.Next = Done
		return fmt.Sprintf("Group %s has been indexed", cs.Title), nil
	default:
		return "", nil
	}
}
