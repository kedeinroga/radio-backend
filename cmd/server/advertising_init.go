package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"radio-backend/internal/handlers"
	"radio-backend/internal/infrastructure/cache"
	"radio-backend/internal/infrastructure/database"
	"radio-backend/internal/repositories/postgres"
	"radio-backend/internal/server"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
	redislib "github.com/redis/go-redis/v9"
)

// InitializeAdvertisingSystem initializes all advertising components
// Call this from your main.go after database and Redis are initialized
func InitializeAdvertisingSystem(
	router *gin.Engine,
	db *database.Connection,
	redisClient *redislib.Client,
	logger *slog.Logger,
) error {
	ctx := context.Background()

	logger.Info("Initializing advertising system...")

	// ============================================
	// 1. Test Redis connection
	// ============================================
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		return fmt.Errorf("redis connection failed: %w", err)
	}
	logger.Info("Redis connection established")

	// ============================================
	// 2. Load Configuration
	// ============================================
	cfg := getAdConfig()
	logger.Info("Advertising configuration loaded",
		"max_ads_per_hour", cfg.MaxAdsPerHour,
		"max_ads_per_day", cfg.MaxAdsPerDay,
		"fraud_threshold", cfg.FraudScoreThreshold,
	)

	// ============================================
	// 3. Initialize Repositories
	// ============================================

	// PostgreSQL Repositories
	campaignRepo := postgres.NewAdCampaignRepository(db)
	adRepo := postgres.NewAdvertisementRepository(db)
	impressionRepo := postgres.NewAdImpressionRepository(db)
	clickRepo := postgres.NewAdClickRepository(db)
	userAdProfileRepo := postgres.NewUserAdProfileRepository(db)

	// Ad Cache (15 minutes TTL)
	adCache := cache.NewAdCache(redisClient, 15*time.Minute)

	logger.Info("Repositories initialized")

	// ============================================
	// 4. Initialize Services
	// ============================================

	// Security Service
	securityService := services.NewAdSecurityService(
		cfg.HMACSecret,
		cfg.FraudScoreThreshold,
	)

	// Campaign Service
	campaignService := services.NewAdCampaignService(campaignRepo)

	// Advertisement Service
	adService := services.NewAdvertisementService(
		adRepo,
		campaignRepo,
		adCache,
	)

	// Impression Service
	impressionService := services.NewImpressionService(
		impressionRepo,
		adRepo,
		userAdProfileRepo,
		adCache,
		securityService,
		cfg.FraudScoreThreshold,
	)

	// Click Service
	clickService := services.NewClickService(
		clickRepo,
		impressionRepo,
		adRepo,
		adCache,
		logger,
	)

	// User Ad Profile Service
	userAdProfileService := services.NewUserAdProfileService(
		userAdProfileRepo,
		adCache,
		logger,
	)

	logger.Info("Services initialized")

	// ============================================
	// 5. Initialize Handlers
	// ============================================

	campaignHandler := handlers.NewAdCampaignHandler(
		campaignService,
		logger,
	)

	adHandler := handlers.NewAdvertisementHandler(
		adService,
		logger,
	)

	impressionHandler := handlers.NewImpressionHandler(
		impressionService,
		clickService,
		logger,
	)

	profileHandler := handlers.NewUserAdProfileHandler(
		userAdProfileService,
		logger,
	)

	analyticsHandler := handlers.NewAdminAnalyticsHandler(
		campaignService,
		logger,
	)

	logger.Info("Handlers initialized")

	// ============================================
	// 6. Register Routes
	// ============================================

	server.RegisterAdRoutes(
		router,
		campaignHandler,
		adHandler,
		impressionHandler,
		profileHandler,
		analyticsHandler,
		impressionService,
		clickService,
		redisClient,
		logger,
	)

	logger.Info("✅ Advertising routes registered")
	logger.Info("✅ Advertising system fully initialized and ready")

	return nil
}

// AdConfig holds advertising configuration
type AdConfig struct {
	HMACSecret          []byte
	FraudScoreThreshold float64
	MaxAdsPerHour       int
	MaxAdsPerDay        int
}

// getAdConfig loads advertising configuration from environment
func getAdConfig() AdConfig {
	cfg := AdConfig{
		HMACSecret:          []byte(getEnv("AD_HMAC_SECRET", "change-me-in-production")),
		FraudScoreThreshold: 0.5, // Block if fraud score >= 50%
		MaxAdsPerHour:       6,
		MaxAdsPerDay:        30,
	}

	// Warn if using default secret
	if string(cfg.HMACSecret) == "change-me-in-production" {
		log.Println("⚠️  WARNING: Using default HMAC secret. Set AD_HMAC_SECRET environment variable!")
	}

	return cfg
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Example usage in your main.go:
//
// func main() {
//     // ... existing initialization ...
//
//     // Initialize database
//     db, err := database.NewDB(cfg.DatabaseURL)
//     if err != nil {
//         log.Fatal(err)
//     }
//
//     // Initialize Redis
//     redisClient := redis.NewClient(&redis.Options{
//         Addr:     cfg.RedisAddr,
//         Password: cfg.RedisPassword,
//         DB:       0,
//     })
//
//     // Initialize Gin router
//     router := gin.Default()
//
//     // Initialize advertising system
//     if err := InitializeAdvertisingSystem(router, db, redisClient, logger); err != nil {
//         log.Fatal("Failed to initialize advertising system:", err)
//     }
//
//     // Start server
//     router.Run(":8080")
// }
