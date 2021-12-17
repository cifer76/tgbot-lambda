package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Handler func(ctx context.Context, update *tgbotapi.Update)

func handleSearch(ctx context.Context, update *tgbotapi.Update) {
	tokens := strings.Fields(update.Message.Text)

	var rsp string
	defer func() {
		if rsp == "" {
			return
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, rsp)
		msg.ParseMode = tgbotapi.ModeHTML
		msg.DisableWebPagePreview = true
		_, err := bot.Send(msg)
		if err != nil {
			log.Println(err)
		}
	}()

	keywords := []string{}
	for _, t := range tokens {
		if patternGroupTag.MatchString(t) {
			keywords = append(keywords, t)
		}
	}

	keys := []map[string]types.AttributeValue{}
	for _, k := range keywords {
		keys = append(keys, map[string]types.AttributeValue{
			"tag": &types.AttributeValueMemberS{Value: k},
		})
	}

	// get group usernames
	out, err := dynsvc.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			"tags": {
				Keys: keys,
			},
		},
	})
	if err != nil {
		log.Printf("error during batch get tags, err %v\n", err)
		return
	}

	recs := []TagRecord{}
	err = attributevalue.UnmarshalListOfMaps(out.Responses["tags"], &recs)
	if err != nil {
		log.Printf("error during unmarshal tags, err %v\n", err)
		return
	}

	// sort by how many tags the group matches
	type kv struct {
		k string
		v int
	}
	ss := []kv{}
	check := map[string]int{}
	for _, r := range recs {
		for _, g := range r.Groups {
			check[g]++
		}
	}
	for name, count := range check {
		ss = append(ss, kv{k: name, v: count})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].v > ss[j].v
	})
	keys = []map[string]types.AttributeValue{}
	for _, kv := range ss {
		keys = append(keys, map[string]types.AttributeValue{
			"username": &types.AttributeValueMemberS{Value: kv.k},
		})
	}

	if len(keys) < 1 {
		return
	}

	// batch get groups by group usernames
	out, err = dynsvc.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			"groups": {
				Keys: keys,
			},
		},
	})
	if err != nil {
		log.Printf("error during batch get groups, err %v\n", err)
		return
	}

	groups := []GroupRecord{}
	err = attributevalue.UnmarshalListOfMaps(out.Responses["groups"], &groups)
	if err != nil {
		log.Printf("error during unmarshal groups, err %v\n", err)
		return
	}

	rsp = `
找到如下结果:

`

	for i, g := range groups {
		icon := "👥"
		if g.Type == "channel" {
			icon = "📢"
		}

		line := fmt.Sprintf("%d. %s <a href=\"https://t.me/%s\">%s</a> (%s)\n", i+1, icon, g.Username, g.Title, formatMemberCount(g.MemberCount))
		rsp += line
	}
	return
}

func handleUpdate(ctx context.Context, update tgbotapi.Update) {
	log.Printf("TG Update: %+v\n", update)

	var chatID int64
	if chatID = getChatIDFromUpdate(&update); chatID == 0 {
		log.Println("not chatID found, unsupported update type")
		return
	}

	s := getState(ctx, chatID)
	if updateIsCommand(&update) {
		// 1. in reality, it isn't necessary for every command to have a state machine
		//    but we take it as so, it makes our code simple and consistent
		// 2. any command interrupts an another command's state machine

		s = &CommandState{
			ChatID:  chatID,
			Command: update.Message.Command(),
			Stage:   CommandReceived,
		}
		writeState(s)
	}

	if s != nil {
		h := getCommandHandler(s.Command)
		h(ctx, &update, s)
		return
	}

	// not command and no state, then it's the simplest case: keyword search
	if update.Message != nil && update.Message.Text != "" {
		handleSearch(ctx, &update)
	} else {
		fmt.Println("unsupported update")
	}
}
