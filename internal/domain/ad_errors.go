package domain

// Campaign Errors
var (
	ErrInvalidCampaignName     = &DomainError{Code: "INVALID_CAMPAIGN_NAME", Message: "campaign name is required"}
	ErrInvalidBudget           = &DomainError{Code: "INVALID_BUDGET", Message: "budget must be greater than zero"}
	ErrInvalidCampaignDates    = &DomainError{Code: "INVALID_CAMPAIGN_DATES", Message: "end date must be after start date"}
	ErrCampaignNotFound        = &DomainError{Code: "CAMPAIGN_NOT_FOUND", Message: "campaign not found"}
	ErrCampaignBudgetExhausted = &DomainError{Code: "CAMPAIGN_BUDGET_EXHAUSTED", Message: "campaign budget exhausted"}
)

// Advertisement Errors
var (
	ErrInvalidAdTitle         = &DomainError{Code: "INVALID_AD_TITLE", Message: "advertisement title is required"}
	ErrInvalidAdFormat        = &DomainError{Code: "INVALID_AD_FORMAT", Message: "invalid advertisement format"}
	ErrInvalidAdType          = &DomainError{Code: "INVALID_AD_TYPE", Message: "invalid advertisement type"}
	ErrInvalidMediaURL        = &DomainError{Code: "INVALID_MEDIA_URL", Message: "media URL is required"}
	ErrInvalidClickURL        = &DomainError{Code: "INVALID_CLICK_URL", Message: "click URL is required"}
	ErrInvalidPricingModel    = &DomainError{Code: "INVALID_PRICING_MODEL", Message: "either CPM or CPC rate must be set"}
	ErrInvalidDimensions      = &DomainError{Code: "INVALID_DIMENSIONS", Message: "banner ads require width and height"}
	ErrInvalidDuration        = &DomainError{Code: "INVALID_DURATION", Message: "audio/video ads require duration"}
	ErrAdvertisementNotFound  = &DomainError{Code: "ADVERTISEMENT_NOT_FOUND", Message: "advertisement not found"}
	ErrAdvertisementNotActive = &DomainError{Code: "ADVERTISEMENT_NOT_ACTIVE", Message: "advertisement is not active"}
)

// Impression Errors
var (
	ErrInvalidImpressionData  = &DomainError{Code: "INVALID_IMPRESSION_DATA", Message: "invalid impression data"}
	ErrImpressionNotFound     = &DomainError{Code: "IMPRESSION_NOT_FOUND", Message: "impression not found"}
	ErrInvalidImpressionToken = &DomainError{Code: "INVALID_IMPRESSION_TOKEN", Message: "invalid or expired impression token"}
	ErrDuplicateImpression    = &DomainError{Code: "DUPLICATE_IMPRESSION", Message: "duplicate impression detected"}
)

// Click Errors
var (
	ErrInvalidClickData = &DomainError{Code: "INVALID_CLICK_DATA", Message: "invalid click data"}
	ErrClickNotFound    = &DomainError{Code: "CLICK_NOT_FOUND", Message: "click not found"}
)

// User Ad Profile Errors
var (
	ErrUserProfileNotFound  = &DomainError{Code: "USER_PROFILE_NOT_FOUND", Message: "user ad profile not found"}
	ErrFrequencyCapExceeded = &DomainError{Code: "FREQUENCY_CAP_EXCEEDED", Message: "user has reached ad frequency cap"}
	ErrPremiumUserNoAds     = &DomainError{Code: "PREMIUM_USER_NO_ADS", Message: "premium users should not see ads"}
)

// Security Errors (nuevos, no duplicados)
var (
	ErrForbidden          = &DomainError{Code: "FORBIDDEN", Message: "forbidden: insufficient permissions"}
	ErrSuspiciousActivity = &DomainError{Code: "SUSPICIOUS_ACTIVITY", Message: "suspicious activity detected"}
	ErrRateLimitExceeded  = &DomainError{Code: "RATE_LIMIT_EXCEEDED", Message: "rate limit exceeded"}
	ErrTokenExpired       = &DomainError{Code: "TOKEN_EXPIRED", Message: "token expired"}
)

// General Errors
var (
	ErrInternalServer = &DomainError{Code: "INTERNAL_SERVER_ERROR", Message: "internal server error"}
	ErrDatabaseError  = &DomainError{Code: "DATABASE_ERROR", Message: "database error"}
	ErrCacheError     = &DomainError{Code: "CACHE_ERROR", Message: "cache error"}
)
