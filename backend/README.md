# Tickets by UMA - E-Ticket Backend Service

A Go backend service for selling e-tickets to virtual events using UMA (Universal Money Address) payments with Lightning Network integration.

## Features

- **Event Management**: Create, read, update, and delete virtual events
- **Ticket Sales**: Manage ticket inventory with capacity limits
- **UMA Payments**: Lightning Network payments using UMA protocol
- **Real-time Tracking**: Monitor payment status and ticket delivery
- **User Authentication**: JWT-based authentication system
- **Admin Panel**: Administrative functions for event and payment management

## Architecture

- **Language**: Go 1.21+
- **Router**: gorilla/mux with standard library net/http
- **Database**: PostgreSQL with jmoiron/sqlx
- **Payment**: UMA protocol with Lightspark Go SDK and UMA Go SDK
- **Authentication**: JWT tokens
- **Logging**: Structured JSON logging with log/slog

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 12 or higher
- Docker and Docker Compose (optional)

## Quick Start

### Local Development

1. Clone the repository:
```bash
git clone <repository-url>
cd tickets-by-uma/backend
```

2. Set up PostgreSQL database:
```sql
CREATE DATABASE tickets_uma;
```

3. Set environment variables:
```bash
export DATABASE_URL="postgres://username:password@localhost:5432/tickets_uma?sslmode=disable"
export LIGHTSPARK_API_TOKEN="your_token"
export LIGHTSPARK_NODE_ID="your_node_id"
export JWT_SECRET="your_secret_key"
export ADMIN_EMAILS="admin@example.com"
```

4. Run the service:
```bash
go run main.go
```

The service will be available at `http://localhost:8080`

### AWS Staging Deployment

Since you have existing Terraform infrastructure in the parent directory:

1. Build and push Docker image to ECR:
```bash
# Set your AWS credentials and ECR details
export AWS_ACCOUNT_ID="your-account-id"
export AWS_REGION="us-east-1"
export ECR_REPOSITORY="tickets-uma"
export IMAGE_TAG="staging"

# Build and push
docker build -t ${ECR_REPOSITORY}:${IMAGE_TAG} .
docker tag ${ECR_REPOSITORY}:${IMAGE_TAG} ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}:${IMAGE_TAG}
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}:${IMAGE_TAG}
```

2. Update your Terraform variables with the new image URI and deploy from the parent directory.



## Configuration

The service uses environment variables for configuration:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:password@localhost:5432/tickets_uma?sslmode=disable` |
| `LIGHTSPARK_API_TOKEN` | Lightspark API token | Required |
| `LIGHTSPARK_NODE_ID` | Lightspark node ID | Required |
| `JWT_SECRET` | JWT signing secret | `your-secret-key-change-in-production` |
| `ADMIN_EMAILS` | Comma-separated admin email addresses | `admin@example.com` |

### Environment Setup

#### Local Development
Set environment variables directly in your shell for local testing.

#### AWS Staging
Set environment variables in your ECS task definition or use AWS Systems Manager Parameter Store:

```bash
# Required for staging
LIGHTSPARK_API_TOKEN=your_actual_token
LIGHTSPARK_NODE_ID=your_actual_node_id
JWT_SECRET=your_secure_random_string
DB_PASSWORD=your_rds_password

# Optional (have sensible defaults)
PORT=8080
LOG_LEVEL=info
ADMIN_EMAILS=admin@yourcompany.com
```

## API Endpoints

### Public Endpoints

#### Events
- `GET /api/events` - List all active events
- `GET /api/events/{id}` - Get event details

#### Users
- `POST /api/users` - Create new user
- `POST /api/users/login` - User login

#### Tickets
- `POST /api/tickets/purchase` - Purchase ticket with UMA
- `GET /api/tickets/{id}/status` - Check ticket status
- `POST /api/tickets/validate` - Validate ticket for event access

#### Webhooks
- `POST /api/webhooks/payment` - Lightning payment webhook

### Protected Endpoints (Require JWT)

#### Users
- `GET /api/users/me` - Get current user
- `PUT /api/users/{id}` - Update user
- `DELETE /api/users/{id}` - Delete user

#### Tickets
- `GET /api/users/{user_id}/tickets` - Get user's tickets

#### Payments
- `GET /api/payments/{invoice_id}/status` - Check payment status

### Admin Endpoints (Require Admin Privileges)

#### Events
- `POST /api/admin/events` - Create new event
- `PUT /api/admin/events/{id}` - Update event
- `DELETE /api/admin/events/{id}` - Delete event

#### Payments
- `GET /api/admin/payments/pending` - Get pending payments
- `POST /api/admin/payments/{id}/retry` - Retry failed payment

## Authentication

The service uses JWT tokens for authentication. Include the token in the `Authorization` header:

```
Authorization: Bearer <your_jwt_token>
```

## UMA Payment Flow

1. **Ticket Purchase**: User requests ticket purchase with UMA address
2. **Invoice Creation**: Service creates Lightning invoice via UMA protocol
3. **Payment Processing**: User pays using Lightning Network
4. **Webhook Notification**: Payment status updated via webhook
5. **Ticket Delivery**: Ticket status updated to "paid"

## Database Schema

### Users
- `id`: Primary key
- `email`: Unique email address
- `name`: User's full name
- `created_at`, `updated_at`: Timestamps

### Events
- `id`: Primary key
- `title`: Event title
- `description`: Event description
- `start_time`, `end_time`: Event timing
- `capacity`: Maximum ticket capacity
- `price_sats`: Ticket price in satoshis
- `stream_url`: Virtual event stream URL
- `is_active`: Event availability status

### Tickets
- `id`: Primary key
- `event_id`: Reference to event
- `user_id`: Reference to user
- `ticket_code`: Unique ticket identifier
- `payment_status`: Payment status (pending/paid/expired/failed)
- `invoice_id`: Lightning invoice ID
- `uma_address`: UMA address for payment

### Payments
- `id`: Primary key
- `ticket_id`: Reference to ticket
- `invoice_id`: Lightning invoice ID
- `amount_sats`: Payment amount in satoshis
- `status`: Payment status
- `paid_at`: Payment completion timestamp

## Development

### Running Tests
```bash
go test ./...
```

### Code Structure
```
.
├── config/          # Configuration management
├── database/        # Database migrations and seeding
├── handlers/        # HTTP request handlers
├── middleware/      # Authentication and logging middleware
├── models/          # Data models and structs
├── repositories/    # Database access layer
├── server/          # Server setup and routing
├── services/        # Business logic (UMA integration)
├── main.go          # Application entry point
└── Dockerfile       # Container configuration
```

### Adding New Features

1. **Models**: Add new structs in `models/`
2. **Database**: Create migrations in `database/`
3. **Repository**: Implement data access in `repositories/`
4. **Service**: Add business logic in `services/`
5. **Handler**: Create HTTP handlers in `handlers/`
6. **Routes**: Add endpoints in `server/server.go`

## Deployment

### Docker
- Containerized with multi-stage build for minimal image size
- Health check endpoint at `/health`
- Non-root user for security
- Graceful shutdown handling

### AWS Staging
- Build and push to ECR
- Update existing Terraform infrastructure in parent directory
- ECS Fargate with Application Load Balancer
- RDS PostgreSQL in private subnets
- CloudWatch logging and monitoring

### Database
- Automatic migrations on startup
- Sample data seeding for development
- Connection pooling and timeout handling

## Security Considerations

- JWT tokens expire after 24 hours
- Admin access controlled by email whitelist
- Input validation on all endpoints
- CORS configured for web frontend
- Secure ticket code generation
- Payment webhook signature verification (implement in production)

## Production Deployment

1. **Environment Variables**: Set secure values for all environment variables
2. **Database**: Use production-grade PostgreSQL with proper backups
3. **HTTPS**: Configure TLS/SSL certificates
4. **Monitoring**: Implement health checks and logging
5. **Rate Limiting**: Add rate limiting for payment endpoints
6. **Backup**: Regular database backups and disaster recovery plan

## Troubleshooting

### Common Issues

1. **Database Connection**: Verify PostgreSQL is running and accessible
2. **UMA Integration**: Check Lightspark API token and node ID
3. **Port Conflicts**: Ensure port 8080 is available
4. **Permissions**: Check file permissions for Docker volumes

### Logs

The service uses structured JSON logging. Check logs for detailed error information:

```bash
docker-compose logs app
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Create an issue in the repository
- Check the documentation
- Review the code examples

## Roadmap

- [ ] Webhook signature verification
- [ ] Rate limiting implementation
- [ ] Email notifications
- [ ] Analytics and reporting
- [ ] Multi-currency support
- [ ] Advanced UMA features
- [ ] Mobile app support
