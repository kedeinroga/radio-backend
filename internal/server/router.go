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
	authMiddleware      *middleware.AuthMiddleware
	analyticsMiddleware *middleware.AnalyticsMiddleware
	corsMiddleware      gin.HandlerFunc

	// Handlers
	authHandler        *handlers.AuthHandler
	stationHandler     *handlers.StationHandler
	analyticsHandler   *handlers.AnalyticsHandler
	favoriteHandler    *handlers.FavoriteHandler
	seoHandler         *handlers.SEOHandler         // NUEVO: Handler SEO
	translationHandler *handlers.TranslationHandler // NUEVO: Handler de traducciones
}

// NewRouter creates a new router
func NewRouter(
	authMiddleware *middleware.AuthMiddleware,
	analyticsMiddleware *middleware.AnalyticsMiddleware,
	corsMiddleware gin.HandlerFunc,
	authHandler *handlers.AuthHandler,
	stationHandler *handlers.StationHandler,
	analyticsHandler *handlers.AnalyticsHandler,
	favoriteHandler *handlers.FavoriteHandler,
	seoHandler *handlers.SEOHandler, // NUEVO: Handler SEO
	translationHandler *handlers.TranslationHandler, // NUEVO: Handler de traducciones
) *Router {
	return &Router{
		engine:              gin.New(),
		authMiddleware:      authMiddleware,
		analyticsMiddleware: analyticsMiddleware,
		corsMiddleware:      corsMiddleware,
		authHandler:         authHandler,
		stationHandler:      stationHandler,
		analyticsHandler:    analyticsHandler,
		favoriteHandler:     favoriteHandler,
		seoHandler:          seoHandler,         // NUEVO
		translationHandler:  translationHandler, // NUEVO
	}
}

// Setup sets up all routes
func (r *Router) Setup() *gin.Engine {
	// Global middleware
	r.engine.Use(gin.Recovery())
	r.engine.Use(r.corsMiddleware)
	r.engine.Use(middleware.LoggingMiddleware())
	r.engine.Use(middleware.LanguageDetector()) // NUEVO: Middleware de detección de idioma
	r.engine.Use(r.analyticsMiddleware.Track())

	// Health check
	r.engine.GET("/health", r.healthCheck)

	// Swagger documentation
	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", r.authHandler.Register)
			auth.POST("/login", r.authHandler.Login)
			auth.POST("/refresh", r.authHandler.RefreshToken)
			auth.GET("/me", r.authMiddleware.Required(), r.authHandler.Me)
		}

		// Station routes (public with optional auth)
		stations := v1.Group("/stations")
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
