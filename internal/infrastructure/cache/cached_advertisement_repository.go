package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"radio-backend/internal/domain"
	"radio-backend/internal/infrastructure/database"
	"radio-backend/internal/repositories/postgres"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// CachedAdvertisementRepository envuelve AdvertisementRepository con caching
type CachedAdvertisementRepository struct {
	repo  *postgres.AdvertisementRepository
	cache *redis.Client
	ttl   time.Duration
}

// NewCachedAdvertisementRepository crea una nueva instancia con cache
func NewCachedAdvertisementRepository(db *database.Connection, cache *redis.Client, ttl time.Duration) *CachedAdvertisementRepository {
	return &CachedAdvertisementRepository{
		repo:  postgres.NewAdvertisementRepository(db),
		cache: cache,
		ttl:   ttl,
	}
}

// Create crea un anuncio e invalida cache
func (r *CachedAdvertisementRepository) Create(ctx context.Context, ad *domain.Advertisement) error {
	err := r.repo.Create(ctx, ad)
	if err != nil {
		return err
	}

	// Invalidar cache de eligible ads
	r.cache.Del(ctx,
		fmt.Sprintf("ads:campaign:%s", ad.CampaignID),
		"ads:eligible:*", // Patrón para invalidar todos los eligible ads
	)

	return nil
}

// GetByID obtiene anuncio por ID con cache
func (r *CachedAdvertisementRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Advertisement, error) {
	cacheKey := fmt.Sprintf("ad:%s", id)

	// Intentar obtener del cache
	cached, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var ad domain.Advertisement
		if err := json.Unmarshal([]byte(cached), &ad); err == nil {
			return &ad, nil
		}
	}

	// Si no está en cache, obtener de BD
	ad, err := r.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Guardar en cache
	if data, err := json.Marshal(ad); err == nil {
		r.cache.Set(ctx, cacheKey, data, r.ttl)
	}

	return ad, nil
}

// GetByCampaignID obtiene anuncios de una campaña (con cache)
func (r *CachedAdvertisementRepository) GetByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*domain.Advertisement, error) {
	cacheKey := fmt.Sprintf("ads:campaign:%s", campaignID)

	// Intentar obtener del cache
	cached, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var ads []*domain.Advertisement
		if err := json.Unmarshal([]byte(cached), &ads); err == nil {
			return ads, nil
		}
	}

	// Si no está en cache, obtener de BD
	ads, err := r.repo.GetByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	// Guardar en cache
	if data, err := json.Marshal(ads); err == nil {
		r.cache.Set(ctx, cacheKey, data, r.ttl)
	}

	return ads, nil
}

// Update actualiza anuncio e invalida cache
func (r *CachedAdvertisementRepository) Update(ctx context.Context, ad *domain.Advertisement) error {
	err := r.repo.Update(ctx, ad)
	if err != nil {
		return err
	}

	// Invalidar cache
	r.cache.Del(ctx,
		fmt.Sprintf("ad:%s", ad.ID),
		fmt.Sprintf("ads:campaign:%s", ad.CampaignID),
	)

	// Invalidar eligible ads usando SCAN (más seguro que patrón *)
	r.invalidateEligibleAdsCache(ctx)

	return nil
}

// GetEligibleAds obtiene anuncios elegibles con cache por criterios
// IMPORTANTE: Cache agresivo para reducir carga de DB en queries complejas
func (r *CachedAdvertisementRepository) GetEligibleAds(ctx context.Context, country, genre, language, device string) ([]*domain.Advertisement, error) {
	// Cache key específico para estos criterios
	cacheKey := fmt.Sprintf("ads:eligible:%s:%s:%s:%s", country, genre, language, device)

	// Cache corto (2 minutos) para datos dinámicos
	shortTTL := 2 * time.Minute

	// Intentar obtener del cache
	cached, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var ads []*domain.Advertisement
		if err := json.Unmarshal([]byte(cached), &ads); err == nil {
			return ads, nil
		}
	}

	// Si no está en cache, obtener de BD (query compleja)
	ads, err := r.repo.GetEligibleAds(ctx, country, genre, language, device)
	if err != nil {
		return nil, err
	}

	// Guardar en cache (TTL corto)
	if data, err := json.Marshal(ads); err == nil {
		r.cache.Set(ctx, cacheKey, data, shortTTL)
	}

	return ads, nil
}

// IncrementImpressions incrementa contador (bypass cache)
func (r *CachedAdvertisementRepository) IncrementImpressions(ctx context.Context, adID uuid.UUID) error {
	err := r.repo.IncrementImpressions(ctx, adID)
	if err != nil {
		return err
	}

	// Invalidar cache del anuncio
	r.cache.Del(ctx, fmt.Sprintf("ad:%s", adID))

	return nil
}

// IncrementClicks incrementa contador (bypass cache)
func (r *CachedAdvertisementRepository) IncrementClicks(ctx context.Context, adID uuid.UUID) error {
	err := r.repo.IncrementClicks(ctx, adID)
	if err != nil {
		return err
	}

	// Invalidar cache del anuncio
	r.cache.Del(ctx, fmt.Sprintf("ad:%s", adID))

	return nil
}

// IncrementSpend incrementa gasto (bypass cache)
func (r *CachedAdvertisementRepository) IncrementSpend(ctx context.Context, adID uuid.UUID, amountCents int) error {
	err := r.repo.IncrementSpend(ctx, adID, amountCents)
	if err != nil {
		return err
	}

	// Invalidar cache
	r.cache.Del(ctx, fmt.Sprintf("ad:%s", adID))

	// Si se agotó presupuesto, invalidar eligible ads
	r.invalidateEligibleAdsCache(ctx)

	return nil
}

// GetActiveAds obtiene anuncios activos (con cache corto)
func (r *CachedAdvertisementRepository) GetActiveAds(ctx context.Context) ([]*domain.Advertisement, error) {
	cacheKey := "ads:active"

	// Cache corto (5 minutos)
	shortTTL := 5 * time.Minute

	// Intentar obtener del cache
	cached, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var ads []*domain.Advertisement
		if err := json.Unmarshal([]byte(cached), &ads); err == nil {
			return ads, nil
		}
	}

	// Si no está en cache, obtener de BD
	ads, err := r.repo.GetActiveAds(ctx)
	if err != nil {
		return nil, err
	}

	// Guardar en cache
	if data, err := json.Marshal(ads); err == nil {
		r.cache.Set(ctx, cacheKey, data, shortTTL)
	}

	return ads, nil
}

// Delete elimina un anuncio
func (r *CachedAdvertisementRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// Invalidar cache
	r.cache.Del(ctx, fmt.Sprintf("ad:%s", id))
	r.invalidateEligibleAdsCache(ctx)

	return nil
}

// invalidateEligibleAdsCache invalida todas las keys de eligible ads
func (r *CachedAdvertisementRepository) invalidateEligibleAdsCache(ctx context.Context) {
	// Usar SCAN para encontrar todas las keys con el patrón
	iter := r.cache.Scan(ctx, 0, "ads:eligible:*", 0).Iterator()
	for iter.Next(ctx) {
		r.cache.Del(ctx, iter.Val())
	}
}
