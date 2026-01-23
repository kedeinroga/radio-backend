#!/bin/bash

# Clean Infrastructure and Start Fresh
# This script destroys existing infrastructure and prepares for Terraform

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PROJECT_ID="---"
REGION="---"

echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo -e "${BLUE}Clean Infrastructure - Start Fresh${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo ""
echo -e "${RED}⚠️  WARNING: This will DELETE ALL infrastructure!${NC}"
echo -e "${RED}This includes:${NC}"
echo "  • Cloud Run services"
echo "  • Service Accounts"
echo "  • Workload Identity Pool"
echo "  • IAM bindings"
echo "  • Secrets in Secret Manager"
echo ""
echo -e "${YELLOW}Preserved:${NC}"
echo "  • Artifact Registry (container images)"
echo ""
read -p "Are you sure? Type 'yes' to continue: " -r
echo
if [[ ! $REPLY == "yes" ]]; then
    echo -e "${YELLOW}Cancelled${NC}"
    exit 0
fi

echo ""
echo -e "${BLUE}Step 1: Deleting Cloud Run services...${NC}"
echo ""

# Delete Cloud Run services
for service in radio-backend radio-backend-staging; do
  if gcloud run services describe $service --region=$REGION --project=$PROJECT_ID &>/dev/null; then
    echo -e "${YELLOW}Deleting service: $service${NC}"
    gcloud run services delete $service \
      --region=$REGION \
      --project=$PROJECT_ID \
      --quiet
    echo -e "${GREEN}✓ Deleted $service${NC}"
  else
    echo -e "${YELLOW}⊘ Service $service does not exist${NC}"
  fi
done

echo ""
echo -e "${BLUE}Step 2: Deleting Workload Identity Pool...${NC}"
echo ""

# Delete Workload Identity Provider
if gcloud iam workload-identity-pools providers describe github-provider \
    --workload-identity-pool=github-pool \
    --location=global \
    --project=$PROJECT_ID &>/dev/null; then
  echo -e "${YELLOW}Deleting Workload Identity Provider...${NC}"
  gcloud iam workload-identity-pools providers delete github-provider \
    --workload-identity-pool=github-pool \
    --location=global \
    --project=$PROJECT_ID \
    --quiet
  echo -e "${GREEN}✓ Deleted provider${NC}"
fi

# Delete Workload Identity Pool
if gcloud iam workload-identity-pools describe github-pool \
    --location=global \
    --project=$PROJECT_ID &>/dev/null; then
  echo -e "${YELLOW}Deleting Workload Identity Pool...${NC}"
  gcloud iam workload-identity-pools delete github-pool \
    --location=global \
    --project=$PROJECT_ID \
    --quiet
  echo -e "${GREEN}✓ Deleted pool${NC}"
fi

echo ""
echo -e "${BLUE}Step 3: Removing IAM bindings...${NC}"
echo ""

# Remove IAM bindings from service accounts
for sa in radio-backend-sa radio-backend-sa-github; do
  if gcloud iam service-accounts describe ${sa}@${PROJECT_ID}.iam.gserviceaccount.com \
      --project=$PROJECT_ID &>/dev/null; then

    echo -e "${YELLOW}Removing IAM bindings for: $sa${NC}"

    # Get all roles for this service account
    roles=$(gcloud projects get-iam-policy $PROJECT_ID \
      --flatten="bindings[].members" \
      --filter="bindings.members:serviceAccount:${sa}@${PROJECT_ID}.iam.gserviceaccount.com" \
      --format="value(bindings.role)" 2>/dev/null || true)

    # Remove each role
    for role in $roles; do
      echo "  Removing role: $role"
      gcloud projects remove-iam-policy-binding $PROJECT_ID \
        --member="serviceAccount:${sa}@${PROJECT_ID}.iam.gserviceaccount.com" \
        --role="$role" \
        --project=$PROJECT_ID \
        --quiet &>/dev/null || true
    done

    echo -e "${GREEN}✓ Removed IAM bindings for $sa${NC}"
  fi
done

echo ""
echo -e "${BLUE}Step 4: Deleting Service Accounts...${NC}"
echo ""

# Delete Service Accounts
for sa in radio-backend-sa radio-backend-sa-github; do
  if gcloud iam service-accounts describe ${sa}@${PROJECT_ID}.iam.gserviceaccount.com \
      --project=$PROJECT_ID &>/dev/null; then
    echo -e "${YELLOW}Deleting service account: $sa${NC}"
    gcloud iam service-accounts delete ${sa}@${PROJECT_ID}.iam.gserviceaccount.com \
      --project=$PROJECT_ID \
      --quiet
    echo -e "${GREEN}✓ Deleted $sa${NC}"
  else
    echo -e "${YELLOW}⊘ Service account $sa does not exist${NC}"
  fi
done

echo ""
echo -e "${BLUE}Step 5: Artifact Registry (keeping it)...${NC}"
echo ""
echo ""
echo -e "${BLUE}Step 5: Artifact Registry (keeping it)...${NC}"
echo ""
echo -e "${YELLOW}ℹ Artifact Registry will be kept (contains container images)${NC}"
echo -e "${YELLOW}If you want to delete it manually:${NC}"
echo "  gcloud artifacts repositories delete radio-backend --location=$REGION --project=$PROJECT_ID"
echo ""

echo ""
echo -e "${BLUE}Step 6: Deleting Secrets from Secret Manager...${NC}"
echo ""

# List all secrets
secrets=$(gcloud secrets list --project=$PROJECT_ID --format="value(name)" 2>/dev/null || true)

if [ -z "$secrets" ]; then
  echo -e "${YELLOW}⊘ No secrets found${NC}"
else
  for secret in $secrets; do
    echo -e "${YELLOW}Deleting secret: $secret${NC}"
    gcloud secrets delete $secret \
      --project=$PROJECT_ID \
      --quiet
    echo -e "${GREEN}✓ Deleted $secret${NC}"
  done
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✓ Infrastructure Cleanup Complete!${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo "1. Create GCS bucket for Terraform state:"
echo -e "   ${GREEN}make tf-setup-backend${NC}"
echo ""
echo "2. Initialize Terraform:"
echo -e "   ${GREEN}make tf-init${NC}"
echo ""
echo "3. Review and apply Terraform:"
echo -e "   ${GREEN}make tf-plan${NC}"
echo -e "   ${GREEN}make tf-apply${NC}"
echo "   (This will create secrets with 'CHANGE_ME' values)"
echo ""
echo "4. Update secret values:"
echo -e "   ${GREEN}./scripts/setup-secrets.sh${NC}"
echo ""
echo "5. Configure GitHub Actions secrets:"
echo -e "   ${GREEN}make tf-github-secrets${NC}"
echo ""
echo -e "${GREEN}Everything is clean! Ready to start fresh with Terraform.${NC}"
echo ""
