package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// InitRedis initializes the Redis client
func InitRedis(config RedisConfig) error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     config.Host + ":" + config.Port,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test the connection
	ctx := context.Background()
	_, err := redisClient.Ping(ctx).Result()
	return err
}

// GetRedisClient returns the Redis client instance
func GetRedisClient() *redis.Client {
	return redisClient
}

// SetCache stores data in Redis cache
func SetCache(ctx context.Context, key string, data interface{}, expiration time.Duration) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return redisClient.Set(ctx, key, dataJSON, expiration).Err()
}

// GetCache retrieves data from Redis cache
func GetCache(ctx context.Context, key string, dest interface{}) error {
	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// DeleteCache removes data from Redis cache
func DeleteCache(ctx context.Context, key string) error {
	return redisClient.Del(ctx, key).Err()
}

// DeleteByPattern deletes all keys matching a pattern
func DeleteByPattern(ctx context.Context, pattern string) error {
	iter := redisClient.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := redisClient.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

const (
	// Cache key patterns
	UserListPattern      = "users:*"
	UserDetailPattern    = "user:%s"
	ProductListPattern   = "products:*"
	ProductDetailPattern = "product:%s"
)
