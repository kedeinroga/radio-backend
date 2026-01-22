# Secret Manager Configuration

# Create secrets in Secret Manager
resource "google_secret_manager_secret" "secrets" {
  for_each = var.secrets

  secret_id = each.value

  replication {
    auto {}
  }

  labels = {
    environment = var.environment
    service     = var.service_name
    managed_by  = "terraform"
  }
}

# Create initial secret versions with placeholder values
# This allows the infrastructure to be created without failing
# You MUST update these values before deploying the application
resource "google_secret_manager_secret_version" "secret_versions" {
  for_each = var.secrets

  secret = google_secret_manager_secret.secrets[each.key].id

  # Placeholder value - MUST be changed before production use
  secret_data = "CHANGE_ME_${upper(replace(each.key, "_", "-"))}"

  lifecycle {
    ignore_changes = [secret_data]
  }
}

# Note: Secret values are created with placeholder "CHANGE_ME" values.
# The lifecycle.ignore_changes prevents Terraform from overwriting your real values.
# You must manually update secret versions using:
# - gcloud secrets versions add <secret-name> --data-file=<file>
# - Google Cloud Console
# - Your existing setup-secrets.sh script

# Output instructions for adding secret values
output "secret_instructions" {
  description = "Instructions for adding secret values"
  value = <<-EOT
    Secrets have been created in Secret Manager. Add values using:

    # Database URL (Supabase)
    printf "your-database-url" | gcloud secrets versions add ${var.secrets.database_url} --data-file=-

    # Redis URL (Upstash)
    printf "your-redis-url" | gcloud secrets versions add ${var.secrets.redis_url} --data-file=-

    # JWT Private Key
    cat keys/jwt-private.pem | gcloud secrets versions add ${var.secrets.jwt_private_key} --data-file=-

    # JWT Public Key
    cat keys/jwt-public.pem | gcloud secrets versions add ${var.secrets.jwt_public_key} --data-file=-

    # Ad Impression Token Secret
    printf "your-ad-token-secret" | gcloud secrets versions add ${var.secrets.ad_impression_token_secret} --data-file=-
  EOT
}
