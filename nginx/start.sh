#!/bin/bash

if [ "$ENV" = "dev" ]; then
    echo "Using development configuration..."
    cp /etc/nginx/nginx.dev.conf /etc/nginx/nginx.conf
else
    echo "Using production configuration..."
    cp /etc/nginx/nginx.prod.conf /etc/nginx/nginx.conf
fi

exec nginx -g 'daemon off;'