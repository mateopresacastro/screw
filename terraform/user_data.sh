#!/bin/bash

# Wait for cloud-init to complete
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do
    echo 'Waiting for cloud-init...'
    sleep 1
done

# Update system packages
dnf update -y
dnf install -y docker curl make

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