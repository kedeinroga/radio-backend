# Terraform Outputs

output "project_id" {
  description = "Google Cloud Project ID"
  value       = var.project_id
}

output "region" {
  description = "Google Cloud Region"
  value       = var.region
}

output "environment" {
  description = "Environment name"
  value       = var.environment
}

# Service Accounts
output "cloudrun_service_account_email" {
  description = "Email of the Cloud Run service account"
  value       = google_service_account.cloudrun.email
}

output "github_actions_service_account_email" {
  description = "Email of the GitHub Actions service account"
  value       = google_service_account.github_actions.email
}

# Cloud Run
output "cloudrun_service_url" {
  description = "URL of the deployed Cloud Run service"
  value       = google_cloud_run_v2_service.service.uri
}

output "cloudrun_service_name" {
  description = "Name of the Cloud Run service"
  value       = google_cloud_run_v2_service.service.name
}

# Artifact Registry
output "artifact_registry_repository" {
  description = "Artifact Registry repository name"
  value       = google_artifact_registry_repository.repository.name
}

output "artifact_registry_url" {
  description = "Artifact Registry repository URL"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repository.repository_id}"
}

# Workload Identity
output "workload_identity_provider" {
  description = "Workload Identity Provider resource name"
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "workload_identity_pool" {
  description = "Workload Identity Pool resource name"
  value       = google_iam_workload_identity_pool.github.name
}

# GitHub Actions Configuration
output "github_actions_secrets" {
  description = "GitHub Actions secrets configuration"
  value = {
    GCP_PROJECT_ID          = var.project_id
    GCP_REGION              = var.region
    GCP_SERVICE_ACCOUNT     = google_service_account.github_actions.email
    WORKLOAD_IDENTITY_PROVIDER = google_iam_workload_identity_pool_provider.github.name
    ARTIFACT_REGISTRY_URL   = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repository.repository_id}"
    CLOUDRUN_SERVICE_NAME   = var.service_name
  }
  sensitive = false
}

# Secrets
output "secret_names" {
  description = "Names of secrets in Secret Manager"
  value       = var.secrets
}

# Configuration Summary
output "infrastructure_summary" {
  description = "Summary of created infrastructure"
  value = <<-EOT
    ═══════════════════════════════════════════════════════════════
    Radio Backend Infrastructure - ${upper(var.environment)}
    ═══════════════════════════════════════════════════════════════

    📦 Project Details:
       Project ID: ${var.project_id}
       Region:     ${var.region}
       Environment: ${var.environment}

    🚀 Cloud Run Service:
       Name:       ${google_cloud_run_v2_service.service.name}
       URL:        ${google_cloud_run_v2_service.service.uri}
       Instances:  ${var.cloudrun_config.min_instances} - ${var.cloudrun_config.max_instances}
       CPU:        ${var.cloudrun_config.cpu}
       Memory:     ${var.cloudrun_config.memory}

    🔐 Service Accounts:
       Cloud Run:       ${google_service_account.cloudrun.email}
       GitHub Actions:  ${google_service_account.github_actions.email}

    📦 Artifact Registry:
       Repository: ${google_artifact_registry_repository.repository.name}
       URL:        ${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repository.repository_id}

    🔑 Workload Identity:
       Pool:       ${google_iam_workload_identity_pool.github.workload_identity_pool_id}
       Provider:   ${google_iam_workload_identity_pool_provider.github.workload_identity_pool_provider_id}
       Repository: ${var.github_repository}

    🔒 Secrets (Secret Manager):
       - ${join("\n       - ", values(var.secrets))}

    ═══════════════════════════════════════════════════════════════

    📝 Next Steps:
       1. Add secret values (see 'secret_instructions' output)
       2. Configure GitHub Actions secrets
       3. Push container image to Artifact Registry
       4. Deploy application to Cloud Run

  EOT
}
