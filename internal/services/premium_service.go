package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"radio-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/subscription"
)

// PremiumService maneja suscripciones premium con Stripe
type PremiumService struct {
	userAdProfileRepo domain.UserAdProfileRepository
	stripeSecretKey   string
	priceIDMonthly    string
	priceIDYearly     string
	successURL        string
	cancelURL         string
	logger            *slog.Logger
}

// NewPremiumService crea una nueva instancia del servicio premium
func NewPremiumService(
	userAdProfileRepo domain.UserAdProfileRepository,
	stripeSecretKey string,
	priceIDMonthly string,
	priceIDYearly string,
	successURL string,
	cancelURL string,
	logger *slog.Logger,
) *PremiumService {
	// Configurar Stripe API key
	stripe.Key = stripeSecretKey

	return &PremiumService{
		userAdProfileRepo: userAdProfileRepo,
		stripeSecretKey:   stripeSecretKey,
		priceIDMonthly:    priceIDMonthly,
		priceIDYearly:     priceIDYearly,
		successURL:        successURL,
		cancelURL:         cancelURL,
		logger:            logger,
	}
}

// CreateCheckoutSession crea una sesión de checkout de Stripe
func (s *PremiumService) CreateCheckoutSession(userID uuid.UUID, plan string, email string) (*stripe.CheckoutSession, error) {
	s.logger.Info("creating checkout session",
		"user_id", userID,
		"plan", plan,
		"email", email,
	)

	// Determinar precio según plan
	priceID := s.priceIDMonthly
	if plan == "yearly" {
		priceID = s.priceIDYearly
	}

	if priceID == "" {
		return nil, fmt.Errorf("stripe price ID not configured for plan: %s", plan)
	}

	// Crear parámetros de checkout session
	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(s.successURL + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(s.cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		ClientReferenceID: stripe.String(userID.String()),
		CustomerEmail:     stripe.String(email),
		Metadata: map[string]string{
			"user_id": userID.String(),
			"plan":    plan,
		},
		AllowPromotionCodes: stripe.Bool(true),
	}

	// Crear sesión
	sess, err := checkoutsession.New(params)
	if err != nil {
		s.logger.Error("failed to create checkout session",
			"error", err,
			"user_id", userID,
		)
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	s.logger.Info("checkout session created",
		"session_id", sess.ID,
		"user_id", userID,
	)

	return sess, nil
}

// HandleCheckoutComplete procesa una sesión de checkout completada
func (s *PremiumService) HandleCheckoutComplete(ctx context.Context, sessionID string) error {
	s.logger.Info("handling checkout complete", "session_id", sessionID)

	// Recuperar sesión de Stripe
	sess, err := checkoutsession.Get(sessionID, nil)
	if err != nil {
		s.logger.Error("failed to retrieve session", "error", err, "session_id", sessionID)
		return fmt.Errorf("failed to retrieve session: %w", err)
	}

	// Verificar que el pago fue exitoso
	if sess.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		s.logger.Warn("payment not completed",
			"session_id", sessionID,
			"payment_status", sess.PaymentStatus,
		)
		return fmt.Errorf("payment not completed: %s", sess.PaymentStatus)
	}

	// Extraer user_id de metadata
	userIDStr, ok := sess.Metadata["user_id"]
	if !ok {
		return fmt.Errorf("user_id not found in session metadata")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user_id in metadata: %w", err)
	}

	// Recuperar suscripción
	if sess.Subscription == nil {
		return fmt.Errorf("no subscription found in session")
	}

	sub, err := subscription.Get(sess.Subscription.ID, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve subscription: %w", err)
	}

	// Activar premium en el perfil del usuario
	expiresAt := time.Unix(sub.CurrentPeriodEnd, 0)

	if err := s.ActivatePremium(ctx, userID, sess.Customer.ID, sess.Subscription.ID, expiresAt); err != nil {
		return fmt.Errorf("failed to activate premium: %w", err)
	}

	s.logger.Info("premium activated successfully",
		"user_id", userID,
		"subscription_id", sess.Subscription.ID,
		"expires_at", expiresAt,
	)

	return nil
}

// ActivatePremium activa la suscripción premium para un usuario
func (s *PremiumService) ActivatePremium(
	ctx context.Context,
	userID uuid.UUID,
	stripeCustomerID string,
	stripeSubscriptionID string,
	expiresAt time.Time,
) error {
	s.logger.Info("activating premium",
		"user_id", userID,
		"customer_id", stripeCustomerID,
		"subscription_id", stripeSubscriptionID,
	)

	// Obtener perfil del usuario
	profile, err := s.userAdProfileRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user profile: %w", err)
	}

	// Actualizar a premium
	profile.IsPremium = true
	profile.PremiumExpiresAt = &expiresAt
	profile.StripeCustomerID = &stripeCustomerID
	profile.StripeSubscriptionID = &stripeSubscriptionID
	profile.UpdatedAt = time.Now()

	// Guardar cambios
	if err := s.userAdProfileRepo.Update(profile); err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	s.logger.Info("premium activated",
		"user_id", userID,
		"expires_at", expiresAt,
	)

	return nil
}

// DeactivatePremium desactiva la suscripción premium
func (s *PremiumService) DeactivatePremium(ctx context.Context, userID uuid.UUID) error {
	s.logger.Info("deactivating premium", "user_id", userID)

	// Obtener perfil
	profile, err := s.userAdProfileRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user profile: %w", err)
	}

	if !profile.IsPremium {
		return fmt.Errorf("user is not premium")
	}

	// Cancelar suscripción en Stripe si existe
	if profile.StripeSubscriptionID != nil {
		_, err := subscription.Cancel(*profile.StripeSubscriptionID, nil)
		if err != nil {
			s.logger.Error("failed to cancel stripe subscription",
				"error", err,
				"subscription_id", *profile.StripeSubscriptionID,
			)
			// Continuar con la desactivación local aunque falle en Stripe
		}
	}

	// Desactivar premium
	profile.IsPremium = false
	profile.PremiumExpiresAt = nil
	profile.StripeSubscriptionID = nil

	// Guardar cambios
	if err := s.userAdProfileRepo.Update(profile); err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	s.logger.Info("premium deactivated", "user_id", userID)

	return nil
}

// HandleSubscriptionDeleted maneja la eliminación de una suscripción
func (s *PremiumService) HandleSubscriptionDeleted(ctx context.Context, subscriptionID string) error {
	s.logger.Info("handling subscription deleted", "subscription_id", subscriptionID)

	// Buscar perfil por subscription_id
	profile, err := s.userAdProfileRepo.GetByStripeSubscriptionID(subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to find profile with subscription_id: %w", err)
	}

	// Desactivar premium
	profile.IsPremium = false
	profile.PremiumExpiresAt = nil
	profile.StripeSubscriptionID = nil

	// Guardar cambios
	if err := s.userAdProfileRepo.Update(profile); err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	s.logger.Info("premium deactivated due to subscription deletion",
		"user_id", profile.UserID,
		"subscription_id", subscriptionID,
	)

	return nil
}

// HandleSubscriptionUpdated maneja la actualización de una suscripción
func (s *PremiumService) HandleSubscriptionUpdated(ctx context.Context, subscriptionID string) error {
	s.logger.Info("handling subscription updated", "subscription_id", subscriptionID)

	// Recuperar suscripción de Stripe
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve subscription: %w", err)
	}

	// Buscar perfil
	profile, err := s.userAdProfileRepo.GetByStripeSubscriptionID(subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to find profile: %w", err)
	}

	// Actualizar fecha de expiración
	expiresAt := time.Unix(sub.CurrentPeriodEnd, 0)
	profile.PremiumExpiresAt = &expiresAt

	// Actualizar estado según el status de la suscripción
	if sub.Status == stripe.SubscriptionStatusActive || sub.Status == stripe.SubscriptionStatusTrialing {
		profile.IsPremium = true
	} else {
		profile.IsPremium = false
	}

	// Guardar cambios
	if err := s.userAdProfileRepo.Update(profile); err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	s.logger.Info("subscription updated",
		"user_id", profile.UserID,
		"subscription_id", subscriptionID,
		"status", sub.Status,
		"expires_at", expiresAt,
	)

	return nil
}

// GetCustomerPortalURL genera URL al portal del cliente de Stripe
func (s *PremiumService) GetCustomerPortalURL(userID uuid.UUID) (string, error) {
	s.logger.Info("getting customer portal URL", "user_id", userID)

	// Obtener perfil
	profile, err := s.userAdProfileRepo.GetByUserID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user profile: %w", err)
	}

	if profile.StripeCustomerID == nil {
		return "", fmt.Errorf("user has no stripe customer ID")
	}

	// Crear sesión del portal del cliente
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(*profile.StripeCustomerID),
		ReturnURL: stripe.String(s.successURL),
	}

	portalSession, err := portalsession.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create portal session: %w", err)
	}

	return portalSession.URL, nil
}

// CheckExpiredSubscriptions verifica y desactiva suscripciones expiradas
func (s *PremiumService) CheckExpiredSubscriptions(ctx context.Context) (int, error) {
	s.logger.Info("checking for expired subscriptions")

	// Esta función debería ser llamada por un cron job
	// Por ahora, retornamos 0 (no implementado)
	// En una implementación completa, buscaríamos perfiles con premium expirado

	return 0, nil
}

// GetUserProfile obtiene el perfil de un usuario
func (s *PremiumService) GetUserProfile(ctx context.Context, userID uuid.UUID) (*domain.UserAdProfile, error) {
	return s.userAdProfileRepo.GetByUserID(userID)
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}
