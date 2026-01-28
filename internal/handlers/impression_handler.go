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
