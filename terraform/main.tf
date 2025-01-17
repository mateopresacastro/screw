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

# Create the OIDC Provider for GitHub
resource "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"

  client_id_list = ["sts.amazonaws.com"]

  thumbprint_list = [
    "6938fd4d98bab03faadb97b34396831e3780aea1" # GitHub's OIDC thumbprint
  ]
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

# Create an Elastic IP
resource "aws_eip" "screw_eip" {
  instance = aws_instance.screw_server.id
  domain   = "vpc"
}

# Security group
resource "aws_security_group" "screw_sg" {
  name = "screw-sg"

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
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
  ami           = "ami-00a830443b0381486" 
  instance_type = "t2.micro"

  iam_instance_profile = aws_iam_instance_profile.ssm_profile.name
  security_groups      = [aws_security_group.screw_sg.name]

  user_data = file("${path.module}/user_data.sh")

  root_block_device {
    volume_size = 30  # Increase root volume size to 30GB
  }

  tags = {
    Name = "screw-server"
  }
}

# SSL Certificate
resource "aws_acm_certificate" "cert" {
  domain_name       = var.domain_name
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
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
          "ssm:GetParameter"
        ]
        Resource = "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter/screw/*"
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:SendCommand"
        ]
        Resource = [
          "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:document/AWS-RunShellScript",
          "arn:aws:ec2:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:instance/${aws_instance.screw_server.id}"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:SendCommand"
        ]
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

output "certificate_arn" {
  value = aws_acm_certificate.cert.arn
}

output "github_actions_role_arn" {
  value = aws_iam_role.github_actions.arn
  description = "ARN of the IAM role for GitHub Actions"
}

output "github_actions_setup_instructions" {
  value = <<EOT
To complete GitHub Actions setup:
1. Add these secrets to your GitHub repository:
   - AWS_ROLE_ARN: ${aws_iam_role.github_actions.arn}
   - INSTANCE_ID: ${aws_instance.screw_server.id}
   - DOMAIN_NAME: ${var.domain_name}
   - CERTBOT_EMAIL: [Your email address]
2. Configure OIDC provider in your GitHub repository settings
EOT
}