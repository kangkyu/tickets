#!/bin/bash
set -e

echo "🚀 Starting UMA Tickets Backend..."

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "❌ ERROR: DATABASE_URL environment variable is not set!"
    echo "   This is required for the application to start."
    echo "   Please ensure DATABASE_URL is configured in your ECS task definition."
    exit 1
fi

echo "✅ DATABASE_URL is configured"

# Wait for database to be ready
echo "⏳ Waiting for database to be ready..."

# Use DATABASE_URL directly with pg_isready
until pg_isready -d "$DATABASE_URL"; do
    echo "Database is not ready yet. Waiting..."
    sleep 2
done

echo "✅ Database is ready!"

# Run database migrations
echo "🔄 Running database migrations..."
dbmate up

echo "✅ Database migrations completed!"

# Start the application
echo "🚀 Starting application..."
exec ./main
