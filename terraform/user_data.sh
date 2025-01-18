#!/bin/bash
# Wait for cloud-init to complete
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do
    echo 'Waiting for cloud-init...'
    sleep 1
done

# Update system packages
sudo dnf update -y
sudo dnf install -y docker make

# Start and enable Docker
systemctl start docker
systemctl enable docker

# Add ec2-user to docker group
usermod -aG docker ec2-user

# Install Docker Compose
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# Create app directory with correct permissions
mkdir -p /home/ec2-user/app
chown ec2-user:ec2-user /home/ec2-user/app
chmod 755 /home/ec2-user/app

# Create docker directory with correct permissions
mkdir -p /home/ec2-user/.docker
chown -R ec2-user:ec2-user /home/ec2-user/.docker
chmod -R 755 /home/ec2-user/.docker

# Create directories for certbot
mkdir -p /home/ec2-user/app/data/certbot/conf
mkdir -p /home/ec2-user/app/data/certbot/www
chown -R ec2-user:ec2-user /home/ec2-user/app/data

# Create SSL configuration file
cat > /home/ec2-user/app/data/certbot/conf/options-ssl-nginx.conf << EOL
ssl_session_cache shared:le_nginx_SSL:10m;
ssl_session_timeout 1440m;
ssl_session_tickets off;

ssl_protocols TLSv1.2 TLSv1.3;
ssl_prefer_server_ciphers off;

ssl_ciphers "ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384";
EOL

# Generate DH parameters
openssl dhparam -out /home/ec2-user/app/data/certbot/conf/ssl-dhparams.pem 2048

# Set permissions for SSL files
chmod -R 755 /home/ec2-user/app/data/certbot/conf

# Generate initial SSL certificate
sudo docker run -it --rm \
-v /home/ec2-user/app/data/certbot/conf:/etc/letsencrypt \
-v /home/ec2-user/app/data/certbot/www:/var/www/certbot \
-p 80:80 \
certbot/certbot certonly --standalone \
-d screw.mateo.id \
--email mateopresacastro@gmail.com \
--agree-tos \
--no-eff-email

# Set final permissions
chown -R ec2-user:ec2-user /home/ec2-user/app/data/certbotata