#!/bin/bash
set -e

echo "🚀 Starting UMA Tickets Backend..."

# Wait for database to be ready
echo "⏳ Waiting for database to be ready..."
until pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER; do
  echo "Database is not ready yet. Waiting..."
  sleep 2
done

echo "✅ Database is ready!"

# Run database migrations
echo "🔄 Running database migrations..."
if [ -n "$DATABASE_URL" ]; then
    echo "Using DATABASE_URL for migrations"
    dbmate up
elif [ -n "$DB_HOST" ] && [ -n "$DB_PORT" ] && [ -n "$DB_USER" ] && [ -n "$DB_NAME" ]; then
    echo "Using individual DB environment variables for migrations"
    export DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"
    dbmate up
else
    echo "⚠️  No database configuration found. Skipping migrations."
fi

echo "✅ Database migrations completed!"

# Start the application
echo "🚀 Starting application..."
exec ./main
