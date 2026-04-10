package middleware

import (
	"net/url"
	"strings"
	"time"

	"radio-backend/internal/domain"

	"github.com/gin-gonic/gin"
)

// RequestSource identifies the likely origin type of an HTTP request.
type RequestSource string

const (
	SourceBrowser RequestSource = "browser"
	SourcePostman RequestSource = "postman"
	SourceCurl    RequestSource = "curl"
	SourceScript  RequestSource = "script"
	SourceUnknown RequestSource = "unknown"
)

// ContextKeyRequestSource is the gin context key for the classified request source.
const ContextKeyRequestSource = "request_source"

// sensitivePathPrefixes are paths that warrant logging when accessed by non-browser clients.
var sensitivePathPrefixes = []string{
	"/api/v1/auth/",
	"/api/v1/admin/",
	"/api/v1/favorites",
}

// ClassifyRequest determines the likely source of a request by inspecting
// the User-Agent and Sec-Fetch-* headers.
//
// Limitations: any header can be spoofed by a determined attacker.
// This is a heuristic that raises the cost of undetected probing,
// not a cryptographic guarantee of origin.
func ClassifyRequest(c *gin.Context) RequestSource {
	ua := c.Request.UserAgent()
	uaLower := strings.ToLower(ua)

	switch {
	case strings.Contains(ua, "PostmanRuntime"):
		return SourcePostman
	case strings.HasPrefix(uaLower, "curl/"):
		return SourceCurl
	case strings.Contains(uaLower, "python-requests"),
		strings.Contains(uaLower, "python-httpx"),
		strings.Contains(uaLower, "go-http-client"),
		strings.Contains(uaLower, "node-fetch"),
		strings.Contains(uaLower, "got/"),
		strings.Contains(uaLower, "httpie"),
		strings.Contains(uaLower, "insomnia/"):
		return SourceScript
	}

	// Sec-Fetch-* headers are injected automatically by browsers and are absent
	// from most HTTP libraries unless explicitly set.
	if c.GetHeader("Sec-Fetch-Site") != "" {
		return SourceBrowser
	}

	return SourceUnknown
}

// GetRequestSource retrieves the classified source from the gin context.
// Returns SourceUnknown if the fingerprint middleware has not run.
func GetRequestSource(c *gin.Context) RequestSource {
	if v, exists := c.Get(ContextKeyRequestSource); exists {
		if src, ok := v.(RequestSource); ok {
			return src
		}
	}
	return SourceUnknown
}

// isSensitivePath returns true when the path warrants source monitoring.
func isSensitivePath(path string) bool {
	for _, prefix := range sensitivePathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// isTrustedOrigin checks whether the request Origin or Referer belongs to
// one of the trusted hosts (e.g. from cfg.CORS.AllowedOrigins).
// Wildcard "*" is intentionally NOT treated as trusted.
func isTrustedOrigin(origin, referer string, trustedOrigins []string) bool {
	normalize := func(raw string) string { return strings.TrimRight(strings.ToLower(raw), "/") }

	for _, trusted := range trustedOrigins {
		if trusted == "*" {
			continue
		}
		t := normalize(trusted)
		if normalize(origin) == t {
			return true
		}
		if referer != "" {
			if u, err := url.Parse(referer); err == nil {
				base := u.Scheme + "://" + u.Host
				if normalize(base) == t {
					return true
				}
			}
		}
	}
	return false
}

// RequestFingerprintMiddleware classifies each request by likely source and
// asynchronously logs suspicious non-browser accesses to sensitive endpoints
// into the security_events table.
//
// It NEVER blocks requests — it is purely observational.
// Rate limiting and authentication remain the enforcement layers.
func RequestFingerprintMiddleware(securityRepo domain.SecurityRepository, trustedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		source := ClassifyRequest(c)
		c.Set(ContextKeyRequestSource, source)

		path := c.Request.URL.Path

		if source != SourceBrowser && isSensitivePath(path) {
			origin := c.GetHeader("Origin")
			referer := c.GetHeader("Referer")

			if !isTrustedOrigin(origin, referer, trustedOrigins) {
				ip := c.ClientIP()
				ua := c.Request.UserAgent()
				method := c.Request.Method
				secFetchSite := c.GetHeader("Sec-Fetch-Site")
				secFetchMode := c.GetHeader("Sec-Fetch-Mode")
				secFetchDest := c.GetHeader("Sec-Fetch-Dest")

				go func() {
					_ = securityRepo.LogSecurityEvent(&domain.SecurityEvent{
						Timestamp: time.Now(),
						Event:     "suspicious_request_source",
						IPAddress: ip,
						UserAgent: ua,
						Reason:    "non-browser client on sensitive endpoint",
						Metadata: map[string]interface{}{
							"source":         string(source),
							"path":           path,
							"method":         method,
							"origin":         origin,
							"referer":        referer,
							"sec_fetch_site": secFetchSite,
							"sec_fetch_mode": secFetchMode,
							"sec_fetch_dest": secFetchDest,
						},
					})
				}()
			}
		}

		c.Next()
	}
}
