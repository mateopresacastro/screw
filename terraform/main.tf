terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "eu-central-1"
}

# Variables
variable "domain_name" {
  type        = string
  description = "Your domain name"
}

variable "github_repository" {
  type        = string
  description = "GitHub repository name in format owner/repo"
}

# VPC Data Source
data "aws_vpc" "default" {
  default = true
}

# Create the OIDC Provider for GitHub
resource "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"
  client_id_list = ["sts.amazonaws.com"]
  thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
}

# IAM role for EC2
resource "aws_iam_role" "ssm_role" {
  name = "screw_ssm_role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })
}

# Attach SSM policy to role
resource "aws_iam_role_policy_attachment" "ssm_policy" {
  role       = aws_iam_role.ssm_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

# Create instance profile
resource "aws_iam_instance_profile" "ssm_profile" {
  name = "screw_ssm_profile"
  role = aws_iam_role.ssm_role.name
}

# Security group
resource "aws_security_group" "screw_sg" {
  name        = "screw-sg"
  description = "Security group for Screw application"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTP"
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTPS"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "screw-sg"
  }
}

# EC2 instance
resource "aws_instance" "screw_server" {
  ami           = "ami-0669b163befffbdfc"  # Amazon Linux 2023 AMI for eu-central-1
  instance_type = "t2.micro"

  iam_instance_profile = aws_iam_instance_profile.ssm_profile.name
  vpc_security_group_ids = [aws_security_group.screw_sg.id]

  root_block_device {
    volume_size = 30
    volume_type = "gp3"
    encrypted   = true
  }

  user_data = file("${path.module}/user_data.sh")

  tags = {
    Name = "screw-server"
  }
}

# Create an Elastic IP
resource "aws_eip" "screw_eip" {
  instance = aws_instance.screw_server.id
  domain   = "vpc"
}

# Create IAM Role for GitHub Actions
resource "aws_iam_role" "github_actions" {
  name = "screw_github_actions_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRoleWithWebIdentity"
        Effect = "Allow"
        Principal = {
          Federated = aws_iam_openid_connect_provider.github.arn
        }
        Condition = {
          StringEquals = {
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          }
          StringLike = {
            "token.actions.githubusercontent.com:sub" = "repo:${var.github_repository}:*"
          }
        }
      }
    ]
  })
}

# Policy for GitHub Actions role
resource "aws_iam_role_policy" "github_actions" {
  name = "screw_github_actions_policy"
  role = aws_iam_role.github_actions.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ssm:PutParameter",
          "ssm:GetParameter",
          "ssm:GetParameters"
        ]
        Resource = "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter/screw/*"
      },
      {
        Effect = "Allow"
        Action = ["ssm:SendCommand"]
        Resource = [
          "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:document/AWS-RunShellScript",
          "arn:aws:ec2:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:instance/${aws_instance.screw_server.id}"
        ]
      },
      {
        Effect = "Allow"
        Action = ["ssm:SendCommand"]
        Resource = "arn:aws:ssm:${data.aws_region.current.name}::document/AWS-RunShellScript"
      }
    ]
  })
}

# Data sources
data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

# Outputs
output "public_ip" {
  value = aws_eip.screw_eip.public_ip
}

output "instance_id" {
  value = aws_instance.screw_server.id
}

output "github_actions_role_arn" {
  value = aws_iam_role.github_actions.arn
}

output "setup_instructions" {
  value = <<EOT
To complete the setup:
1. Add these secrets to your GitHub repository:
   - AWS_ROLE_ARN: ${aws_iam_role.github_actions.arn}
   - INSTANCE_ID: ${aws_instance.screw_server.id}
   - DOMAIN_NAME: ${var.domain_name}
   - CERTBOT_EMAIL: [Your email address]
2. SSH into the instance and run:
   docker-compose up -d
3. Your server IP is: ${aws_eip.screw_eip.public_ip}
EOT
}