// data structures related with storing in dynamodb
package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
	Username    string   `json:"username"`
	ChatID      int64    `json:"chat_id" dynamodbav:"chat_id"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	MemberCount int      `json:"member_count" dynamodbav:"member_count"`
	Category    string   `json:"category"` // group category specified by the requestor
	Tags        []string `json:"tags"`     // group tags specified by the requestor
}

// GroupRecords implements sort.Interface based on the MemberCount field.
type GroupRecords []GroupRecord

func (a GroupRecords) Len() int           { return len(a) }
func (a GroupRecords) Less(i, j int) bool { return a[i].MemberCount > a[j].MemberCount }
func (a GroupRecords) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// Tag Record
type TagRecord struct {
	Tag    string
	Groups []string
}

// User Record
type UserRecord struct {
	ID           int64
	Username     string
	FirstName    string
	LastName     string
	LanguageCode string
}
