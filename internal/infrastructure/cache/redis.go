package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the Redis client
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client
func NewRedisClient(url, password string, db int) (*RedisClient, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	if password != "" {
		opts.Password = password
	}
	opts.DB = db

	client := redis.NewClient(opts)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return &RedisClient{client: client}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// IncrementStationPlay increments the play count for a station
func (r *RedisClient) IncrementStationPlay(stationID string) error {
	ctx := context.Background()
	key := "analytics:stations:plays"
	return r.client.ZIncrBy(ctx, key, 1, stationID).Err()
}

// IncrementSearch increments the search count for a query
func (r *RedisClient) IncrementSearch(query string) error {
	ctx := context.Background()
	key := "analytics:searches:trending"
	return r.client.ZIncrBy(ctx, key, 1, query).Err()
}

// GetPopularStations returns the most popular stations
func (r *RedisClient) GetPopularStations(limit int) ([]string, error) {
	ctx := context.Background()
	key := "analytics:stations:plays"

	result, err := r.client.ZRevRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetTrendingSearches returns the most trending searches
func (r *RedisClient) GetTrendingSearches(limit int) ([]string, error) {
	ctx := context.Background()
	key := "analytics:searches:trending"

	result, err := r.client.ZRevRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	return result, nil
}

// AddActiveUser adds a user to the active users set
func (r *RedisClient) AddActiveUser(userID string) error {
	ctx := context.Background()
	key := "analytics:users:active"
	return r.client.PFAdd(ctx, key, userID).Err()
}

// GetActiveUsersCount returns the count of active users
func (r *RedisClient) GetActiveUsersCount() (int64, error) {
	ctx := context.Background()
	key := "analytics:users:active"
	return r.client.PFCount(ctx, key).Result()
}

// Set sets a key-value pair with expiration
func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	ctx := context.Background()
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Get gets a value by key
func (r *RedisClient) Get(key string) (string, error) {
	ctx := context.Background()
	return r.client.Get(ctx, key).Result()
}

// Delete deletes a key
func (r *RedisClient) Delete(key string) error {
	ctx := context.Background()
	return r.client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (r *RedisClient) Exists(key string) (bool, error) {
	ctx := context.Background()
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Increment increments a counter
func (r *RedisClient) Increment(key string) (int64, error) {
	ctx := context.Background()
	return r.client.Incr(ctx, key).Result()
}

// SetWithNX sets a key only if it doesn't exist
func (r *RedisClient) SetWithNX(key string, value interface{}, expiration time.Duration) (bool, error) {
	ctx := context.Background()
	return r.client.SetNX(ctx, key, value, expiration).Result()
}

// GetInt gets an integer value
func (r *RedisClient) GetInt(key string) (int, error) {
	val, err := r.Get(key)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(val)
}
