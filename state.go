package main

import (
	"context"
	"log"
	"strconv"
	"time"

	cache "github.com/patrickmn/go-cache"
)

const (
	stateKey       = "state:"
	expireDuration = 5 * time.Minute
)

var (
	mcache = cache.New(5*time.Minute, 10*time.Minute)
)

func getState(ctx context.Context, chatID int64) *CommandState {
	key := stateKey + strconv.FormatInt(chatID, 10)

	if x, found := mcache.Get(key); found {
		state := x.(*CommandState)
		log.Printf("getState, chatID: %v, command: %v, stage: %v\n", state.ChatID, state.Command, state.Stage)
		return state
	}

	return nil
}

func writeState(state *CommandState) {
	key := stateKey + strconv.FormatInt(state.ChatID, 10)
	log.Printf("writeState, chatID: %v, command: %v, stage: %v\n", state.ChatID, state.Command, state.Stage)
	mcache.Set(key, state, cache.DefaultExpiration)
}

func clearState(chatID int64) {
	key := stateKey + strconv.FormatInt(chatID, 10)
	log.Printf("clearState, chatID: %v\n", key)
	mcache.Delete(key)
}
