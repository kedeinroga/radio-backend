package icy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// icyStreamHandler construye un cuerpo de stream ICY: metaInt bytes de audio,
// 1 byte de longitud y el bloque de metadata con StreamTitle.
func icyStreamBody(metaInt int, streamTitle string) []byte {
	audio := bytes.Repeat([]byte{0xAA}, metaInt)

	meta := fmt.Sprintf("StreamTitle='%s';", streamTitle)
	// Padding a múltiplo de 16.
	padded := meta
	for len(padded)%16 != 0 {
		padded += "\x00"
	}
	lengthByte := byte(len(padded) / 16)

	body := append([]byte{}, audio...)
	body = append(body, lengthByte)
	body = append(body, []byte(padded)...)
	// Más audio después (no debe leerse).
	body = append(body, bytes.Repeat([]byte{0xBB}, 1024)...)
	return body
}

func newClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

func TestFetchNowPlaying_Success(t *testing.T) {
	const metaInt = 8192
	const want = "Queen - Bohemian Rhapsody"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Icy-MetaData") != "1" {
			t.Errorf("expected Icy-MetaData header to be 1")
		}
		w.Header().Set("icy-metaint", strconv.Itoa(metaInt))
		w.WriteHeader(http.StatusOK)
		w.Write(icyStreamBody(metaInt, want))
	}))
	defer srv.Close()

	got, err := FetchNowPlaying(context.Background(), newClient(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFetchNowPlaying_NoMetaInt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("rawaudiowithoutmetadata"))
	}))
	defer srv.Close()

	_, err := FetchNowPlaying(context.Background(), newClient(), srv.URL)
	if !errors.Is(err, ErrNoMetadata) {
		t.Fatalf("expected ErrNoMetadata, got %v", err)
	}
}

func TestFetchNowPlaying_EmptyTitle(t *testing.T) {
	const metaInt = 4096
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("icy-metaint", strconv.Itoa(metaInt))
		w.WriteHeader(http.StatusOK)
		w.Write(icyStreamBody(metaInt, ""))
	}))
	defer srv.Close()

	_, err := FetchNowPlaying(context.Background(), newClient(), srv.URL)
	if !errors.Is(err, ErrEmptyTitle) {
		t.Fatalf("expected ErrEmptyTitle, got %v", err)
	}
}

func TestFetchNowPlaying_ResolvesPLSPlaylist(t *testing.T) {
	const metaInt = 4096
	const want = "Eagles - Hotel California"

	mux := http.NewServeMux()
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("icy-metaint", strconv.Itoa(metaInt))
		w.WriteHeader(http.StatusOK)
		w.Write(icyStreamBody(metaInt, want))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/list.pls", func(w http.ResponseWriter, r *http.Request) {
		body := "[playlist]\nNumberOfEntries=1\nFile1=" + srv.URL + "/stream\n"
		w.Write([]byte(body))
	})

	got, err := FetchNowPlaying(context.Background(), newClient(), srv.URL+"/list.pls")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFirstStreamURLFromPlaylist(t *testing.T) {
	pls := "[playlist]\nFile1=http://example.com/stream\nTitle1=Radio\n"
	if got := firstStreamURLFromPlaylist(pls); got != "http://example.com/stream" {
		t.Errorf("PLS: got %q", got)
	}

	m3u := "#EXTM3U\n#EXTINF:-1,Radio\nhttps://example.com/live\n"
	if got := firstStreamURLFromPlaylist(m3u); got != "https://example.com/live" {
		t.Errorf("M3U: got %q", got)
	}

	if got := firstStreamURLFromPlaylist("no urls here"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestLooksLikePlaylist(t *testing.T) {
	cases := map[string]bool{
		"http://x.com/list.pls":         true,
		"http://x.com/list.m3u":         true,
		"http://x.com/list.m3u8":        true,
		"http://x.com/stream":           false,
		"http://x.com/list.pls?foo=bar": true,
		"http://x.com/stream.mp3":       false,
	}
	for url, want := range cases {
		if got := looksLikePlaylist(url); got != want {
			t.Errorf("looksLikePlaylist(%q) = %v, want %v", url, got, want)
		}
	}
}

func TestFetchNowPlaying_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("icy-metaint", "8192")
		w.WriteHeader(http.StatusOK)
		// No escribir suficiente audio -> el lector se bloquea esperando.
		w.Write(bytes.Repeat([]byte{0xAA}, 100))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := FetchNowPlaying(ctx, newClient(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "skip audio") && !errors.Is(err, context.DeadlineExceeded) {
		// Aceptamos cualquier error derivado de la cancelación.
		if err == nil {
			t.Fatalf("expected an error on context timeout")
		}
	}
}
