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

// CachedAdCampaignRepository envuelve AdCampaignRepository con caching
type CachedAdCampaignRepository struct {
	repo  *postgres.AdCampaignRepository
	cache *redis.Client
	ttl   time.Duration
}

// NewCachedAdCampaignRepository crea una nueva instancia con cache
func NewCachedAdCampaignRepository(db *database.Connection, cache *redis.Client, ttl time.Duration) *CachedAdCampaignRepository {
	return &CachedAdCampaignRepository{
		repo:  postgres.NewAdCampaignRepository(db),
		cache: cache,
		ttl:   ttl,
	}
}

// Create crea una campaña e invalida cache relacionado
func (r *CachedAdCampaignRepository) Create(ctx context.Context, campaign *domain.AdCampaign) error {
	err := r.repo.Create(ctx, campaign)
	if err != nil {
		return err
	}

	// Invalidar cache de campañas activas
	r.cache.Del(ctx, "campaigns:active")
	r.cache.Del(ctx, fmt.Sprintf("campaigns:advertiser:%s", campaign.AdvertiserID))

	return nil
}

// GetByID obtiene campaña por ID con cache
func (r *CachedAdCampaignRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AdCampaign, error) {
	cacheKey := fmt.Sprintf("campaign:%s", id)

	// Intentar obtener del cache
	cached, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var campaign domain.AdCampaign
		if err := json.Unmarshal([]byte(cached), &campaign); err == nil {
			return &campaign, nil
		}
	}

	// Si no está en cache, obtener de BD
	campaign, err := r.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Guardar en cache
	if data, err := json.Marshal(campaign); err == nil {
		r.cache.Set(ctx, cacheKey, data, r.ttl)
	}

	return campaign, nil
}

// GetByAdvertiserID obtiene campañas de un advertiser (con cache)
func (r *CachedAdCampaignRepository) GetByAdvertiserID(ctx context.Context, advertiserID uuid.UUID) ([]*domain.AdCampaign, error) {
	cacheKey := fmt.Sprintf("campaigns:advertiser:%s", advertiserID)

	// Intentar obtener del cache
	cached, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var campaigns []*domain.AdCampaign
		if err := json.Unmarshal([]byte(cached), &campaigns); err == nil {
			return campaigns, nil
		}
	}

	// Si no está en cache, obtener de BD
	campaigns, err := r.repo.GetByAdvertiserID(ctx, advertiserID)
	if err != nil {
		return nil, err
	}

	// Guardar en cache
	if data, err := json.Marshal(campaigns); err == nil {
		r.cache.Set(ctx, cacheKey, data, r.ttl)
	}

	return campaigns, nil
}

// Update actualiza campaña e invalida cache
func (r *CachedAdCampaignRepository) Update(ctx context.Context, campaign *domain.AdCampaign) error {
	err := r.repo.Update(ctx, campaign)
	if err != nil {
		return err
	}

	// Invalidar cache
	r.cache.Del(ctx,
		fmt.Sprintf("campaign:%s", campaign.ID),
		"campaigns:active",
		fmt.Sprintf("campaigns:advertiser:%s", campaign.AdvertiserID),
	)

	return nil
}

// GetActiveCampaigns obtiene campañas activas (con cache agresivo)
func (r *CachedAdCampaignRepository) GetActiveCampaigns(ctx context.Context) ([]*domain.AdCampaign, error) {
	cacheKey := "campaigns:active"

	// Cache más corto para datos dinámicos (5 minutos)
	shortTTL := 5 * time.Minute

	// Intentar obtener del cache
	cached, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var campaigns []*domain.AdCampaign
		if err := json.Unmarshal([]byte(cached), &campaigns); err == nil {
			return campaigns, nil
		}
	}

	// Si no está en cache, obtener de BD
	campaigns, err := r.repo.GetActiveCampaigns(ctx)
	if err != nil {
		return nil, err
	}

	// Guardar en cache (TTL corto)
	if data, err := json.Marshal(campaigns); err == nil {
		r.cache.Set(ctx, cacheKey, data, shortTTL)
	}

	return campaigns, nil
}

// IncrementSpent incrementa gasto (bypass cache, actualiza BD)
func (r *CachedAdCampaignRepository) IncrementSpent(ctx context.Context, campaignID uuid.UUID, amountCents int) error {
	err := r.repo.IncrementSpent(ctx, campaignID, amountCents)
	if err != nil {
		return err
	}

	// Invalidar cache
	r.cache.Del(ctx,
		fmt.Sprintf("campaign:%s", campaignID),
		"campaigns:active",
	)

	return nil
}

// Delete elimina una campaña
func (r *CachedAdCampaignRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// Invalidar cache
	r.cache.Del(ctx,
		fmt.Sprintf("campaign:%s", id),
		"campaigns:active",
	)

	return nil
}

// GetExpiredCampaigns obtiene campañas expiradas (sin cache)
func (r *CachedAdCampaignRepository) GetExpiredCampaigns(ctx context.Context) ([]*domain.AdCampaign, error) {
	// No cachear queries administrativas
	return r.repo.GetExpiredCampaigns(ctx)
}
