#!/bin/bash

# Create required directories
mkdir -p data/certbot/conf
mkdir -p data/certbot/www

# Stop any existing services
make down

# Check if certificates exist
if [ ! -d "/etc/letsencrypt/live/screw.mateo.id" ]; then
  # Start nginx temporarily for certificate acquisition
  docker compose -f compose.yaml up -d proxy

  # Run certbot
  docker compose run --rm certbot certonly --webroot --webroot-path=/var/www/certbot \
    --email $CERTBOT_EMAIL --agree-tos --no-eff-email \
    --force-renewal -d screw.mateo.id
fi

# Start all services in production mode
make prod

# Setup automatic certificate renewal
(crontab -l 2>/dev/null; echo "0 12 * * * /usr/bin/docker compose run --rm certbot renew --quiet && /usr/bin/docker compose exec proxy nginx -s reload") | crontab -