#!/bin/bash

# Deployment script for camera-viewer
# This script should be run on the server

set -e  # Exit on any error

echo "🚀 Starting camera-viewer deployment..."

# Configuration
APP_DIR="/home/$(whoami)/camera-viewer"
IMAGE_NAME="camera-viewer:latest"

# Create app directory if it doesn't exist
mkdir -p "$APP_DIR"
cd "$APP_DIR"

# Check if .env exists
if [ ! -f .env ]; then
    echo "❌ .env file not found!"
    echo "Please create .env file with your configuration:"
    echo "  AWS_REGION=us-east-1"
    echo "  AWS_ACCESS_KEY_ID=your-key"
    echo "  AWS_SECRET_ACCESS_KEY=your-secret"
    echo "  BUCKET_NAME=your-bucket"
    echo "  USERNAME=admin"
    echo "  PASSWORD=your-password"
    exit 1
fi

# Stop existing containers
echo "🛑 Stopping existing containers..."
docker-compose down || true

# Remove old images (keep last 2)
echo "🧹 Cleaning up old images..."
docker images camera-viewer --format "table {{.ID}}\t{{.CreatedAt}}" | tail -n +4 | awk '{print $1}' | xargs -r docker rmi || true

# Pull/load latest image (this will be done by GitHub Actions)
# docker-compose pull  # Uncomment if using registry

# Start services
echo "▶️  Starting services..."
docker-compose up -d

# Wait for service to be ready
echo "⏳ Waiting for service to start..."
sleep 15

# Health check
echo "🏥 Performing health check..."
if curl -f http://localhost:8080/list-years >/dev/null 2>&1; then
    echo "✅ Deployment successful! Service is healthy."
else
    echo "❌ Health check failed!"
    echo "Service logs:"
    docker-compose logs --tail=20
    exit 1
fi

# Show status
echo "📊 Service status:"
docker-compose ps

echo "🎉 Deployment completed successfully!"
echo "🌐 Access your application at: http://$(hostname -I | awk '{print $1}'):8080"