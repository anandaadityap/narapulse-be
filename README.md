# NaraPulse Backend API

A robust Go backend API built with Fiber, GORM, PostgreSQL, and comprehensive authentication/authorization using JWT and Casbin RBAC.

## 🚀 Features

- **Web Framework**: Fiber v2 for high-performance HTTP server
- **Database**: PostgreSQL with GORM ORM
- **Authentication**: JWT-based authentication
- **Authorization**: Casbin RBAC (Role-Based Access Control)
- **Migrations**: Goose for database migrations
- **Documentation**: Swagger/OpenAPI documentation
- **Architecture**: Repository pattern with clean architecture
- **Validation**: Request validation with go-playground/validator
- **Security**: Password hashing with bcrypt
- **Standardized Responses**: Consistent API response format

## 📋 Prerequisites

- Go 1.21 or higher
- PostgreSQL 12 or higher
- Git

## 🛠️ Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd narapulse-be
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up PostgreSQL database**
   ```sql
   CREATE DATABASE narapulsedb;
   CREATE USER postgres WITH PASSWORD 'postgres';
   GRANT ALL PRIVILEGES ON DATABASE narapulsedb TO postgres;
   ```

4. **Set environment variables** (optional)
   ```bash
   export PORT=8080
   export DATABASE_URL="postgres://postgres:postgres@localhost:5432/narapulsedb?sslmode=disable"
   export JWT_SECRET="your-secret-key"
   export ENVIRONMENT="development"
   ```

5. **Run the application**
   ```bash
   go run main.go
   ```

## 🗄️ Database Migrations

This project uses Goose for database migrations and GORM auto-migration.

### Using Goose (Recommended for production)

1. **Install Goose**
   ```bash
   go install github.com/pressly/goose/v3/cmd/goose@latest
   ```

2. **Run migrations**
   ```bash
   goose -dir migrations postgres "postgres://postgres:postgres@localhost:5432/narapulsedb?sslmode=disable" up
   ```

3. **Create new migration**
   ```bash
   goose -dir migrations create migration_name sql
   ```

### Auto-Migration (Development)

GORM auto-migration runs automatically when starting the application in development mode.

## 🏗️ Project Structure

```
narapulse-be/
├── main.go                     # Application entry point
├── go.mod                      # Go module dependencies
├── go.sum                      # Go module checksums
├── README.md                   # Project documentation
├── configs/                    # Configuration files
│   ├── rbac_model.conf        # Casbin RBAC model
│   └── rbac_policy.csv        # Casbin RBAC policies
├── docs/                       # Swagger documentation
│   └── swagger.go             # Swagger configuration
├── migrations/                 # Database migrations
│   └── 00001_create_users_table.sql
└── internal/                   # Internal application code
    ├── config/                 # Configuration management
    │   └── config.go
    ├── database/               # Database connection and migration
    │   ├── database.go
    │   └── migrate.go
    ├── handlers/               # HTTP request handlers
    │   ├── auth_handler.go
    │   └── user_handler.go
    ├── middleware/             # HTTP middleware
    │   └── auth.go
    ├── models/                 # Data models and DTOs
    │   ├── user.go
    │   └── response.go
    ├── repositories/           # Data access layer
    │   └── user_repository.go
    ├── routes/                 # Route definitions
    │   └── routes.go
    ├── services/               # Business logic layer
    │   ├── user_service.go
    │   └── casbin_service.go
    └── utils/                  # Utility functions
        ├── jwt.go
        └── password.go
```

## 🔐 Authentication & Authorization

### Authentication
- JWT tokens for stateless authentication
- Token expiration: 24 hours
- Password hashing with bcrypt

### Authorization (RBAC)
- Casbin for role-based access control
- Roles: `admin`, `user`
- Policies defined in `configs/rbac_policy.csv`

### Default Users
- **Admin User**:
  - Email: `admin@narapulse.com`
  - Password: `password`
  - Role: `admin`

## 📚 API Documentation

### Swagger UI
Access the interactive API documentation at: `http://localhost:8080/swagger/`

### API Endpoints

#### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - User login

#### User Management
- `GET /api/v1/profile` - Get user profile (authenticated)
- `PUT /api/v1/profile` - Update user profile (authenticated)

#### Admin Endpoints
- `GET /api/v1/admin/users` - Get all users (admin only)
- `DELETE /api/v1/admin/users/:id` - Delete user (admin only)

#### Health Check
- `GET /health` - Server health status

### Standard Response Format

All API responses follow this standard format:

```json
{
  "success": true,
  "message": "Operation successful",
  "data": {},
  "error": null,
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 100,
    "total_pages": 10
  }
}
```

## 🧪 Testing

### Example API Calls

1. **Register a new user**
   ```bash
   curl -X POST http://localhost:8080/api/v1/auth/register \
     -H "Content-Type: application/json" \
     -d '{
       "email": "user@example.com",
       "username": "testuser",
       "password": "password123",
       "first_name": "Test",
       "last_name": "User"
     }'
   ```

2. **Login**
   ```bash
   curl -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{
       "email": "user@example.com",
       "password": "password123"
     }'
   ```

3. **Get profile (with token)**
   ```bash
   curl -X GET http://localhost:8080/api/v1/profile \
     -H "Authorization: Bearer YOUR_JWT_TOKEN"
   ```

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/narapulsedb?sslmode=disable` | PostgreSQL connection string |
| `JWT_SECRET` | `your-secret-key` | JWT signing secret |
| `ENVIRONMENT` | `development` | Application environment |

## 🏛️ Architecture Patterns

### Repository Pattern
- **Repository Layer**: Data access abstraction
- **Service Layer**: Business logic
- **Handler Layer**: HTTP request handling
- **Middleware Layer**: Cross-cutting concerns

### Clean Architecture Benefits
- **Separation of Concerns**: Each layer has a specific responsibility
- **Testability**: Easy to unit test business logic
- **Maintainability**: Changes in one layer don't affect others
- **Scalability**: Easy to add new features and endpoints

## 🚀 Deployment

### Docker (Recommended)

1. **Create Dockerfile**
   ```dockerfile
   FROM golang:1.21-alpine AS builder
   WORKDIR /app
   COPY go.mod go.sum ./
   RUN go mod download
   COPY . .
   RUN go build -o main .
   
   FROM alpine:latest
   RUN apk --no-cache add ca-certificates
   WORKDIR /root/
   COPY --from=builder /app/main .
   COPY --from=builder /app/configs ./configs
   CMD ["./main"]
   ```

2. **Build and run**
   ```bash
   docker build -t narapulse-be .
   docker run -p 8080:8080 narapulse-be
   ```

### Production Considerations

- Use environment-specific configuration
- Set up proper logging
- Configure HTTPS/TLS
- Set up database connection pooling
- Implement rate limiting
- Add monitoring and health checks
- Use secrets management for sensitive data

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 📞 Support

For support, email support@narapulse.com or create an issue in the repository.