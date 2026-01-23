# Production Environment Configuration

project_id   = "radio-485022"
region       = "us-central1"
environment  = "production"
service_name = "radio-backend"

github_repository = "kedeinroga/radio-backend"
github_branch     = "main"

# Cloud Run Configuration - Production
cloudrun_config = {
  min_instances   = 1
  max_instances   = 3
  cpu             = "1000m"
  memory          = "512Mi"
  timeout_seconds = 300
  concurrency     = 80
  cpu_throttling  = true
}

# Production URLs
server_base_url       = "https://api.rradio.online"
cors_allowed_origins  = "https://rradio.online"

# Container Image
container_image = "gcr.io/radio-485022/radio-backend:latest"
