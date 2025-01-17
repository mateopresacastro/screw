FROM nginx:alpine

# Create directory for certbot challenge
RUN mkdir -p /var/www/certbot

# Copy nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Create directory for Let's Encrypt certificates
RUN mkdir -p /etc/letsencrypt

# Create directory for logs
RUN mkdir -p /var/log/nginx
RUN touch /var/log/nginx/access.log /var/log/nginx/error.log

EXPOSE 80 443