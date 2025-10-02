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
║   Turso Database Setup Script         ║
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

# Step 1: Check if Turso CLI is installed
print_info "Checking for Turso CLI installation..."

if ! command -v turso &> /dev/null; then
    print_warning "Turso CLI is not installed."
    echo ""
    echo "Please install it using one of the following methods:"
    echo ""
    echo "  macOS/Linux (curl):"
    echo "    curl -sSfL https://get.tur.so/install.sh | bash"
    echo ""
    echo "  macOS (Homebrew):"
    echo "    brew install tursodatabase/tap/turso"
    echo ""
    echo "  Windows (PowerShell):"
    echo "    iwr -useb https://get.tur.so/install.ps1 | iex"
    echo ""
    echo "For more options, visit: https://docs.turso.tech/cli/installation"
    echo ""

    read -p "Would you like to install via the install script now? (y/N): " install_choice
    if [[ "$install_choice" =~ ^[Yy]$ ]]; then
        print_info "Installing Turso CLI..."
        if curl -sSfL https://get.tur.so/install.sh | bash; then
            print_success "Turso CLI installed successfully!"
            # Reload shell to get turso in PATH
            export PATH="$HOME/.turso:$PATH"
        else
            print_error "Failed to install Turso CLI. Please install it manually."
            exit 1
        fi
    else
        print_error "Turso CLI is required. Please install it and run this script again."
        exit 1
    fi
else
    print_success "Turso CLI is already installed ($(turso --version))"
fi

# Step 2: Login to Turso
print_info "Checking Turso authentication..."

# Check if already logged in by checking the output
WHOAMI_OUTPUT=$(turso auth whoami 2>&1)

if echo "$WHOAMI_OUTPUT" | grep -q "not logged in"; then
    print_warning "Not logged in to Turso"
    print_info "Logging into Turso (this will open your browser)..."
    if turso auth login; then
        print_success "Successfully logged in to Turso!"
        CURRENT_USER=$(turso auth whoami 2>/dev/null)
        print_info "Logged in as: $CURRENT_USER"
    else
        print_error "Failed to login to Turso"
        exit 1
    fi
else
    print_success "Already logged in as: $WHOAMI_OUTPUT"
fi

# Step 3: List databases and let user choose or create new
print_info "Fetching your Turso databases..."

# Get databases list
databases_output=$(turso db list 2>&1)

# Check if command failed due to authentication
if echo "$databases_output" | grep -q "not logged in"; then
    print_error "Authentication failed. Please run 'turso auth login' manually and try again."
    exit 1
fi

if [ -z "$databases_output" ] || ! echo "$databases_output" | grep -q "^[[:alnum:]]"; then
    print_warning "No Turso databases found!"
    echo ""
    read -p "Would you like to create a new database? (Y/n): " create_choice
    if [[ ! "$create_choice" =~ ^[Nn]$ ]]; then
        read -p "Enter database name (default: mix-db): " DB_NAME
        DB_NAME=${DB_NAME:-mix-db}

        print_info "Creating database '$DB_NAME'..."
        if turso db create "$DB_NAME"; then
            print_success "Database '$DB_NAME' created successfully!"
        else
            print_error "Failed to create database"
            exit 1
        fi
    else
        print_error "Cannot proceed without a database. Please create one and run this script again."
        exit 1
    fi
else
    # Parse and display databases
    echo ""
    echo "Your Turso Databases:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Display databases with index
    index=1
    declare -a db_names

    while IFS= read -r line; do
        # Skip header line
        if [[ "$line" =~ ^NAME ]]; then
            continue
        fi

        # Extract database name (first column)
        db_name=$(echo "$line" | awk '{print $1}')

        if [ -n "$db_name" ]; then
            printf "%d) %s\n" "$index" "$db_name"
            db_names+=("$db_name")
            ((index++))
        fi
    done <<< "$databases_output"

    # Add option to create new database
    printf "%d) Create new database\n" "$index"

    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    # Get user selection
    while true; do
        read -p "Select a database (1-$index): " selection

        if [[ "$selection" =~ ^[0-9]+$ ]] && [ "$selection" -ge 1 ] && [ "$selection" -le "$index" ]; then
            if [ "$selection" -eq "$index" ]; then
                # Create new database
                read -p "Enter database name (default: mix-db): " DB_NAME
                DB_NAME=${DB_NAME:-mix-db}

                print_info "Creating database '$DB_NAME'..."
                if turso db create "$DB_NAME"; then
                    print_success "Database '$DB_NAME' created successfully!"
                else
                    print_error "Failed to create database"
                    exit 1
                fi
            else
                DB_NAME="${db_names[$((selection-1))]}"
            fi
            break
        else
            print_error "Invalid selection. Please enter a number between 1 and $index"
        fi
    done
fi

print_success "Selected database: $DB_NAME"

# Step 4: Get database URL and create auth token
print_info "Fetching database URL..."

DB_URL=$(turso db show "$DB_NAME" --url 2>&1)
if [ -z "$DB_URL" ] || echo "$DB_URL" | grep -q "not logged in\|error\|Error"; then
    print_error "Failed to get database URL"
    echo "Error: $DB_URL"
    exit 1
fi

print_success "Database URL: $DB_URL"

print_info "Creating authentication token..."

AUTH_TOKEN=$(turso db tokens create "$DB_NAME" 2>&1)
if [ -z "$AUTH_TOKEN" ] || echo "$AUTH_TOKEN" | grep -q "not logged in\|error\|Error"; then
    print_error "Failed to create authentication token"
    echo "Error: $AUTH_TOKEN"
    exit 1
fi

print_success "Authentication token created successfully"

# Step 5: Update .env file
print_info "Updating .env file..."

# Remove old Turso database config if exists
sed -i.bak '/^MIX_DB_TYPE=/d' .env 2>/dev/null || true
sed -i.bak '/^MIX_DB_TURSO_URL=/d' .env 2>/dev/null || true
sed -i.bak '/^MIX_DB_TURSO_AUTH_TOKEN=/d' .env 2>/dev/null || true

# Append new configuration
cat >> .env << EOF

# Turso Database (Auto-configured by setup-turso.sh)
MIX_DB_TYPE=turso
MIX_DB_TURSO_URL=${DB_URL}
MIX_DB_TURSO_AUTH_TOKEN=${AUTH_TOKEN}
EOF

print_success ".env file updated successfully"

# Clean up backup file
rm -f .env.bak

# Step 6: Summary
echo ""
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║              ✅ Setup Complete!                           ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "Configuration Summary:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Database: $DB_NAME"
echo "  URL:      $DB_URL"
echo "  Token:    ${AUTH_TOKEN:0:20}... (truncated)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Next steps:"
echo "  1. Run 'make dev' to start your application"
echo "  2. Database migrations will run automatically on startup"
echo ""
print_success "You're all set! 🚀"
