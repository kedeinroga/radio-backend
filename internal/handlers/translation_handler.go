package handlers

import (
	"net/http"

	"radio-backend/internal/domain"
	"radio-backend/internal/i18n"
	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// TranslationHandler maneja las peticiones HTTP relacionadas con traducciones
type TranslationHandler struct {
	translationService *services.TranslationService
}

// NewTranslationHandler crea una nueva instancia del handler de traducciones
func NewTranslationHandler(translationService *services.TranslationService) *TranslationHandler {
	return &TranslationHandler{
		translationService: translationService,
	}
}

// Translation endpoints wrap their payload in {"success": true, ...}.

// TranslationEnvelope is the envelope for a single translation.
type TranslationEnvelope struct {
	Success bool                       `json:"success" example:"true"`
	Data    domain.TranslationResponse `json:"data"`
}

// TranslationListResponse is the envelope for a list of translations.
type TranslationListResponse struct {
	Success bool                          `json:"success" example:"true"`
	Data    []*domain.TranslationResponse `json:"data"`
	Count   int                           `json:"count" example:"3"`
}

// SuccessMessageResponse is a {"success": true, "message": "..."} envelope.
type SuccessMessageResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"Translation deleted successfully"`
}

// BulkTranslationResponse is the envelope for the bulk-create operation.
type BulkTranslationResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"Translations created successfully"`
	Count   int    `json:"count" example:"10"`
}

// AvailableLanguagesResponse is the envelope listing available language codes.
type AvailableLanguagesResponse struct {
	Success bool     `json:"success" example:"true"`
	Data    []string `json:"data" example:"es,en,fr"`
	Count   int      `json:"count" example:"3"`
}

// CreateTranslation godoc
// @Summary Create a new translation
// @Description Creates a new translation for a station in a specific language (admin only). Request example: {"station_id": "abc123", "language_code": "en", "title": "Rock FM - Free Online Radio", "description": "Listen to Rock FM live from USA", "keywords": ["rock", "usa", "music", "online", "free"]}
// @Tags Translations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body domain.CreateTranslationRequest true "Translation data"
// @Success 201 {object} TranslationEnvelope "Translation created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} SimpleErrorResponse "Not authenticated"
// @Failure 403 {object} SimpleErrorResponse "Not authorized - Admin only"
// @Failure 404 {object} ErrorResponse "Station not found"
// @Failure 409 {object} ErrorResponse "Translation already exists"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/translations [post]
func (h *TranslationHandler) CreateTranslation(c *gin.Context) {
	var req domain.CreateTranslationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Create the translation
	translation, err := h.translationService.CreateTranslation(c.Request.Context(), &req)
	if err != nil {
		switch err {
		case domain.ErrStationNotFound:
			RespondWithError(c, http.StatusNotFound, "station_not_found", "Station not found")
		case domain.ErrUnsupportedLanguage:
			RespondWithError(c, http.StatusBadRequest, "unsupported_language", "Unsupported language")
		case domain.ErrTranslationExists:
			RespondWithError(c, http.StatusConflict, "translation_exists", "Translation already exists for this station and language")
		default:
			RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to create translation")
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    translation.ToResponse(),
	})
}

// GetTranslation godoc
// @Summary Get a specific translation
// @Description Gets a station translation in a specific language (admin only)
// @Tags Translations
// @Produce json
// @Security BearerAuth
// @Param stationId path string true "Station ID"
// @Param lang path string true "Language code (es, en, fr, de)"
// @Success 200 {object} TranslationEnvelope "Translation found"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} SimpleErrorResponse "Not authenticated"
// @Failure 403 {object} SimpleErrorResponse "Not authorized - Admin only"
// @Failure 404 {object} ErrorResponse "Translation not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/translations/{stationId}/{lang} [get]
func (h *TranslationHandler) GetTranslation(c *gin.Context) {
	stationID := c.Param("stationId")
	langCode := c.Param("lang")

	// Validate language
	lang := i18n.ParseLanguage(langCode)
	if !i18n.IsSupported(lang) {
		RespondWithError(c, http.StatusBadRequest, "invalid_language", "Invalid or unsupported language code")
		return
	}

	// Get the translation
	translation, err := h.translationService.GetTranslation(stationID, lang)
	if err != nil {
		if err == domain.ErrTranslationNotFound {
			RespondWithError(c, http.StatusNotFound, "translation_not_found", "Translation not found")
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to get translation")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    translation.ToResponse(),
	})
}

// ListTranslations godoc
// @Summary List all translations for a station
// @Description Gets all available translations for a station (admin only)
// @Tags Translations
// @Produce json
// @Security BearerAuth
// @Param stationId path string true "Station ID"
// @Success 200 {object} TranslationListResponse "List of translations"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} SimpleErrorResponse "Not authenticated"
// @Failure 403 {object} SimpleErrorResponse "Not authorized - Admin only"
// @Failure 404 {object} ErrorResponse "Station not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/translations/{stationId} [get]
func (h *TranslationHandler) ListTranslations(c *gin.Context) {
	stationID := c.Param("stationId")

	// List translations
	translations, err := h.translationService.ListTranslationsByStation(c.Request.Context(), stationID)
	if err != nil {
		if err == domain.ErrStationNotFound {
			RespondWithError(c, http.StatusNotFound, "station_not_found", "Station not found")
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to list translations")
		return
	}

	// Convert to responses
	responses := make([]*domain.TranslationResponse, len(translations))
	for i, t := range translations {
		responses[i] = t.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responses,
		"count":   len(responses),
	})
}

// UpdateTranslation godoc
// @Summary Update a translation
// @Description Updates an existing translation (admin only)
// @Tags Translations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param stationId path string true "Station ID"
// @Param lang path string true "Language code (es, en, fr, de)"
// @Param request body domain.UpdateTranslationRequest true "Updated translation data"
// @Success 200 {object} TranslationEnvelope "Translation updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} SimpleErrorResponse "Not authenticated"
// @Failure 403 {object} SimpleErrorResponse "Not authorized - Admin only"
// @Failure 404 {object} ErrorResponse "Translation not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/translations/{stationId}/{lang} [put]
func (h *TranslationHandler) UpdateTranslation(c *gin.Context) {
	stationID := c.Param("stationId")
	langCode := c.Param("lang")

	// Validate language
	lang := i18n.ParseLanguage(langCode)
	if !i18n.IsSupported(lang) {
		RespondWithError(c, http.StatusBadRequest, "invalid_language", "Invalid or unsupported language code")
		return
	}

	var req domain.UpdateTranslationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Update the translation
	translation, err := h.translationService.UpdateTranslation(c.Request.Context(), stationID, lang, &req)
	if err != nil {
		if err == domain.ErrTranslationNotFound {
			RespondWithError(c, http.StatusNotFound, "translation_not_found", "Translation not found")
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to update translation")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    translation.ToResponse(),
	})
}

// DeleteTranslation godoc
// @Summary Delete a translation
// @Description Deletes an existing translation (admin only)
// @Tags Translations
// @Produce json
// @Security BearerAuth
// @Param stationId path string true "Station ID"
// @Param lang path string true "Language code (es, en, fr, de)"
// @Success 200 {object} SuccessMessageResponse "Translation deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} SimpleErrorResponse "Not authenticated"
// @Failure 403 {object} SimpleErrorResponse "Not authorized - Admin only"
// @Failure 404 {object} ErrorResponse "Translation not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/translations/{stationId}/{lang} [delete]
func (h *TranslationHandler) DeleteTranslation(c *gin.Context) {
	stationID := c.Param("stationId")
	langCode := c.Param("lang")

	// Validate language
	lang := i18n.ParseLanguage(langCode)
	if !i18n.IsSupported(lang) {
		RespondWithError(c, http.StatusBadRequest, "invalid_language", "Invalid or unsupported language code")
		return
	}

	// Delete the translation
	err := h.translationService.DeleteTranslation(stationID, lang)
	if err != nil {
		if err == domain.ErrTranslationNotFound {
			RespondWithError(c, http.StatusNotFound, "translation_not_found", "Translation not found")
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to delete translation")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Translation deleted successfully",
	})
}

// BulkCreateTranslations godoc
// @Summary Create multiple translations
// @Description Creates multiple translations in a single operation (admin only). Useful for populating translations for multiple stations. Example: [{"station_id":"abc","language_code":"en","title":"Radio FM","description":"Listen live","keywords":["radio","music"]},{"station_id":"abc","language_code":"fr","title":"Radio FM","description":"Écoutez en direct","keywords":["radio","musique"]}]
// @Tags Translations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body []domain.CreateTranslationRequest true "List of translations to create"
// @Success 201 {object} BulkTranslationResponse "Translations created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} SimpleErrorResponse "Not authenticated"
// @Failure 403 {object} SimpleErrorResponse "Not authorized - Admin only"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/translations/bulk [post]
func (h *TranslationHandler) BulkCreateTranslations(c *gin.Context) {
	var requests []domain.CreateTranslationRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if len(requests) == 0 {
		RespondWithError(c, http.StatusBadRequest, "empty_request", "At least one translation is required")
		return
	}

	// Convert requests to StationTranslation
	translations := make([]*domain.StationTranslation, len(requests))
	for i, req := range requests {
		translations[i] = &domain.StationTranslation{
			StationID:    req.StationID,
			LanguageCode: i18n.ParseLanguage(req.LanguageCode),
			Title:        req.Title,
			Description:  req.Description,
			Keywords:     req.Keywords,
		}
	}

	// Create translations in bulk
	err := h.translationService.BulkCreateTranslations(translations)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to create translations")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Translations created successfully",
		"count":   len(translations),
	})
}

// GetAvailableLanguages godoc
// @Summary Get available languages for a station
// @Description Gets the list of languages for which translations exist for a station. Useful for showing a language selector in the frontend. Returns language codes: es, en, fr, de
// @Tags Translations
// @Produce json
// @Param stationId path string true "Station ID"
// @Success 200 {object} AvailableLanguagesResponse "List of available languages"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /translations/{stationId}/languages [get]
func (h *TranslationHandler) GetAvailableLanguages(c *gin.Context) {
	stationID := c.Param("stationId")

	languages, err := h.translationService.GetAvailableLanguages(stationID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to get available languages")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    languages,
		"count":   len(languages),
	})
}
