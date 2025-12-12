package handlers

import (
	"net/http"

	"radio-backend/internal/domain"

	"github.com/gin-gonic/gin"
)

// RespondWithSuccess sends a success response
func RespondWithSuccess(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

// RespondWithError sends an error response
func RespondWithError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

// RespondWithDomainError sends a domain error response
func RespondWithDomainError(c *gin.Context, err *domain.DomainError) {
	status := mapDomainErrorToHTTPStatus(err)

	response := gin.H{
		"error": gin.H{
			"code":    err.Code,
			"message": err.Message,
		},
	}

	if err.Field != "" {
		response["error"].(gin.H)["field"] = err.Field
	}

	c.JSON(status, response)
}

// mapDomainErrorToHTTPStatus maps domain errors to HTTP status codes
func mapDomainErrorToHTTPStatus(err *domain.DomainError) int {
	switch err.Code {
	case "USER_NOT_FOUND", "STATION_NOT_FOUND":
		return http.StatusNotFound
	case "USER_ALREADY_EXISTS":
		return http.StatusConflict
	case "INVALID_CREDENTIALS", "INVALID_TOKEN", "UNAUTHORIZED":
		return http.StatusUnauthorized
	case "VALIDATION_ERROR", "INVALID_QUERY", "INVALID_USER_TYPE":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
