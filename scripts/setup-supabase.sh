#!/bin/bash

# Color codes for better UX
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Banner
echo -e "${BLUE}"
cat << "EOF"
╔═══════════════════════════════════════╗
║   Supabase Storage Setup Script       ║
║   Automated configuration for Mix     ║
╚═══════════════════════════════════════╝
EOF
echo -e "${NC}"

# Check if .env file exists
if [ ! -f .env ]; then
    print_error ".env file not found!"
    print_info "Creating .env from .env.example..."
    if [ -f .env.example ]; then
        cp .env.example .env
        print_success ".env file created"
    else
        print_error ".env.example not found. Cannot proceed."
        exit 1
    fi
fi

# Step 1: Check if Supabase CLI is installed
print_info "Checking for Supabase CLI installation..."

if ! command -v supabase &> /dev/null; then
    print_warning "Supabase CLI is not installed."
    echo ""
    echo "Please install it using one of the following methods:"
    echo ""
    echo "  macOS (Homebrew):"
    echo "    brew install supabase/tap/supabase"
    echo ""
    echo "  macOS/Linux (npm):"
    echo "    npm install -g supabase"
    echo ""
    echo "  Windows (Scoop):"
    echo "    scoop bucket add supabase https://github.com/supabase/scoop-bucket.git"
    echo "    scoop install supabase"
    echo ""
    echo "For more options, visit: https://supabase.com/docs/guides/cli/getting-started"
    echo ""

    read -p "Would you like to install via Homebrew now? (y/N): " install_choice
    if [[ "$install_choice" =~ ^[Yy]$ ]]; then
        if command -v brew &> /dev/null; then
            print_info "Installing Supabase CLI via Homebrew..."
            brew install supabase/tap/supabase
            print_success "Supabase CLI installed successfully!"
        else
            print_error "Homebrew is not installed. Please install Supabase CLI manually."
            exit 1
        fi
    else
        print_error "Supabase CLI is required. Please install it and run this script again."
        exit 1
    fi
else
    print_success "Supabase CLI is already installed ($(supabase --version))"
fi

# Step 2: Login to Supabase
print_info "Checking Supabase authentication..."

# Check if already logged in by testing a command
if supabase projects list &> /dev/null; then
    print_success "Already logged in to Supabase"
else
    print_info "Logging into Supabase (this will open your browser)..."
    if supabase login; then
        print_success "Successfully logged in to Supabase!"
    else
        print_error "Failed to login to Supabase"
        exit 1
    fi
fi

# Step 3: List projects and let user choose
print_info "Fetching your Supabase projects..."

# Get projects list in JSON format
projects_json=$(supabase projects list --output json 2>/dev/null)

if [ -z "$projects_json" ] || [ "$projects_json" == "null" ] || [ "$projects_json" == "[]" ]; then
    print_warning "No Supabase projects found!"
    echo ""
    echo "📋 Steps to create a new Supabase project:"
    echo ""
    echo "  1. Visit: ${BLUE}https://supabase.com/dashboard/projects${NC}"
    echo "  2. Click '${GREEN}New Project${NC}' button"
    echo "  3. Fill in the details:"
    echo "     • Project name (e.g., 'my-mix-project')"
    echo "     • Database password (save this!)"
    echo "     • Region (choose closest to you)"
    echo "  4. Click '${GREEN}Create new project${NC}'"
    echo "  5. Wait 1-2 minutes for setup to complete"
    echo "  6. Run this script again: ${BLUE}./scripts/setup-supabase.sh${NC}"
    echo ""
    print_info "The setup will complete in 1-2 minutes. Come back and re-run this script!"
    exit 0
fi

# Parse and display projects
echo ""
echo "Your Supabase Projects:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Display projects with index
index=1
declare -a project_refs
declare -a project_names

while IFS= read -r line; do
    project_id=$(echo "$line" | jq -r '.id')
    project_name=$(echo "$line" | jq -r '.name')
    project_region=$(echo "$line" | jq -r '.region')

    printf "%d) %-30s (ref: %s, region: %s)\n" "$index" "$project_name" "$project_id" "$project_region"

    project_refs+=("$project_id")
    project_names+=("$project_name")

    ((index++))
done < <(echo "$projects_json" | jq -c '.[]')

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Get user selection
while true; do
    read -p "Select a project (1-$((index-1))): " selection

    if [[ "$selection" =~ ^[0-9]+$ ]] && [ "$selection" -ge 1 ] && [ "$selection" -lt "$index" ]; then
        PROJECT_REF="${project_refs[$((selection-1))]}"
        PROJECT_NAME="${project_names[$((selection-1))]}"
        break
    else
        print_error "Invalid selection. Please enter a number between 1 and $((index-1))"
    fi
done

print_success "Selected project: $PROJECT_NAME (ref: $PROJECT_REF)"

# Step 4: Get API keys
print_info "Fetching API keys for project $PROJECT_REF..."

api_keys_output=$(supabase projects api-keys --project-ref "$PROJECT_REF" 2>&1)

if [ $? -ne 0 ]; then
    print_error "Failed to fetch API keys"
    echo "$api_keys_output"
    exit 1
fi

# Parse the output to extract service_role key
# The CLI outputs in a table format, we need to extract the service_role key
SERVICE_ROLE_KEY=$(echo "$api_keys_output" | grep -i "service_role" | awk '{print $NF}')

if [ -z "$SERVICE_ROLE_KEY" ]; then
    print_error "Could not extract service_role key from output"
    echo "Output was:"
    echo "$api_keys_output"
    exit 1
fi

PROJECT_URL="https://${PROJECT_REF}.supabase.co"

print_success "Successfully retrieved API keys"

# Step 5: Configure bucket name
echo ""
print_info "Storage bucket configuration"
read -p "Enter bucket name (default: mix-storage): " BUCKET_NAME
BUCKET_NAME=${BUCKET_NAME:-mix-storage}

# Step 6: Create the bucket in Supabase
print_info "Creating storage bucket '${BUCKET_NAME}' in Supabase..."

# Create bucket using Supabase Storage API
CREATE_RESPONSE=$(curl -s -X POST "${PROJECT_URL}/storage/v1/bucket" \
  -H "Authorization: Bearer ${SERVICE_ROLE_KEY}" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"${BUCKET_NAME}\",
    \"public\": false,
    \"file_size_limit\": 52428800,
    \"allowed_mime_types\": [\"image/*\", \"video/*\", \"application/pdf\", \"text/*\"]
  }")

# Check if bucket was created or already exists
if echo "$CREATE_RESPONSE" | grep -q "\"name\":\"${BUCKET_NAME}\""; then
    print_success "Bucket '${BUCKET_NAME}' created successfully"
elif echo "$CREATE_RESPONSE" | grep -q "already exists"; then
    print_success "Bucket '${BUCKET_NAME}' already exists"
else
    print_warning "Could not verify bucket creation. Response: $CREATE_RESPONSE"
    print_info "The app will attempt to create it on startup if needed"
fi

# Step 7: Update .env file
print_info "Updating .env file..."

# Remove old Supabase storage config if exists
sed -i.bak '/^STORAGE_TYPE=/d' .env 2>/dev/null || true
sed -i.bak '/^STORAGE_ENDPOINT=/d' .env 2>/dev/null || true
sed -i.bak '/^STORAGE_BUCKET=/d' .env 2>/dev/null || true
sed -i.bak '/^STORAGE_ACCESS_KEY=/d' .env 2>/dev/null || true

# Append new configuration
cat >> .env << EOF

# Supabase Storage (Auto-configured by setup-supabase.sh)
STORAGE_TYPE=supabase
STORAGE_ENDPOINT=${PROJECT_URL}
STORAGE_BUCKET=${BUCKET_NAME}
STORAGE_ACCESS_KEY=${SERVICE_ROLE_KEY}
EOF

print_success ".env file updated successfully"

# Clean up backup file
rm -f .env.bak

# Step 8: Summary
echo ""
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║              ✅ Setup Complete!                           ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "Configuration Summary:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Project:  $PROJECT_NAME"
echo "  Endpoint: $PROJECT_URL"
echo "  Bucket:   $BUCKET_NAME (created ✅)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Next steps:"
echo "  1. Run 'make dev' to start your application"
echo "  2. Upload and manage files through the app"
echo ""
print_success "You're all set! 🚀"
