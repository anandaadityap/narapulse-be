# NaraPulse Backend Setup Guide

## Quick Start

### Prerequisites
- Go 1.21 or higher
- Docker and Docker Compose
- PostgreSQL (or use Docker)

### 1. Environment Setup

Copy the environment file:
```bash
cp .env.example .env
```

Edit `.env` file with your configuration:
```env
# Server Configuration
PORT=8080
ENVIRONMENT=development

# Database Configuration
DATABASE_URL=postgres://postgres:password@localhost:5432/narapulsedb?sslmode=disable
POSTGRES_DB=narapulsedb
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
```

### 2. Database Setup

#### Option A: Using Docker (Recommended)

1. Start Docker Desktop
2. Run the database:
```bash
docker-compose up -d postgres
```

#### Option B: Local PostgreSQL

1. Install PostgreSQL locally
2. Create database:
```sql
CREATE DATABASE narapulsedb;
```

### 3. Install Dependencies

```bash
go mod download
```

### 4. Run Database Migrations

```bash
# Install goose if not already installed
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run migrations
make migrate-up
# OR manually:
goose -dir migrations postgres "$DATABASE_URL" up
```

### 5. Start the Application

```bash
# Development mode with hot reload
make dev
# OR
air

# OR run directly
go run main.go
```

### 6. Verify Installation

1. Check health endpoint:
```bash
curl http://localhost:8080/health
```

2. Visit Swagger documentation:
```
http://localhost:8080/swagger/
```

## API Endpoints

### Public Endpoints
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login
- `GET /health` - Health check
- `GET /swagger/*` - API documentation

### Protected Endpoints (Requires JWT)
- `GET /api/v1/profile` - Get user profile
- `PUT /api/v1/profile` - Update user profile

### Admin Endpoints (Requires admin role)
- `GET /api/v1/admin/users` - Get all users
- `DELETE /api/v1/admin/users/:id` - Delete user

## Testing the API

### 1. Register a new user
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "username": "testuser",
    "password": "password123",
    "full_name": "Test User"
  }'
```

### 2. Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

### 3. Access protected endpoint
```bash
curl -X GET http://localhost:8080/api/v1/profile \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## Default Admin User

The migration creates a default admin user:
- **Email**: admin@narapulse.com
- **Password**: admin123
- **Role**: admin

## Development Commands

```bash
# Run with hot reload
make dev

# Build the application
make build

# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# Generate Swagger docs
make swagger

# Database migrations
make migrate-up
make migrate-down
make migrate-status

# Docker operations
make docker-build
make docker-run
```

## Troubleshooting

### Database Connection Issues
1. Ensure PostgreSQL is running
2. Check database credentials in `.env`
3. Verify database exists
4. Check firewall settings

### Build Issues
1. Ensure Go 1.21+ is installed
2. Run `go mod tidy`
3. Check for missing dependencies

### JWT Issues
1. Ensure JWT_SECRET is set in `.env`
2. Check token expiration
3. Verify token format

## Project Structure

```
.
├── cmd/                    # Application entrypoints
├── configs/               # Configuration files
├── docs/                  # Swagger documentation
├── internal/              # Private application code
│   ├── config/           # Configuration management
│   ├── database/         # Database connection
│   ├── handlers/         # HTTP handlers
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Data models
│   ├── repositories/     # Data access layer
│   ├── routes/           # Route definitions
│   ├── services/         # Business logic
│   └── utils/            # Utility functions
├── migrations/           # Database migrations
├── main.go              # Application entry point
├── Dockerfile           # Docker configuration
├── docker-compose.yml   # Docker Compose configuration
├── Makefile            # Build automation
└── README.md           # Project documentation
```

## Next Steps

1. Customize the user model for your needs
2. Add more business logic in services
3. Implement additional endpoints
4. Add comprehensive tests
5. Set up CI/CD pipeline
6. Configure production environment
7. Add logging and monitoring
8. Implement rate limiting
9. Add API versioning
10. Set up database backups