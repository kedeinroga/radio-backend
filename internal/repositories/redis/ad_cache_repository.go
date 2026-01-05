package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"radio-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// AdCacheRepository gestiona el caché de anuncios y frequency capping en Redis
type AdCacheRepository struct {
	client *redis.Client
	ctx    context.Context
}

// NewAdCacheRepository crea una nueva instancia del repositorio de caché
func NewAdCacheRepository(client *redis.Client) *AdCacheRepository {
	return &AdCacheRepository{
		client: client,
		ctx:    context.Background(),
	}
}

// CacheAdvertisement guarda un anuncio en caché
func (r *AdCacheRepository) CacheAdvertisement(ad *domain.Advertisement, ttl time.Duration) error {
	key := fmt.Sprintf("ad:cache:%s", ad.ID.String())

	data, err := json.Marshal(ad)
	if err != nil {
		return fmt.Errorf("failed to marshal advertisement: %w", err)
	}

	return r.client.Set(r.ctx, key, data, ttl).Err()
}

// GetCachedAdvertisement obtiene un anuncio del caché
func (r *AdCacheRepository) GetCachedAdvertisement(adID uuid.UUID) (*domain.Advertisement, error) {
	key := fmt.Sprintf("ad:cache:%s", adID.String())

	data, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // No está en caché
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	var ad domain.Advertisement
	if err := json.Unmarshal([]byte(data), &ad); err != nil {
		return nil, fmt.Errorf("failed to unmarshal advertisement: %w", err)
	}

	return &ad, nil
}

// CacheEligibleAds guarda la lista de anuncios elegibles para un contexto específico
func (r *AdCacheRepository) CacheEligibleAds(cacheKey string, ads []*domain.Advertisement, ttl time.Duration) error {
	key := fmt.Sprintf("ad:eligible:%s", cacheKey)

	data, err := json.Marshal(ads)
	if err != nil {
		return fmt.Errorf("failed to marshal ads: %w", err)
	}

	return r.client.Set(r.ctx, key, data, ttl).Err()
}

// GetCachedEligibleAds obtiene la lista de anuncios elegibles del caché
func (r *AdCacheRepository) GetCachedEligibleAds(cacheKey string) ([]*domain.Advertisement, error) {
	key := fmt.Sprintf("ad:eligible:%s", cacheKey)

	data, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // No está en caché
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	var ads []*domain.Advertisement
	if err := json.Unmarshal([]byte(data), &ads); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ads: %w", err)
	}

	return ads, nil
}

// InvalidateAdCache invalida el caché de un anuncio específico
func (r *AdCacheRepository) InvalidateAdCache(adID uuid.UUID) error {
	key := fmt.Sprintf("ad:cache:%s", adID.String())
	return r.client.Del(r.ctx, key).Err()
}

// InvalidateEligibleAdsCache invalida el caché de anuncios elegibles
func (r *AdCacheRepository) InvalidateEligibleAdsCache(pattern string) error {
	key := fmt.Sprintf("ad:eligible:%s", pattern)
	return r.client.Del(r.ctx, key).Err()
}

// ============================================================================
// FREQUENCY CAPPING
// ============================================================================

// IncrementHourlyAdCount incrementa el contador de anuncios por hora para un usuario
func (r *AdCacheRepository) IncrementHourlyAdCount(userID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("ad:freq:hour:%s", userID.String())

	// Incrementar contador
	count, err := r.client.Incr(r.ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment hourly count: %w", err)
	}

	// Si es el primer incremento, establecer TTL de 1 hora
	if count == 1 {
		r.client.Expire(r.ctx, key, time.Hour)
	}

	return count, nil
}

// IncrementDailyAdCount incrementa el contador de anuncios por día para un usuario
func (r *AdCacheRepository) IncrementDailyAdCount(userID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("ad:freq:day:%s", userID.String())

	// Incrementar contador
	count, err := r.client.Incr(r.ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment daily count: %w", err)
	}

	// Si es el primer incremento, establecer TTL de 24 horas
	if count == 1 {
		r.client.Expire(r.ctx, key, 24*time.Hour)
	}

	return count, nil
}

// GetHourlyAdCount obtiene el contador de anuncios por hora para un usuario
func (r *AdCacheRepository) GetHourlyAdCount(userID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("ad:freq:hour:%s", userID.String())

	count, err := r.client.Get(r.ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil // No hay contador
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get hourly count: %w", err)
	}

	return count, nil
}

// GetDailyAdCount obtiene el contador de anuncios por día para un usuario
func (r *AdCacheRepository) GetDailyAdCount(userID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("ad:freq:day:%s", userID.String())

	count, err := r.client.Get(r.ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil // No hay contador
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get daily count: %w", err)
	}

	return count, nil
}

// ResetHourlyAdCount resetea el contador de anuncios por hora (para testing)
func (r *AdCacheRepository) ResetHourlyAdCount(userID uuid.UUID) error {
	key := fmt.Sprintf("ad:freq:hour:%s", userID.String())
	return r.client.Del(r.ctx, key).Err()
}

// ResetDailyAdCount resetea el contador de anuncios por día (para testing)
func (r *AdCacheRepository) ResetDailyAdCount(userID uuid.UUID) error {
	key := fmt.Sprintf("ad:freq:day:%s", userID.String())
	return r.client.Del(r.ctx, key).Err()
}

// ============================================================================
// IMPRESSION TOKEN TRACKING (Anti-Replay)
// ============================================================================

// MarkTokenAsUsed marca un token de impresión como usado para prevenir replay attacks
func (r *AdCacheRepository) MarkTokenAsUsed(token string, ttl time.Duration) error {
	key := fmt.Sprintf("ad:token:used:%s", token)
	return r.client.Set(r.ctx, key, "1", ttl).Err()
}

// IsTokenUsed verifica si un token ya fue usado
func (r *AdCacheRepository) IsTokenUsed(token string) (bool, error) {
	key := fmt.Sprintf("ad:token:used:%s", token)

	exists, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token: %w", err)
	}

	return exists > 0, nil
}

// ============================================================================
// FRAUD DETECTION
// ============================================================================

// IncrementIPRequestCount incrementa el contador de requests por IP (ventana de 5 minutos)
func (r *AdCacheRepository) IncrementIPRequestCount(ipAddress string) (int64, error) {
	key := fmt.Sprintf("ad:fraud:ip:%s", ipAddress)

	count, err := r.client.Incr(r.ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment IP count: %w", err)
	}

	// Si es el primer incremento, establecer TTL de 5 minutos
	if count == 1 {
		r.client.Expire(r.ctx, key, 5*time.Minute)
	}

	return count, nil
}

// GetIPRequestCount obtiene el contador de requests por IP
func (r *AdCacheRepository) GetIPRequestCount(ipAddress string) (int64, error) {
	key := fmt.Sprintf("ad:fraud:ip:%s", ipAddress)

	count, err := r.client.Get(r.ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get IP count: %w", err)
	}

	return count, nil
}

// AddIPToBlacklist agrega una IP a la lista negra
func (r *AdCacheRepository) AddIPToBlacklist(ipAddress string, ttl time.Duration) error {
	key := fmt.Sprintf("ad:fraud:blacklist:%s", ipAddress)
	return r.client.Set(r.ctx, key, "1", ttl).Err()
}

// IsIPBlacklisted verifica si una IP está en la lista negra
func (r *AdCacheRepository) IsIPBlacklisted(ipAddress string) (bool, error) {
	key := fmt.Sprintf("ad:fraud:blacklist:%s", ipAddress)

	exists, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}

	return exists > 0, nil
}

// TrackSessionImpression registra una impresión por sesión para detectar replays
func (r *AdCacheRepository) TrackSessionImpression(sessionID string, adID uuid.UUID) error {
	key := fmt.Sprintf("ad:session:%s:%s", sessionID, adID.String())
	// TTL de 24 horas
	return r.client.Set(r.ctx, key, "1", 24*time.Hour).Err()
}

// HasSessionSeenAd verifica si una sesión ya vio un anuncio
func (r *AdCacheRepository) HasSessionSeenAd(sessionID string, adID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("ad:session:%s:%s", sessionID, adID.String())

	exists, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session: %w", err)
	}

	return exists > 0, nil
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

// Ping verifica la conectividad con Redis
func (r *AdCacheRepository) Ping() error {
	return r.client.Ping(r.ctx).Err()
}
