package handlers

import (
	"net/http"

	"radio-backend/internal/domain"
	"radio-backend/internal/middleware"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// FavoriteHandler handles favorite endpoints
type FavoriteHandler struct {
	favoriteService *services.FavoriteService
}

// NewFavoriteHandler creates a new favorite handler
func NewFavoriteHandler(favoriteService *services.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{
		favoriteService: favoriteService,
	}
}

// AddFavoriteRequest represents the request to add a favorite
type AddFavoriteRequest struct {
	StationID string `json:"station_id" binding:"required"`
}

// GetFavorites returns user's favorite stations
// @Summary Get user favorites
// @Description Returns the list of favorite stations for the authenticated user
// @Tags Favorites
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} FavoriteListResponse "List of favorite stations"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /favorites [get]
func (h *FavoriteHandler) GetFavorites(c *gin.Context) {
	userID := middleware.GetUserID(c)

	stations, err := h.favoriteService.GetUserFavorites(c.Request.Context(), *userID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "fetch_failed", "Failed to fetch favorites")
		return
	}

	// Convert to DTOs
	stationDTOs := make([]gin.H, 0, len(stations))
	for _, station := range stations {
		stationDTOs = append(stationDTOs, gin.H{
			"id":              station.ID,
			"name":            station.Name,
			"stream_url":      station.StreamURL,
			"image_url":       station.ImageURL,
			"tags":            station.Tags,
			"country":         station.Country,
			"votes":           station.Votes,
			"is_premium_only": station.IsPremiumOnly,
		})
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"data": stationDTOs,
		"meta": gin.H{
			"count": len(stationDTOs),
		},
	})
}

// AddFavorite adds a station to user's favorites
// @Summary Add station to favorites
// @Description Adds a station to the authenticated user's favorites list
// @Tags Favorites
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AddFavoriteRequest true "Station ID"
// @Success 201 {object} SuccessResponse "Favorite added successfully"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Access denied - Premium only station"
// @Failure 404 {object} ErrorResponse "Station not found"
// @Failure 409 {object} ErrorResponse "Station already in favorites"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /favorites [post]
func (h *FavoriteHandler) AddFavorite(c *gin.Context) {
	var req AddFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	userType := middleware.GetUserType(c)
	lang := middleware.GetLanguage(c)

	err := h.favoriteService.AddFavorite(c.Request.Context(), *userID, req.StationID, userType, lang)
	if err != nil {
		if err == domain.ErrFavoriteAlreadyExists {
			RespondWithError(c, http.StatusConflict, "already_exists", "Station is already in favorites")
			return
		}
		if err == domain.ErrStationNotFound {
			RespondWithError(c, http.StatusNotFound, "station_not_found", "Station not found")
			return
		}
		if err == domain.ErrUnauthorized {
			RespondWithError(c, http.StatusForbidden, "premium_only", "This station is only available for premium users")
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "add_failed", "Failed to add favorite")
		return
	}

	RespondWithSuccess(c, http.StatusCreated, gin.H{
		"message": "Favorite added successfully",
	})
}

// RemoveFavorite removes a station from user's favorites
// @Summary Remove station from favorites
// @Description Removes a station from the authenticated user's favorites list
// @Tags Favorites
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param stationId path string true "Station ID"
// @Success 200 {object} SuccessResponse "Favorite removed successfully"
// @Failure 400 {object} ErrorResponse "Station ID is required"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 404 {object} ErrorResponse "Favorite not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /favorites/{stationId} [delete]
func (h *FavoriteHandler) RemoveFavorite(c *gin.Context) {
	stationID := c.Param("stationId")
	if stationID == "" {
		RespondWithError(c, http.StatusBadRequest, "invalid_id", "Station ID is required")
		return
	}

	userID := middleware.GetUserID(c)

	err := h.favoriteService.RemoveFavorite(c.Request.Context(), *userID, stationID)
	if err != nil {
		if err == domain.ErrFavoriteNotFound {
			RespondWithError(c, http.StatusNotFound, "not_found", "Favorite not found")
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "remove_failed", "Failed to remove favorite")
		return
	}

	RespondWithSuccess(c, http.StatusOK, gin.H{
		"message": "Favorite removed successfully",
	})
}
