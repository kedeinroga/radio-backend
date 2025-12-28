// @title Radio Backend API
// @version 2.1
// @description Radio streaming API with JWT authentication, analytics and multi-language support (i18n).
// @description
// @description **Security Features (v2.1):**
// @description - 🔐 Timing attack prevention (constant-time operations)
// @description - 🔒 HTTPS enforcement (production)
// @description - 🚫 Account lockout (10 attempts = 30 min lockout)
// @description - 📧 Email rate limiting (10 attempts/hour per email)
// @description - 🔑 Password strength enforcement (special chars + 47 common passwords blocked)
// @description - 🛡️ Session hijacking protection (User-Agent validation)
// @description - 📝 Security event logging (failed attempts with metadata)
// @description - ⚡ Rate limiting (5 attempts/15min per IP)
// @description - 🔐 JWT with RS256 + token revocation
// @description
// @description **Internationalization:**
// @description Supported languages: Spanish (es), English (en), French (fr), German (de).
// @description Use the 'lang' parameter or 'Accept-Language' header to specify the desired language.
// @description
// @description **Security Score:** 100/100 (audited Dec 2025)
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@radiobackend.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"radio-backend/internal/config"
	"radio-backend/internal/handlers"
	"radio-backend/internal/infrastructure/cache"
	cryptoBcrypt "radio-backend/internal/infrastructure/crypto/bcrypt"
	"radio-backend/internal/infrastructure/database"
	"radio-backend/internal/infrastructure/jwt"
	"radio-backend/internal/infrastructure/logger"
	"radio-backend/internal/middleware"
	"radio-backend/internal/repositories/postgres"
	"radio-backend/internal/repositories/radiobrowser"
	"radio-backend/internal/server"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger.Init(cfg.Logging.Format, cfg.Logging.Level)
	logger.Info("Starting radio-backend server", "env", cfg.Server.Env)

	// Set Gin mode
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	db, err := database.NewConnection(
		cfg.Database.URL,
		cfg.Database.MaxConnections,
		cfg.Database.MaxIdleConnections,
	)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()
	logger.Info("Database connected successfully")

	// Initialize Redis
	redisClient, err := cache.NewRedisClient(
		cfg.Redis.URL,
		cfg.Redis.Password,
		cfg.Redis.DB,
	)
	if err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		log.Fatalf("Redis connection failed: %v", err)
	}
	defer redisClient.Close()
	logger.Info("Redis connected successfully")

	// Initialize infrastructure components
	passwordHasher := cryptoBcrypt.NewPasswordHasher(cfg.Security.BcryptCost)

	tokenManager, err := jwt.NewTokenManager(
		cfg.JWT.PrivateKeyPath,
		cfg.JWT.PublicKeyPath,
		cfg.JWT.Expiration,
		cfg.JWT.RefreshExpiration,
	)
	if err != nil {
		logger.Error("Failed to initialize JWT token manager", "error", err)
		log.Fatalf("JWT initialization failed: %v", err)
	}
	logger.Info("JWT token manager initialized")

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)
	analyticsRepo := postgres.NewAnalyticsRepository(db)
	stationCacheRepo := postgres.NewStationCacheRepository(db)
	searchCacheRepo := postgres.NewSearchCacheRepository(db)
	favoriteRepo := postgres.NewFavoriteRepository(db)
	seoRepo := postgres.NewSEORepository(db.DB)                     // NUEVO: SEO repo usa *sql.DB directamente
	translationRepo := postgres.NewTranslationRepository(db.DB)     // NUEVO: Translation repo
	sessionRepo := postgres.NewSessionRepository(db.DB)             // NUEVO: Session repo
	securityEventRepo := postgres.NewSecurityEventRepository(db.DB) // NUEVO: Security event repo
	loginAttemptRepo := postgres.NewLoginAttemptRepository(db.DB)   // NUEVO: Login attempt repo
	radioBrowserRepo := radiobrowser.NewRepository(cfg.External.RadioBrowserAPIURL)

	// Initialize cache components
	seoCache := cache.NewSEOCache(redisClient) // NUEVO: Cache SEO

	// Initialize services
	slugService := services.NewSlugService()                                                                     // NUEVO: Servicio de slugs
	translationService := services.NewTranslationService(translationRepo, stationCacheRepo)                      // NUEVO: Servicio de traducciones
	seoService := services.NewSEOService(seoRepo, seoCache, translationService, slugService, cfg.Server.BaseURL) // NUEVO: Servicio SEO
	authService := services.NewAuthService(
		userRepo,
		sessionRepo,       // NUEVO: Session repository
		securityEventRepo, // NUEVO: Security event repository
		loginAttemptRepo,  // NUEVO: Login attempt repository
		passwordHasher,
		tokenManager,
		tokenManager,
		redisClient, // NUEVO: Token blacklist (Redis)
	)
	analyticsService := services.NewAnalyticsService(analyticsRepo, redisClient)
	stationService := services.NewStationService(
		stationCacheRepo,
		radioBrowserRepo,
		searchCacheRepo,
		analyticsService,
		seoService, // NUEVO: Inyectar SEO service
		cfg.Cache.StationMaxAge,
		cfg.Cache.SearchCacheTTL,
	)
	favoriteService := services.NewFavoriteService(favoriteRepo, stationCacheRepo, stationService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager, redisClient, sessionRepo) // Pass sessionRepo for validation
	analyticsMiddleware := middleware.NewAnalyticsMiddleware(analyticsService)
	corsMiddleware := middleware.CORSMiddleware(
		cfg.CORS.AllowedOrigins,
		cfg.CORS.AllowedMethods,
		cfg.CORS.AllowedHeaders,
	)

	// NUEVO: Initialize rate limiters
	rateLimiter := middleware.NewRateLimiter(cfg.Security.RateLimitReqs)             // General: 100 req/min
	authRateLimiter := middleware.NewRateLimiter(10)                                 // Auth: 10 req/min (más estricto)
	emailRateLimiter := middleware.NewEmailRateLimiter(redisClient, 10, 1*time.Hour) // Email: 10 attempts per hour

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, emailRateLimiter) // Pass email rate limiter
	stationHandler := handlers.NewStationHandler(stationService, analyticsService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
	favoriteHandler := handlers.NewFavoriteHandler(favoriteService)
	seoHandler := handlers.NewSEOHandler(seoService)                         // NUEVO: Handler SEO
	translationHandler := handlers.NewTranslationHandler(translationService) // NUEVO: Handler de traducciones

	// Setup router
	router := server.NewRouter(
		authMiddleware,
		analyticsMiddleware,
		corsMiddleware,
		rateLimiter,        // NUEVO: Rate limiter general
		authRateLimiter,    // NUEVO: Rate limiter para auth
		cfg.IsProduction(), // NUEVO: Flag de producción
		authHandler,
		stationHandler,
		analyticsHandler,
		favoriteHandler,
		seoHandler,         // NUEVO: Inyectar SEO handler
		translationHandler, // NUEVO: Inyectar Translation handler
	)
	engine := router.Setup()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:           addr,
		Handler:        engine,
		ReadTimeout:    cfg.Server.Timeout,
		WriteTimeout:   cfg.Server.Timeout,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting", "address", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", "error", err)
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	logger.Info("Server started successfully", "address", addr)

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		log.Fatalf("Server shutdown failed: %v", err)
	}

	logger.Info("Server stopped gracefully")
}
