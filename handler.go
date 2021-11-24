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
		rsp = "An error occurred, empty result"
		return
	}

	recs := []TagRecord{}
	err = attributevalue.UnmarshalListOfMaps(out.Responses["tags"], &recs)
	if err != nil {
		log.Printf("error during unmarshal tags, err %v\n", err)
		rsp = "An error occurred, empty result"
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
		rsp = "no results found"
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
		rsp = "An error occurred, empty result"
		return
	}

	groups := []GroupRecord{}
	err = attributevalue.UnmarshalListOfMaps(out.Responses["groups"], &groups)
	if err != nil {
		log.Printf("error during unmarshal groups, err %v\n", err)
		rsp = "An error occurred, empty result"
		return
	}

	rsp = `
Found the following results:

`

	for i, g := range groups {
		icon := "👥"
		if g.Type == "channel" {
			icon = "📢"
		}
		line := fmt.Sprintf("%d. %s <a href=\"https://t.me/%s\">%s</a> <pre>%s</pre>\n", i+1, icon, g.Username, g.Title, g.Description)
		rsp += line
	}
	return
}
