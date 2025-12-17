package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig controls redis client behavior.
// Keep it config-driven; defaults should be safe and conservative.
type RedisConfig struct {
	Addr string

	// Basic timeouts
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// Pool tuning
	PoolSize           int
	MinIdleConns       int
	PoolTimeout        time.Duration
	ConnMaxIdleTime    time.Duration
	ConnMaxLifetime    time.Duration

	PingTimeout time.Duration
}

func (c RedisConfig) withDefaults() RedisConfig {
	out := c
	if out.DialTimeout <= 0 {
		out.DialTimeout = 3 * time.Second
	}
	if out.ReadTimeout <= 0 {
		out.ReadTimeout = 2 * time.Second
	}
	if out.WriteTimeout <= 0 {
		out.WriteTimeout = 2 * time.Second
	}
	if out.PoolSize <= 0 {
		out.PoolSize = 20
	}
	if out.MinIdleConns < 0 {
		out.MinIdleConns = 0
	}
	if out.PoolTimeout <= 0 {
		out.PoolTimeout = 4 * time.Second
	}
	if out.ConnMaxIdleTime <= 0 {
		out.ConnMaxIdleTime = 5 * time.Minute
	}
	if out.ConnMaxLifetime <= 0 {
		out.ConnMaxLifetime = 30 * time.Minute
	}
	if out.PingTimeout <= 0 {
		out.PingTimeout = 2 * time.Second
	}
	return out
}

// OpenRedis initializes a Redis client and validates connectivity via PING.
func OpenRedis(ctx context.Context, cfg RedisConfig) (*redis.Client, error) {
	cfg = cfg.withDefaults()
	if cfg.Addr == "" {
		return nil, fmt.Errorf("redis addr is required")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:            cfg.Addr,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		PoolTimeout:     cfg.PoolTimeout,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	})

	pingCtx, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return rdb, nil
}

var concurrencyAcquireScript = redis.NewScript(`
-- KEYS[1] = counter key
-- ARGV[1] = limit (int)
-- ARGV[2] = ttl_ms (int)
--
-- Returns:
--  1 if acquired
--  0 if rejected (limit reached)
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[2])
else
  -- Ensure TTL exists even if key already existed without TTL
  if redis.call('PTTL', KEYS[1]) < 0 then
    redis.call('PEXPIRE', KEYS[1], ARGV[2])
  end
end

if current > tonumber(ARGV[1]) then
  redis.call('DECR', KEYS[1])
  return 0
end
return 1
`)

var concurrencyReleaseScript = redis.NewScript(`
-- KEYS[1] = counter key
-- Decrement, and delete if <= 0
local current = redis.call('DECR', KEYS[1])
if current <= 0 then
  redis.call('DEL', KEYS[1])
end
return 1
`)

// AcquireConcurrencyCap attempts to acquire a slot for a given key.
// This is intended for concurrency caps (e.g., per-workspace call limits).
//
// Safety properties:
// - Atomic acquire using Lua.
// - TTL prevents leaked caps on process crash.
func AcquireConcurrencyCap(ctx context.Context, rdb *redis.Client, key string, limit int, ttl time.Duration) (bool, error) {
	if rdb == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	if key == "" {
		return false, fmt.Errorf("key is required")
	}
	if limit <= 0 {
		return false, fmt.Errorf("limit must be > 0")
	}
	if ttl <= 0 {
		return false, fmt.Errorf("ttl must be > 0")
	}

	res, err := concurrencyAcquireScript.Run(ctx, rdb, []string{key}, limit, ttl.Milliseconds()).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// ReleaseConcurrencyCap releases a previously acquired slot.
func ReleaseConcurrencyCap(ctx context.Context, rdb *redis.Client, key string) error {
	if rdb == nil {
		return fmt.Errorf("redis client is nil")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}
	_, err := concurrencyReleaseScript.Run(ctx, rdb, []string{key}).Result()
	return err
}
