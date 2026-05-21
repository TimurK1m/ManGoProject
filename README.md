# manGo - Service Monitoring Application

A production-ready Go service monitoring application that continuously checks the health and availability of registered web services.

## Features

- **Service Registration**: REST API to register web services for monitoring
- **Health Checks**: Automated health checks every 30 seconds with 10-second timeout
- **Response Tracking**: Records response times and status (UP/DOWN) for each check
- **Database Persistence**: PostgreSQL with automatic migrations
- **Graceful Shutdown**: Proper signal handling and resource cleanup
- **Error Handling**: Comprehensive error handling and logging
- **Input Validation**: URL validation and request validation
- **Configuration Management**: Environment variable support for configuration
- **Authentication**: JWT-based user authentication and multi-tenant isolation
- **Telegram Alerts**: Instant notifications when services go UP or DOWN

## Project Structure

```
.
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go          # Configuration management
│   ├── database/
│   │   └── db.go              # Database connection & migrations
│   ├── handlers/
│   │   └── service.go         # API route handlers
│   ├── models/
│   │   └── models.go          # Data models (Service, Check)
│   └── workers/
│       └── worker.go          # Background health check worker
├── go.mod
├── go.sum
├── .env.example               # Environment variable template
└── README.md
```

## Prerequisites

- Go 1.25+
- PostgreSQL 12+

## Setup

### 1. Clone and Install Dependencies

```bash
go mod download
```

### 2. Configure Environment Variables

Copy `.env.example` to `.env` and update with your database credentials:

```bash
cp .env.example .env
```

Edit `.env`:

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_db_user
DB_PASSWORD=your_secure_password
DB_NAME=mango
DB_SSLMODE=require  # Use require for production
SERVER_PORT=8080
```

### 3. Create PostgreSQL Database

```sql
CREATE DATABASE mango;
```

The application will automatically create tables on first run.

### 4. Run the Application

```bash
go run cmd/main.go
```

The server will start on the configured port (default: 8080).

### 5. (Alternative) Run with Docker Compose

You can easily spin up the entire stack (PostgreSQL, Backend API, and Frontend) using Docker Compose.

```bash
docker-compose up -d --build
```

- **Frontend**: Available at `http://localhost:80`
- **Backend API**: Available at `http://localhost:8080`
- **Database**: PostgreSQL on port `5432`

## API Endpoints

### Health Check

```
GET /
```

Returns: `{"status": "ok", "message": "server running"}`

### Register Service

```
POST /services
Content-Type: application/json

{
  "url": "https://example.com"
}
```

Response (201 Created):
```json
{
  "message": "service created successfully",
  "service": {
    "id": 1,
    "url": "https://example.com",
    "created_at": "2024-03-25T10:30:00Z",
    "updated_at": "2024-03-25T10:30:00Z"
  }
}
```

### List All Services

```
GET /services
```

Response:
```json
{
  "count": 2,
  "services": [
    {
      "id": 1,
      "url": "https://example.com",
      "created_at": "2024-03-25T10:30:00Z",
      "updated_at": "2024-03-25T10:30:00Z"
    }
  ]
}
```

### Get Service Check History

```
GET /services/:id/checks
```

Response (returns last 100 checks):
```json
{
  "count": 3,
  "checks": [
    {
      "id": 5,
      "service_id": 1,
      "status": "UP",
      "response_time": 245,
      "created_at": "2024-03-25T11:00:00Z"
    }
  ]
}
```

## Key Improvements Made

### Critical Bug Fixes
- ✅ Fixed nil pointer dereference when HTTP request fails
- ✅ Fixed unbounded goroutines with sync.WaitGroup for proper resource cleanup
- ✅ Added HTTP timeout (10 seconds) to prevent hanging requests
- ✅ Removed hardcoded database credentials (now uses environment variables)

### Reliability Enhancements
- ✅ Comprehensive error handling on all database operations
- ✅ Graceful shutdown with signal handling (SIGTERM/SIGINT)
- ✅ Proper response body cleanup to prevent resource leaks
- ✅ URL validation before storing and checking services

### Production Readiness
- ✅ Configuration management via environment variables
- ✅ Improved API responses with proper HTTP status codes
- ✅ Request validation and input sanitization
- ✅ Structured logging throughout the application
- ✅ Automatic database migrations
- ✅ Database connection pooling

### Developer Experience
- ✅ Clean code structure and separation of concerns
- ✅ Clear error messages for debugging
- ✅ Proper logging with context (worker:, handlers:)
- ✅ Timestamps on data models for audit trail

## Configuration Options

All configuration is done via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_USER` | appuser | Database user |
| `DB_PASSWORD` | appuser | Database password |
| `DB_NAME` | mango | Database name |
| `DB_SSLMODE` | require | SSL mode (require/disable) |
| `SERVER_PORT` | 8080 | Server port |

## Monitoring

The application provides logging output that tracks:
- Server startup and shutdown events
- Worker health checks and status changes
- Database operations and errors
- API request errors

Example logs:
```
2024/03/25 10:30:00 connected to database successfully
2024/03/25 10:30:00 starting server on port 8080
2024/03/25 10:30:00 worker: started
2024/03/25 10:30:30 handlers: failed to create service: duplicate URL
2024/03/25 10:31:00 worker: stopping gracefully
```

## Performance Characteristics

- **Health Check Interval**: 30 seconds
- **HTTP Timeout**: 10 seconds per request
- **Concurrent Checks**: All services checked in parallel using goroutines
- **Database**: Connection pooling handled by GORM
- **Memory**: Bounded by number of services and check history

## Known Limitations

- Check history is stored indefinitely (consider adding cleanup for old records in production)
- Single-instance deployment (no clustering)

## Future Enhancements

- Database cleanup for old check records
- Webhooks integrations
- Prometheus metrics export
- Health check statistics (uptime percentage, avg response time)
- Service grouping/tagging
- Delete service endpoint
- Update service endpoint
- Uptime SLA reporting

## Troubleshooting

### Database Connection Error
```
failed to connect to database: connection refused
```
- Ensure PostgreSQL is running
- Check DB_HOST, DB_PORT, DB_USER, DB_PASSWORD in .env
- Verify database exists: `createdb mango`

### Services Not Being Checked
- Check logs for worker startup message
- Verify services are registered: `GET /services`
- Check PostgreSQL for data: `SELECT * FROM services;`

### Timeout Errors
- Increase `checkTimeout` in worker.go if services are slow
- Check network connectivity to monitored services

## License

MIT
