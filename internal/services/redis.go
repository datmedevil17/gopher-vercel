package services

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	client *redis.Client
}

func NewRedisService(redisURL string) *RedisService {
	// Parse Redis URL if possible, or just use address
	opt, err := redis.ParseURL(redisURL)
	var client *redis.Client
	if err != nil {
		// Fallback to simple address if URL parsing fails (e.g. "localhost:6379")
		client = redis.NewClient(&redis.Options{
			Addr: redisURL,
		})
	} else {
		client = redis.NewClient(opt)
	}

	return &RedisService{
		client: client,
	}
}

func (s *RedisService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

func (s *RedisService) Get(ctx context.Context, key string) ([]byte, error) {
	return s.client.Get(ctx, key).Bytes()
}

func (s *RedisService) SetContentType(ctx context.Context, key string, contentType string, ttl time.Duration) error {
	return s.client.Set(ctx, key+":content-type", contentType, ttl).Err()
}

func (s *RedisService) GetContentType(ctx context.Context, key string) (string, error) {
	return s.client.Get(ctx, key+":content-type").Result()
}
