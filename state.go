package main

import (
	"context"
	"encoding/json"
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

	state := &CommandState{}
	_ = json.Unmarshal([]byte(output), state)
	return state, nil
}

func writeState(ctx context.Context, state *CommandState) error {
	key := stateKey + strconv.FormatInt(state.ChatID, 10)

	bytes, _ := json.Marshal(state)
	_, err := rdb.Set(ctx, key, bytes, expireDuration).Result()
	if err != nil {
		return err
	}

	return nil
}

func clearState(ctx context.Context, chatID int64) error {
	key := stateKey + strconv.FormatInt(chatID, 10)
	_, err := rdb.Set(ctx, key, nil, 0).Result()
	if err != nil {
		return err
	}

	return nil
}
