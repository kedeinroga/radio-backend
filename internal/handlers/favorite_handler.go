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
// @Summary Obtener favoritos del usuario
// @Description Retorna la lista de estaciones favoritas del usuario autenticado
// @Tags Favorites
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} handlers.StationListResponse "Lista de estaciones favoritas"
// @Failure 401 {object} map[string]interface{} "No autenticado"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /favorites [get]
func (h *FavoriteHandler) GetFavorites(c *gin.Context) {
	userID := middleware.GetUserID(c)

	stations, err := h.favoriteService.GetUserFavorites(*userID)
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
// @Summary Agregar estación a favoritos
// @Description Agrega una estación a la lista de favoritos del usuario autenticado
// @Tags Favorites
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AddFavoriteRequest true "ID de la estación"
// @Success 201 {object} map[string]interface{} "Favorito agregado exitosamente"
// @Failure 400 {object} map[string]interface{} "Solicitud inválida"
// @Failure 401 {object} map[string]interface{} "No autenticado"
// @Failure 403 {object} map[string]interface{} "Acceso denegado - Estación solo para Premium"
// @Failure 404 {object} map[string]interface{} "Estación no encontrada"
// @Failure 409 {object} map[string]interface{} "Estación ya está en favoritos"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
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

	err := h.favoriteService.AddFavorite(*userID, req.StationID, userType, lang)
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
// @Summary Eliminar estación de favoritos
// @Description Elimina una estación de la lista de favoritos del usuario autenticado
// @Tags Favorites
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param stationId path string true "ID de la estación"
// @Success 200 {object} map[string]interface{} "Favorito eliminado exitosamente"
// @Failure 401 {object} map[string]interface{} "No autenticado"
// @Failure 404 {object} map[string]interface{} "Favorito no encontrado"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /favorites/{stationId} [delete]
func (h *FavoriteHandler) RemoveFavorite(c *gin.Context) {
	stationID := c.Param("stationId")
	if stationID == "" {
		RespondWithError(c, http.StatusBadRequest, "invalid_id", "Station ID is required")
		return
	}

	userID := middleware.GetUserID(c)

	err := h.favoriteService.RemoveFavorite(*userID, stationID)
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
