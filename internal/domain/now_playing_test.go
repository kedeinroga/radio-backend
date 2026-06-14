package domain

import "testing"

func TestParseStreamTitle(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantArtist string
		wantTitle  string
	}{
		{
			name:       "artist and song",
			raw:        "Queen - Bohemian Rhapsody",
			wantArtist: "Queen",
			wantTitle:  "Bohemian Rhapsody",
		},
		{
			name:       "only title",
			raw:        "Morning News",
			wantArtist: "",
			wantTitle:  "Morning News",
		},
		{
			name:       "multiple separators keeps remainder in title",
			raw:        "Eagles - Hotel California - Live",
			wantArtist: "Eagles",
			wantTitle:  "Hotel California - Live",
		},
		{
			name:       "trims surrounding whitespace",
			raw:        "  Oasis  -  Wonderwall  ",
			wantArtist: "Oasis",
			wantTitle:  "Wonderwall",
		},
		{
			name:       "empty string",
			raw:        "",
			wantArtist: "",
			wantTitle:  "",
		},
		{
			name:       "empty artist falls back to full title",
			raw:        " - Wonderwall",
			wantArtist: "",
			wantTitle:  "- Wonderwall",
		},
		{
			name:       "empty title falls back to full title",
			raw:        "Oasis - ",
			wantArtist: "",
			wantTitle:  "Oasis -",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artist, title := ParseStreamTitle(tt.raw)
			if artist != tt.wantArtist {
				t.Errorf("artist = %q, want %q", artist, tt.wantArtist)
			}
			if title != tt.wantTitle {
				t.Errorf("title = %q, want %q", title, tt.wantTitle)
			}
		})
	}
}
