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
	stationRepo := radiobrowser.NewRepository(cfg.External.RadioBrowserAPIURL)

	// Initialize services
	authService := services.NewAuthService(userRepo, passwordHasher, tokenManager, tokenManager)
	analyticsService := services.NewAnalyticsService(analyticsRepo, redisClient)
	stationService := services.NewStationService(stationRepo, analyticsService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)
	analyticsMiddleware := middleware.NewAnalyticsMiddleware(analyticsService)
	corsMiddleware := middleware.CORSMiddleware(
		cfg.CORS.AllowedOrigins,
		cfg.CORS.AllowedMethods,
		cfg.CORS.AllowedHeaders,
	)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	stationHandler := handlers.NewStationHandler(stationService, analyticsService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)

	// Setup router
	router := server.NewRouter(
		authMiddleware,
		analyticsMiddleware,
		corsMiddleware,
		authHandler,
		stationHandler,
		analyticsHandler,
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
