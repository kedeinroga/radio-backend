package i18n

import (
	"strings"
)

// Language representa un código de idioma soportado
type Language string

const (
	// LanguageES representa español
	LanguageES Language = "es"
	// LanguageEN representa inglés
	LanguageEN Language = "en"
	// LanguageFR representa francés
	LanguageFR Language = "fr"
	// LanguageDE representa alemán
	LanguageDE Language = "de"
)

// DefaultLanguage es el idioma por defecto del sistema
const DefaultLanguage = LanguageES

// SupportedLanguages contiene todos los idiomas soportados
var SupportedLanguages = []Language{
	LanguageES,
	LanguageEN,
	LanguageFR,
	LanguageDE,
}

// ParseLanguage parsea un string a Language y valida que sea soportado
// Si no es válido, retorna el idioma por defecto
func ParseLanguage(lang string) Language {
	// Normalizar: lowercase y tomar solo los primeros 2 caracteres
	lang = strings.ToLower(strings.TrimSpace(lang))
	if len(lang) >= 2 {
		lang = lang[:2]
	}

	// Validar contra idiomas soportados
	for _, supported := range SupportedLanguages {
		if Language(lang) == supported {
			return Language(lang)
		}
	}

	return DefaultLanguage
}

// IsSupported verifica si un idioma está soportado
func IsSupported(lang Language) bool {
	for _, supported := range SupportedLanguages {
		if lang == supported {
			return true
		}
	}
	return false
}

// String retorna la representación en string del idioma
func (l Language) String() string {
	return string(l)
}

// ParseAcceptLanguage parsea el header Accept-Language y retorna el mejor match
// Ejemplo: "en-US,en;q=0.9,es;q=0.8" -> "en"
func ParseAcceptLanguage(acceptLanguage string) Language {
	if acceptLanguage == "" {
		return DefaultLanguage
	}

	// Dividir por comas
	languages := strings.Split(acceptLanguage, ",")
	
	// Tomar el primer idioma (tiene mayor prioridad)
	if len(languages) > 0 {
		// Extraer solo el código del idioma (antes de ; o -)
		firstLang := strings.TrimSpace(languages[0])
		firstLang = strings.Split(firstLang, ";")[0] // Remover quality factor
		firstLang = strings.Split(firstLang, "-")[0] // Remover región (en-US -> en)
		
		return ParseLanguage(firstLang)
	}

	return DefaultLanguage
}

// GetLanguageName retorna el nombre del idioma en su propio idioma
func GetLanguageName(lang Language) string {
	names := map[Language]string{
		LanguageES: "Español",
		LanguageEN: "English",
		LanguageFR: "Français",
		LanguageDE: "Deutsch",
	}
	
	if name, ok := names[lang]; ok {
		return name
	}
	
	return names[DefaultLanguage]
}
