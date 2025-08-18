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

# Provider for us-east-1 (required for CloudFront ACM certificate)
provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
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

variable "domain_name" {
  description = "Domain name for the application"
  type        = string
  default     = "fanmeeting.org"
}

variable "use_custom_domain" {
  description = "Whether to use custom domain with CloudFront"
  type        = bool
  default     = false
}

variable "github_repo_url" {
  description = "GitHub repository URL"
  type        = string
}

variable "github_access_token" {
  description = "GitHub access token for Amplify"
  type        = string
  sensitive   = true
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

variable "main_branch_name" {
  description = "Main branch name (e.g., master, main)"
  type        = string
  default     = "master"
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
          aws_ssm_parameter.lightspark_node_password.arn
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

resource "random_password" "jwt_secret" {
  length  = 32
  special = true
}

# AWS Amplify App - Simplified
resource "aws_amplify_app" "frontend" {
  name         = "${var.app_name}-frontend"
  repository   = var.github_repo_url
  access_token = var.github_access_token

  build_spec = <<-EOT
version: 1
applications:
  - appRoot: frontend
    frontend:
      phases:
        preBuild:
          commands:
            - npm ci --include=dev
            - 'export PATH="$PWD/node_modules/.bin:$PATH"'
        build:
          commands:
            - 'export PATH="$PWD/node_modules/.bin:$PATH"'
            - npm run build
      artifacts:
        baseDirectory: dist
        files:
          - '**/*'
      cache:
        paths:
          - node_modules/**/*
EOT

  # Environment variables for frontend
  environment_variables = {
    NODE_ENV                  = "production"
    AMPLIFY_MONOREPO_APP_ROOT = "frontend"
  }

  tags = {
    Name        = "${var.app_name}-frontend"
    Environment = var.environment
  }
}

# Updated Amplify branch with CloudFront support
resource "aws_amplify_branch" "main" {
  app_id      = aws_amplify_app.frontend.id
  branch_name = var.main_branch_name
  enable_auto_build = true

  # Updated environment variables for the new setup
  environment_variables = {
    VITE_API_BASE_URL = var.use_custom_domain ? "https://api.${var.domain_name}" : "http://${aws_lb.main.dns_name}"
    NODE_ENV          = "production"
  }

  tags = {
    Name        = "${var.app_name}-frontend-${var.main_branch_name}"
    Environment = var.environment
  }
}

# Add custom domain to Amplify app
resource "aws_amplify_domain_association" "main" {
  count       = var.use_custom_domain ? 1 : 0
  app_id      = aws_amplify_app.frontend.id
  domain_name = var.domain_name

  # Add subdomain for API if needed
  sub_domain {
    branch_name = aws_amplify_branch.main.branch_name
    prefix      = ""  # Root domain
  }
}

# ACM certificate for ALB (in your main region)
resource "aws_acm_certificate" "alb" {
  count       = var.use_custom_domain ? 1 : 0
  domain_name = "api.${var.domain_name}"  # api.fanmeeting.org

  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Name        = "${var.app_name}-alb-cert"
    Environment = var.environment
  }
}

# HTTPS listener for ALB
resource "aws_lb_listener" "https" {
  count             = var.use_custom_domain ? 1 : 0
  load_balancer_arn = aws_lb.main.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS-1-2-2017-01"
  certificate_arn   = aws_acm_certificate.alb[0].arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.backend.arn
  }
}

output "amplify_dns_record" {
  description = "DNS record to add at Name.com for Amplify"
  value = var.use_custom_domain ? {
    message = "Add this CNAME record at Name.com:"
    type    = "CNAME"
    name    = var.domain_name
    target  = aws_amplify_domain_association.main[0].certificate_verification_dns_record
  } : null
}

output "backend_dns_record" {
  description = "DNS record for backend API"
  value = var.use_custom_domain ? {
    message = "Add this CNAME record at Name.com:"
    type    = "CNAME"
    name    = "api.${var.domain_name}"
    target  = aws_lb.main.dns_name
  } : null
}

output "backend_url" {
  description = "Backend API URL"
  value       = "https://${aws_lb.main.dns_name}"
}

output "frontend_url" {
  description = "Frontend URL"
  value       = "https://${aws_amplify_branch.main.branch_name}.${aws_amplify_app.frontend.default_domain}"
}

output "database_endpoint" {
  description = "RDS endpoint"
  value       = aws_db_instance.postgres.endpoint
  sensitive   = true
}

output "amplify_app_id" {
  description = "Amplify App ID"
  value       = aws_amplify_app.frontend.id
}

output "custom_domain_url" {
  description = "Custom domain URL"
  value       = var.use_custom_domain ? "https://${var.domain_name}" : null
}
