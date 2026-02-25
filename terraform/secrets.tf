# Secret Manager Configuration
# Free Tier: 6 active secret versions, 10,000 access operations/month
#
# Strategy: ONE single secret "app-secrets" containing a JSON bundle
# with all sensitive key-value pairs. This reduces:
#   - Active secret versions:  5 → 1
#   - IAM bindings:            5 → 1
#   - Secret access ops/startup: 5 → 1

resource "google_secret_manager_secret" "app_secrets" {
  secret_id = var.app_secrets_name

  replication {
    auto {}
  }

  labels = {
    environment = var.environment
    service     = var.service_name
    managed_by  = "terraform"
  }
}

# Create initial secret version with placeholder JSON.
# IMPORTANT: This resource only creates the INITIAL version.
# After you manually update the secret values, Terraform will NOT overwrite them.
# The lifecycle block prevents Terraform from recreating or modifying versions.
resource "google_secret_manager_secret_version" "app_secrets_version" {
  secret = google_secret_manager_secret.app_secrets.id

  # Placeholder JSON — only used for initial creation.
  # Update the real values using:
  #   gcloud secrets versions add app-secrets --data-file=secrets.json
  # or via the Google Cloud Console.
  secret_data = jsonencode({
    DATABASE_URL               = "CHANGE_ME"
    REDIS_URL                  = "CHANGE_ME"
    JWT_PRIVATE_KEY            = "CHANGE_ME"
    JWT_PUBLIC_KEY             = "CHANGE_ME"
    AD_IMPRESSION_TOKEN_SECRET = "CHANGE_ME"
    API_SECRET_KEY             = "CHANGE_ME"
  })

  lifecycle {
    # Prevent Terraform from ever recreating this resource.
    # This ensures manually updated secret values are preserved.
    ignore_changes = all
  }
}

output "secret_instructions" {
  description = "Instructions for updating the secrets bundle"
  value       = <<-EOT
    One secret has been created: "${var.app_secrets_name}"
    Update all sensitive values at once using a local JSON file:

    cat > /tmp/secrets.json <<'EOF'
    {
      "DATABASE_URL":               "your-supabase-url",
      "REDIS_URL":                  "your-upstash-url",
      "JWT_PRIVATE_KEY":            "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----",
      "JWT_PUBLIC_KEY":             "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----",
      "AD_IMPRESSION_TOKEN_SECRET": "your-ad-token-secret",
      "API_SECRET_KEY":             "your-shared-api-secret"
    }
    EOF

    gcloud secrets versions add ${var.app_secrets_name} --data-file=/tmp/secrets.json
    rm /tmp/secrets.json
  EOT
}
