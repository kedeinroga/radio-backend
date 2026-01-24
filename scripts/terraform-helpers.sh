#!/bin/bash

# Terraform Helper Script
# Simplifies common Terraform operations for different environments

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="$SCRIPT_DIR/../terraform"

usage() {
  echo -e "${BLUE}Usage:${NC} $0 <command> <environment>"
  echo ""
  echo "Commands:"
  echo "  init       - Initialize Terraform"
  echo "  plan       - Show execution plan"
  echo "  apply      - Apply changes"
  echo "  destroy    - Destroy infrastructure"
  echo "  output     - Show outputs"
  echo "  validate   - Validate configuration"
  echo "  fmt        - Format Terraform files"
  echo "  state      - Show state list"
  echo ""
  echo "Environments:"
  echo "  production"
  echo "  staging"
  echo ""
  echo "Examples:"
  echo "  $0 plan production"
  echo "  $0 apply staging"
  echo "  $0 output production"
  exit 1
}

if [ $# -lt 1 ]; then
  usage
fi

COMMAND=$1
ENVIRONMENT=${2:-production}

if [[ ! "$ENVIRONMENT" =~ ^(production|staging)$ ]]; then
  echo -e "${RED}Error: Invalid environment. Use 'production' or 'staging'${NC}"
  exit 1
fi

TFVARS_FILE="$TERRAFORM_DIR/environments/$ENVIRONMENT/terraform.tfvars"
BACKEND_CONFIG="$TERRAFORM_DIR/environments/$ENVIRONMENT/backend.tf"

if [ ! -f "$TFVARS_FILE" ]; then
  echo -e "${RED}Error: tfvars file not found: $TFVARS_FILE${NC}"
  exit 1
fi

cd "$TERRAFORM_DIR"

echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo -e "${BLUE}Terraform - Radio Backend${NC}"
echo -e "${BLUE}Environment: ${YELLOW}$ENVIRONMENT${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo ""

case $COMMAND in
  init)
    echo -e "${GREEN}Initializing Terraform...${NC}"
    terraform init -backend-config="$BACKEND_CONFIG" -reconfigure
    ;;

  plan)
    echo -e "${GREEN}Generating execution plan...${NC}"
    terraform plan -var-file="$TFVARS_FILE" -out="tfplan-$ENVIRONMENT"
    echo ""
    echo -e "${YELLOW}Plan saved to: tfplan-$ENVIRONMENT${NC}"
    echo -e "${YELLOW}Apply with: terraform apply tfplan-$ENVIRONMENT${NC}"
    ;;

  apply)
    echo -e "${GREEN}Applying changes...${NC}"
    if [ -f "tfplan-$ENVIRONMENT" ]; then
      echo -e "${YELLOW}Using saved plan: tfplan-$ENVIRONMENT${NC}"
      terraform apply "tfplan-$ENVIRONMENT"
      rm "tfplan-$ENVIRONMENT"
    else
      echo -e "${YELLOW}No saved plan found, generating new one...${NC}"
      terraform apply -var-file="$TFVARS_FILE"
    fi
    ;;

  destroy)
    echo -e "${RED}⚠️  WARNING: This will destroy all infrastructure!${NC}"
    echo -e "${RED}Environment: $ENVIRONMENT${NC}"
    read -p "Are you sure? Type 'yes' to confirm: " -r
    echo
    if [[ $REPLY == "yes" ]]; then
      terraform destroy -var-file="$TFVARS_FILE"
    else
      echo -e "${YELLOW}Destroy cancelled${NC}"
      exit 0
    fi
    ;;

  output)
    echo -e "${GREEN}Terraform Outputs:${NC}"
    echo ""
    if [ $# -eq 3 ]; then
      terraform output "$3"
    else
      terraform output
    fi
    ;;

  validate)
    echo -e "${GREEN}Validating configuration...${NC}"
    terraform validate
    echo -e "${GREEN}✓ Configuration is valid${NC}"
    ;;

  fmt)
    echo -e "${GREEN}Formatting Terraform files...${NC}"
    terraform fmt -recursive
    echo -e "${GREEN}✓ Files formatted${NC}"
    ;;

  state)
    echo -e "${GREEN}Terraform State:${NC}"
    echo ""
    terraform state list
    ;;

  *)
    echo -e "${RED}Error: Unknown command '$COMMAND'${NC}"
    echo ""
    usage
    ;;
esac

echo ""
echo -e "${GREEN}✓ Done!${NC}"
