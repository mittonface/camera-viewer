#!/bin/bash

# Nginx startup script that handles SSL certificate presence
SSL_CERT="/etc/letsencrypt/live/camera.mittn.ca/fullchain.pem"

if [ -f "$SSL_CERT" ]; then
    echo "SSL certificate found, using SSL-enabled configuration"
    cp /etc/nginx/nginx-ssl.conf /etc/nginx/nginx.conf
else
    echo "No SSL certificate found, using HTTP-only configuration"
    cp /etc/nginx/nginx-http.conf /etc/nginx/nginx.conf
fi

# Start nginx
nginx -g "daemon off;"