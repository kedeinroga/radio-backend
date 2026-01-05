package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"radio-backend/internal/handlers"
	"radio-backend/internal/middleware"
	"radio-backend/internal/services"
)

// RegisterAdRoutes registers all advertising-related routes
func RegisterAdRoutes(
	router *gin.Engine,
	// Handlers
	campaignHandler *handlers.AdCampaignHandler,
	adHandler *handlers.AdvertisementHandler,
	impressionHandler *handlers.ImpressionHandler,
	profileHandler *handlers.UserAdProfileHandler,
	analyticsHandler *handlers.AdminAnalyticsHandler,
	// Services
	impressionService *services.ImpressionService,
	clickService *services.ClickService,
	// Infrastructure
	redis *redis.Client,
	logger *slog.Logger,
) {
	// Initialize middleware
	rateLimiter := middleware.NewAdRateLimiter(redis, logger)
	fraudDetector := middleware.NewAdFraudDetector(redis, logger)
	adAuth := middleware.NewAdAuthMiddleware(logger)

	// Public ad serving routes (no auth required, rate limited)
	publicAds := router.Group("/api/v1/ads")
	{
		// Apply fraud detection and rate limiting
		publicAds.Use(fraudDetector.CheckIPBlacklist())
		publicAds.Use(fraudDetector.DetectFraud())
		publicAds.Use(rateLimiter.LimitByIP(middleware.DefaultAdRateLimitConfig()))

		// Get eligible ads for user (most important endpoint)
		publicAds.GET("/eligible", adAuth.OptionalAuth(), adHandler.GetEligibleAds)

		// Get single ad details
		publicAds.GET("/:id", adHandler.GetAdvertisement)
	}

	// Ad tracking routes (public, but heavily rate limited and fraud detected)
	tracking := router.Group("/api/v1/tracking")
	{
		tracking.Use(fraudDetector.CheckIPBlacklist())
		tracking.Use(fraudDetector.DetectFraud())

		// Impression tracking
		impressions := tracking.Group("/impressions")
		{
			impressions.Use(rateLimiter.LimitImpressionTracking())
			impressions.POST("", impressionHandler.TrackImpression)
			impressions.GET("/validate", impressionHandler.ValidateImpressionToken)
		}

		// Click tracking
		clicks := tracking.Group("/clicks")
		{
			clicks.Use(rateLimiter.LimitClickTracking())
			clicks.POST("", impressionHandler.TrackClick)
			clicks.POST("/:click_id/conversion", impressionHandler.TrackConversion)
		}
	}

	// User profile routes (auth required)
	userProfiles := router.Group("/api/v1/users/:user_id/ad-profile")
	// Note: Would need to add main auth middleware here in production
	{
		userProfiles.GET("", profileHandler.GetUserAdProfile)
		userProfiles.PUT("", profileHandler.UpdateUserAdProfile)
		userProfiles.GET("/stats", profileHandler.GetUserAdStats)
		userProfiles.GET("/can-show-ad", profileHandler.CanShowAd)

		// Premium subscription management
		premium := userProfiles.Group("/premium")
		{
			premium.POST("", profileHandler.ActivatePremium)
			premium.DELETE("", profileHandler.DeactivatePremium)
		}
	}

	// Advertiser routes (auth + advertiser role required)
	advertiser := router.Group("/api/v1/advertiser")
	advertiser.Use(adAuth.RequireAdvertiser())
	{
		// Campaign management
		campaigns := advertiser.Group("/campaigns")
		{
			campaigns.POST("", campaignHandler.CreateCampaign)
			campaigns.GET("/:id", campaignHandler.GetCampaign)
			campaigns.PUT("/:id", campaignHandler.UpdateCampaign)
			campaigns.DELETE("/:id", campaignHandler.DeleteCampaign)
			campaigns.POST("/:id/pause", campaignHandler.PauseCampaign)
			campaigns.POST("/:id/resume", campaignHandler.ResumeCampaign)
			campaigns.GET("/:id/stats", campaignHandler.GetCampaignStats)
		}

		// Advertisement management
		advertisements := advertiser.Group("/ads")
		{
			advertisements.POST("", adHandler.CreateAdvertisement)
			advertisements.GET("/:id", adHandler.GetAdvertisement)
			advertisements.PUT("/:id", adHandler.UpdateAdvertisement)
			advertisements.DELETE("/:id", adHandler.DeleteAdvertisement)
			advertisements.GET("/:id/stats", adHandler.GetAdvertisementStats)
			advertisements.GET("/:id/impressions", impressionHandler.GetImpressionsByAdvertisement)
			advertisements.GET("/:id/viewable", impressionHandler.CountViewableImpressions)
			advertisements.GET("/:id/clicks", impressionHandler.GetClicksByAdvertisement)
			advertisements.GET("/:id/clicks/stats", impressionHandler.GetClickStats)
		}

		// List resources
		advertiser.GET("/campaigns", campaignHandler.GetCampaignsByAdvertiser)
		advertiser.GET("/campaigns/:id/ads", adHandler.GetAdvertisementsByCampaign)
	}

	// Admin analytics routes (auth + admin role required)
	admin := router.Group("/api/v1/admin")
	admin.Use(adAuth.RequireAdmin())
	{
		analytics := admin.Group("/analytics")
		{
			analytics.GET("/revenue", analyticsHandler.GetRevenueAnalytics)
			analytics.GET("/campaigns", analyticsHandler.GetCampaignPerformance)
			analytics.GET("/fraud", analyticsHandler.GetFraudMetrics)
			analytics.GET("/top-ads", analyticsHandler.GetTopAds)
			analytics.GET("/dashboard", analyticsHandler.GetDashboardOverview)
		}

		// Active campaigns (admin only)
		admin.GET("/campaigns/active", campaignHandler.GetActiveCampaigns)
	}

	logger.Info("Advertising routes registered successfully")
}
