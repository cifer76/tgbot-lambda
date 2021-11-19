package main

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	stateKey       = "state:"
	expireDuration = 5 * time.Minute
)

func getState(ctx context.Context, chatID int64) (*CommandState, error) {
	key := stateKey + strconv.FormatInt(chatID, 10)

	output, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil { // key does not exist
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	log.Printf("getState: %v\n", output)

	state := &CommandState{}
	_ = json.Unmarshal([]byte(output), state)
	return state, nil
}

func writeState(ctx context.Context, state *CommandState) error {
	key := stateKey + strconv.FormatInt(state.ChatID, 10)

	bytes, _ := json.Marshal(state)

	log.Printf("writeState: %v\n", string(bytes))

	_, err := rdb.Set(ctx, key, bytes, expireDuration).Result()
	if err != nil {
		return err
	}

	return nil
}

func clearState(ctx context.Context, chatID int64) error {
	key := stateKey + strconv.FormatInt(chatID, 10)
	log.Printf("clearState: %v\n", key)
	_, err := rdb.Expire(ctx, key, 0).Result()
	if err != nil {
		return err
	}

	return nil
}
