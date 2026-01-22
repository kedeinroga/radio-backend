# Production Environment - Backend Configuration

terraform {
  backend "gcs" {
    bucket = "radio-485022-terraform-state"
    prefix = "production/terraform/state"
  }
}
