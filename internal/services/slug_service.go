package services

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// SlugService maneja la normalización de URLs a formato SEO-friendly
type SlugService struct {
	transformer transform.Transformer
	regexNonAlnum *regexp.Regexp
	regexMultiDash *regexp.Regexp
}

// NewSlugService crea una nueva instancia del servicio de slugs
func NewSlugService() *SlugService {
	// Transformer para normalizar Unicode y remover diacríticos
	transformer := transform.Chain(
		norm.NFD,
		runes.Remove(runes.In(unicode.Mn)), // Remover marcas no espaciadas (acentos)
		norm.NFC,
	)

	return &SlugService{
		transformer:    transformer,
		regexNonAlnum:  regexp.MustCompile(`[^a-z0-9]+`),
		regexMultiDash: regexp.MustCompile(`-+`),
	}
}

// Slugify convierte texto a formato URL-friendly
// Ejemplo: "Radio Rock & Pop 100.1 🎵" -> "radio-rock-pop-100-1"
func (s *SlugService) Slugify(text string) string {
	if text == "" {
		return ""
	}

	// 1. Normalizar Unicode (á -> a, ñ -> n, etc.)
	normalized, _, _ := transform.String(s.transformer, text)

	// 2. Convertir a minúsculas
	lower := strings.ToLower(normalized)

	// 3. Reemplazar caracteres no alfanuméricos por guiones
	slug := s.regexNonAlnum.ReplaceAllString(lower, "-")

	// 4. Reemplazar múltiples guiones consecutivos por uno solo
	slug = s.regexMultiDash.ReplaceAllString(slug, "-")

	// 5. Remover guiones al inicio y final
	slug = strings.Trim(slug, "-")

	// 6. Limitar longitud a 100 caracteres para evitar URLs muy largas
	if len(slug) > 100 {
		slug = slug[:100]
		// Asegurar que no termina en guión después del corte
		slug = strings.TrimRight(slug, "-")
	}

	// Si después de todo el slug está vacío, usar un fallback
	if slug == "" {
		return "station"
	}

	return slug
}

// SlugifyWithID crea un slug único agregando un ID al final
// Útil para manejar nombres duplicados
func (s *SlugService) SlugifyWithID(text, id string) string {
	baseSlug := s.Slugify(text)
	
	if id == "" {
		return baseSlug
	}

	// Limpiar el ID también
	cleanID := s.Slugify(id)
	
	// Si el ID ya está al final del slug, no duplicar
	if strings.HasSuffix(baseSlug, "-"+cleanID) {
		return baseSlug
	}

	return baseSlug + "-" + cleanID
}

// Validate verifica si un slug es válido
func (s *SlugService) Validate(slug string) bool {
	if slug == "" {
		return false
	}

	// Solo letras minúsculas, números y guiones
	match := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	return match.MatchString(slug)
}
