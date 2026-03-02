# IAM Configuration - Service Accounts and Permissions

# Service Account for Cloud Run
resource "google_service_account" "cloudrun" {
  account_id   = var.service_account_name
  display_name = "Radio Backend Cloud Run Service Account"
  description  = "Service account for running the radio-backend Cloud Run service"
}

# IAM Roles for Cloud Run Service Account
resource "google_project_iam_member" "cloudrun_roles" {
  for_each = toset([
    "roles/run.invoker",        # Invoke Cloud Run services
    "roles/logging.logWriter",  # Write logs
    "roles/cloudtrace.agent",   # Write traces
    "roles/monitoring.metricWriter", # Write metrics
  ])

  project = var.project_id
  role    = each.key
  member  = "serviceAccount:${google_service_account.cloudrun.email}"
}

# Secret Manager access for Cloud Run Service Account (project-level)
# Using project_iam_member instead of secret_iam_member avoids needing
# secretmanager.secrets.setIamPolicy, which would require secretmanager.admin.
resource "google_project_iam_member" "cloudrun_secret_access" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.cloudrun.email}"
}

# Service Account for GitHub Actions (Workload Identity)
resource "google_service_account" "github_actions" {
  account_id   = "${var.service_account_name}-github"
  display_name = "Radio Backend GitHub Actions"
  description  = "Service account for GitHub Actions CI/CD"
}

# IAM Roles for GitHub Actions Service Account
resource "google_project_iam_member" "github_actions_roles" {
  for_each = toset([
    "roles/run.admin",                    # Manage Cloud Run services
    "roles/iam.serviceAccountUser",       # Use service accounts
    "roles/artifactregistry.writer",      # Push container images
    "roles/secretmanager.secretAccessor", # Read secrets (for validation/audit)
  ])

  project = var.project_id
  role    = each.key
  member  = "serviceAccount:${google_service_account.github_actions.email}"
}

# Allow GitHub Actions SA to act as Cloud Run SA
resource "google_service_account_iam_member" "github_actions_impersonate" {
  service_account_id = google_service_account.cloudrun.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.github_actions.email}"
}
