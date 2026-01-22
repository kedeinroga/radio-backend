# Staging Environment - Backend Configuration

terraform {
  backend "gcs" {
    bucket = "radio-485022-terraform-state"
    prefix = "staging/terraform/state"
  }
}
