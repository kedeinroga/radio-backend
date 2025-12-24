package middleware

import (
	"radio-backend/internal/i18n"

	"github.com/gin-gonic/gin"
)

const (
	// LanguageContextKey es la clave para obtener el idioma del contexto
	LanguageContextKey = "language"

	// LanguageQueryParam es el parámetro de query para especificar el idioma
	LanguageQueryParam = "lang"

	// AcceptLanguageHeader es el header HTTP para especificar el idioma
	AcceptLanguageHeader = "Accept-Language"
)

// LanguageDetector es un middleware que detecta el idioma del usuario
// Prioridad: 1) Query param (?lang=en), 2) Accept-Language header, 3) Idioma por defecto
func LanguageDetector() gin.HandlerFunc {
	return func(c *gin.Context) {
		var lang i18n.Language

		// 1. Intentar obtener del query parameter
		if queryLang := c.Query(LanguageQueryParam); queryLang != "" {
			lang = i18n.ParseLanguage(queryLang)
		} else {
			// 2. Intentar obtener del header Accept-Language
			acceptLang := c.GetHeader(AcceptLanguageHeader)
			lang = i18n.ParseAcceptLanguage(acceptLang)
		}

		// Guardar en el contexto
		c.Set(LanguageContextKey, lang)

		c.Next()
	}
}

// GetLanguage obtiene el idioma del contexto de Gin
// Si no existe, retorna el idioma por defecto
func GetLanguage(c *gin.Context) i18n.Language {
	if lang, exists := c.Get(LanguageContextKey); exists {
		if language, ok := lang.(i18n.Language); ok {
			return language
		}
	}
	return i18n.DefaultLanguage
}

// SetLanguage establece el idioma en el contexto de Gin
func SetLanguage(c *gin.Context, lang i18n.Language) {
	c.Set(LanguageContextKey, lang)
}

// WithLanguage es un helper para probar handlers con un idioma específico
func WithLanguage(lang i18n.Language) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(LanguageContextKey, lang)
		c.Next()
	}
}
