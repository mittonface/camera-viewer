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

# Health check with retries (check both HTTP and HTTPS)
echo "🏥 Performing health check..."
for i in {1..6}; do
    echo "Health check attempt $i/6..."
    # Try HTTP first (via nginx proxy)
    if curl -f -s http://localhost/health >/dev/null 2>&1; then
        echo "✅ HTTP health check successful!"
        # If HTTPS is configured, try HTTPS as well
        if curl -f -s -k https://localhost/health >/dev/null 2>&1; then
            echo "✅ HTTPS health check successful!"
            echo "Health check response:"
            curl -s http://localhost/health | jq . || curl -s http://localhost/health
        else
            echo "⚠️  HTTPS not yet configured, but HTTP is working"
            echo "Health check response:"
            curl -s http://localhost/health | jq . || curl -s http://localhost/health
        fi
        break
    elif [ $i -eq 6 ]; then
        echo "❌ Health check failed after 6 attempts!"
        echo "Service logs:"
        docker-compose logs --tail=30
        echo ""
        echo "Container status:"
        docker-compose ps
        exit 1
    else
        echo "Service not ready yet, waiting 10 seconds..."
        sleep 10
    fi
done

# Show status
echo "📊 Service status:"
docker-compose ps

echo "🎉 Deployment completed successfully!"
echo "🌐 Access your application at:"
echo "   - HTTP:  http://camera.mittn.ca"
echo "   - HTTPS: https://camera.mittn.ca (after SSL setup)"
echo ""
echo "📋 Next steps for SSL:"
echo "   1. Ensure DNS points camera.mittn.ca to this server"
echo "   2. Run: ./init-letsencrypt.sh"
echo "   3. Your site will be available with SSL!"