package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	opensearch "github.com/opensearch-project/opensearch-go"
)

var bot *tgbotapi.BotAPI

func main() {
	// initialize tgbot
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatalln("environment BOT_TOKEN empty!")
	}
	botDebug := os.Getenv("BOT_DEBUG")

	var err error
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = botDebug == "true"

	// initialize opensearch client
	server := os.Getenv("OPENSEARCH_SERVER")
	if server == "" {
		log.Fatalln("environment OPENSEARCH_SERVER empty!")
	}

	// Initialize the client with SSL/TLS enabled.
	opensvc, err = opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Addresses: []string{server},
	})
	if err != nil {
		log.Panic("cannot initialize", err)
	}

	groups := []string{
		"tkt02",
		"expressnode",
		"cryptomiao",
		"fetchai_cn",
		"rnb_jb",
		"chinese_characters",
		"gansutg",
		"mklennongroup",
		"fedora_tw",
		"hxcloud",
		"xdosc",
		"bocai188",
		"kaichequn88",
		"lavatech",
		"machixofficial",
		"karochi_kelishurchi",
		"hkparliament",
		"testdasts",
		"cndogshome",
		"acg18moe",
		"antigoogle",
		"nohkip",
		"pcredivezh",
		"mycosschinese",
		"xianyitw8",
		"brothersisterfacebookclub",
		"grabluckychinese",
		"asiaedx",
		"letstalkinsql",
		"beehive_game_china",
		"japaniscool",
		"eospowcommunity",
		"singbady",
		"masschinese",
		"kickico_china",
		"bocaisj",
		"ltonetwork_cn",
		"happyjobhk",
		"macauguy",
		"brothersisterhk_yautsimmong",
		"eosjackscn",
		"dsechihis",
		"populstaycn",
		"giftochinese",
		"greenlightusa",
		"cloudcoin_china",
		"swagliver",
		"nutopiaio_cn",
		"t1b2h3t4k5",
		"kanaheiforest",
		"mithrilchat_zh",
		"amtbtn_tg_room",
		"dsels",
		"hkaa1",
		"movehk",
		"linfinitycn",
		"shadiaofriend",
		"abccexofficial_cn",
		"kb_tgdc5",
		"tiyachan",
		"fflfanfenlie",
		"teletowers",
		"eosfans",
		"ultrain_cn",
		"sm520",
		"hz_tc",
		"peacefulhk",
		"tgfxzk2",
		"hrzyq",
		"money4uoffical",
		"hkpromotion",
		"fairgameonline",
		"lifeintaiwan",
		"bandprotocolcn",
		"coinseason",
		"blockchainshanghai",
		"freechatclub",
		"guowengui17",
		"main2amoi",
		"chromawayofficial",
		"arcs_arx_cn_tw",
		"lanews2020",
		"btmfans",
		"canadatraders",
		"talknchat",
		"giftwgroup",
		"windows2okey",
		"gjcomex",
		"sw728",
		"molofficial2",
		"mimemi_cloud",
		"log_trans_hk",
		"nb250",
		"afanti001",
		"cuhkprotestorsgroup",
		"teahouse2nd",
		"greenplanetearth",
		"fangong",
		"cateringindustrystrike",
		"taipodistrictcouncilfrontline",
		"crypto_raiders_china",
		"shatinindependent",
		"neimenggutg",
		"nekopen",
		"discusscrossgfw",
		"freeislandeast",
		"coinboba",
		"antitotalism929",
		"porn580",
		"ballballgame",
		"wfpornbot",
		"haiwaipin888",
		"stickergroup",
		"zh_hookup",
		"xxdeep",
		"flbwm",
		"yuanma_66",
		"tgcar_ph",
		"jimmyguide",
		"zhenzhutiaocao",
		"bcpay",
	}

	for _, g := range groups {
		s, memberCount, err := getGroupInfo(context.Background(), g)
		if err != nil {
			continue
		}
		fmt.Printf("New group, ID: %v, name: %s, type: %s, memberCount: %d, description: %s\n", s.ID, s.Title, s.Type, memberCount, s.Description)
		opensearchWriteGroup(context.Background(), GroupRecord{
			Username:    s.UserName,
			ChatID:      s.ID,
			Title:       s.Title,
			Type:        s.Type,
			Description: s.Description,
			MemberCount: memberCount,
		})
		time.Sleep(5 * time.Second)
	}
}
