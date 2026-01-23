# Artifact Registry Configuration
# Free Tier: 0.5 GB storage per month

resource "google_artifact_registry_repository" "repository" {
  location      = var.region
  repository_id = var.artifact_registry_repository
  description   = "Docker repository for radio-backend container images"
  format        = "DOCKER"

  labels = {
    environment = var.environment
    service     = var.service_name
    managed_by  = "terraform"
  }

  # Keep only 5 recent versions to stay within free tier storage (0.5 GB)
  cleanup_policies {
    id     = "keep-recent"
    action = "KEEP"

    most_recent_versions {
      keep_count = 5  # Reduced from 10 for free tier
    }
  }

  # Delete untagged images after 2 days (instead of 7) to save storage
  cleanup_policies {
    id     = "delete-old-untagged"
    action = "DELETE"

    condition {
      tag_state  = "UNTAGGED"
      older_than = "172800s" # 2 days (reduced from 7 for free tier)
    }
  }
}

# IAM binding for GitHub Actions to push images
resource "google_artifact_registry_repository_iam_member" "github_actions_writer" {
  repository = google_artifact_registry_repository.repository.name
  location   = google_artifact_registry_repository.repository.location
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.github_actions.email}"
}

# IAM binding for Cloud Run to pull images
resource "google_artifact_registry_repository_iam_member" "cloudrun_reader" {
  repository = google_artifact_registry_repository.repository.name
  location   = google_artifact_registry_repository.repository.location
  role       = "roles/artifactregistry.reader"
  member     = "serviceAccount:${google_service_account.cloudrun.email}"
}
