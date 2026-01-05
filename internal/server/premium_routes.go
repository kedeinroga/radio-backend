package server

import (
	"log/slog"

	"radio-backend/internal/handlers"
	"radio-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterPremiumRoutes registra todas las rutas relacionadas con suscripciones premium
func RegisterPremiumRoutes(
	router *gin.Engine,
	premiumHandler *handlers.PremiumHandler,
	webhookHandler *handlers.StripeWebhookHandler,
	authMiddleware *middleware.AuthMiddleware,
	logger *slog.Logger,
) {
	logger.Info("Registering premium subscription routes...")

	// ============================================
	// Stripe Webhook Endpoint (No Auth - Stripe signature validation)
	// ============================================
	router.POST("/api/v1/webhooks/stripe", webhookHandler.HandleWebhook)

	// ============================================
	// Premium Subscription Endpoints (Auth Required)
	// ============================================
	premiumGroup := router.Group("/api/v1/premium")
	premiumGroup.Use(authMiddleware.Required())
	{
		// Create checkout session
		// POST /api/v1/premium/checkout
		// Body: {"plan": "monthly|yearly", "email": "user@example.com"}
		premiumGroup.POST("/checkout", premiumHandler.CreateCheckoutSession)

		// Get premium status
		// GET /api/v1/premium/status
		premiumGroup.GET("/status", premiumHandler.GetPremiumStatus)

		// Cancel subscription
		// POST /api/v1/premium/cancel
		premiumGroup.POST("/cancel", premiumHandler.CancelSubscription)

		// Get customer portal URL
		// GET /api/v1/premium/portal
		premiumGroup.GET("/portal", premiumHandler.GetCustomerPortalURL)
	}

	logger.Info("✅ Premium subscription routes registered successfully")
	logger.Info("  - POST   /api/v1/webhooks/stripe (webhook)")
	logger.Info("  - POST   /api/v1/premium/checkout")
	logger.Info("  - GET    /api/v1/premium/status")
	logger.Info("  - POST   /api/v1/premium/cancel")
	logger.Info("  - GET    /api/v1/premium/portal")
}
