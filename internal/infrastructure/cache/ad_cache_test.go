package cache

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis crea un cliente Redis para testing
func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Usar DB 1 para tests
	})

	ctx := context.Background()
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Skip("Redis not available for testing")
	}

	// Limpiar DB de test
	client.FlushDB(ctx)

	return client
}

func TestAdCache_FrequencyCapping(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	cache := NewAdCache(client, 10*time.Minute)
	ctx := context.Background()
	userID := uuid.New()

	// Test hourly counter
	count1, err := cache.IncrementUserAdCountHourly(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count1)

	count2, err := cache.IncrementUserAdCountHourly(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count2)

	// Test daily counter
	countDaily, err := cache.IncrementUserAdCountDaily(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), countDaily)

	// Get counts
	hourlyCount, err := cache.GetUserAdCountHourly(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), hourlyCount)

	dailyCount, err := cache.GetUserAdCountDaily(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), dailyCount)
}

func TestAdCache_RateLimiting(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	cache := NewAdCache(client, 10*time.Minute)
	ctx := context.Background()
	key := "test-user-123"
	limit := int64(5)
	window := 1 * time.Minute

	// Primeras 5 solicitudes deben pasar
	for i := 0; i < 5; i++ {
		allowed, remaining, err := cache.RateLimitCheck(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
		assert.Equal(t, int64(4-i), remaining)
	}

	// La sexta debe fallar
	allowed, remaining, err := cache.RateLimitCheck(ctx, key, limit, window)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, int64(0), remaining)
}

func TestAdCache_ImpressionMetrics(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	cache := NewAdCache(client, 10*time.Minute)
	ctx := context.Background()
	adID := uuid.New()

	// Incrementar impresiones
	err := cache.IncrementImpressions(ctx, adID)
	require.NoError(t, err)

	err = cache.IncrementImpressions(ctx, adID)
	require.NoError(t, err)

	// Verificar contador
	count, err := cache.GetImpressionCount(ctx, adID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Incrementar clicks
	err = cache.IncrementClicks(ctx, adID)
	require.NoError(t, err)

	clickCount, err := cache.GetClickCount(ctx, adID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), clickCount)
}

func TestAdCache_FraudDetection(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	cache := NewAdCache(client, 10*time.Minute)
	ctx := context.Background()
	ipAddress := "192.168.1.100"
	window := 5 * time.Minute

	// Track multiple impressions
	for i := 0; i < 10; i++ {
		impressionID := uuid.New()
		err := cache.TrackIPImpression(ctx, ipAddress, impressionID, window)
		require.NoError(t, err)
	}

	// Count impressions from IP
	count, err := cache.CountIPImpressions(ctx, ipAddress, window)
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)

	// Track clicks
	for i := 0; i < 5; i++ {
		clickID := uuid.New()
		err := cache.TrackIPClick(ctx, ipAddress, clickID, window)
		require.NoError(t, err)
	}

	// Count clicks from IP
	clickCount, err := cache.CountIPClicks(ctx, ipAddress, window)
	require.NoError(t, err)
	assert.Equal(t, int64(5), clickCount)
}

func TestAdCache_CacheAdvertisement(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	cache := NewAdCache(client, 10*time.Minute)
	ctx := context.Background()
	adID := uuid.New()

	type TestAd struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}

	testAd := TestAd{
		ID:    adID.String(),
		Title: "Test Advertisement",
	}

	// Cache ad
	err := cache.CacheAdvertisement(ctx, adID, testAd)
	require.NoError(t, err)

	// Retrieve ad
	var retrievedAd TestAd
	err = cache.GetCachedAdvertisement(ctx, adID, &retrievedAd)
	require.NoError(t, err)
	assert.Equal(t, testAd.ID, retrievedAd.ID)
	assert.Equal(t, testAd.Title, retrievedAd.Title)

	// Invalidate ad
	err = cache.InvalidateAdvertisement(ctx, adID)
	require.NoError(t, err)

	// Should not be found
	err = cache.GetCachedAdvertisement(ctx, adID, &retrievedAd)
	assert.Error(t, err)
}
