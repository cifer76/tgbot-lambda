package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

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

func addCommandHandler(ctx context.Context, update *tgbotapi.Update, s *CommandState) {
	// get user data from
	var content string

	chatID := getChatIDFromUpdate(update)
	message := getChatMessageFromUpdate(update)

	defer func() {
		msg := tgbotapi.NewMessage(chatID, content)
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
		groupUsername := message
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
		content = getLocalizedText(ctx, InputTags)
		log.Printf("Group info:\nID: %v\nname: %s\ntype: %s\ndescription: %s\n",
			chat.ID, chat.Title, chat.Type, chat.Description)
	case GroupTagsReceived:
		tags := strings.Fields(message)
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

		go ddbWriteGroup(ctx, s)

		s.Stage = Done
		content = fmt.Sprintf(getLocalizedText(ctx, IndexSuccess), s.Title, s.Description, tags, time.Now().Format("2006/01/02 15:04:05"))
	default:
	}
}

func startCommandHandler(ctx context.Context, update *tgbotapi.Update, s *CommandState) {
	if update.Message == nil {
		// invalid update message, just ignore it
		return
	}

	chatID := update.Message.Chat.ID
	content := getStartContent(ctx)
	_, err := bot.Send(tgbotapi.NewMessage(chatID, content))
	if err != nil {
		log.Println(err)
	}

	clearState(s.ChatID)
}

func getCommandHandler(command string) CommandHandler {
	switch command {
	case "add":
		return addCommandHandler
	default:
		return startCommandHandler
	}
}

func init() {
	patternGroupUsername = regexp.MustCompile("^[a-zA-Z]+[0-9_a-zA-Z]+$")
	patternGroupTag = regexp.MustCompile("^[\u4e00-\u9fa5a-zA-Z0-9]+$")
	patternGroupCategory = regexp.MustCompile("^[\u4e00-\u9fa5a-zA-Z0-9]+$")
}
