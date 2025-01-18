#!/bin/bash

# Check if instance ID and email were provided
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <instance-id> <email>"
    echo "Example: $0 i-1234567890abcdef0 your@email.com"
    exit 1
fi

INSTANCE_ID=$1
EMAIL=$2
DOMAIN="screw.mateo.id"

echo "Starting initial setup for instance $INSTANCE_ID..."

# Function to wait for command completion
wait_for_command() {
    local COMMAND_ID=$1
    while true; do
        STATUS=$(aws ssm get-command-invocation --command-id "$COMMAND_ID" --instance-id "$INSTANCE_ID" --query "Status" --output text)

        # Print command output for debugging
        echo "Command output:"
        aws ssm get-command-invocation \
            --command-id "$COMMAND_ID" \
            --instance-id "$INSTANCE_ID" \
            --query "StandardOutputContent" --output text

        echo "Error output:"
        aws ssm get-command-invocation \
            --command-id "$COMMAND_ID" \
            --instance-id "$INSTANCE_ID" \
            --query "StandardErrorContent" --output text

        if [ "$STATUS" = "Success" ]; then
            echo "Command completed successfully"
            break
        elif [ "$STATUS" = "Failed" ] || [ "$STATUS" = "TimedOut" ]; then
            echo "Command failed with status: $STATUS"
            exit 1
        fi
        echo "Waiting for command completion... (Status: $STATUS)"
        sleep 5
    done
}

# First, verify required files exist
echo "Checking required files..."
COMMAND_ID=$(aws ssm send-command \
    --instance-ids "$INSTANCE_ID" \
    --document-name "AWS-RunShellScript" \
    --parameters commands="\
        if [ ! -f /home/ec2-user/app/compose.yaml ]; then echo 'compose.yaml missing'; exit 1; fi && \
        if [ ! -f /home/ec2-user/app/Makefile ]; then echo 'Makefile missing'; exit 1; fi && \
        echo 'All required files present'" \
    --query "Command.CommandId" --output text)
wait_for_command "$COMMAND_ID"

# Create initial directories
echo "Creating directories..."
COMMAND_ID=$(aws ssm send-command \
    --instance-ids "$INSTANCE_ID" \
    --document-name "AWS-RunShellScript" \
    --parameters commands="\
        sudo -u ec2-user mkdir -p /home/ec2-user/app/data/certbot/conf && \
        sudo -u ec2-user mkdir -p /home/ec2-user/app/data/certbot/www" \
    --query "Command.CommandId" --output text)
wait_for_command "$COMMAND_ID"

# Wait for Docker to be ready
echo "Waiting for Docker to be ready..."
sleep 30

# Run certbot
echo "Obtaining SSL certificate..."
COMMAND_ID=$(aws ssm send-command \
    --instance-ids "$INSTANCE_ID" \
    --document-name "AWS-RunShellScript" \
    --parameters commands="\
        cd /home/ec2-user/app && \
        sudo -u ec2-user docker compose run --rm certbot certonly --webroot \
        --webroot-path=/var/www/certbot -d $DOMAIN \
        --email $EMAIL --agree-tos --no-eff-email" \
    --query "Command.CommandId" --output text)
wait_for_command "$COMMAND_ID"

# Start services using make
echo "Starting services..."
COMMAND_ID=$(aws ssm send-command \
    --instance-ids "$INSTANCE_ID" \
    --document-name "AWS-RunShellScript" \
    --parameters commands="\
        cd /home/ec2-user/app && \
        sudo -u ec2-user docker compose pull && \
        sudo -u ec2-user make prod" \
    --query "Command.CommandId" --output text)
wait_for_command "$COMMAND_ID"

echo "Setup completed successfully!"
echo "Next steps:"
echo "1. Add an A record in Namecheap for $DOMAIN pointing to the Elastic IP"
echo "2. Push your code to GitHub to trigger the CI/CD pipeline"