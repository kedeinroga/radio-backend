package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetUUIDParam parses a UUID from a URL parameter.
// If the parameter is invalid, it sends a 400 Bad Request response and returns false.
// If valid, it returns the parsed UUID and true.
func GetUUIDParam(c *gin.Context, paramName string) (uuid.UUID, bool) {
	idStr := c.Param(paramName)
	if idStr == "" {
		RespondWithError(c, http.StatusBadRequest, "invalid_id", "Missing "+paramName)
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_id", "Invalid "+paramName+" format")
		return uuid.Nil, false
	}

	return id, true
}
