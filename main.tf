terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Data sources
data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

# Variables
variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "app_name" {
  description = "Application name"
  type        = string
  default     = "uma-tickets"
}

variable "lightspark_client_id" {
  description = "Lightspark API Client ID"
  type        = string
  sensitive   = true
}

variable "lightspark_client_secret" {
  description = "Lightspark API Client Secret"
  type        = string
  sensitive   = true
}

variable "lightspark_node_id" {
  description = "Lightspark Node ID"
  type        = string
  sensitive   = true
}

variable "lightspark_node_password" {
  description = "Lightspark Node Password"
  type        = string
  sensitive   = true
}

variable "webhook_signing_key" {
  description = "Lightspark webhook signing key"
  type        = string
  sensitive   = true
}

variable "uma_signing_privkey" {
  description = "UMA signing private key (hex)"
  type        = string
  sensitive   = true
}

variable "uma_signing_cert_chain" {
  description = "UMA signing certificate chain (PEM)"
  type        = string
  sensitive   = true
}

variable "uma_encryption_privkey" {
  description = "UMA encryption private key (hex)"
  type        = string
  sensitive   = true
}

variable "uma_encryption_cert_chain" {
  description = "UMA encryption certificate chain (PEM)"
  type        = string
  sensitive   = true
}

variable "uma_auth_app_identity_pubkey" {
  description = "UMA Auth app identity public key (Nostr hex pubkey)"
  type        = string
}

variable "database_name" {
  description = "Database name"
  type        = string
  default     = "umatickets"
}

variable "database_username" {
  description = "Database username"
  type        = string
  default     = "postgres"
}

variable "database_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t3.micro"
}

variable "database_allocated_storage" {
  description = "RDS allocated storage in GB"
  type        = number
  default     = 20
}

variable "local_database_url" {
  description = "Local database URL for development"
  type        = string
  default     = "postgres://postgres:password@localhost:5432/umatickets?sslmode=disable"
}

# VPC Configuration
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "${var.app_name}-vpc"
    Environment = var.environment
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name        = "${var.app_name}-igw"
    Environment = var.environment
  }
}

# Public Subnets
resource "aws_subnet" "public" {
  count = 2

  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.${count.index + 1}.0/24"
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name        = "${var.app_name}-public-${count.index + 1}"
    Environment = var.environment
  }
}

# Private Subnets
resource "aws_subnet" "private" {
  count = 2

  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.${count.index + 10}.0/24"
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = {
    Name        = "${var.app_name}-private-${count.index + 1}"
    Environment = var.environment
  }
}

data "aws_availability_zones" "available" {
  state = "available"
}

# Route Tables
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name        = "${var.app_name}-public-rt"
    Environment = var.environment
  }
}

resource "aws_route_table_association" "public" {
  count = length(aws_subnet.public)

  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# NAT Gateway for private subnets
resource "aws_eip" "nat" {
  domain = "vpc"

  tags = {
    Name        = "${var.app_name}-nat-eip"
    Environment = var.environment
  }
}

resource "aws_nat_gateway" "main" {
  allocation_id = aws_eip.nat.id
  subnet_id     = aws_subnet.public[0].id

  tags = {
    Name        = "${var.app_name}-nat"
    Environment = var.environment
  }

  depends_on = [aws_internet_gateway.main]
}

resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main.id
  }

  tags = {
    Name        = "${var.app_name}-private-rt"
    Environment = var.environment
  }
}

resource "aws_route_table_association" "private" {
  count = length(aws_subnet.private)

  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private.id
}

# Security Groups
resource "aws_security_group" "alb" {
  name_prefix = "${var.app_name}-alb-"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
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
    Name        = "${var.app_name}-alb-sg"
    Environment = var.environment
  }
}

resource "aws_security_group" "ecs" {
  name_prefix = "${var.app_name}-ecs-"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "${var.app_name}-ecs-sg"
    Environment = var.environment
  }
}

resource "aws_security_group" "rds" {
  name_prefix = "${var.app_name}-rds-"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.ecs.id]
  }

  tags = {
    Name        = "${var.app_name}-rds-sg"
    Environment = var.environment
  }
}

# RDS PostgreSQL Database
resource "aws_db_subnet_group" "main" {
  name       = "${var.app_name}-db-subnet-group"
  subnet_ids = aws_subnet.private[*].id

  tags = {
    Name        = "${var.app_name}-db-subnet-group"
    Environment = var.environment
  }
}

resource "aws_db_instance" "postgres" {
  identifier            = "${var.app_name}-postgres"
  engine                = "postgres"
  engine_version        = "15.12"
  instance_class        = var.database_instance_class
  allocated_storage     = var.database_allocated_storage
  max_allocated_storage = 100
  storage_encrypted     = true

  db_name  = var.database_name
  username = var.database_username
  password = random_password.db_password.result

  vpc_security_group_ids = [aws_security_group.rds.id]
  db_subnet_group_name   = aws_db_subnet_group.main.name

  backup_retention_period = 7
  backup_window           = "03:00-04:00"
  maintenance_window      = "sun:04:00-sun:05:00"

  skip_final_snapshot = true
  deletion_protection = false

  tags = {
    Name        = "${var.app_name}-postgres"
    Environment = var.environment
  }
}

resource "random_password" "db_password" {
  length  = 20
  special = false
}

# ECS Cluster
resource "aws_ecs_cluster" "main" {
  name = "${var.app_name}-cluster"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }

  tags = {
    Name        = "${var.app_name}-cluster"
    Environment = var.environment
  }
}

# ECS Task Definition
resource "aws_ecs_task_definition" "backend" {
  family                   = "${var.app_name}-backend"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = aws_iam_role.ecs_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name  = "backend"
      image = "${aws_ecr_repository.backend.repository_url}:latest"

      portMappings = [
        {
          containerPort = 8080
          protocol      = "tcp"
        }
      ]

      environment = [
        {
          name  = "PORT"
          value = "8080"
        },
        {
          name  = "DATABASE_URL"
          value = "postgres://${var.database_username}:${random_password.db_password.result}@${aws_db_instance.postgres.endpoint}/${var.database_name}?sslmode=require"
        },
        {
          name  = "JWT_SECRET"
          value = random_password.jwt_secret.result
        },
        {
          name  = "LIGHTSPARK_WEBHOOK_SIGNING_KEY"
          value = var.webhook_signing_key
        },
        {
          name  = "DOMAIN"
          value = aws_cloudfront_distribution.main.domain_name
        }
      ]

      secrets = [
        {
          name      = "LIGHTSPARK_CLIENT_ID"
          valueFrom = aws_ssm_parameter.lightspark_client_id.arn
        },
        {
          name      = "LIGHTSPARK_CLIENT_SECRET"
          valueFrom = aws_ssm_parameter.lightspark_client_secret.arn
        },
        {
          name      = "LIGHTSPARK_NODE_ID"
          valueFrom = aws_ssm_parameter.lightspark_node_id.arn
        },
        {
          name      = "LIGHTSPARK_NODE_PASSWORD"
          valueFrom = aws_ssm_parameter.lightspark_node_password.arn
        },
        {
          name      = "UMA_SIGNING_PRIVKEY"
          valueFrom = aws_ssm_parameter.uma_signing_privkey.arn
        },
        {
          name      = "UMA_SIGNING_CERT_CHAIN"
          valueFrom = aws_ssm_parameter.uma_signing_cert_chain.arn
        },
        {
          name      = "UMA_ENCRYPTION_PRIVKEY"
          valueFrom = aws_ssm_parameter.uma_encryption_privkey.arn
        },
        {
          name      = "UMA_ENCRYPTION_CERT_CHAIN"
          valueFrom = aws_ssm_parameter.uma_encryption_cert_chain.arn
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.backend.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }

      healthCheck = {
        command     = ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"]
        interval    = 30
        timeout     = 5
        retries     = 3
        startPeriod = 60
      }
    }
  ])

  tags = {
    Name        = "${var.app_name}-backend-task"
    Environment = var.environment
  }
}

# ECS Service
resource "aws_ecs_service" "backend" {
  name            = "${var.app_name}-backend"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.backend.arn
  desired_count   = 2
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = aws_subnet.private[*].id
    security_groups = [aws_security_group.ecs.id]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.backend.arn
    container_name   = "backend"
    container_port   = 8080
  }

  depends_on = [aws_lb_listener.backend]

  tags = {
    Name        = "${var.app_name}-backend-service"
    Environment = var.environment
  }
}

# Application Load Balancer
resource "aws_lb" "main" {
  name               = "${var.app_name}-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = aws_subnet.public[*].id

  enable_deletion_protection = false

  tags = {
    Name        = "${var.app_name}-alb"
    Environment = var.environment
  }
}

resource "aws_lb_target_group" "backend" {
  name        = "${var.app_name}-backend-tg"
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = aws_vpc.main.id
  target_type = "ip"

  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 2
    timeout             = 5
    interval            = 30
    path                = "/health"
    matcher             = "200"
    port                = "traffic-port"
    protocol            = "HTTP"
  }

  tags = {
    Name        = "${var.app_name}-backend-tg"
    Environment = var.environment
  }
}

resource "aws_lb_listener" "backend" {
  load_balancer_arn = aws_lb.main.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.backend.arn
  }
}

# ECR Repository
resource "aws_ecr_repository" "backend" {
  name = "${var.app_name}/backend"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = {
    Name        = "${var.app_name}-backend-ecr"
    Environment = var.environment
  }
}

# IAM Roles
resource "aws_iam_role" "ecs_execution" {
  name = "${var.app_name}-ecs-execution-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "${var.app_name}-ecs-execution-role"
    Environment = var.environment
  }
}

resource "aws_iam_role_policy_attachment" "ecs_execution" {
  role       = aws_iam_role.ecs_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role_policy" "ecs_execution_ssm" {
  name = "${var.app_name}-ecs-execution-ssm"
  role = aws_iam_role.ecs_execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:GetParameters",
          "ssm:GetParametersByPath"
        ]
        Resource = [
          aws_ssm_parameter.lightspark_client_id.arn,
          aws_ssm_parameter.lightspark_client_secret.arn,
          aws_ssm_parameter.lightspark_node_id.arn,
          aws_ssm_parameter.lightspark_node_password.arn,
          aws_ssm_parameter.uma_signing_privkey.arn,
          aws_ssm_parameter.uma_signing_cert_chain.arn,
          aws_ssm_parameter.uma_encryption_privkey.arn,
          aws_ssm_parameter.uma_encryption_cert_chain.arn
        ]
      }
    ]
  })
}

resource "aws_iam_role" "ecs_task" {
  name = "${var.app_name}-ecs-task-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "${var.app_name}-ecs-task-role"
    Environment = var.environment
  }
}

# CloudWatch Log Group
resource "aws_cloudwatch_log_group" "backend" {
  name              = "/ecs/${var.app_name}-backend"
  retention_in_days = 7

  tags = {
    Name        = "${var.app_name}-backend-logs"
    Environment = var.environment
  }
}

# SSM Parameters for secrets
resource "aws_ssm_parameter" "lightspark_client_id" {
  name  = "/${var.app_name}/lightspark/client_id"
  type  = "SecureString"
  value = var.lightspark_client_id

  tags = {
    Name        = "${var.app_name}-lightspark-client-id"
    Environment = var.environment
  }
}

resource "aws_ssm_parameter" "lightspark_client_secret" {
  name  = "/${var.app_name}/lightspark/client_secret"
  type  = "SecureString"
  value = var.lightspark_client_secret

  tags = {
    Name        = "${var.app_name}-lightspark-client-secret"
    Environment = var.environment
  }
}

resource "aws_ssm_parameter" "lightspark_node_id" {
  name  = "/${var.app_name}/lightspark/node_id"
  type  = "SecureString"
  value = var.lightspark_node_id

  tags = {
    Name        = "${var.app_name}-lightspark-node-id"
    Environment = var.environment
  }
}

resource "aws_ssm_parameter" "lightspark_node_password" {
  name  = "/${var.app_name}/lightspark/node_password"
  type  = "SecureString"
  value = var.lightspark_node_password

  tags = {
    Name        = "${var.app_name}-lightspark-node-password"
    Environment = var.environment
  }
}

resource "aws_ssm_parameter" "uma_signing_privkey" {
  name  = "/${var.app_name}/uma/signing_privkey"
  type  = "SecureString"
  value = var.uma_signing_privkey

  tags = {
    Name        = "${var.app_name}-uma-signing-privkey"
    Environment = var.environment
  }
}

resource "aws_ssm_parameter" "uma_signing_cert_chain" {
  name  = "/${var.app_name}/uma/signing_cert_chain"
  type  = "SecureString"
  value = var.uma_signing_cert_chain

  tags = {
    Name        = "${var.app_name}-uma-signing-cert-chain"
    Environment = var.environment
  }
}

resource "aws_ssm_parameter" "uma_encryption_privkey" {
  name  = "/${var.app_name}/uma/encryption_privkey"
  type  = "SecureString"
  value = var.uma_encryption_privkey

  tags = {
    Name        = "${var.app_name}-uma-encryption-privkey"
    Environment = var.environment
  }
}

resource "aws_ssm_parameter" "uma_encryption_cert_chain" {
  name  = "/${var.app_name}/uma/encryption_cert_chain"
  type  = "SecureString"
  value = var.uma_encryption_cert_chain

  tags = {
    Name        = "${var.app_name}-uma-encryption-cert-chain"
    Environment = var.environment
  }
}

resource "random_password" "jwt_secret" {
  length  = 32
  special = true
}

# S3 bucket for frontend static files
resource "aws_s3_bucket" "frontend" {
  bucket = "${var.app_name}-frontend"

  tags = {
    Name        = "${var.app_name}-frontend"
    Environment = var.environment
  }
}

resource "aws_s3_bucket_public_access_block" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# CloudFront OAC for S3 access
resource "aws_cloudfront_origin_access_control" "frontend" {
  name                              = "${var.app_name}-frontend-oac"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# S3 bucket policy allowing CloudFront access
resource "aws_s3_bucket_policy" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowCloudFrontServicePrincipal"
        Effect = "Allow"
        Principal = {
          Service = "cloudfront.amazonaws.com"
        }
        Action   = "s3:GetObject"
        Resource = "${aws_s3_bucket.frontend.arn}/*"
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = aws_cloudfront_distribution.main.arn
          }
        }
      }
    ]
  })
}

# CloudFront Function for SPA routing (rewrites non-file paths to /index.html)
resource "aws_cloudfront_function" "spa_routing" {
  name    = "${var.app_name}-spa-routing"
  runtime = "cloudfront-js-2.0"
  code    = <<-EOT
function handler(event) {
  var request = event.request;
  var uri = request.uri;
  if (!uri.includes('.')) {
    request.uri = '/index.html';
  }
  return request;
}
EOT
}

# Single CloudFront distribution for frontend (S3) and backend (ALB)
resource "aws_cloudfront_distribution" "main" {
  enabled             = true
  default_root_object = "index.html"
  comment             = var.app_name

  # S3 origin for frontend
  origin {
    domain_name              = aws_s3_bucket.frontend.bucket_regional_domain_name
    origin_id                = "s3"
    origin_access_control_id = aws_cloudfront_origin_access_control.frontend.id
  }

  # ALB origin for backend API
  origin {
    domain_name = aws_lb.main.dns_name
    origin_id   = "alb"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "http-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  # Default: serve frontend from S3
  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "s3"
    viewer_protocol_policy = "redirect-to-https"

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }

    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.spa_routing.arn
    }

    min_ttl     = 0
    default_ttl = 86400
    max_ttl     = 31536000
  }

  # /api/* → ALB
  ordered_cache_behavior {
    path_pattern           = "/api/*"
    allowed_methods        = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "alb"
    viewer_protocol_policy = "redirect-to-https"

    forwarded_values {
      query_string = true
      headers      = ["*"]
      cookies {
        forward = "all"
      }
    }

    min_ttl     = 0
    default_ttl = 0
    max_ttl     = 0
  }

  # /uma/* → ALB
  ordered_cache_behavior {
    path_pattern           = "/uma/*"
    allowed_methods        = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "alb"
    viewer_protocol_policy = "redirect-to-https"

    forwarded_values {
      query_string = true
      headers      = ["*"]
      cookies {
        forward = "all"
      }
    }

    min_ttl     = 0
    default_ttl = 0
    max_ttl     = 0
  }

  # /.well-known/* → ALB
  ordered_cache_behavior {
    path_pattern           = "/.well-known/*"
    allowed_methods        = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "alb"
    viewer_protocol_policy = "redirect-to-https"

    forwarded_values {
      query_string = true
      headers      = ["*"]
      cookies {
        forward = "all"
      }
    }

    min_ttl     = 0
    default_ttl = 0
    max_ttl     = 0
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  tags = {
    Name        = "${var.app_name}-cf"
    Environment = var.environment
  }
}

output "app_url" {
  description = "Application URL (CloudFront)"
  value       = "https://${aws_cloudfront_distribution.main.domain_name}"
}

output "frontend_bucket" {
  description = "S3 bucket for frontend deployment"
  value       = aws_s3_bucket.frontend.bucket
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID (for cache invalidation)"
  value       = aws_cloudfront_distribution.main.id
}

output "database_endpoint" {
  description = "RDS endpoint"
  value       = aws_db_instance.postgres.endpoint
  sensitive   = true
}
