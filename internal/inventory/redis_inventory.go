package inventory

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisInventory uses Redis + Lua script for atomic check-and-decrement.
type RedisInventory struct {
	client *redis.Client
	script *redis.Script
	prefix string
}

// NewRedis creates a Redis-backed inventory. addr should be "host:port".
func NewRedis(ctx context.Context, addr string) (*RedisInventory, error) {
	opt := &redis.Options{Addr: addr}
	client := redis.NewClient(opt)
	// quick ping
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	script := redis.NewScript(RedisLuaScript)
	return &RedisInventory{client: client, script: script, prefix: "stock:"}, nil
}

// Initialize sets the starting stock values in Redis.
func (r *RedisInventory) Initialize(items map[string]int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pipe := r.client.Pipeline()
	for k, v := range items {
		pipe.Set(ctx, r.prefix+k, v, 0)
	}
	_, _ = pipe.Exec(ctx)
}

// CheckAndDecrement runs the Lua script to atomically check and decrement stock.
// It returns (true, remaining) if decremented, or (false, 0) if sold out.
func (r *RedisInventory) CheckAndDecrement(itemID string) (bool, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	key := r.prefix + itemID
	cmd := r.script.Run(ctx, r.client, []string{key})
	n, err := cmd.Int64()
	if err != nil {
		return false, 0
	}
	if n < 0 {
		return false, 0
	}
	return true, int(n)
}
