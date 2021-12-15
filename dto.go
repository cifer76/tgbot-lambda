package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type GroupInfo struct {
	tgbotapi.Chat

	MemberCount int      `json:"memberCount"`
	Category    string   `json:"category"` // group category specified by the requestor
	Tags        []string `json:"tags"`     // group tags specified by the requestor
}

// state
type CommandState struct {
	GroupInfo // the basic info of the group being requested index, used in the /request command

	ChatID  int64  `json:"chatID"` // who is initiating the command?
	Command string `json:"command"`
	Stage   string `json:"stage"` // the current stage
}

// Group Record
type GroupRecord struct {
	Username    string
	ChatID      int64
	Title       string
	Type        string
	Description string
	MemberCount int
	Category    string   `json:"category"` // group category specified by the requestor
	Tags        []string `json:"tags"`     // group tags specified by the requestor
}

// Tag Record
type TagRecord struct {
	Tag    string
	Groups []string
}
