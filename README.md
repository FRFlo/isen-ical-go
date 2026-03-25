# ISEN iCal Generator

A Go-based web service that converts ISEN Aurion planning data into iCal format, compatible with Google Calendar, Apple Calendar, Outlook, and other calendar applications.

## Features

- **Secure Token-Based Access**: Generate secure tokens to access your calendar without sharing credentials
- **iCal Format Export**: Standard iCalendar format supported by all major calendar applications
- **AES-256-GCM Encryption**: Credentials are encrypted before storage with client-side keys
- **Valkey Caching**: Efficient caching of planning data to reduce load on Aurion servers
- **Concurrency Control**: Distributed locks prevent duplicate planning fetches
- **Worker-Compatible Contract**: Compatibility routes and trace headers align with the original Worker behavior
- **Docker Support**: Easy deployment with Docker and docker-compose
- **Health Monitoring**: Built-in `/api/health` endpoint with Valkey connectivity status
- **Privacy First**: No data sharing with third parties, transparent privacy policy

## Quick Start

### Using Docker Compose

1. Clone the repository:
```bash
git clone https://github.com/FRFlo/isen-ical-go.git
cd isen-ical-go
```

2. Copy the example environment file:
```bash
cp .env.example .env
```

3. Edit `.env` and set your encryption key (32-byte hex string):
```bash
# Generate a secure key
openssl rand -hex 32
```

4. Start the services:
```bash
docker-compose up -d
```

5. Access the application at `http://localhost:8080`

### Manual Setup

1. Install Go 1.21 or later
2. Install Valkey (or Redis-compatible server)
3. Copy `.env.example` to `.env` and configure
4. Run the application:
```bash
go run cmd/server/main.go
```

## API Documentation

### Endpoints

#### Health Check
```
GET /api/health
```

Returns the health status of the service.

**Responses:**
- `200 OK`
```json
{
  "status": "healthy"
}
```
- `503 Service Unavailable` when Valkey is disconnected
```json
{
  "status": "unhealthy",
  "valkey": "disconnected"
}
```

`/health` is not registered and returns `404 Not Found`.

#### Home Page
```
GET /
```

Returns the HTML homepage with API documentation.

**Headers:**
- `Accept: text/html` - Returns HTML documentation
- `Authorization: Basic <base64>` - Returns iCal data directly

#### Direct iCal Access with Basic Auth
```
GET /
Authorization: Basic <base64(email:password)>
```

Returns iCal directly when `Accept` does not include `text/html`.

**Successful Response Headers:**
- `Content-Type: text/calendar; charset=utf-8`
- `Content-Disposition: attachment; filename="isen-ical.ics"`

**Error Responses:**
- `401 Unauthorized` + `WWW-Authenticate: Basic realm="Identifiants Aurion"`
- `403 Forbidden` - Invalid credentials

#### Generate Token
```
POST /api/generate-token
```

Generates a secure token for calendar access.

**Request Body:**
```json
{
  "username": "prenom.nom@student.junia.com",
  "password": "votre_mot_de_passe"
}
```

`email` is also accepted for backward compatibility if `username` is not provided.

**Response:**
```json
{
  "token": "550e8400-e29b-41d4-a716-446655440000",
  "encryptionKey": "base64url-encoded-key",
  "url": "https://example.com/calendar/550e8400-e29b-41d4-a716-446655440000?key=..."
}
```

**Error Responses:**
- `400 Bad Request` - Invalid request format
- `403 Forbidden` - Invalid credentials

#### Access Calendar
```
GET /calendar/:token?key=<encryption_key>
```

Returns the calendar in iCal format using a token.

**Parameters:**
- `token` (path) - The token ID received from generate-token
- `key` (query) - The encryption key received from generate-token

**Response:**
```
Content-Type: text/calendar; charset=utf-8

BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//ISEN iCal Generator//EN
...
END:VCALENDAR
```

**Error Responses:**
- `400 Bad Request` - Missing or invalid parameters
- `403 Forbidden` - Failed to decrypt token
- `404 Not Found` - Token not found

#### Privacy Policy
```
GET /privacy
```

Returns the privacy policy page in HTML format.

#### Frontend Tracking
```
POST /api/track
```

Accepts frontend telemetry payloads and validates the event name.

**Request Body:**
```json
{
  "event": "frontend_page_view",
  "distinctId": "visitor-123",
  "properties": {
    "path": "/"
  }
}
```

**Behavior:**
- Returns `202 Accepted` for events prefixed with `frontend_`
- Returns `400 Bad Request` for invalid payloads or event names
- Returns `415 Unsupported Media Type` when `Content-Type` is not JSON

#### Compatibility Routes
```
GET /favicon.ico
GET /.well-known/appspecific/com.chrome.devtools.json
```

Both routes return `204 No Content` for client compatibility.

## Environment Variables

| Variable              | Description                               | Default                    | Required |
|-----------------------|-------------------------------------------|----------------------------|----------|
| `VALKEY_URL`          | Valkey/Redis-compatible connection URL    | `redis://localhost:6379`   | No       |
| `PORT`                | HTTP server port                          | `8080`                     | No       |
| `AURION_BASE_URL`     | Aurion system base URL                    | `https://aurion.junia.com` | No       |
| `MAX_TOKENS_PER_USER` | Maximum tokens per user                   | `3`                        | No       |
| `SESSION_TTL`         | Session cache TTL in seconds              | `3600`                     | No       |
| `CACHE_TTL`           | Events cache TTL in seconds               | `3600`                     | No       |
| `ENCRYPTION_KEY`      | 32-byte hex key for additional encryption | -                          | No       |
| `GIN_MODE`            | Gin framework mode (`release` or `debug`) | `debug`                    | No       |
| `ADMIN_API_TOKEN`     | Admin bearer token (currently reserved routes) | -                     | No       |

### Valkey URL Format

```
redis://[:password@]host[:port][/db]
rediss://[:password@]host[:port][/db]   # TLS enabled
valkey://[:password@]host[:port][/db]   # format used in .env.example/tests
valkeys://[:password@]host[:port][/db]  # format used in .env.example/tests
```

Examples:
- `redis://localhost:6379`
- `redis://:mypassword@valkey.example.com:6379/0`
- `rediss://secure.valkey.com:6380`

### Encryption Key

The `ENCRYPTION_KEY` is optional but recommended for production. It must be:
- 32 bytes (64 hex characters)
- Generated securely using: `openssl rand -hex 32`

## Architecture Overview

```
┌─────────────────┐
│   HTTP Client   │
│  (Browser/App)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Gin Router     │
│  + Middleware   │
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌────────┐ ┌──────────┐
│Handlers│ │ Services │
└───┬────┘ └────┬─────┘
    │           │
    ▼           ▼
┌────────┐ ┌──────────┐
│ Valkey │ │  Aurion  │
│Storage │ │  Client  │
└────────┘ └──────────┘
```

### Components

**Handlers** (`internal/handlers/`)
- HTTP request handlers for all endpoints
- Request validation and response formatting
- Business logic orchestration

**Services** (`internal/services/`)
- `aurion/` - HTTP client for Aurion system integration
- `auth/` - Basic authentication parsing
- `ical/` - iCalendar format generation
- `session/` - Session management and caching
- `token/` - Token generation and encryption
- `template/` - HTML template rendering helper used by handlers

**Storage** (`internal/storage/`)
- Valkey client wrapper with connection pooling
- Key pattern management
- Distributed locking for concurrency control

**Middleware** (`internal/middleware/`)
- Request logging
- Recovery from panics
- Security headers

**Models** (`internal/models/`)
- Data structures for events, credentials, and tokens

**Config** (`internal/config/`)
- Shared runtime config struct consumed by handlers/services
- Values are assembled from CLI flags + env sources in `cmd/server/main.go`

## Usage Examples

### Generate Token with cURL

```bash
curl -X POST http://localhost:8080/api/generate-token \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john.doe@student.junia.com",
    "password": "your_password"
  }'
```

### Subscribe in Google Calendar

1. Generate a token using the API
2. Copy the `url` from the response
3. In Google Calendar, click the "+" next to "Other calendars"
4. Select "From URL"
5. Paste the calendar URL
6. Click "Add calendar"

### Subscribe in Apple Calendar

1. Generate a token using the API
2. Open Calendar app
3. File → New Calendar Subscription
4. Paste the calendar URL
5. Configure refresh settings and click OK

### Direct iCal Access with Basic Auth

```bash
# Encode credentials
credentials=$(echo -n 'john.doe@student.junia.com:your_password' | base64)

# Request calendar
curl -H "Authorization: Basic $credentials" \
  http://localhost:8080/
```

## Security Considerations

1. **Credential Encryption**: All credentials are encrypted with AES-256-GCM using randomly generated keys
2. **Key Management**: Encryption keys are never stored server-side; they are provided by the client in the URL
3. **Token Limits**: Each user can have maximum 3 active tokens (configurable)
4. **HTTPS Required**: Always use HTTPS in production to protect tokens in URLs
5. **Credential Logging Scope**: Passwords are not logged in plain text, but URL query parameters may be logged (including `key` on `/calendar/:token?key=...`)
6. **Concurrency Locks**: Distributed locks deduplicate simultaneous fetches for the same planning window

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run integration tests (requires Valkey)
go test ./tests/...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/handlers/...
```

### Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/               # Configuration management
│   ├── handlers/             # HTTP handlers
│   ├── middleware/           # HTTP middleware
│   ├── models/               # Data models
│   ├── services/             # Business logic
│   │   ├── aurion/           # Aurion client
│   │   ├── auth/             # Authentication
│   │   ├── ical/             # iCal generation
│   │   ├── session/          # Session management
│   │   ├── token/            # Token service
│   │   └── template/         # HTML template helper
│   ├── storage/              # Valkey storage
├── tests/
│   └── integration_test.go   # Integration tests
├── docker-compose.yml        # Docker composition
├── Dockerfile                # Container image
├── go.mod                    # Go module definition
├── go.sum                    # Go dependencies
├── .env.example              # Example environment file
└── README.md                 # This file
```

### Building

```bash
# Build binary
go build -o isen-ical cmd/server/main.go

# Build Docker image
docker build -t isen-ical .
```

## Troubleshooting

### Valkey Connection Issues

If you see "Failed to connect to Valkey" errors:
1. Verify Valkey is running: `valkey-cli ping`
2. Check `VALKEY_URL` environment variable
3. Ensure network connectivity between app and Valkey

### Aurion Login Failures

If token generation fails with "Invalid credentials":
1. Verify your ISEN email and password
2. Check that `AURION_BASE_URL` is correct
3. Ensure your account is active in Aurion

### Calendar Not Updating

Calendar applications cache iCal feeds. To force refresh:
- **Google Calendar**: Can take up to 24 hours; remove and re-add the calendar
- **Apple Calendar**: Right-click calendar → Refresh
- **Outlook**: Send/Receive → Update Folder

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Commit changes: `git commit -am 'Add new feature'`
4. Push to branch: `git push origin feature/my-feature`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Disclaimer

This is an unofficial tool and is not affiliated with ISEN or Junia. Use at your own risk. Always protect your credentials and use HTTPS in production.

## Support

For issues, questions, or contributions, please use the GitHub issue tracker.
