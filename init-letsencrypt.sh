#!/bin/bash

# Let's Encrypt SSL certificate initialization script
# Run this script once to set up SSL certificates

set -e

DOMAIN="camera.mittn.ca"
EMAIL="admin@mittn.ca"  # Change this to your email
DATA_PATH="./data/certbot"

echo "🔐 Initializing Let's Encrypt certificates for $DOMAIN..."

# Create directory structure
mkdir -p "$DATA_PATH/conf"
mkdir -p "$DATA_PATH/www"

# Check if certificates already exist
if [ -d "$DATA_PATH/conf/live/$DOMAIN" ]; then
    echo "⚠️  Certificates for $DOMAIN already exist. Skipping initialization."
    echo "Run 'docker-compose exec certbot certbot renew' to renew certificates."
    exit 0
fi

# Create dummy certificate to start nginx
echo "📝 Creating dummy certificate for $DOMAIN..."
mkdir -p "$DATA_PATH/conf/live/$DOMAIN"
docker-compose run --rm --entrypoint "\
  openssl req -x509 -nodes -newkey rsa:4096 -days 1\
    -keyout '/etc/letsencrypt/live/$DOMAIN/privkey.pem' \
    -out '/etc/letsencrypt/live/$DOMAIN/fullchain.pem' \
    -subj '/CN=localhost'" certbot

# Start nginx with dummy certificate
echo "🚀 Starting nginx with dummy certificate..."
docker-compose up -d nginx

# Remove dummy certificate
echo "🗑️  Removing dummy certificate..."
docker-compose run --rm --entrypoint "\
  rm -Rf /etc/letsencrypt/live/$DOMAIN && \
  rm -Rf /etc/letsencrypt/archive/$DOMAIN && \
  rm -Rf /etc/letsencrypt/renewal/$DOMAIN.conf" certbot

# Request real certificate
echo "📋 Requesting SSL certificate from Let's Encrypt..."
docker-compose run --rm --entrypoint "\
  certbot certonly --webroot -w /var/www/certbot \
    --email $EMAIL \
    --agree-tos \
    --no-eff-email \
    -d $DOMAIN" certbot

# Switch to SSL-enabled nginx configuration
echo "🔄 Switching to SSL-enabled nginx configuration..."
# Update docker-compose to use the SSL-enabled config
sed -i.backup 's|nginx-initial.conf|nginx.conf|g' docker-compose.yml
# Restart nginx with the new configuration
docker-compose stop nginx
docker-compose up -d nginx

echo "✅ SSL certificate successfully obtained for $DOMAIN!"
echo "🔒 Your site is now accessible at https://$DOMAIN"
echo ""
echo "📝 To renew certificates in the future, run:"
echo "   docker-compose exec certbot certbot renew --dry-run"