# Variables for Radio Backend Infrastructure

variable "project_id" {
  description = "Google Cloud Project ID"
  type        = string
  default     = "radio-485022"
}

variable "region" {
  description = "Google Cloud Region"
  type        = string
  default     = "us-central1"
}

variable "environment" {
  description = "Environment (production, staging, development)"
  type        = string
  default     = "production"

  validation {
    condition     = contains(["production", "staging", "development"], var.environment)
    error_message = "Environment must be production, staging, or development."
  }
}

variable "service_name" {
  description = "Name of the Cloud Run service"
  type        = string
  default     = "radio-backend"
}

variable "service_account_name" {
  description = "Name of the service account"
  type        = string
  default     = "radio-backend-sa"
}

variable "github_repository" {
  description = "GitHub repository in format owner/repo"
  type        = string
  default     = "kedeinroga/radio-backend"
}

variable "github_branch" {
  description = "GitHub branch for deployments"
  type        = string
  default     = "main"
}

# Cloud Run Configuration
variable "cloudrun_config" {
  description = "Cloud Run service configuration"
  type = object({
    min_instances      = number
    max_instances      = number
    cpu                = string
    memory             = string
    timeout_seconds    = number
    concurrency        = number
    cpu_throttling     = bool
  })

  default = {
    min_instances      = 1
    max_instances      = 3
    cpu                = "1000m"
    memory             = "512Mi"
    timeout_seconds    = 300
    concurrency        = 80
    cpu_throttling     = true
  }
}

# Application Configuration
variable "app_config" {
  description = "Application configuration variables"
  type = object({
    server_port               = string
    server_host               = string
    server_timeout            = string
    jwt_expiration            = string
    jwt_refresh_expiration    = string
    log_level                 = string
    log_format                = string
    default_language          = string
    supported_languages       = string
    analytics_batch_size      = string
    analytics_flush_interval  = string
    bcrypt_cost               = string
    rate_limit_requests       = string
    rate_limit_window         = string
    cors_allowed_methods      = string
    cors_allowed_headers      = string
    feature_premium_content   = string
    feature_analytics         = string
    feature_vault_integration = string
    radio_browser_api_url     = string
  })

  default = {
    server_port               = "8080"
    server_host               = "0.0.0.0"
    server_timeout            = "30s"
    jwt_expiration            = "24h"
    jwt_refresh_expiration    = "168h"
    log_level                 = "info"
    log_format                = "json"
    default_language          = "en"
    supported_languages       = "en,es,fr,de"
    analytics_batch_size      = "100"
    analytics_flush_interval  = "10s"
    bcrypt_cost               = "12"
    rate_limit_requests       = "100"
    rate_limit_window         = "1m"
    cors_allowed_methods      = "GET,POST,PUT,DELETE,OPTIONS"
    cors_allowed_headers      = "Content-Type,Authorization,X-Language,X-Request-ID"
    feature_premium_content   = "true"
    feature_analytics         = "true"
    feature_vault_integration = "false"
    radio_browser_api_url     = "https://de1.api.radio-browser.info"
  }
}

# Advertising Configuration
variable "ad_config" {
  description = "Advertising configuration variables"
  type = object({
    impression_token_max_age = string
    cache_ttl                = string
    frequency_cap_hourly     = string
    frequency_cap_daily      = string
    fraud_score_threshold    = string
    rate_limit_requests      = string
    rate_limit_window        = string
  })

  default = {
    impression_token_max_age = "5m"
    cache_ttl                = "10m"
    frequency_cap_hourly     = "6"
    frequency_cap_daily      = "30"
    fraud_score_threshold    = "0.7"
    rate_limit_requests      = "50"
    rate_limit_window        = "1m"
  }
}

# CORS Origins (environment-specific)
variable "cors_allowed_origins" {
  description = "CORS allowed origins"
  type        = string
  default     = "https://your-frontend-domain.com"
}

# Server Base URL (environment-specific)
variable "server_base_url" {
  description = "Server base URL"
  type        = string
  default     = "https://radio-backend-296736956418.us-central1.run.app"
}

# Secrets Configuration
variable "secrets" {
  description = "Secret names in Secret Manager"
  type = object({
    database_url              = string
    redis_url                 = string
    jwt_private_key           = string
    jwt_public_key            = string
    ad_impression_token_secret = string
  })

  default = {
    database_url              = "database-url"
    redis_url                 = "redis-url"
    jwt_private_key           = "jwt-private-key"
    jwt_public_key            = "jwt-public-key"
    ad_impression_token_secret = "ad-impression-token-secret"
  }
}

# Container Image
variable "container_image" {
  description = "Container image URL"
  type        = string
  default     = "gcr.io/radio-485022/radio-backend:latest"
}

# Artifact Registry
variable "artifact_registry_repository" {
  description = "Artifact Registry repository name"
  type        = string
  default     = "radio-backend"
}
