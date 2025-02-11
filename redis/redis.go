package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/GetStream/stream-backend-homework-assignment/api"
	"github.com/redis/go-redis/v9"
)

// Redis provides caching in Redis.
type Redis struct {
	cli *redis.Client
}

// Connect connects to the Redis server and pings the server to ensure the
// connection is working.
func Connect(ctx context.Context, addr string) (*Redis, error) {
	cli := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	if err := cli.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &Redis{
		cli: cli,
	}, nil
}

const (
	messagePrefix = "messages"
	maxSize       = 10
)

// ListMessages returns a list of message from Redis. The messages are sorted
// by the timestamp in descending order.
func (r *Redis) ListMessages(ctx context.Context) ([]api.Message, error) {
	vals, err := r.cli.ZRevRangeByScore(ctx, messagePrefix, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", time.Now().UnixNano()),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("zrange: %w", err)
	}

	out := make([]api.Message, len(vals))
	for i, key := range vals {
		var msg message
		err = r.cli.HGetAll(ctx, key).Scan(&msg)
		if err != nil {
			return nil, fmt.Errorf("hgetall: %w", err)
		}

		reactions, err := r.ListReactions(ctx, msg.ID)
		if err != nil {
			return nil, fmt.Errorf("list reactions: %w", err)
		}

		msg.Reactions = reactions
		out[i] = msg.APIMessage()
	}

	return out, nil
}

// InsertMessage adds the message to Redis with the message:MESSAGE_ID as the key and adds the key to a sorted set.
func (r *Redis) InsertMessage(ctx context.Context, msg api.Message) error {
	m := &message{
		ID:        msg.ID,
		Text:      msg.Text,
		UserID:    msg.UserID,
		CreatedAt: msg.CreatedAt,
	}

	err := r.cli.Watch(ctx, func(tx *redis.Tx) error {
		_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			key := fmt.Sprintf("%s:%s", messagePrefix, m.ID)
			pipe.HSet(ctx, key, m)
			pipe.ZAdd(ctx, messagePrefix, redis.Z{
				Score:  float64(msg.CreatedAt.UnixNano()),
				Member: key,
			})

			return nil
		})
		return err
	}, m.ID)

	if err != nil {
		return fmt.Errorf("redis insert message: %w", err)
	}

	// Simulate an eviction strategy by removing the oldest key in case the max cache size is exceeded.
	err = r.evictOldest(ctx)
	if err != nil {
		return fmt.Errorf("evict oldest: %w", err)
	}
	return nil
}

// ListReactions fetches all reactions associated with a given message ID.
func (r *Redis) ListReactions(ctx context.Context, msgId string) ([]reaction, error) {
	key := fmt.Sprintf("%s:%s:reactions", messagePrefix, msgId)
	vals, err := r.cli.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", time.Now().UnixNano()),
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("zrange: %w", err)
	}
	out := make([]reaction, len(vals))
	for i, key := range vals {
		var rc reaction
		err := r.cli.HGetAll(ctx, key).Scan(&rc)
		if err != nil {
			return nil, fmt.Errorf("hgetall: %w", err)
		}

		out[i] = rc
	}

	return out, nil
}

// InsertReaction adds a reaction to the specified message in Redis identified by msgId.
func (r *Redis) InsertReaction(ctx context.Context, msgId string, mr api.Reaction) error {
	reaction_ := &reaction{
		ID:        mr.ID,
		MessageID: mr.MessageID,
		UserID:    mr.UserID,
		Type:      mr.Type,
		Score:     mr.Score,
	}

	err := r.cli.Watch(ctx, func(tx *redis.Tx) error {
		_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			keyPrefix := fmt.Sprintf("%s:%s:reactions", messagePrefix, msgId)
			key := fmt.Sprintf("%s:%s", keyPrefix, mr.ID)
			pipe.HSet(ctx, key, reaction_)

			pipe.ZAdd(ctx, keyPrefix, redis.Z{
				Score:  float64(mr.CreatedAt.UnixNano()),
				Member: key,
			})
			return nil
		})

		return err
	}, mr.ID)

	if err != nil {
		return fmt.Errorf("could not insert reaction: %w", err)
	}

	return nil
}

func (r *Redis) evictOldest(ctx context.Context) error {
	vals, err := r.cli.ZRange(ctx, messagePrefix, 0, int64(-maxSize-1)).Result()
	if err != nil {
		return fmt.Errorf("zrevrange: %w", err)
	}

	for _, key := range vals {
		_ = r.cli.ZRem(ctx, messagePrefix, key).Err()
		_ = r.cli.Del(ctx, key).Err()
		_ = r.cli.Del(ctx, fmt.Sprintf("%s:reactions", key)).Err()
	}

	return nil
}
