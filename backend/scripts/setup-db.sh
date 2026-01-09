#!/bin/bash

# =============================================================================
# Aegis v13 Database Setup Script
# =============================================================================
# This script creates the PostgreSQL user and database for Aegis v13
# =============================================================================

set -e  # Exit on error

echo "=== Aegis v13 Database Setup ==="
echo ""

# Database configuration
DB_USER="aegis_v13"
DB_PASSWORD="aegis_v13_won"
DB_NAME="aegis_v13"

echo "📋 Configuration:"
echo "   User: $DB_USER"
echo "   Database: $DB_NAME"
echo ""

# Check if psql is available
if ! command -v psql &> /dev/null; then
    echo "❌ Error: psql command not found"
    echo "   Please install PostgreSQL client"
    exit 1
fi

echo "🔍 Checking PostgreSQL connection..."
if ! psql -d postgres -c '\q' 2>/dev/null; then
    echo "❌ Error: Cannot connect to PostgreSQL"
    echo "   Please ensure PostgreSQL is running and you have access"
    echo ""
    echo "💡 Try: brew services start postgresql@14"
    exit 1
fi

echo "✅ PostgreSQL connection OK"
echo ""

# Create user if not exists
echo "👤 Creating user '$DB_USER'..."
psql -d postgres -tc "SELECT 1 FROM pg_user WHERE usename = '$DB_USER'" | grep -q 1 || \
psql -d postgres <<EOF
CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
ALTER USER $DB_USER CREATEDB;
EOF

echo "✅ User created/verified"
echo ""

# Create database if not exists
echo "🗄️  Creating database '$DB_NAME'..."
psql -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1 || \
psql -d postgres <<EOF
CREATE DATABASE $DB_NAME OWNER $DB_USER;
EOF

echo "✅ Database created/verified"
echo ""

# Grant privileges
echo "🔐 Granting privileges..."
psql -d postgres <<EOF
GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;
EOF

psql -d $DB_NAME <<EOF
GRANT ALL ON SCHEMA public TO $DB_USER;
EOF

echo "✅ Privileges granted"
echo ""

# Test connection
echo "🧪 Testing connection..."
if PGPASSWORD=$DB_PASSWORD psql -U $DB_USER -d $DB_NAME -c '\q' 2>/dev/null; then
    echo "✅ Connection test successful!"
else
    echo "❌ Connection test failed"
    exit 1
fi

echo ""
echo "🎉 Database setup complete!"
echo ""
echo "📝 Connection details:"
echo "   Host: localhost"
echo "   Port: 5432"
echo "   User: $DB_USER"
echo "   Database: $DB_NAME"
echo "   URL: postgresql://$DB_USER:$DB_PASSWORD@localhost:5432/$DB_NAME"
echo ""
echo "Next steps:"
echo "  1. Run migrations: make migrate-up"
echo "  2. Test connection: go run ./cmd/test-db"
