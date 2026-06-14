// Package icy lee el metadata ICY (Shoutcast/Icecast) de un stream de audio
// para extraer el título de la canción que suena ("now playing"), sin reproducir
// el stream completo: abre la conexión, lee hasta el primer bloque de metadata y cierra.
package icy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var (
	// ErrNoMetadata indica que el stream no expone metadata ICY (sin header icy-metaint).
	ErrNoMetadata = errors.New("stream does not expose ICY metadata")
	// ErrPlaylistUnresolved indica que no se pudo resolver una playlist a una URL de stream.
	ErrPlaylistUnresolved = errors.New("could not resolve playlist to a stream URL")
	// ErrEmptyTitle indica que el bloque de metadata no contenía StreamTitle.
	ErrEmptyTitle = errors.New("metadata block had no StreamTitle")
)

const (
	// maxMetaInt acota el intervalo de metadata aceptado para limitar el ancho de banda leído.
	maxMetaInt = 256 * 1024
	// maxPlaylistBytes acota la lectura de archivos de playlist.
	maxPlaylistBytes = 64 * 1024
)

var streamTitleRe = regexp.MustCompile(`StreamTitle='([^']*)'`)

// FetchNowPlaying obtiene el StreamTitle ICY actual de una estación.
//
// Resuelve playlists (.pls/.m3u/.m3u8) a la primera URL de stream real, solicita
// metadata ICY y lee solo hasta el primer bloque de metadata. Devuelve el título crudo.
//
// Errores esperables (no son fallos del sistema): ErrNoMetadata, ErrEmptyTitle,
// ErrPlaylistUnresolved.
func FetchNowPlaying(ctx context.Context, client *http.Client, streamURL string) (string, error) {
	resolvedURL, err := resolveStreamURL(ctx, client, streamURL, 0)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolvedURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build stream request: %w", err)
	}
	req.Header.Set("Icy-MetaData", "1")
	req.Header.Set("User-Agent", "rradio-nowplaying/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("stream returned status %d", resp.StatusCode)
	}

	metaInt, err := strconv.Atoi(resp.Header.Get("icy-metaint"))
	if err != nil || metaInt <= 0 {
		return "", ErrNoMetadata
	}
	if metaInt > maxMetaInt {
		return "", fmt.Errorf("%w: icy-metaint too large (%d)", ErrNoMetadata, metaInt)
	}

	return readFirstStreamTitle(resp.Body, metaInt)
}

// readFirstStreamTitle descarta metaInt bytes de audio, lee el bloque de metadata
// y extrae el StreamTitle.
func readFirstStreamTitle(body io.Reader, metaInt int) (string, error) {
	reader := bufio.NewReader(body)

	// Descartar metaInt bytes de audio.
	if _, err := io.CopyN(io.Discard, reader, int64(metaInt)); err != nil {
		return "", fmt.Errorf("failed to skip audio bytes: %w", err)
	}

	// El siguiente byte indica la longitud del bloque de metadata en múltiplos de 16.
	lengthByte, err := reader.ReadByte()
	if err != nil {
		return "", fmt.Errorf("failed to read metadata length: %w", err)
	}
	metaLen := int(lengthByte) * 16
	if metaLen == 0 {
		return "", ErrEmptyTitle
	}

	metaBuf := make([]byte, metaLen)
	if _, err := io.ReadFull(reader, metaBuf); err != nil {
		return "", fmt.Errorf("failed to read metadata block: %w", err)
	}

	matches := streamTitleRe.FindSubmatch(metaBuf)
	if matches == nil {
		return "", ErrEmptyTitle
	}

	title := strings.TrimSpace(string(matches[1]))
	if title == "" {
		return "", ErrEmptyTitle
	}
	return title, nil
}

// resolveStreamURL detecta y resuelve playlists (.pls/.m3u/.m3u8) a una URL de stream.
// maxDepth previene bucles de playlists anidadas.
func resolveStreamURL(ctx context.Context, client *http.Client, rawURL string, depth int) (string, error) {
	if depth > 2 {
		return "", ErrPlaylistUnresolved
	}

	if !looksLikePlaylist(rawURL) {
		return rawURL, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build playlist request: %w", err)
	}
	req.Header.Set("User-Agent", "rradio-nowplaying/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch playlist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("playlist returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPlaylistBytes))
	if err != nil {
		return "", fmt.Errorf("failed to read playlist: %w", err)
	}

	streamURL := firstStreamURLFromPlaylist(string(data))
	if streamURL == "" {
		return "", ErrPlaylistUnresolved
	}

	// La URL extraída podría ser otra playlist anidada.
	return resolveStreamURL(ctx, client, streamURL, depth+1)
}

// looksLikePlaylist determina si la URL apunta a un archivo de playlist por su extensión.
func looksLikePlaylist(rawURL string) bool {
	lower := strings.ToLower(rawURL)
	// Recortar query string para evaluar la extensión del path.
	if i := strings.IndexAny(lower, "?#"); i >= 0 {
		lower = lower[:i]
	}
	return strings.HasSuffix(lower, ".pls") ||
		strings.HasSuffix(lower, ".m3u") ||
		strings.HasSuffix(lower, ".m3u8")
}

// firstStreamURLFromPlaylist extrae la primera URL http(s) de un .pls o .m3u/.m3u8.
func firstStreamURLFromPlaylist(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Formato PLS: "FileN=http://..."
		if strings.HasPrefix(strings.ToLower(line), "file") {
			if idx := strings.Index(line, "="); idx >= 0 {
				if url := strings.TrimSpace(line[idx+1:]); isHTTPURL(url) {
					return url
				}
			}
			continue
		}

		// Formato M3U: comentarios empiezan con '#'; las demás líneas son URLs.
		if strings.HasPrefix(line, "#") {
			continue
		}
		if isHTTPURL(line) {
			return line
		}
	}
	return ""
}

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
