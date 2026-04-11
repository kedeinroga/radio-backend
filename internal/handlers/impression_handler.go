package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"radio-backend/internal/domain"
	"radio-backend/internal/services"
)

// ImpressionHandler handles HTTP requests for ad impressions
type ImpressionHandler struct {
	impressionService *services.ImpressionService
	clickService      *services.ClickService
	logger            *slog.Logger
}

// NewImpressionHandler creates a new impression handler
func NewImpressionHandler(
	impressionService *services.ImpressionService,
	clickService *services.ClickService,
	logger *slog.Logger,
) *ImpressionHandler {
	return &ImpressionHandler{
		impressionService: impressionService,
		clickService:      clickService,
		logger:            logger,
	}
}

// TrackImpressionRequest represents the request body for tracking an impression
type TrackImpressionRequest struct {
	AdvertisementID string  `json:"advertisement_id" binding:"required,uuid"`
	UserID          *string `json:"user_id,omitempty" binding:"omitempty,uuid"`
	SessionID       string  `json:"session_id" binding:"required"`
	StationID       *string `json:"station_id,omitempty"`
	IPAddress       string  `json:"ip_address" binding:"required,ip"`
	UserAgent       string  `json:"user_agent" binding:"required"`
	CountryCode     *string `json:"country_code,omitempty"`
	City            *string `json:"city,omitempty"`
	DeviceType      string  `json:"device_type" binding:"required,oneof=mobile tablet desktop"`
}

// TrackClickRequest represents the request body for tracking a click
type TrackClickRequest struct {
	ImpressionID    string  `json:"impression_id" binding:"required,uuid"`
	AdvertisementID string  `json:"advertisement_id" binding:"required,uuid"`
	UserID          *string `json:"user_id,omitempty" binding:"omitempty,uuid"`
	IPAddress       string  `json:"ip_address" binding:"required,ip"`
	UserAgent       string  `json:"user_agent" binding:"required"`
	Referrer        *string `json:"referrer,omitempty"`
	ClickPosition   *string `json:"click_position,omitempty"`
}

// TrackConversionRequest represents the request body for tracking a conversion
type TrackConversionRequest struct {
	ConversionValueCents int `json:"conversion_value_cents" binding:"required,min=0"`
}

// TrackImpression tracks an ad impression
// @Summary Track ad impression
// @Description Records an ad impression event. Returns 403 for suspicious activity and 429 when the frequency cap is exceeded.
// @Tags Ad Tracking
// @Accept json
// @Produce json
// @Param request body TrackImpressionRequest true "Impression data"
// @Success 201 {object} map[string]interface{} "Impression recorded with impression_id and impression_token"
// @Failure 400 {object} map[string]interface{} "Invalid request body or UUID fields"
// @Failure 403 {object} map[string]interface{} "Suspicious activity detected"
// @Failure 429 {object} map[string]interface{} "Frequency cap exceeded"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/tracking/impressions [post]
func (h *ImpressionHandler) TrackImpression(c *gin.Context) {
	var req TrackImpressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid impression tracking request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adID, err := uuid.Parse(req.AdvertisementID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid advertisement_id"})
		return
	}

	var userIDPtr *uuid.UUID
	if req.UserID != nil {
		parsedUserID, err := uuid.Parse(*req.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}
		userIDPtr = &parsedUserID
	}

	impression := &domain.AdImpression{
		ID:              uuid.New(),
		AdvertisementID: adID,
		UserID:          userIDPtr,
		SessionID:       req.SessionID,
		StationID:       req.StationID,
		IPAddress:       req.IPAddress,
		UserAgent:       req.UserAgent,
		CountryCode:     req.CountryCode,
		City:            req.City,
		DeviceType:      req.DeviceType,
		Viewable:        false,
		CreatedAt:       time.Now(),
	}

	if err := h.impressionService.RecordImpression(c.Request.Context(), impression); err != nil {
		h.logger.Error("Failed to record impression", "error", err)

		// Check if fraud was detected
		if err == domain.ErrSuspiciousActivity {
			c.JSON(http.StatusForbidden, gin.H{"error": "suspicious activity detected"})
			return
		}

		if err == domain.ErrFrequencyCapExceeded {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "frequency cap exceeded"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record impression"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"impression_id":    impression.ID,
		"impression_token": impression.ImpressionToken,
		"success":          true,
	})
}

// ValidateImpressionToken validates an impression token
// @Summary Validate impression token
// @Description Validates an impression token and returns the associated advertisement ID and session ID.
// @Tags Ad Tracking
// @Produce json
// @Param token query string true "Impression token to validate"
// @Success 200 {object} map[string]interface{} "Token is valid with advertisement_id and session_id"
// @Failure 400 {object} map[string]interface{} "Token query parameter is required"
// @Failure 401 {object} map[string]interface{} "Invalid or expired token"
// @Router /api/v1/tracking/impressions/validate [get]
func (h *ImpressionHandler) ValidateImpressionToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	impressionToken, err := h.impressionService.ValidateImpressionToken(token)
	if err != nil {
		h.logger.Warn("Invalid impression token", "error", err, "token", token)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":            true,
		"advertisement_id": impressionToken.AdvertisementID,
		"session_id":       impressionToken.SessionID,
		"timestamp":        impressionToken.Timestamp,
	})
}

// GetImpressionsByAdvertisement retrieves impressions for an advertisement
// @Summary Get impressions by advertisement
// @Description Returns a list of impressions for the specified advertisement. Accepts optional limit query parameter (default: 100).
// @Tags Ad Tracking
// @Produce json
// @Security BearerAuth
// @Param ad_id path string true "Advertisement ID (UUID)"
// @Param limit query int false "Maximum number of records to return" default(100)
// @Success 200 {object} map[string]interface{} "List of impressions with count and applied limit"
// @Failure 400 {object} map[string]interface{} "Invalid advertisement ID format"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/advertiser/ads/{ad_id}/impressions [get]
func (h *ImpressionHandler) GetImpressionsByAdvertisement(c *gin.Context) {
	adID, err := uuid.Parse(c.Param("ad_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid advertisement_id"})
		return
	}

	// Parse limit (default: 100)
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	impressions, err := h.impressionService.GetImpressionsByAdvertisement(c.Request.Context(), adID, limit)
	if err != nil {
		h.logger.Error("Failed to get impressions", "error", err, "ad_id", adID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get impressions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"impressions": impressions,
		"count":       len(impressions),
		"limit":       limit,
	})
}

// CountViewableImpressions counts viewable impressions for an advertisement
// @Summary Count viewable impressions
// @Description Returns the number of viewable impressions for an advertisement since a given timestamp (defaults to last 24 hours).
// @Tags Ad Tracking
// @Produce json
// @Security BearerAuth
// @Param ad_id path string true "Advertisement ID (UUID)"
// @Param since query string false "Start timestamp (RFC3339). Defaults to 24 hours ago." example("2024-01-15T00:00:00Z")
// @Success 200 {object} map[string]interface{} "Viewable impression count with advertisement_id and since timestamp"
// @Failure 400 {object} map[string]interface{} "Invalid advertisement ID or since date format"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/advertiser/ads/{ad_id}/viewable [get]
func (h *ImpressionHandler) CountViewableImpressions(c *gin.Context) {
	adID, err := uuid.Parse(c.Param("ad_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid advertisement_id"})
		return
	}

	since := time.Now().Add(-24 * time.Hour)
	if sinceStr := c.Query("since"); sinceStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid since format"})
			return
		}
		since = parsedTime
	}

	count, err := h.impressionService.CountViewableImpressions(c.Request.Context(), adID, since)
	if err != nil {
		h.logger.Error("Failed to count viewable impressions", "error", err, "ad_id", adID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count impressions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"advertisement_id": adID,
		"viewable_count":   count,
		"since":            since,
	})
}

// TrackClick tracks an ad click
// @Summary Track ad click
// @Description Records an ad click event. Returns 403 for suspicious clicks and 400 for invalid click data.
// @Tags Ad Tracking
// @Accept json
// @Produce json
// @Param request body TrackClickRequest true "Click data"
// @Success 201 {object} map[string]interface{} "Click recorded with click_id"
// @Failure 400 {object} map[string]interface{} "Invalid request body, UUID fields, or click data"
// @Failure 403 {object} map[string]interface{} "Suspicious click detected"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/tracking/clicks [post]
func (h *ImpressionHandler) TrackClick(c *gin.Context) {
	var req TrackClickRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid click tracking request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	impressionID, err := uuid.Parse(req.ImpressionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid impression_id"})
		return
	}

	adID, err := uuid.Parse(req.AdvertisementID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid advertisement_id"})
		return
	}

	var userIDPtr *uuid.UUID
	if req.UserID != nil {
		parsedUserID, err := uuid.Parse(*req.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}
		userIDPtr = &parsedUserID
	}

	click := &domain.AdClick{
		ID:              uuid.New(),
		ImpressionID:    impressionID,
		AdvertisementID: adID,
		UserID:          userIDPtr,
		IPAddress:       req.IPAddress,
		UserAgent:       req.UserAgent,
		Referrer:        req.Referrer,
		ClickPosition:   req.ClickPosition,
		Converted:       false,
		CreatedAt:       time.Now(),
	}

	if err := h.clickService.RecordClick(c.Request.Context(), click); err != nil {
		h.logger.Error("Failed to record click", "error", err)

		if err == domain.ErrSuspiciousActivity {
			c.JSON(http.StatusForbidden, gin.H{"error": "suspicious click detected"})
			return
		}

		if err == domain.ErrInvalidClickData {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid click data"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record click"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"click_id": click.ID,
		"success":  true,
	})
}

// TrackConversion tracks a conversion for a click
// @Summary Track click conversion
// @Description Records a conversion event associated with a previously tracked click.
// @Tags Ad Tracking
// @Accept json
// @Produce json
// @Param click_id path string true "Click ID (UUID)"
// @Param request body TrackConversionRequest true "Conversion data"
// @Success 200 {object} map[string]interface{} "Conversion recorded with click_id and conversion_value_cents"
// @Failure 400 {object} map[string]interface{} "Invalid click ID or request body"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/tracking/clicks/{click_id}/conversion [post]
func (h *ImpressionHandler) TrackConversion(c *gin.Context) {
	clickID, err := uuid.Parse(c.Param("click_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid click_id"})
		return
	}

	var req TrackConversionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid conversion tracking request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.clickService.RecordConversion(c.Request.Context(), clickID, req.ConversionValueCents); err != nil {
		h.logger.Error("Failed to record conversion", "error", err, "click_id", clickID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record conversion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"click_id":               clickID,
		"conversion_value_cents": req.ConversionValueCents,
		"success":                true,
	})
}

// GetClicksByAdvertisement retrieves clicks for an advertisement
// @Summary Get clicks by advertisement
// @Description Returns a list of clicks for the specified advertisement (default limit: 100).
// @Tags Ad Tracking
// @Produce json
// @Security BearerAuth
// @Param ad_id path string true "Advertisement ID (UUID)"
// @Success 200 {object} map[string]interface{} "List of clicks with count"
// @Failure 400 {object} map[string]interface{} "Invalid advertisement ID format"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/advertiser/ads/{ad_id}/clicks [get]
func (h *ImpressionHandler) GetClicksByAdvertisement(c *gin.Context) {
	adID, err := uuid.Parse(c.Param("ad_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid advertisement_id"})
		return
	}

	limit := 100 // Default limit
	clicks, err := h.clickService.GetClicksByAdvertisement(c.Request.Context(), adID, limit)
	if err != nil {
		h.logger.Error("Failed to get clicks", "error", err, "ad_id", adID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get clicks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clicks": clicks,
		"count":  len(clicks),
	})
}

// GetClickStats retrieves click statistics for an advertisement
// @Summary Get click statistics
// @Description Returns click statistics for an advertisement since a given timestamp (defaults to last 24 hours).
// @Tags Ad Tracking
// @Produce json
// @Security BearerAuth
// @Param ad_id path string true "Advertisement ID (UUID)"
// @Param since query string false "Start timestamp (RFC3339). Defaults to 24 hours ago." example("2024-01-15T00:00:00Z")
// @Success 200 {object} map[string]interface{} "Click statistics"
// @Failure 400 {object} map[string]interface{} "Invalid advertisement ID or since date format"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/advertiser/ads/{ad_id}/clicks/stats [get]
func (h *ImpressionHandler) GetClickStats(c *gin.Context) {
	adID, err := uuid.Parse(c.Param("ad_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid advertisement_id"})
		return
	}

	since := time.Now().Add(-24 * time.Hour)
	if sinceStr := c.Query("since"); sinceStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid since format"})
			return
		}
		since = parsedTime
	}

	stats, err := h.clickService.GetClickStats(c.Request.Context(), adID, since)
	if err != nil {
		h.logger.Error("Failed to get click stats", "error", err, "ad_id", adID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get click stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
