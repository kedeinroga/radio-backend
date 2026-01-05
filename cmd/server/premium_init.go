package main

import (
	"fmt"
	"log/slog"

	"radio-backend/internal/handlers"
	"radio-backend/internal/infrastructure/database"
	"radio-backend/internal/middleware"
	"radio-backend/internal/repositories/postgres"
	"radio-backend/internal/server"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// InitializePremiumSystem inicializa el sistema de suscripciones premium
func InitializePremiumSystem(
	router *gin.Engine,
	db *database.Connection,
	authMiddleware *middleware.AuthMiddleware,
	logger *slog.Logger,
	stripeSecretKey string,
	stripeWebhookSecret string,
	stripePriceMonthly string,
	stripePriceYearly string,
	stripeSuccessURL string,
	stripeCancelURL string,
) error {
	logger.Info("Initializing premium subscription system...")

	// Verificar configuración de Stripe
	if stripeSecretKey == "" {
		logger.Warn("Stripe secret key not configured, premium features will be disabled")
		return fmt.Errorf("stripe secret key not configured")
	}

	if stripeWebhookSecret == "" {
		logger.Warn("Stripe webhook secret not configured, webhook events will not be validated")
	}

	// ============================================
	// 1. Initialize Repositories
	// ============================================
	userAdProfileRepo := postgres.NewUserAdProfileRepository(db)
	logger.Info("Premium repositories initialized")

	// ============================================
	// 2. Initialize Services
	// ============================================
	premiumService := services.NewPremiumService(
		userAdProfileRepo,
		stripeSecretKey,
		stripePriceMonthly,
		stripePriceYearly,
		stripeSuccessURL,
		stripeCancelURL,
		logger,
	)
	logger.Info("Premium service initialized")

	// ============================================
	// 3. Initialize Handlers
	// ============================================
	premiumHandler := handlers.NewPremiumHandler(premiumService, logger)
	webhookHandler := handlers.NewStripeWebhookHandler(
		premiumService,
		stripeWebhookSecret,
		logger,
	)
	logger.Info("Premium handlers initialized")

	// ============================================
	// 4. Register Routes
	// ============================================
	server.RegisterPremiumRoutes(
		router,
		premiumHandler,
		webhookHandler,
		authMiddleware,
		logger,
	)

	logger.Info("✅ Premium subscription system fully initialized and ready")
	logger.Info("  - Stripe checkout sessions enabled")
	logger.Info("  - Webhook endpoint: POST /api/v1/webhooks/stripe")
	logger.Info("  - Premium endpoints: 4 routes registered")

	return nil
}
