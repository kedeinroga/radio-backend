package server

import (
	"radio-backend/internal/handlers"
	"radio-backend/internal/middleware"

	_ "radio-backend/docs" // Import generated docs

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Router holds all route handlers and middleware
type Router struct {
	engine *gin.Engine

	// Middleware
	authMiddleware             *middleware.AuthMiddleware
	analyticsMiddleware        *middleware.AnalyticsMiddleware
	corsMiddleware             gin.HandlerFunc
	rateLimiter                *middleware.RateLimiter // NUEVO: Rate limiter
	authRateLimiter            *middleware.RateLimiter // NUEVO: Rate limiter estricto para auth
	isProduction               bool                    // NUEVO: Flag para producción
	sharedSecretKey            string                  // NUEVO: Shared secret for bot protection
	requestFingerprintMiddleware gin.HandlerFunc        // NUEVO: Request source classifier

	// Handlers
	authHandler        *handlers.AuthHandler
	stationHandler     *handlers.StationHandler
	analyticsHandler   *handlers.AnalyticsHandler
	favoriteHandler    *handlers.FavoriteHandler
	seoHandler         *handlers.SEOHandler         // NUEVO: Handler SEO
	translationHandler *handlers.TranslationHandler // NUEVO: Handler de traducciones
	securityHandler    *handlers.SecurityHandler    // NUEVO: Handler de seguridad
	maintenanceHandler *handlers.MaintenanceHandler // NUEVO: Handler de mantenimiento
	monitoringHandler  *handlers.MonitoringHandler  // NUEVO: Handler de monitoring
}

// NewRouter creates a new router
func NewRouter(
	authMiddleware *middleware.AuthMiddleware,
	analyticsMiddleware *middleware.AnalyticsMiddleware,
	corsMiddleware gin.HandlerFunc,
	rateLimiter *middleware.RateLimiter, // NUEVO
	authRateLimiter *middleware.RateLimiter, // NUEVO
	isProduction bool, // NUEVO
	sharedSecretKey string, // NUEVO: bot protection
	requestFingerprintMiddleware gin.HandlerFunc, // NUEVO: request source classifier
	authHandler *handlers.AuthHandler,
	stationHandler *handlers.StationHandler,
	analyticsHandler *handlers.AnalyticsHandler,
	favoriteHandler *handlers.FavoriteHandler,
	seoHandler *handlers.SEOHandler, // NUEVO: Handler SEO
	translationHandler *handlers.TranslationHandler, // NUEVO: Handler de traducciones
	securityHandler *handlers.SecurityHandler, // NUEVO: Handler de seguridad
	maintenanceHandler *handlers.MaintenanceHandler, // NUEVO: Handler de mantenimiento
	monitoringHandler *handlers.MonitoringHandler, // NUEVO: Handler de monitoring
) *Router {
	return &Router{
		engine:                       gin.New(),
		authMiddleware:               authMiddleware,
		analyticsMiddleware:          analyticsMiddleware,
		corsMiddleware:               corsMiddleware,
		rateLimiter:                  rateLimiter,                  // NUEVO
		authRateLimiter:              authRateLimiter,              // NUEVO
		isProduction:                 isProduction,                 // NUEVO
		sharedSecretKey:              sharedSecretKey,              // NUEVO
		requestFingerprintMiddleware: requestFingerprintMiddleware, // NUEVO
		authHandler:         authHandler,
		stationHandler:      stationHandler,
		analyticsHandler:    analyticsHandler,
		favoriteHandler:     favoriteHandler,
		seoHandler:          seoHandler,         // NUEVO
		translationHandler:  translationHandler, // NUEVO
		securityHandler:     securityHandler,    // NUEVO
		maintenanceHandler:  maintenanceHandler, // NUEVO
		monitoringHandler:   monitoringHandler,  // NUEVO
	}
}

// Setup sets up all routes
func (r *Router) Setup() *gin.Engine {
	// Global middleware
	r.engine.Use(gin.Recovery())
	r.engine.Use(middleware.ForceHTTPS(r.isProduction))      // NUEVO: Force HTTPS in production
	r.engine.Use(middleware.SecurityHeaders(r.isProduction)) // NUEVO: Security headers
	r.engine.Use(middleware.MaxRequestSize(10 << 20))        // NUEVO: Limit to 10MB
	r.engine.Use(r.corsMiddleware)
	r.engine.Use(middleware.LoggingMiddleware())
	r.engine.Use(r.requestFingerprintMiddleware) // NUEVO: Request source classifier
	r.engine.Use(middleware.LanguageDetector())  // NUEVO: Middleware de detección de idioma
	r.engine.Use(r.rateLimiter.Middleware())     // NUEVO: Global rate limiting
	r.engine.Use(r.analyticsMiddleware.Track())

	// Health check
	r.engine.GET("/health", r.healthCheck)

	// Swagger documentation
	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Auth routes (public) with strict rate limiting
		auth := v1.Group("/auth")
		auth.Use(r.authRateLimiter.StrictMiddleware()) // NUEVO: Rate limiting estricto para auth
		{
			auth.POST("/register", r.authHandler.Register)
			auth.POST("/login", r.authHandler.Login)
			auth.POST("/refresh", r.authHandler.RefreshToken)
			auth.POST("/validate", r.authHandler.ValidateToken)                          // NUEVO: Validate token
			auth.POST("/revoke", r.authMiddleware.Required(), r.authHandler.RevokeToken) // NUEVO: Revoke tokens
			auth.POST("/logout", r.authMiddleware.Required(), r.authHandler.Logout)      // NUEVO: Logout endpoint
			auth.GET("/me", r.authMiddleware.Required(), r.authHandler.Me)
			auth.GET("/sessions", r.authMiddleware.Required(), r.authHandler.GetSessions)                 // NUEVO: List sessions
			auth.DELETE("/sessions/:sessionId", r.authMiddleware.Required(), r.authHandler.DeleteSession) // NUEVO: Delete session
		}

		// Station routes (public with optional auth)
		stations := v1.Group("/stations")
		stations.Use(middleware.SharedSecretAuth(r.sharedSecretKey)) // NUEVO: Bot protection
		stations.Use(r.authMiddleware.Optional())
		{
			stations.GET("/popular", r.stationHandler.GetPopular)
			stations.GET("/search", r.stationHandler.Search)
			stations.GET("/:id", r.stationHandler.GetByID)
		}

		// Analytics routes (admin only)
		analytics := v1.Group("/analytics")
		analytics.Use(r.authMiddleware.Required(), r.authMiddleware.AdminOnly())
		{
			analytics.GET("/stations/popular", r.analyticsHandler.GetPopularStations)
			analytics.GET("/searches/trending", r.analyticsHandler.GetTrendingSearches)
			analytics.GET("/users/active", r.analyticsHandler.GetActiveUsers)
			analytics.GET("/users/guest", r.analyticsHandler.GetGuestUsers)
		}

		// Favorites routes (authenticated)
		favorites := v1.Group("/favorites")
		favorites.Use(r.authMiddleware.Required())
		{
			favorites.GET("", r.favoriteHandler.GetFavorites)
			favorites.POST("", r.favoriteHandler.AddFavorite)
			favorites.DELETE("/:stationId", r.favoriteHandler.RemoveFavorite)
		}

		// SEO routes (public, no auth required)
		seo := v1.Group("/seo")
		seo.Use(middleware.SharedSecretAuth(r.sharedSecretKey)) // NUEVO: Bot protection
		{
			seo.GET("/sitemap-data", r.seoHandler.GetSitemapData)
			seo.GET("/popular-tags", r.seoHandler.GetPopularTags)
			seo.GET("/popular-countries", r.seoHandler.GetPopularCountries)
		}

		// Admin SEO routes
		adminSEO := v1.Group("/admin/seo")
		adminSEO.Use(r.authMiddleware.Required(), r.authMiddleware.AdminOnly())
		{
			adminSEO.POST("/refresh-stats", r.seoHandler.RefreshSEOStats)
		}

		// Translation routes (admin only)
		adminTranslations := v1.Group("/admin/translations")
		adminTranslations.Use(r.authMiddleware.Required(), r.authMiddleware.AdminOnly())
		{
			adminTranslations.POST("", r.translationHandler.CreateTranslation)
			adminTranslations.POST("/bulk", r.translationHandler.BulkCreateTranslations)
			adminTranslations.GET("/:stationId", r.translationHandler.ListTranslations)
			adminTranslations.GET("/:stationId/:lang", r.translationHandler.GetTranslation)
			adminTranslations.PUT("/:stationId/:lang", r.translationHandler.UpdateTranslation)
			adminTranslations.DELETE("/:stationId/:lang", r.translationHandler.DeleteTranslation)
		}

		// Public translation routes
		translations := v1.Group("/translations")
		{
			translations.GET("/:stationId/languages", r.translationHandler.GetAvailableLanguages)
		}

		// Admin Security routes
		adminSecurity := v1.Group("/admin/security")
		adminSecurity.Use(r.authMiddleware.Required(), r.authMiddleware.AdminOnly())
		{
			adminSecurity.GET("/metrics", r.securityHandler.GetMetrics)
			adminSecurity.GET("/logs", r.securityHandler.GetLogs)
			adminSecurity.GET("/suspicious-sources", r.securityHandler.GetSuspiciousSources)
		}

		// Admin Maintenance routes
		adminMaintenance := v1.Group("/admin/maintenance")
		adminMaintenance.Use(r.authMiddleware.Required(), r.authMiddleware.AdminOnly())
		{
			adminMaintenance.GET("/recommendations", r.maintenanceHandler.GetRecommendations)
			adminMaintenance.POST("/refresh-views", r.maintenanceHandler.RefreshViews)
			adminMaintenance.GET("/refresh-stats", r.maintenanceHandler.GetRefreshStatistics)
			adminMaintenance.POST("/cleanup-partitions", r.maintenanceHandler.CleanupPartitions)
			adminMaintenance.GET("/check-partitions", r.maintenanceHandler.CheckPartitions)
			adminMaintenance.GET("/partition-status", r.maintenanceHandler.GetPartitionStatus)
			adminMaintenance.POST("/full", r.maintenanceHandler.PerformFullMaintenance)
		}

		// Admin Monitoring routes
		adminMonitoring := v1.Group("/admin/monitoring")
		adminMonitoring.Use(r.authMiddleware.Required(), r.authMiddleware.AdminOnly())
		{
			adminMonitoring.GET("/health", r.monitoringHandler.GetHealthMetrics)
			adminMonitoring.GET("/alerts", r.monitoringHandler.GetAlerts)
		}
	}

	return r.engine
}

// healthCheck is a simple health check endpoint
func (r *Router) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "healthy",
		"service": "radio-backend",
	})
}
