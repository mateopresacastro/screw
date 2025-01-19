FROM nginx:alpine

# Create directory for certbot challenge
RUN mkdir -p /var/www/certbot

# Add nginx template configuration
COPY nginx.conf /etc/nginx/nginx.template.conf

# Install bash and gettext (for envsubst)
RUN apk add --no-cache bash gettext

# Substitute environment variables and create final nginx.conf
CMD envsubst '${NGINX_SERVER_NAME}' < /etc/nginx/nginx.template.conf > /etc/nginx/nginx.conf && nginx -g 'daemon off;'

# Create directory for Let's Encrypt certificates
RUN mkdir -p /etc/letsencrypt

# Create directory for logs
RUN mkdir -p /var/log/nginx
RUN touch /var/log/nginx/access.log /var/log/nginx/error.log

EXPOSE 80 443
