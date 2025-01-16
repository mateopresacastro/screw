provider "aws" {
  region = "eu-central-1"
}

# Use your existing SSH key
resource "aws_key_pair" "screw_key" {
  key_name   = "screw-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

# Basic security group
resource "aws_security_group" "screw_sg" {
  name = "screw-sg"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# EC2 instance
resource "aws_instance" "screw_server" {
  ami             = "ami-00a830443b0381486"  # Amazon Linux 2 AMI
  instance_type   = "t2.micro"
  security_groups = [aws_security_group.screw_sg.name]
  key_name        = aws_key_pair.screw_key.key_name

  tags = {
    Name = "screw-server"
  }
}

output "ssh_command" {
  value = "ssh -i ~/.ssh/id_rsa ubuntu@${aws_instance.screw_server.public_ip}"
}