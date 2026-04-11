package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
)

// StripeWebhookHandler maneja webhooks de Stripe
type StripeWebhookHandler struct {
	premiumService *services.PremiumService
	webhookSecret  string
	logger         *slog.Logger
}

// NewStripeWebhookHandler crea una nueva instancia del handler
func NewStripeWebhookHandler(
	premiumService *services.PremiumService,
	webhookSecret string,
	logger *slog.Logger,
) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		premiumService: premiumService,
		webhookSecret:  webhookSecret,
		logger:         logger,
	}
}

// HandleWebhook procesa eventos de Stripe
// @Summary Handle Stripe webhook
// @Tags premium
// @Accept  json
// @Produce json
// @Param Stripe-Signature header string true "Stripe webhook signature"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H "Invalid body, missing/invalid signature, or unparseable event"
// @Failure 500 {object} gin.H "Failed to process checkout, subscription update, or deletion"
// @Router /webhooks/stripe [post]
func (h *StripeWebhookHandler) HandleWebhook(c *gin.Context) {
	const MaxBodyBytes = int64(65536) // 64KB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	// Leer el body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read webhook body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// Verificar firma de Stripe
	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		h.logger.Error("missing stripe signature header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing signature"})
		return
	}

	// Construir evento verificando la firma
	event, err := webhook.ConstructEvent(body, signature, h.webhookSecret)
	if err != nil {
		h.logger.Error("failed to verify webhook signature",
			"error", err,
			"signature", signature,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
		return
	}

	h.logger.Info("received stripe webhook",
		"event_type", event.Type,
		"event_id", event.ID,
	)

	// Procesar según tipo de evento
	switch event.Type {
	case "checkout.session.completed":
		h.handleCheckoutCompleted(c, event)
	case "customer.subscription.created":
		h.handleSubscriptionCreated(c, event)
	case "customer.subscription.updated":
		h.handleSubscriptionUpdated(c, event)
	case "customer.subscription.deleted":
		h.handleSubscriptionDeleted(c, event)
	case "invoice.payment_succeeded":
		h.handleInvoicePaymentSucceeded(c, event)
	case "invoice.payment_failed":
		h.handleInvoicePaymentFailed(c, event)
	default:
		h.logger.Info("unhandled webhook event type", "type", event.Type)
		c.JSON(http.StatusOK, gin.H{"received": true})
	}
}

// handleCheckoutCompleted procesa sesiones de checkout completadas
func (h *StripeWebhookHandler) handleCheckoutCompleted(c *gin.Context, event stripe.Event) {
	var session stripe.CheckoutSession
	if err := h.unmarshalEvent(event, &session); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse event"})
		return
	}

	h.logger.Info("processing checkout completed",
		"session_id", session.ID,
		"customer", session.Customer,
		"subscription", session.Subscription,
	)

	// Procesar checkout completado
	if err := h.premiumService.HandleCheckoutComplete(c.Request.Context(), session.ID); err != nil {
		h.logger.Error("failed to handle checkout complete",
			"error", err,
			"session_id", session.ID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process checkout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// handleSubscriptionCreated procesa nuevas suscripciones
func (h *StripeWebhookHandler) handleSubscriptionCreated(c *gin.Context, event stripe.Event) {
	var subscription stripe.Subscription
	if err := h.unmarshalEvent(event, &subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse event"})
		return
	}

	h.logger.Info("subscription created",
		"subscription_id", subscription.ID,
		"customer", subscription.Customer,
		"status", subscription.Status,
	)

	// Por ahora, solo logeamos. El checkout.session.completed se encarga de activar premium
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// handleSubscriptionUpdated procesa actualizaciones de suscripciones
func (h *StripeWebhookHandler) handleSubscriptionUpdated(c *gin.Context, event stripe.Event) {
	var subscription stripe.Subscription
	if err := h.unmarshalEvent(event, &subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse event"})
		return
	}

	h.logger.Info("subscription updated",
		"subscription_id", subscription.ID,
		"status", subscription.Status,
		"current_period_end", subscription.CurrentPeriodEnd,
	)

	// Actualizar suscripción
	if err := h.premiumService.HandleSubscriptionUpdated(c.Request.Context(), subscription.ID); err != nil {
		h.logger.Error("failed to handle subscription update",
			"error", err,
			"subscription_id", subscription.ID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process update"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// handleSubscriptionDeleted procesa eliminación de suscripciones
func (h *StripeWebhookHandler) handleSubscriptionDeleted(c *gin.Context, event stripe.Event) {
	var subscription stripe.Subscription
	if err := h.unmarshalEvent(event, &subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse event"})
		return
	}

	h.logger.Info("subscription deleted",
		"subscription_id", subscription.ID,
		"customer", subscription.Customer,
	)

	// Desactivar premium
	if err := h.premiumService.HandleSubscriptionDeleted(c.Request.Context(), subscription.ID); err != nil {
		h.logger.Error("failed to handle subscription deletion",
			"error", err,
			"subscription_id", subscription.ID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process deletion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// handleInvoicePaymentSucceeded procesa pagos exitosos
func (h *StripeWebhookHandler) handleInvoicePaymentSucceeded(c *gin.Context, event stripe.Event) {
	var invoice stripe.Invoice
	if err := h.unmarshalEvent(event, &invoice); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse event"})
		return
	}

	h.logger.Info("invoice payment succeeded",
		"invoice_id", invoice.ID,
		"customer", invoice.Customer,
		"subscription", invoice.Subscription,
		"amount_paid", invoice.AmountPaid,
	)

	// Registrar pago exitoso (para analytics/billing)
	// Por ahora solo logeamos
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// handleInvoicePaymentFailed procesa pagos fallidos
func (h *StripeWebhookHandler) handleInvoicePaymentFailed(c *gin.Context, event stripe.Event) {
	var invoice stripe.Invoice
	if err := h.unmarshalEvent(event, &invoice); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse event"})
		return
	}

	h.logger.Warn("invoice payment failed",
		"invoice_id", invoice.ID,
		"customer", invoice.Customer,
		"subscription", invoice.Subscription,
		"amount_due", invoice.AmountDue,
	)

	// TODO: Enviar email de notificación al usuario
	// TODO: Actualizar estado de suscripción si es necesario
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// unmarshalEvent es un helper para deserializar eventos
func (h *StripeWebhookHandler) unmarshalEvent(event stripe.Event, v interface{}) error {
	if err := json.Unmarshal(event.Data.Raw, v); err != nil {
		h.logger.Error("failed to unmarshal event data",
			"error", err,
			"event_type", event.Type,
		)
		return fmt.Errorf("failed to unmarshal event data: %w", err)
	}
	return nil
}
