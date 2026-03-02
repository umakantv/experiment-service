# Experiment Service

A REST API service for managing entity-level experiments to control how new features are exposed to customers.

## Features

- **Database**: SQLite with local file storage
- **HTTP Server**: Standardized routing with authentication
- **Cache**: In-memory caching for performance
- **Logger**: Structured logging with custom format
- **Errors**: Standardized error responses

## Database

### Creating migrations

```bash
go run main.go --command create-migration --name experiment --dir database/migrations
```

## API Endpoints

### Public Endpoints
- `GET /health` - Health check (no auth required)

### Protected Endpoints (Bearer token required)
- `GET /experiments` - List all experiments
- `GET /experiments/{id}` - Get experiment by ID
- `POST /experiments` - Create new experiment
- `PUT /experiments/{id}` - Update experiment
- `DELETE /experiments/{id}` - Delete experiment

## Request/Response Examples

### Create Experiment
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "button-color-test",
    "description": "Testing button color impact on conversion",
    "start_date": "2026-03-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "variants": [
      {"name": "control", "description": "Blue button", "traffic_ratio": 0.5},
      {"name": "treatment", "description": "Green button", "traffic_ratio": 0.5}
    ]
  }'
```

### List Experiments
```bash
curl -X GET http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token"
```

### Get Experiment by ID
```bash
curl -X GET http://localhost:8080/experiments/1 \
  -H "Authorization: Bearer secret-token"
```

### Update Experiment
```bash
curl -X PUT http://localhost:8080/experiments/1 \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{"description": "Updated description"}'
```

### Delete Experiment
```bash
curl -X DELETE http://localhost:8080/experiments/1 \
  -H "Authorization: Bearer secret-token"
```

## Experiment Model

Each experiment contains:
- `id` - Unique identifier
- `name` - Experiment name (unique)
- `description` - Experiment description
- `start_date` - Start date/time
- `end_date` - End date/time
- `variants` - Array of variants with:
  - `name` - Variant name (e.g., "control", "treatment")
  - `description` - Variant description
  - `traffic_ratio` - Traffic allocation (must sum to 1.0 across all variants)
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp

## Validation Rules

- Experiment name is required and must be unique
- Start and end dates are required
- End date must be after start date
- At least 2 variants are required (e.g., control and treatment)
- All variants must have a name
- Traffic ratios must sum to 1.0 (with 0.01 tolerance)

## Authentication

All API endpoints (except health check) require Bearer token authentication:

```
Authorization: Bearer secret-token
```

## Running the Service

1. **Build and run (from project root):**
   ```bash
   go run main.go
   ```

2. **Check health:**
   ```bash
   curl http://localhost:8080/health
   ```

## Database

The service uses SQLite with a local file `./experiment_service.db`. The schema will be created via migrations.

## Logging

All requests are logged with the format:
```
2023-12-01 10:30:45 - RouteName - METHOD - /path - client:experiment-service-client
```

## Error Responses

Standardized error responses using the errs package:

```json
{
  "Code": 404,
  "Message": "Resource not found"
}
```

## Architecture

```
main.go          - Service entry point
handlers/        - Request handlers
models/          - Data models
database/        - Database schema and migrations
cache/           - Cache initialization
server/          - Server configuration and routes
```

This service demonstrates enterprise-ready patterns for building microservices with the go-utils packages.