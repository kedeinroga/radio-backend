package services

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	slugService := NewSlugService()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple text",
			input:    "Radio Station",
			expected: "radio-station",
		},
		{
			name:     "With special characters",
			input:    "Radio Rock & Pop 100.1",
			expected: "radio-rock-pop-100-1",
		},
		{
			name:     "With accents (Spanish)",
			input:    "Estación de México",
			expected: "estacion-de-mexico",
		},
		{
			name:     "With accents (French)",
			input:    "Émission Française",
			expected: "emission-francaise",
		},
		{
			name:     "With emojis",
			input:    "Rock!!! 🎸 Radio 🎵",
			expected: "rock-radio",
		},
		{
			name:     "Multiple spaces",
			input:    "  Multiple   Spaces  ",
			expected: "multiple-spaces",
		},
		{
			name:     "Numbers and letters",
			input:    "FM 102.5",
			expected: "fm-102-5",
		},
		{
			name:     "All special chars",
			input:    "@#$%^&*()",
			expected: "station", // Fallback
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Already slugified",
			input:    "already-slugified",
			expected: "already-slugified",
		},
		{
			name:     "With underscores",
			input:    "Radio_Station_Name",
			expected: "radio-station-name",
		},
		{
			name:     "Mixed case",
			input:    "RaDiO StAtIoN",
			expected: "radio-station",
		},
		{
			name:     "Portuguese accents",
			input:    "Rádio São Paulo",
			expected: "radio-sao-paulo",
		},
		{
			name:     "German umlauts",
			input:    "Über Radio",
			expected: "uber-radio",
		},
		{
			name:     "Multiple dashes",
			input:    "Radio---Station",
			expected: "radio-station",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slugService.Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSlugifyWithID(t *testing.T) {
	slugService := NewSlugService()

	tests := []struct {
		name     string
		text     string
		id       string
		expected string
	}{
		{
			name:     "With UUID",
			text:     "Radio Station",
			id:       "abc123",
			expected: "radio-station-abc123",
		},
		{
			name:     "Empty ID",
			text:     "Radio Station",
			id:       "",
			expected: "radio-station",
		},
		{
			name:     "ID already in name",
			text:     "Radio Station abc123",
			id:       "abc123",
			expected: "radio-station-abc123",
		},
		{
			name:     "Complex ID",
			text:     "Radio",
			id:       "550e8400-e29b-41d4-a716-446655440000",
			expected: "radio-550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slugService.SlugifyWithID(tt.text, tt.id)
			if result != tt.expected {
				t.Errorf("SlugifyWithID(%q, %q) = %q, want %q", tt.text, tt.id, result, tt.expected)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	slugService := NewSlugService()

	tests := []struct {
		name     string
		slug     string
		expected bool
	}{
		{
			name:     "Valid slug",
			slug:     "radio-station",
			expected: true,
		},
		{
			name:     "Valid with numbers",
			slug:     "fm-102-5",
			expected: true,
		},
		{
			name:     "Invalid - uppercase",
			slug:     "Radio-Station",
			expected: false,
		},
		{
			name:     "Invalid - spaces",
			slug:     "radio station",
			expected: false,
		},
		{
			name:     "Invalid - special chars",
			slug:     "radio@station",
			expected: false,
		},
		{
			name:     "Invalid - empty",
			slug:     "",
			expected: false,
		},
		{
			name:     "Invalid - starting with dash",
			slug:     "-radio",
			expected: false,
		},
		{
			name:     "Invalid - ending with dash",
			slug:     "radio-",
			expected: false,
		},
		{
			name:     "Invalid - double dash",
			slug:     "radio--station",
			expected: false,
		},
		{
			name:     "Valid - only numbers",
			slug:     "102-5",
			expected: true,
		},
		{
			name:     "Valid - only letters",
			slug:     "radio",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slugService.Validate(tt.slug)
			if result != tt.expected {
				t.Errorf("Validate(%q) = %v, want %v", tt.slug, result, tt.expected)
			}
		})
	}
}

func TestSlugifyLongText(t *testing.T) {
	slugService := NewSlugService()

	// Texto muy largo que debe ser truncado
	longText := "This is a very long station name that should be truncated to avoid having extremely long URLs that could cause issues with some browsers and systems"
	result := slugService.Slugify(longText)

	if len(result) > 100 {
		t.Errorf("Slug length %d exceeds maximum of 100 characters", len(result))
	}

	// Verificar que no termina en guión
	if len(result) > 0 && result[len(result)-1] == '-' {
		t.Error("Slug should not end with a dash after truncation")
	}

	// Verificar que sigue siendo válido
	if !slugService.Validate(result) {
		t.Errorf("Truncated slug %q is not valid", result)
	}
}

func BenchmarkSlugify(b *testing.B) {
	slugService := NewSlugService()
	testText := "Radio Rock & Pop 100.1 🎵"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slugService.Slugify(testText)
	}
}

func BenchmarkSlugifyWithAccents(b *testing.B) {
	slugService := NewSlugService()
	testText := "Rádio São Paulo - Émission Française Über"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slugService.Slugify(testText)
	}
}
