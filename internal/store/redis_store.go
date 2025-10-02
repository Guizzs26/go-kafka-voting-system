package store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/model"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(ctx context.Context, addr string) (*RedisStore, error) {
	opts, err := redis.ParseURL(addr)
	if err != nil {
		return nil, fmt.Errorf("error parsing redis URL: %v", err)
	}

	c := redis.NewClient(opts)

	if err := c.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("error connecting to redis: %v", err)
	}

	return &RedisStore{client: c}, nil
}

func (rs *RedisStore) RegisterVote(ctx context.Context, vote model.Vote) (bool, error) {
	vkey := fmt.Sprintf("poll:%s:votes", vote.PollID)
	rkey := fmt.Sprintf("poll:%s:results", vote.PollID)

	pipe := rs.client.Pipeline()
	saddR := pipe.SAdd(ctx, vkey, vote.UserID)
	pipe.HIncrBy(ctx, rkey, vote.OptionID, 1)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("error executing redis pipeline: %v", err)
	}

	r, err := saddR.Result()
	if err != nil {
		return false, fmt.Errorf("error getting SAdd result: %v", err)
	}

	isNewVote := r > 0

	return isNewVote, nil
}

func (rs *RedisStore) GetResults(ctx context.Context, pollID string) (map[string]int, error) {
	vkey := fmt.Sprintf("poll:%s:results", pollID)

	rstr, err := rs.client.HGetAll(ctx, vkey).Result()
	if err != nil {
		return nil, fmt.Errorf("error getting results from redis: %v", err)
	}

	result := make(map[string]int, len(rstr))
	for optionID, countStr := range rstr {
		count, err := strconv.Atoi(countStr)
		if err != nil {
			return nil, fmt.Errorf("error converting count to int: %v", err)
		}
		result[optionID] = count
	}

	return result, nil
}

func (rs *RedisStore) Close() error {
	if err := rs.client.Close(); err != nil {
		return fmt.Errorf("error closing redis client: %v", err)
	}
	return nil
}
