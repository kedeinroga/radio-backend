package handlers

// This file centralizes the shared response DTOs used across handlers for
// Swagger/Scalar documentation. They are kept faithful to what the handlers
// actually return through RespondWithSuccess / RespondWithError /
// RespondWithDomainError (see response.go).

// ErrorResponse is the canonical error envelope returned by every endpoint on
// failure. It mirrors exactly the JSON produced by RespondWithError and
// RespondWithDomainError: {"error": {"code": "...", "message": "...", "field": "..."}}.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail holds the machine-readable code and a human message. The code and
// message vary per endpoint (e.g. STATION_NOT_FOUND, INVALID_CREDENTIALS,
// fetch_failed); the values shown here are only illustrative. `field` is
// OPTIONAL and is included exclusively in validation errors (VALIDATION_ERROR)
// to point at the offending input field.
type ErrorDetail struct {
	Code    string `json:"code" example:"STATION_NOT_FOUND"`
	Message string `json:"message" example:"station not found"`
	Field   string `json:"field,omitempty" example:"email (only present on validation errors)"`
}

// SuccessResponse is a generic message-only success envelope: {"message": "..."}.
type SuccessResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// SimpleErrorResponse is the flat error shape produced by the authentication
// middlewares (auth.go / ad_auth.go): {"error": "message"}. It differs from
// ErrorResponse, which is the nested envelope produced by the handlers. Use it
// to document middleware-produced 401/403 responses on protected routes.
type SimpleErrorResponse struct {
	Error string `json:"error" example:"authentication required"`
}

// UserSummary is the public representation of a user returned by auth endpoints.
type UserSummary struct {
	ID       string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Email    string `json:"email" example:"user@example.com"`
	UserType string `json:"user_type" example:"guest"`
}

// UserEnvelope wraps a user under the "user" key, matching {"user": {...}}.
type UserEnvelope struct {
	User UserSummary `json:"user"`
}

// MetaCount is the common "meta" object that carries a result count.
type MetaCount struct {
	Count int `json:"count" example:"20"`
}

// FavoriteStationDTO is the reduced station shape returned in the favorites
// list (no slug / SEO metadata, unlike the full StationDTO).
type FavoriteStationDTO struct {
	ID            string   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name          string   `json:"name" example:"Rock FM 100.1"`
	StreamURL     string   `json:"stream_url" example:"https://stream.rockfm.com/live"`
	ImageURL      string   `json:"image_url,omitempty" example:"https://cdn.rockfm.com/logo.png"`
	Tags          []string `json:"tags" example:"rock,classic rock"`
	Country       string   `json:"country" example:"United States"`
	Votes         int      `json:"votes" example:"1500"`
	IsPremiumOnly bool     `json:"is_premium_only" example:"false"`
}

// FavoriteListResponse is the envelope returned by GET /favorites.
type FavoriteListResponse struct {
	Data []FavoriteStationDTO `json:"data"`
	Meta MetaCount            `json:"meta"`
}
