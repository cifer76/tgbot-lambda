package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/cifer76/gojieba"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jdkato/prose/v2"
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

	jieba *gojieba.Jieba

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

func getCheckGroupUsername(userInput string) string {
	groupUsername := userInput
	if strings.HasPrefix(groupUsername, "https://t.me/") || strings.HasPrefix(groupUsername, "t.me/") {
		// extract the group username
		groupUsername = strings.TrimSpace(groupUsername[strings.Index(groupUsername, "t.me/")+5:])
	}

	// check username validity, telegram allows only letters, numbers and underscore characters in username
	if !patternGroupUsername.MatchString(groupUsername) {
		return ""
	}

	return groupUsername
}

func getGroupInfo(ctx context.Context, groupUsername string) (tgbotapi.Chat, int, error) {
	chatConfig := tgbotapi.ChatConfig{
		// must be proceeded with @, refer to: https://core.telegram.org/bots/api#getchat
		SuperGroupUsername: "@" + groupUsername,
	}

	// query group info
	chat, err := bot.GetChat(tgbotapi.ChatInfoConfig{
		ChatConfig: chatConfig,
	})
	if err != nil {
		log.Printf("getChat for %s error: %v\n", groupUsername, err)
		return chat, 0, fmt.Errorf("GroupNotFound")
	}

	// get chat member count
	count, err := bot.GetChatMembersCount(tgbotapi.ChatMemberCountConfig{
		ChatConfig: chatConfig,
	})
	if err != nil {
		log.Printf("getChatMembersCount for %s error: %v\n", groupUsername, err)
	}

	return chat, count, nil
}

func getGroupTagsCN(ctx context.Context, title, desc string) []string {
	// get tags using gojieba
	tags := jieba.CutForSearch(title, true)
	tags = append(tags, jieba.CutForSearch(desc, true)...)

	// filter out non-noun words
	filtered := []string{}
	for _, t := range tags {
		r := jieba.Tag(t)
		prop := strings.Split(r[0], "/")
		// keep only noun words
		// refer to https://gist.github.com/hscspring/c985355e0814f01437eaf8fd55fd7998
		if strings.HasPrefix(prop[1], "n") || strings.HasPrefix(prop[1], "i") || strings.HasPrefix(prop[1], "q") {
			filtered = append(filtered, t)
		}
	}
	tags = filtered

	// do some validation of the tags
	/*
		filtered = []string{}
		for _, t := range tags {
			// tags less than 2 character and not match regex won't be counted
			if len([]rune(t)) > 1 && patternGroupTag.MatchString(t) {
				filtered = append(filtered, t)
			}
		}
		tags = filtered
	*/

	return tags
}

func getGroupTags(ctx context.Context, title, description string) []string {
	tags := getGroupTagsCN(ctx, title, description)
	// support up to 20 Chinese tags for each group
	if len(tags) > 20 {
		tags = tags[:20]
	}

	tagsEng := getGroupTagsEng(ctx, title, description)
	// support up to 20 Chinese tags for each group
	if len(tagsEng) > 20 {
		tagsEng = tagsEng[:20]
	}

	tags = append(tags, tagsEng...)

	// de-duplication
	dedup := map[string]bool{}
	filtered := []string{}
	for _, t := range tags {
		if !dedup[t] {
			dedup[t] = true
			filtered = append(filtered, t)
		}
	}
	tags = filtered

	return tags
}

func getGroupTagsEng(ctx context.Context, title, description string) []string {
	// get the english words using jieba
	tags := jieba.CutForSearch(title, true)
	tags = append(tags, jieba.CutForSearch(description, true)...)
	// filter out non-english words
	docstring := ""
	for _, t := range tags {
		r := jieba.Tag(t)
		prop := strings.Split(r[0], "/")
		// refer to https://gist.github.com/hscspring/c985355e0814f01437eaf8fd55fd7998
		if prop[1] == "eng" {
			docstring += " "
			docstring += t
		}
	}

	if docstring == "" {
		return []string{}
	}

	// Create a new document with the default configuration:
	doc, err := prose.NewDocument(docstring)
	if err != nil {
		log.Println(err)
		return []string{}
	}

	// Iterate over the doc's tokens:
	tags = []string{}
	for _, tok := range doc.Tokens() {
		fmt.Println(tok)
		if strings.HasPrefix(tok.Tag, "NN") {
			tags = append(tags, tok.Text)
		}
	}

	// Iterate over the doc's named-entities:
	for _, ent := range doc.Entities() {
		tags = append(tags, ent.Text)
		// fmt.Println(ent.Text, ent.Label)
	}

	return tags
}

func addCommandHandler(ctx context.Context, update *tgbotapi.Update, s *CommandState) {
	// get user data from
	var content string

	chatID := getChatIDFromUpdate(update)
	message := getChatMessageFromUpdate(update)

	defer func() {
		msg := tgbotapi.NewMessage(chatID, content)
		msg.DisableWebPagePreview = true
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
		groupUsername := getCheckGroupUsername(message)
		if groupUsername == "" {
			content = getLocalizedText(ctx, UsernameInvalid)
			return
		}

		var err error
		s.Chat, s.MemberCount, err = getGroupInfo(ctx, groupUsername)
		if err != nil {
			content = getLocalizedText(ctx, GroupNotFound)
			return
		}
		log.Printf("Group info:\nID: %v\nname: %s\ntype: %s\nmemberCount: %d\ndescription: %s\n", s.Chat.ID, s.Chat.Title, s.Chat.Type, s.MemberCount, s.Chat.Description)

		s.Tags = getGroupTags(ctx, s.Chat.Title, s.Chat.Description)
		s.Stage = Done

		go ddbWriteGroup(ctx, s.GroupInfo)

		content = fmt.Sprintf(getLocalizedText(ctx, IndexSuccess), s.Title, s.Description, s.Tags, time.Now().Format("2006/01/02 15:04:05"))
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
	patternGroupTag = regexp.MustCompile("^[\u4e00-\u9fa5a-zA-Z0-9._]+$")
	patternGroupCategory = regexp.MustCompile("^[\u4e00-\u9fa5a-zA-Z0-9]+$")
	jieba = gojieba.NewJieba()
}
