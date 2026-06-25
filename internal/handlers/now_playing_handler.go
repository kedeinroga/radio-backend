package handlers

import (
	"net/http"
	"strconv"
	"time"

	"radio-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// NowPlayingHandler handles now-playing / recently-played endpoints.
type NowPlayingHandler struct {
	nowPlayingService *services.NowPlayingService
}

// NewNowPlayingHandler creates a new now-playing handler.
func NewNowPlayingHandler(nowPlayingService *services.NowPlayingService) *NowPlayingHandler {
	return &NowPlayingHandler{nowPlayingService: nowPlayingService}
}

// TrackDTO represents a track in API responses.
type TrackDTO struct {
	StationID string    `json:"station_id"`
	RawTitle  string    `json:"raw_title"`
	Artist    string    `json:"artist,omitempty"`
	Title     string    `json:"title"`
	PlayedAt  time.Time `json:"played_at"`
}

// RecentTracksResponse represents the response for the recent-tracks endpoint.
type RecentTracksResponse struct {
	Data []TrackDTO `json:"data"`
}

// GetNowPlaying returns the track currently playing on a station.
// @Summary Get now-playing track
// @Description Returns the song currently detected on a station from ICY metadata. 204 if unknown.
// @Tags Stations
// @Produce json
// @Security SharedSecret
// @Param id path string true "Station ID"
// @Success 200 {object} TrackDTO "Currently playing track"
// @Success 204 "No now-playing data available"
// @Failure 400 {object} ErrorResponse "Station ID is required"
// @Failure 500 {object} ErrorResponse "Failed to get now-playing data"
// @Router /stations/{id}/now-playing [get]
func (h *NowPlayingHandler) GetNowPlaying(c *gin.Context) {
	stationID := c.Param("id")
	if stationID == "" {
		RespondWithError(c, http.StatusBadRequest, "invalid_id", "Station ID is required")
		return
	}

	track, err := h.nowPlayingService.GetNowPlaying(c.Request.Context(), stationID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "now_playing_error", "Failed to get now-playing data")
		return
	}
	if track == nil {
		c.Status(http.StatusNoContent)
		return
	}

	c.Header("Cache-Control", "public, max-age=30")
	c.JSON(http.StatusOK, TrackDTO{
		StationID: track.StationID,
		RawTitle:  track.RawTitle,
		Artist:    track.Artist,
		Title:     track.Title,
		PlayedAt:  track.PlayedAt,
	})
}

// GetRecentTracks returns the recently played tracks of a station.
// @Summary Get recently-played tracks
// @Description Returns the recent track history of a station from ICY metadata.
// @Tags Stations
// @Produce json
// @Security SharedSecret
// @Param id path string true "Station ID"
// @Param limit query int false "Max tracks (1-50)" default(10)
// @Success 200 {object} RecentTracksResponse "Recent track history"
// @Failure 400 {object} ErrorResponse "Station ID is required"
// @Failure 500 {object} ErrorResponse "Failed to get recent tracks"
// @Router /stations/{id}/recent-tracks [get]
func (h *NowPlayingHandler) GetRecentTracks(c *gin.Context) {
	stationID := c.Param("id")
	if stationID == "" {
		RespondWithError(c, http.StatusBadRequest, "invalid_id", "Station ID is required")
		return
	}

	limit := 10
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	tracks, err := h.nowPlayingService.GetRecentTracks(c.Request.Context(), stationID, limit)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "recent_tracks_error", "Failed to get recent tracks")
		return
	}

	dtos := make([]TrackDTO, 0, len(tracks))
	for _, t := range tracks {
		dtos = append(dtos, TrackDTO{
			StationID: t.StationID,
			RawTitle:  t.RawTitle,
			Artist:    t.Artist,
			Title:     t.Title,
			PlayedAt:  t.PlayedAt,
		})
	}

	c.Header("Cache-Control", "public, max-age=30")
	c.JSON(http.StatusOK, RecentTracksResponse{Data: dtos})
}
