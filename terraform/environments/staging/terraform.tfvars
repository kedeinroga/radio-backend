# Staging Environment Configuration

project_id   = "radio-485022"
region       = "us-central1"
environment  = "staging"
service_name = "radio-backend-staging"

github_repository = "kedeinroga/radio-backend"
github_branch     = "develop"

# Cloud Run Configuration - Staging
cloudrun_config = {
  min_instances   = 0  # Scale to zero when not in use
  max_instances   = 3
  cpu             = "1000m"
  memory          = "512Mi"
  timeout_seconds = 300
  concurrency     = 80
  cpu_throttling  = true
}

# Staging URLs
server_base_url       = "https://radio-backend-staging-296736956418.us-central1.run.app"
cors_allowed_origins  = "http://localhost:3000,http://localhost:5173,https://staging-frontend.com"

# Container Image
container_image = "gcr.io/radio-485022/radio-backend:staging"

# Different log level for staging
app_config = {
  server_port               = "8080"
  server_host               = "0.0.0.0"
  server_timeout            = "30s"
  jwt_expiration            = "24h"
  jwt_refresh_expiration    = "168h"
  log_level                 = "debug"  # More verbose in staging
  log_format                = "json"
  default_language          = "en"
  supported_languages       = "en,es,fr,de"
  analytics_batch_size      = "100"
  analytics_flush_interval  = "10s"
  bcrypt_cost               = "10"  # Faster in staging
  rate_limit_requests       = "100"
  rate_limit_window         = "1m"
  cors_allowed_methods      = "GET,POST,PUT,DELETE,OPTIONS"
  cors_allowed_headers      = "Content-Type,Authorization,X-Language,X-Request-ID"
  feature_premium_content   = "true"
  feature_analytics         = "true"
  feature_vault_integration = "false"
  radio_browser_api_url     = "https://de1.api.radio-browser.info"
}
