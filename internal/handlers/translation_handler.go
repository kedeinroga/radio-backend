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

// CreateTranslation godoc
// @Summary Crear una nueva traducción
// @Description Crea una nueva traducción para una estación en un idioma específico (solo admin). Ejemplo de request: {"station_id": "abc123", "language_code": "en", "title": "Rock FM - Free Online Radio", "description": "Listen to Rock FM live from USA", "keywords": ["rock", "usa", "music", "online", "free"]}
// @Tags Translations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body domain.CreateTranslationRequest true "Datos de la traducción"
// @Success 201 {object} domain.TranslationResponse "Traducción creada exitosamente"
// @Failure 400 {object} map[string]interface{} "Solicitud inválida"
// @Failure 401 {object} map[string]interface{} "No autenticado"
// @Failure 403 {object} map[string]interface{} "No autorizado - Solo admin"
// @Failure 404 {object} map[string]interface{} "Estación no encontrada"
// @Failure 409 {object} map[string]interface{} "La traducción ya existe"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /admin/translations [post]
func (h *TranslationHandler) CreateTranslation(c *gin.Context) {
	var req domain.CreateTranslationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Crear la traducción
	translation, err := h.translationService.CreateTranslation(&req)
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
// @Summary Obtener una traducción específica
// @Description Obtiene la traducción de una estación en un idioma específico (solo admin)
// @Tags Translations
// @Produce json
// @Security BearerAuth
// @Param stationId path string true "ID de la estación"
// @Param lang path string true "Código de idioma (es, en, fr, de)"
// @Success 200 {object} domain.TranslationResponse "Traducción encontrada"
// @Failure 400 {object} map[string]interface{} "Solicitud inválida"
// @Failure 401 {object} map[string]interface{} "No autenticado"
// @Failure 403 {object} map[string]interface{} "No autorizado - Solo admin"
// @Failure 404 {object} map[string]interface{} "Traducción no encontrada"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /admin/translations/{stationId}/{lang} [get]
func (h *TranslationHandler) GetTranslation(c *gin.Context) {
	stationID := c.Param("stationId")
	langCode := c.Param("lang")

	// Validar idioma
	lang := i18n.ParseLanguage(langCode)
	if !i18n.IsSupported(lang) {
		RespondWithError(c, http.StatusBadRequest, "invalid_language", "Invalid or unsupported language code")
		return
	}

	// Obtener la traducción
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
// @Summary Listar todas las traducciones de una estación
// @Description Obtiene todas las traducciones disponibles para una estación (solo admin)
// @Tags Translations
// @Produce json
// @Security BearerAuth
// @Param stationId path string true "ID de la estación"
// @Success 200 {object} map[string]interface{} "Lista de traducciones"
// @Failure 400 {object} map[string]interface{} "Solicitud inválida"
// @Failure 401 {object} map[string]interface{} "No autenticado"
// @Failure 403 {object} map[string]interface{} "No autorizado - Solo admin"
// @Failure 404 {object} map[string]interface{} "Estación no encontrada"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /admin/translations/{stationId} [get]
func (h *TranslationHandler) ListTranslations(c *gin.Context) {
	stationID := c.Param("stationId")

	// Listar traducciones
	translations, err := h.translationService.ListTranslationsByStation(stationID)
	if err != nil {
		if err == domain.ErrStationNotFound {
			RespondWithError(c, http.StatusNotFound, "station_not_found", "Station not found")
			return
		}
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to list translations")
		return
	}

	// Convertir a responses
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
// @Summary Actualizar una traducción
// @Description Actualiza una traducción existente (solo admin)
// @Tags Translations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param stationId path string true "ID de la estación"
// @Param lang path string true "Código de idioma (es, en, fr, de)"
// @Param request body domain.UpdateTranslationRequest true "Datos actualizados de la traducción"
// @Success 200 {object} domain.TranslationResponse "Traducción actualizada exitosamente"
// @Failure 400 {object} map[string]interface{} "Solicitud inválida"
// @Failure 401 {object} map[string]interface{} "No autenticado"
// @Failure 403 {object} map[string]interface{} "No autorizado - Solo admin"
// @Failure 404 {object} map[string]interface{} "Traducción no encontrada"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /admin/translations/{stationId}/{lang} [put]
func (h *TranslationHandler) UpdateTranslation(c *gin.Context) {
	stationID := c.Param("stationId")
	langCode := c.Param("lang")

	// Validar idioma
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

	// Actualizar la traducción
	translation, err := h.translationService.UpdateTranslation(stationID, lang, &req)
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
// @Summary Eliminar una traducción
// @Description Elimina una traducción existente (solo admin)
// @Tags Translations
// @Produce json
// @Security BearerAuth
// @Param stationId path string true "ID de la estación"
// @Param lang path string true "Código de idioma (es, en, fr, de)"
// @Success 200 {object} map[string]interface{} "Traducción eliminada exitosamente"
// @Failure 400 {object} map[string]interface{} "Solicitud inválida"
// @Failure 401 {object} map[string]interface{} "No autenticado"
// @Failure 403 {object} map[string]interface{} "No autorizado - Solo admin"
// @Failure 404 {object} map[string]interface{} "Traducción no encontrada"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /admin/translations/{stationId}/{lang} [delete]
func (h *TranslationHandler) DeleteTranslation(c *gin.Context) {
	stationID := c.Param("stationId")
	langCode := c.Param("lang")

	// Validar idioma
	lang := i18n.ParseLanguage(langCode)
	if !i18n.IsSupported(lang) {
		RespondWithError(c, http.StatusBadRequest, "invalid_language", "Invalid or unsupported language code")
		return
	}

	// Eliminar la traducción
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
// @Summary Crear múltiples traducciones
// @Description Crea múltiples traducciones en una sola operación (solo admin). Útil para poblar traducciones de múltiples estaciones. Ejemplo: [{"station_id":"abc","language_code":"en","title":"Radio FM","description":"Listen live","keywords":["radio","music"]},{"station_id":"abc","language_code":"fr","title":"Radio FM","description":"Écoutez en direct","keywords":["radio","musique"]}]
// @Tags Translations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body []domain.CreateTranslationRequest true "Lista de traducciones a crear"
// @Success 201 {object} map[string]interface{} "Traducciones creadas exitosamente"
// @Failure 400 {object} map[string]interface{} "Solicitud inválida"
// @Failure 401 {object} map[string]interface{} "No autenticado"
// @Failure 403 {object} map[string]interface{} "No autorizado - Solo admin"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
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

	// Convertir requests a StationTranslation
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

	// Crear traducciones en bulk
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
// @Summary Obtener idiomas disponibles para una estación
// @Description Obtiene la lista de idiomas para los cuales existen traducciones de una estación. Útil para mostrar selector de idiomas en el frontend. Retorna códigos de idioma: es, en, fr, de
// @Tags Translations
// @Produce json
// @Param stationId path string true "ID de la estación"
// @Success 200 {object} map[string]interface{} "Lista de idiomas disponibles. Ejemplo: {\"success\":true,\"data\":[\"es\",\"en\",\"fr\"],\"count\":3}"
// @Failure 400 {object} map[string]interface{} "Solicitud inválida"
// @Failure 500 {object} map[string]interface{} "Error interno del servidor"
// @Router /translations/{stationId}/languages [get]
func (h *TranslationHandler) GetAvailableLanguages(c *gin.Context) {
	stationID := c.Param("stationId")

	languages, err := h.translationService.GetAvailableLanguages(stationID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to get available languages")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      languages,
		"count":     len(languages),
	})
}
