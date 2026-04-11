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
// @description - 🤖 Bot protection via shared secret (X-Rradio-Secret header)
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

// @securityDefinitions.apikey SharedSecret
// @in header
// @name X-Rradio-Secret
// @description Shared secret key required for public station and SEO endpoints (bot protection).

package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
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

	// Run pending migrations before starting the server
	if err := database.RunMigrations(cfg.Database.URL, database.DefaultMigrateConfig()); err != nil {
		logger.Error("Database migrations failed", "error", err)
		log.Fatalf("Migration failed: %v", err)
	}

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

	// Prefer raw PEM content from environment variables if provided, otherwise use file paths
	privateKeyInput := cfg.JWT.PrivateKeyPath
	if cfg.JWT.PrivateKey != "" {
		privateKeyInput = cfg.JWT.PrivateKey
	}

	publicKeyInput := cfg.JWT.PublicKeyPath
	if cfg.JWT.PublicKey != "" {
		publicKeyInput = cfg.JWT.PublicKey
	}

	tokenManager, err := jwt.NewTokenManager(
		privateKeyInput,
		publicKeyInput,
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
	securityRepo := postgres.NewSecurityRepository(db.DB)           // NUEVO: Security repo
	maintenanceRepo := postgres.NewMaintenanceRepository(db.DB)     // NUEVO: Maintenance repo
	radioBrowserRepo := radiobrowser.NewRepository(cfg.External.RadioBrowserAPIURL)

	// Initialize cache components
	seoCache := cache.NewSEOCache(redisClient) // NUEVO: Cache SEO

	// Initialize services
	slugService := services.NewSlugService()                                                                     // NUEVO: Servicio de slugs
	translationService := services.NewTranslationService(translationRepo, stationCacheRepo)                      // NUEVO: Servicio de traducciones
	seoService := services.NewSEOService(seoRepo, seoCache, translationService, slugService, cfg.Server.BaseURL) // NUEVO: Servicio SEO
	securityService := services.NewSecurityService(securityRepo)                                                 // NUEVO: Servicio de seguridad
	maintenanceService := services.NewMaintenanceService(maintenanceRepo)                                        // NUEVO: Servicio de mantenimiento
	monitoringService := services.NewMonitoringService(maintenanceService)                                       // NUEVO: Servicio de monitoring
	tokenBlacklistRepo := postgres.NewTokenBlacklistRepository(db.DB)                                            // NUEVO: Token blacklist repository (PostgreSQL)
	authService := services.NewAuthService(
		userRepo,
		sessionRepo,       // NUEVO: Session repository
		securityEventRepo, // NUEVO: Security event repository
		loginAttemptRepo,  // NUEVO: Login attempt repository
		passwordHasher,
		tokenManager,
		tokenManager,
		tokenBlacklistRepo, // NUEVO: Token blacklist (PostgreSQL)
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
	authMiddleware := middleware.NewAuthMiddleware(tokenManager, tokenBlacklistRepo, sessionRepo) // Pass tokenBlacklistRepo for validation
	analyticsMiddleware := middleware.NewAnalyticsMiddleware(analyticsService)
	fingerprintMiddleware := middleware.RequestFingerprintMiddleware(securityRepo, cfg.CORS.AllowedOrigins)
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
	securityHandler := handlers.NewSecurityHandler(securityService)          // NUEVO: Handler de seguridad
	maintenanceHandler := handlers.NewMaintenanceHandler(maintenanceService) // NUEVO: Handler de mantenimiento
	monitoringHandler := handlers.NewMonitoringHandler(monitoringService)    // NUEVO: Handler de monitoring

	// Setup router
	if cfg.Security.APISecretKey == "" {
		logger.Warn("API_SECRET_KEY is not set — shared secret middleware is DISABLED. Set it in production.")
	}
	router := server.NewRouter(
		authMiddleware,
		analyticsMiddleware,
		corsMiddleware,
		rateLimiter,               // NUEVO: Rate limiter general
		authRateLimiter,           // NUEVO: Rate limiter para auth
		cfg.IsProduction(),        // NUEVO: Flag de producción
		cfg.Security.APISecretKey, // NUEVO: Shared secret for bot protection
		fingerprintMiddleware,     // NUEVO: Request source classifier
		authHandler,
		stationHandler,
		analyticsHandler,
		favoriteHandler,
		seoHandler,         // NUEVO: Inyectar SEO handler
		translationHandler, // NUEVO: Inyectar Translation handler
		securityHandler,    // NUEVO: Inyectar Security handler
		maintenanceHandler, // NUEVO: Inyectar Maintenance handler
		monitoringHandler,  // NUEVO: Inyectar Monitoring handler
	)
	engine := router.Setup()

	// ============================================
	// NUEVO: Initialize Advertising System (Phase 4)
	// ============================================
	// Get raw redis client for advertising system
	adLogger := slog.Default()
	if err := InitializeAdvertisingSystem(engine, db, redisClient.GetClient(), adLogger); err != nil {
		logger.Error("Failed to initialize advertising system", "error", err)
		log.Printf("Warning: Advertising system could not be initialized: %v", err)
		// No fatal - permitir que el servidor continúe sin publicidad
	}

	// ============================================
	// NUEVO: Initialize Premium Subscription System (Phase 5)
	// ============================================
	if err := InitializePremiumSystem(
		engine,
		db,
		authMiddleware,
		adLogger,
		cfg.Stripe.SecretKey,
		cfg.Stripe.WebhookSecret,
		cfg.Stripe.PriceIDMonthly,
		cfg.Stripe.PriceIDYearly,
		cfg.Stripe.SuccessURL,
		cfg.Stripe.CancelURL,
	); err != nil {
		logger.Error("Failed to initialize premium system", "error", err)
		log.Printf("Warning: Premium subscription system could not be initialized: %v", err)
		// No fatal - permitir que el servidor continúe sin premium
	}

	// ============================================
	// NUEVO: Initialize Streaming System (Phase 6)
	// ============================================
	// El sistema de streaming requiere:
	// - stationRepo (radioBrowserRepo implementa StationRepository)
	// - adRepo, impressionRepo (del advertising system)
	// - JWT secret para tokens de stream

	// Crear repositorios de publicidad (necesarios para streaming)
	adRepo := postgres.NewAdvertisementRepository(db)
	impressionRepo := postgres.NewAdImpressionRepository(db)

	// Inicializar streaming system
	var streamSessionService *services.StreamSessionService
	if err := InitializeStreamingSystem(
		engine,
		db,
		radioBrowserRepo, // StationRepository
		adRepo,
		impressionRepo,
		authMiddleware,
		analyticsService,
		cfg.Security.APISecretKey,
		adLogger,
		[]byte(cfg.Security.APISecretKey), // JWT secret para stream tokens (reutiliza API_SECRET_KEY)
	); err != nil {
		adLogger.Error("Failed to initialize streaming system", "error", err)
		log.Printf("Warning: Streaming system could not be initialized: %v", err)
		// No fatal - permitir que el servidor continúe sin streaming
	} else {
		// Obtener StreamSessionService para el job system
		streamSessionRepo := postgres.NewStreamSessionRepository(db)
		streamSessionService = services.NewStreamSessionService(
			streamSessionRepo,
			radioBrowserRepo,
			adRepo,
			impressionRepo,
			[]byte(cfg.Security.APISecretKey),
			5*time.Minute,
			adLogger,
		)
	}

	// Inicializar Job System (Background Jobs)
	var jobScheduler interface{ Stop() }
	if streamSessionService != nil {
		scheduler, err := InitializeJobSystem(db, streamSessionService, adLogger)
		if err != nil {
			adLogger.Error("Failed to initialize job system", "error", err)
			log.Printf("Warning: Job system could not be initialized: %v", err)
		} else {
			jobScheduler = scheduler
			scheduler.Start()

			// Registrar rutas de administración de jobs
			RegisterJobRoutes(engine, scheduler)

			adLogger.Info("Job system initialized and started successfully")
		}
	}

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

	// Detener el job scheduler primero (si existe)
	if jobScheduler != nil {
		logger.Info("Stopping job scheduler...")
		jobScheduler.Stop()
		logger.Info("Job scheduler stopped")
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		log.Fatalf("Server shutdown failed: %v", err)
	}

	logger.Info("Server stopped gracefully")
}
