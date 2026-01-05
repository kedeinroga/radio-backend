package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// AdCache maneja el caché de publicidad en Redis
type AdCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewAdCache crea una nueva instancia de AdCache
func NewAdCache(client *redis.Client, ttl time.Duration) *AdCache {
	return &AdCache{
		client: client,
		ttl:    ttl,
	}
}

// CacheAdvertisement guarda un anuncio en caché
func (c *AdCache) CacheAdvertisement(ctx context.Context, adID uuid.UUID, data interface{}) error {
	key := fmt.Sprintf("ad:cache:%s", adID.String())

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal ad data: %w", err)
	}

	return c.client.Set(ctx, key, jsonData, c.ttl).Err()
}

// GetCachedAdvertisement recupera un anuncio del caché
func (c *AdCache) GetCachedAdvertisement(ctx context.Context, adID uuid.UUID, dest interface{}) error {
	key := fmt.Sprintf("ad:cache:%s", adID.String())

	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("ad not found in cache")
		}
		return fmt.Errorf("failed to get ad from cache: %w", err)
	}

	return json.Unmarshal([]byte(data), dest)
}

// InvalidateAdvertisement elimina un anuncio del caché
func (c *AdCache) InvalidateAdvertisement(ctx context.Context, adID uuid.UUID) error {
	key := fmt.Sprintf("ad:cache:%s", adID.String())
	return c.client.Del(ctx, key).Err()
}

// IncrementUserAdCountHourly incrementa el contador de anuncios por hora para un usuario
func (c *AdCache) IncrementUserAdCountHourly(ctx context.Context, userID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("ad:frequency:hourly:%s", userID.String())

	// Incrementar
	count, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment hourly ad count: %w", err)
	}

	// Establecer TTL de 1 hora si es nuevo
	if count == 1 {
		c.client.Expire(ctx, key, 1*time.Hour)
	}

	return count, nil
}

// IncrementUserAdCountDaily incrementa el contador de anuncios por día para un usuario
func (c *AdCache) IncrementUserAdCountDaily(ctx context.Context, userID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("ad:frequency:daily:%s", userID.String())

	// Incrementar
	count, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment daily ad count: %w", err)
	}

	// Establecer TTL de 24 horas si es nuevo
	if count == 1 {
		c.client.Expire(ctx, key, 24*time.Hour)
	}

	return count, nil
}

// GetUserAdCountHourly obtiene el contador de anuncios por hora para un usuario
func (c *AdCache) GetUserAdCountHourly(ctx context.Context, userID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("ad:frequency:hourly:%s", userID.String())

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get hourly ad count: %w", err)
	}

	count, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse hourly ad count: %w", err)
	}

	return count, nil
}

// GetUserAdCountDaily obtiene el contador de anuncios por día para un usuario
func (c *AdCache) GetUserAdCountDaily(ctx context.Context, userID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("ad:frequency:daily:%s", userID.String())

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get daily ad count: %w", err)
	}

	count, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse daily ad count: %w", err)
	}

	return count, nil
}

// RateLimitCheck verifica rate limiting usando sliding window
func (c *AdCache) RateLimitCheck(ctx context.Context, key string, limit int64, window time.Duration) (bool, int64, error) {
	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	rateLimitKey := fmt.Sprintf("ad:ratelimit:%s", key)

	// Usar pipeline para optimizar
	pipe := c.client.Pipeline()

	// Eliminar entradas antiguas
	pipe.ZRemRangeByScore(ctx, rateLimitKey, "0", fmt.Sprintf("%d", windowStart))

	// Contar elementos en la ventana
	countCmd := pipe.ZCard(ctx, rateLimitKey)

	// Ejecutar pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, fmt.Errorf("failed to check rate limit: %w", err)
	}

	count := countCmd.Val()

	// Si está por debajo del límite, agregar nueva entrada
	if count < limit {
		c.client.ZAdd(ctx, rateLimitKey, redis.Z{
			Score:  float64(now),
			Member: fmt.Sprintf("%d", now),
		})
		c.client.Expire(ctx, rateLimitKey, window)
		return true, limit - count - 1, nil
	}

	return false, 0, nil
}

// IncrementImpressions incrementa el contador de impresiones para un anuncio
func (c *AdCache) IncrementImpressions(ctx context.Context, adID uuid.UUID) error {
	key := fmt.Sprintf("ad:metrics:impressions:%s", adID.String())
	return c.client.Incr(ctx, key).Err()
}

// IncrementClicks incrementa el contador de clicks para un anuncio
func (c *AdCache) IncrementClicks(ctx context.Context, adID uuid.UUID) error {
	key := fmt.Sprintf("ad:metrics:clicks:%s", adID.String())
	return c.client.Incr(ctx, key).Err()
}

// GetImpressionCount obtiene el contador de impresiones para un anuncio
func (c *AdCache) GetImpressionCount(ctx context.Context, adID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("ad:metrics:impressions:%s", adID.String())

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}

	return strconv.ParseInt(val, 10, 64)
}

// GetClickCount obtiene el contador de clicks para un anuncio
func (c *AdCache) GetClickCount(ctx context.Context, adID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("ad:metrics:clicks:%s", adID.String())

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}

	return strconv.ParseInt(val, 10, 64)
}

// CountIPImpressions cuenta las impresiones de una IP en un período de tiempo
func (c *AdCache) CountIPImpressions(ctx context.Context, ipAddress string, window time.Duration) (int64, error) {
	key := fmt.Sprintf("ad:fraud:ip:%s", ipAddress)

	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	// Limpiar entradas antiguas
	c.client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// Contar
	count, err := c.client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	return count, nil
}

// TrackIPImpression registra una impresión de una IP
func (c *AdCache) TrackIPImpression(ctx context.Context, ipAddress string, impressionID uuid.UUID, window time.Duration) error {
	key := fmt.Sprintf("ad:fraud:ip:%s", ipAddress)
	now := time.Now().UnixNano()

	err := c.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: impressionID.String(),
	}).Err()

	if err != nil {
		return err
	}

	// Establecer TTL
	return c.client.Expire(ctx, key, window).Err()
}

// CountIPClicks cuenta los clicks de una IP en un período de tiempo
func (c *AdCache) CountIPClicks(ctx context.Context, ipAddress string, window time.Duration) (int64, error) {
	key := fmt.Sprintf("ad:fraud:clicks:ip:%s", ipAddress)

	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	// Limpiar entradas antiguas
	c.client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// Contar
	count, err := c.client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	return count, nil
}

// TrackIPClick registra un click de una IP
func (c *AdCache) TrackIPClick(ctx context.Context, ipAddress string, clickID uuid.UUID, window time.Duration) error {
	key := fmt.Sprintf("ad:fraud:clicks:ip:%s", ipAddress)
	now := time.Now().UnixNano()

	err := c.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: clickID.String(),
	}).Err()

	if err != nil {
		return err
	}

	// Establecer TTL
	return c.client.Expire(ctx, key, window).Err()
}
