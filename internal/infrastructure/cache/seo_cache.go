package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/logger"
)

// SEOCache maneja el cache de datos SEO en Redis
type SEOCache struct {
	redis *RedisClient
}

// NewSEOCache crea una nueva instancia del cache SEO
func NewSEOCache(redis *RedisClient) *SEOCache {
	return &SEOCache{redis: redis}
}

// GetSitemapData obtiene los datos del sitemap desde cache
func (c *SEOCache) GetSitemapData() (*domain.SitemapData, error) {
	ctx := context.Background()
	key := "seo:sitemap:data"

	data, err := c.redis.client.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			logger.Info("sitemap data cache miss")
			return nil, nil // Cache miss
		}
		logger.Error("failed to get sitemap data from cache", "error", err)
		return nil, fmt.Errorf("failed to get sitemap data from cache: %w", err)
	}

	var sitemapData domain.SitemapData
	if err := json.Unmarshal([]byte(data), &sitemapData); err != nil {
		logger.Error("failed to unmarshal sitemap data", "error", err)
		return nil, fmt.Errorf("failed to unmarshal sitemap data: %w", err)
	}

	logger.Info("sitemap data cache hit")
	return &sitemapData, nil
}

// SetSitemapData guarda los datos del sitemap en cache
func (c *SEOCache) SetSitemapData(data *domain.SitemapData, ttl time.Duration) error {
	ctx := context.Background()
	key := "seo:sitemap:data"

	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Error("failed to marshal sitemap data", "error", err)
		return fmt.Errorf("failed to marshal sitemap data: %w", err)
	}

	if err := c.redis.client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		logger.Error("failed to set sitemap data in cache", "error", err, "ttl", ttl)
		return fmt.Errorf("failed to set sitemap data in cache: %w", err)
	}

	logger.Info("sitemap data cached successfully", "ttl", ttl)
	return nil
}

// GetStationSEO obtiene los metadatos SEO de una estación desde cache (con idioma)
func (c *SEOCache) GetStationSEO(stationID string, language string) (*domain.SEOMetadata, error) {
	ctx := context.Background()
	key := fmt.Sprintf("seo:station:%s:%s", stationID, language)

	data, err := c.redis.client.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			logger.Info("station SEO cache miss", "station_id", stationID, "language", language)
			return nil, nil // Cache miss
		}
		logger.Error("failed to get station SEO from cache", "error", err, "station_id", stationID, "language", language)
		return nil, fmt.Errorf("failed to get station SEO from cache: %w", err)
	}

	var metadata domain.SEOMetadata
	if err := json.Unmarshal([]byte(data), &metadata); err != nil {
		logger.Error("failed to unmarshal station SEO", "error", err, "station_id", stationID, "language", language)
		return nil, fmt.Errorf("failed to unmarshal station SEO: %w", err)
	}

	logger.Info("station SEO cache hit", "station_id", stationID, "language", language)
	return &metadata, nil
}

// SetStationSEO guarda los metadatos SEO de una estación en cache (con idioma)
func (c *SEOCache) SetStationSEO(stationID string, language string, metadata *domain.SEOMetadata, ttl time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("seo:station:%s:%s", stationID, language)

	jsonData, err := json.Marshal(metadata)
	if err != nil {
		logger.Error("failed to marshal station SEO", "error", err, "station_id", stationID, "language", language)
		return fmt.Errorf("failed to marshal station SEO: %w", err)
	}

	if err := c.redis.client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		logger.Error("failed to set station SEO in cache", "error", err, "station_id", stationID, "language", language, "ttl", ttl)
		return fmt.Errorf("failed to set station SEO in cache: %w", err)
	}

	logger.Info("station SEO cached successfully", "station_id", stationID, "language", language, "ttl", ttl)
	return nil
}

// InvalidateSitemapData invalida el cache de sitemap data
func (c *SEOCache) InvalidateSitemapData() error {
	ctx := context.Background()
	key := "seo:sitemap:data"

	if err := c.redis.client.Del(ctx, key).Err(); err != nil {
		logger.Error("failed to invalidate sitemap data", "error", err)
		return fmt.Errorf("failed to invalidate sitemap data: %w", err)
	}

	logger.Info("sitemap data cache invalidated")
	return nil
}

// InvalidateStationSEO invalida el cache SEO de una estación específica (todos los idiomas)
func (c *SEOCache) InvalidateStationSEO(stationID string) error {
	ctx := context.Background()
	pattern := fmt.Sprintf("seo:station:%s:*", stationID)

	// Obtener todas las claves que coincidan con el patrón
	keys, err := c.redis.client.Keys(ctx, pattern).Result()
	if err != nil {
		logger.Error("failed to get keys for station SEO invalidation", "error", err, "station_id", stationID)
		return fmt.Errorf("failed to get keys for station SEO invalidation: %w", err)
	}

	if len(keys) == 0 {
		logger.Info("no cache keys found for station", "station_id", stationID)
		return nil
	}

	// Eliminar todas las claves
	if err := c.redis.client.Del(ctx, keys...).Err(); err != nil {
		logger.Error("failed to invalidate station SEO", "error", err, "station_id", stationID)
		return fmt.Errorf("failed to invalidate station SEO: %w", err)
	}

	logger.Info("station SEO cache invalidated for all languages", "station_id", stationID, "keys_deleted", len(keys))
	return nil
}

// GetCacheStats retorna estadísticas del cache SEO
func (c *SEOCache) GetCacheStats() (map[string]interface{}, error) {
	ctx := context.Background()

	stats := make(map[string]interface{})

	// Contar claves de sitemap
	sitemapKeys, err := c.redis.client.Keys(ctx, "seo:sitemap:*").Result()
	if err == nil {
		stats["sitemap_keys"] = len(sitemapKeys)
	}

	// Contar claves de estaciones
	stationKeys, err := c.redis.client.Keys(ctx, "seo:station:*").Result()
	if err == nil {
		stats["station_keys"] = len(stationKeys)
	}

	// Obtener info de memoria (si está disponible)
	info, err := c.redis.client.Info(ctx, "memory").Result()
	if err == nil {
		stats["memory_info"] = info
	}

	return stats, nil
}
