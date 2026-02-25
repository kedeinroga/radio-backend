# Cloud Run Service Configuration

locals {
  # Build environment variables map
  env_vars = merge(
    {
      # Server Configuration
      # Note: PORT is automatically set by Cloud Run, don't override it
      SERVER_PORT              = var.app_config.server_port
      SERVER_HOST              = var.app_config.server_host
      SERVER_ENV               = var.environment
      SERVER_TIMEOUT           = var.app_config.server_timeout
      SERVER_BASE_URL          = var.server_base_url

      # JWT Configuration
      JWT_EXPIRATION           = var.app_config.jwt_expiration
      JWT_REFRESH_EXPIRATION   = var.app_config.jwt_refresh_expiration

      # External API
      RADIO_BROWSER_API_URL    = var.app_config.radio_browser_api_url

      # Logging
      LOG_LEVEL                = var.app_config.log_level
      LOG_FORMAT               = var.app_config.log_format

      # i18n
      DEFAULT_LANGUAGE         = var.app_config.default_language
      SUPPORTED_LANGUAGES      = var.app_config.supported_languages

      # Analytics
      ANALYTICS_BATCH_SIZE     = var.app_config.analytics_batch_size
      ANALYTICS_FLUSH_INTERVAL = var.app_config.analytics_flush_interval

      # Security
      BCRYPT_COST              = var.app_config.bcrypt_cost
      RATE_LIMIT_REQUESTS      = var.app_config.rate_limit_requests
      RATE_LIMIT_WINDOW        = var.app_config.rate_limit_window

      # CORS
      CORS_ALLOWED_ORIGINS     = var.cors_allowed_origins
      CORS_ALLOWED_METHODS     = var.app_config.cors_allowed_methods
      CORS_ALLOWED_HEADERS     = var.app_config.cors_allowed_headers

      # Feature Flags
      FEATURE_PREMIUM_CONTENT   = var.app_config.feature_premium_content
      FEATURE_ANALYTICS         = var.app_config.feature_analytics
      FEATURE_VAULT_INTEGRATION = var.app_config.feature_vault_integration

      # Advertising Configuration
      AD_IMPRESSION_TOKEN_MAX_AGE = var.ad_config.impression_token_max_age
      AD_CACHE_TTL                = var.ad_config.cache_ttl
      AD_FREQUENCY_CAP_HOURLY     = var.ad_config.frequency_cap_hourly
      AD_FREQUENCY_CAP_DAILY      = var.ad_config.frequency_cap_daily
      AD_FRAUD_SCORE_THRESHOLD    = var.ad_config.fraud_score_threshold
      AD_RATE_LIMIT_REQUESTS      = var.ad_config.rate_limit_requests
      AD_RATE_LIMIT_WINDOW        = var.ad_config.rate_limit_window
    }
  )
}

resource "google_cloud_run_v2_service" "service" {
  name     = var.service_name
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    service_account = google_service_account.cloudrun.email

    scaling {
      min_instance_count = var.cloudrun_config.min_instances
      max_instance_count = var.cloudrun_config.max_instances
    }

    timeout = "${var.cloudrun_config.timeout_seconds}s"

    containers {
      image = var.container_image

      ports {
        name           = "http1"
        container_port = tonumber(var.app_config.server_port)
      }

      resources {
        limits = {
          cpu    = var.cloudrun_config.cpu
          memory = var.cloudrun_config.memory
        }

        cpu_idle = var.cloudrun_config.cpu_throttling
      }

      # Environment variables
      dynamic "env" {
        for_each = local.env_vars
        content {
          name  = env.key
          value = env.value
        }
      }

      # Single JSON secret bundle — all sensitive secrets in one Secret Manager entry
      env {
        name = "APP_SECRETS_JSON"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.app_secrets.secret_id
            version = "latest"
          }
        }
      }

      # Startup probe
      startup_probe {
        http_get {
          path = "/health"
          port = tonumber(var.app_config.server_port)
        }
        initial_delay_seconds = 5
        timeout_seconds       = 3
        period_seconds        = 10
        failure_threshold     = 3
      }

      # Liveness probe
      liveness_probe {
        http_get {
          path = "/health"
          port = tonumber(var.app_config.server_port)
        }
        initial_delay_seconds = 10
        timeout_seconds       = 3
        period_seconds        = 30
        failure_threshold     = 3
      }
    }
  }

  labels = {
    environment = var.environment
    service     = var.service_name
    managed_by  = "terraform"
  }

  depends_on = [
    google_project_service.apis,
    google_service_account.cloudrun,
    google_secret_manager_secret.app_secrets,
  ]
}

# Make the service publicly accessible
resource "google_cloud_run_v2_service_iam_member" "public_access" {
  name     = google_cloud_run_v2_service.service.name
  location = google_cloud_run_v2_service.service.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
