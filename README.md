# Multi-Tenant SaaS API with Usage-Based Billing

![CI Tests Status](https://github.com/Mekazstan/multi-tenant-saas-api/actions/workflows/ci.yml/badge.svg)

A production-ready RESTful API platform that demonstrates enterprise SaaS architecture. Organizations can sign up, generate API keys, and consume services with automatic usage tracking and billing—similar to Stripe, Twilio, or Paystack.

## Why This Project?

This isn't just another CRUD app. It showcases:

- **Multi-tenancy architecture** with proper tenant isolation
- **Production-grade security** with JWT authentication and API key management
- **Real-world billing logic** that tracks usage and calculates costs
- **Performance optimization** with Redis-based rate limiting
- **Scalable database design** with PostgreSQL and strategic indexing

Overall, it demonstrates a systems that handle real business requirements.

## Key Features

- 🔐 **Authentication & Authorization**: JWT-based auth with organization-scoped API keys
- 🚦 **Rate Limiting**: Redis-powered per-tenant rate limiting to prevent abuse
- 📊 **Usage Tracking**: Real-time monitoring of API consumption per organization
- 💰 **Billing System**: Automated monthly usage calculations and invoice generation
- 🔔 **Webhook System**: Event notifications for important actions (usage limits, billing, etc.)
- 📈 **Admin Dashboard**: Simple React/HTMX interface for analytics and management
- 🏗️ **Mock API Services**: Simulated SMS/Email sending endpoints to demonstrate the platform

## Tech Stack

- **Backend**: Go (Golang)
- **Database**: PostgreSQL with optimized indexing
- **Cache/Rate Limiting**: Redis
- **Authentication**: JWT tokens
- **Frontend**: React/HTMX (admin dashboard)
- **API Documentation**: (OpenAPI/Swagger)

## Getting Started

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 14+
- Redis 6+
- (Optional) Docker and Docker Compose

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/Mekazstan/multi-tenant-saas-api.git
   cd multi-tenant-saas-api
   ```

2. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your database credentials and configuration
   ```

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Run database migrations**
   ```bash
   goose -dir sql/schema postgres your_db_url up
   ```

5. **Start the server**
   ```bash
   go run cmd/api/main.go
   ```

6. **Start the Cron Scheduler**
   ```bash
   go run cmd/scheduler/main.go
   ```

The API will be available at `http://localhost:8080`

### Using Docker (Alternative)

```bash
docker-compose up -d
```

## Quick Start Guide

### 1. Create an Organization

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "organization_name": "Acme Corporation",
    "email": "admin@acmecorp.com",
    "password": "SecureP@ssw0rd123",
    "full_name": "John Doe"
  }'
```

### 2. Generate an API Key

```bash
curl -X POST http://localhost:8080/api/v1/keys \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production Key"
  }'
```

### 3. Make API Calls

```bash
curl -X POST http://localhost:8080/api/v1/messages/send \
  -H "X-API-Key: sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "to": "+2348012345678",
    "message": "Your verification code is 123456",
    "type": "sms"
  }'
```

### 4. View Usage and Billing

Access the admin dashboard at `http://localhost:8080/dashboard` or use the API:

```bash
curl -X GET http://localhost:8080/api/v1/usage \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## API Documentation

Full API documentation is available at `http://localhost:8080/docs` when the server is running.

### Main Endpoints

- `POST /api/v1/auth/register` - Create organization account
- `POST /api/v1/auth/login` - Authenticate user
- `GET  /api/v1/auth/me` - Get current user info
- `POST /api/v1/keys` - Generate API key
- `GET /api/v1/keys` - List API keys
- `DELETE /api-keys/:id` - Revoke API key
- `POST /api/v1/messages/send` - Send SMS/Email (mocked)
- `GET  /api/v1/messages/:id` - Get message status
- `GET /api/v1/billing/usage` - View usage statistics
- `GET /api/v1/billing/history` - View billing history
- `GET /api/v1/billing/calculate` - Calculate current period bill
- `POST /api/v1/billing/upgrade` - Upgrade organization plan
- `POST /api/v1/billing/initiate-payment` - Initiate a payment plan for an organization
- `GET /api/v1/dashboard/stats` - Overview stats
- `GET /api/v1/dashboard/usage-graph` - Usage over time (last 30 days)
- `GET /api/v1/dashboard/api-keys` - API keys with usage
- `POST /api/v1/webhooks/payment` - Webhook for payment verification

## Configuration

Key configuration options in `.env`:

```env
DATABASE_URL=postgresql://user:password@localhost:5432/saas_api
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-secret-key
API_PORT=8080
RATE_LIMIT_PER_MINUTE=60
BILLING_RATE_PER_REQUEST=0.01
```

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

## Deployment

The application is containerized and can be deployed to:
- AWS ECS/EKS
- Google Cloud Run
- Digital Ocean App Platform
- Any platform supporting Docker

See `docs/deployment.md` for detailed deployment guides.

## Roadmap

- [ ] Advanced analytics dashboard
- [ ] Multiple pricing tiers
- [ ] Webhook retry mechanism
- [ ] API versioning support
- [ ] GraphQL endpoint
- [ ] Real SMS/Email provider integration

## Contributing

Contributions are welcome! Please read `CONTRIBUTING.md` for details on the code of conduct and submission process.

## License

This project is licensed under the MIT License - see the `LICENSE` file for details.

## Acknowledgments

Built as a learning project to demonstrate production-grade Go development practices. Inspired by the API architectures of Stripe, Twilio, and Paystack.

---
 
**Contact**: mekastans@gmail.com