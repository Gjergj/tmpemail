# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tmpemail is a temporary email service that gives users a temporary email address and displays incoming emails in real-time via a React frontend.

The system consists of three main components:
1. **API Service** (Go) - REST API and WebSocket server
2. **Email Service** (Go) - SMTP server for receiving emails
3. **Frontend** (React + TypeScript) - User interface

## Architecture

### Complete Email Flow
```
User Browser → API Service → Generate Address → Save to SQLite
                ↓
         WebSocket Connection
                ↓
External SMTP → Email Service (port 2525) → Validate Address
                ↓
         Save to Filesystem (SHA256 hash)
                ↓
         POST to API /internal/email/:address/store
                ↓
         API saves metadata to SQLite
                ↓
         WebSocket broadcast to connected clients
                ↓
         User sees email in real-time
```

### 1. API Service (`api/`)
**Technology**: Go with go-chi/chi (router) + gorilla/websocket + sqlx + mattn/go-sqlite3

**Responsibilities:**
- Generate temporary email addresses (format: `adjective-noun-4to6digits@tmpemail.xyz`)
- Sole owner of SQLite database (using sqlx for type-safe queries)
- WebSocket server for real-time email delivery
- REST API for frontend (versioned: `/api/v1/`)
- Background cleanup job (configurable interval, default 5 minutes)
- Tiered rate limiting (separate limits for generate, API, and WebSocket)
- CORS handling (configurable origins via environment)
- Health check endpoints (`/health`, `/readiness`)
- Request ID tracking for distributed tracing (`X-Request-ID` header)
- Graceful shutdown with 30-second timeout

**Database Schema:**
- `email_addresses`: id (ULID), address (unique), created_at, expires_at (24h default)
- `emails`: id (ULID), to_address (FK), from_address, subject, body_preview, body_text, body_html, file_path, received_at
- `attachments`: id (ULID), email_id (FK), filename, filepath, size

**Key Files:**
- `main.go` - Server setup, chi router configuration, middleware chain
- `config/config.go` - Environment-based configuration
- `database/db.go` - SQLite operations with sqlx
- `database/schema.sql` - Database schema (embedded)
- `models/models.go` - Data structures, ULID generation, address generator
- `handlers/address_handler.go` - `GET /api/v1/generate`
- `handlers/email_handler.go` - Email retrieval and attachment download
- `handlers/internal_handler.go` - Internal endpoints for Email Service
- `handlers/health_handler.go` - Health check endpoints
- `websocket/hub.go` - Room-based WebSocket broadcasting
- `websocket/handler.go` - WebSocket upgrade handler
- `websocket/client.go` - Client connection management
- `middleware/ratelimit.go` - In-memory rate limiter
- `middleware/cors.go` - CORS middleware
- `middleware/requestid.go` - Request ID middleware
- `cleanup/cleanup.go` - Background job for expired addresses

**Middleware Chain** (in order):
1. `RealIP` - Extracts real client IP from proxy headers
2. `RequestID` - Adds unique request ID to all requests
3. `CORS` - Handles cross-origin requests
4. `Recoverer` - Panic recovery

**Endpoints:**

| Method | Path | Rate Limit | Description |
|--------|------|------------|-------------|
| GET | `/` | - | API info |
| GET | `/health` | - | Liveness check |
| GET | `/readiness` | - | Readiness check (DB connectivity) |
| GET | `/ws?address={email}` | 5/min | WebSocket connection |
| GET | `/api/v1/generate` | 10/min | Generate new email address |
| GET | `/api/v1/emails/{address}` | 60/min | List emails for address |
| GET | `/api/v1/email/{address}/{emailID}` | 60/min | Get email content |
| GET | `/api/v1/email/{address}/{emailID}/attachments` | 60/min | List attachments |
| GET | `/api/v1/email/{address}/{emailID}/attachments/{attachmentID}` | 60/min | Download attachment |
| GET | `/internal/email/{address}` | - | Validate address (internal) |
| POST | `/internal/email/{address}/store` | - | Store email (internal) |

**Note:** Legacy routes without `/v1/` prefix are still supported for backwards compatibility.

**HTTP Server Settings:**
- Read timeout: 15 seconds
- Write timeout: 15 seconds
- Idle timeout: 60 seconds
- Graceful shutdown: 30 seconds

### 2. Email Service (`email-service/`)
**Technology**: Go with emersion/go-smtp

**Responsibilities:**
- SMTP server listening on port 2525 (configurable)
- Receive emails from any SMTP client
- Validate addresses against API Service in real-time
- Parse MIME multipart messages
- Extract text, HTML, and attachments (max 20MB total)
- Save raw email to filesystem with secure SHA256 hash
- Save attachments with sanitized filenames
- Call API Service to store metadata
- Retry logic with exponential backoff

**Key Files:**
- `main.go` - SMTP server and session handling
- `storage/storage.go` - Filesystem operations
- `client/api_client.go` - HTTP client for API Service
- `config/config.go` - Configuration management

**Email Processing:**
- Validates recipient address before accepting (RCPT TO)
- Rejects invalid or expired addresses with proper SMTP codes
- Parses MIME parts: text/plain, text/html, attachments
- Generates filename: `SHA256(timestamp + address + random).eml`
- Attachments: `emailfile_sanitized_attachment_name`

### 3. Frontend (`frontend/`)
**Technology**: React 18 + TypeScript + Vite

**Features:**
- Generate temporary email address on load
- Persist address in localStorage
- Real-time email display via WebSocket
- Email list with preview
- Full email viewer with HTML sanitization (DOMPurify)
- Attachment indicator
- Expiration countdown timer
- Copy email address to clipboard
- Generate new address option

**Key Files:**
- `src/App.tsx` - Main application component
- `src/components/EmailDisplay.tsx` - Email address display and countdown
- `src/components/EmailList.tsx` - Inbox with email previews
- `src/components/EmailViewer.tsx` - Full email modal viewer
- `src/hooks/useWebSocket.ts` - WebSocket connection with auto-reconnect
- `src/services/api.ts` - API client (axios)
- `src/utils/localStorage.ts` - Email address persistence

## Development Commands

### API Service (in `api/` directory)
```bash
make build       # Build for current platform
make run         # Build and run locally
make build-linux # Build for Linux deployment
make build-all   # Build for all platforms
make test        # Run tests
make fmt         # Format code
make vet         # Run go vet
make deps        # Install/update dependencies
```

**Environment Variables:**
- `TMPEMAIL_DB_PATH` - Database path (default: `/var/lib/tmpemail/tmpemail.db`)
- `TMPEMAIL_PORT` - API port (default: `8080`)
- `TMPEMAIL_DOMAIN` - Email domain (default: `tmpemail.xyz`)
- `TMPEMAIL_STORAGE_PATH` - Email storage (default: `/var/mail/tmpemail`)
- `TMPEMAIL_DEFAULT_EXPIRATION` - Expiry duration (default: `24h`)
- `TMPEMAIL_RATE_LIMIT_GENERATE` - Generate endpoint rate limit per minute (default: `10`)
- `TMPEMAIL_RATE_LIMIT_API` - API endpoints rate limit per minute (default: `60`)
- `TMPEMAIL_RATE_LIMIT_WS` - WebSocket connections rate limit per minute (default: `5`)
- `TMPEMAIL_CLEANUP_INTERVAL` - Cleanup job interval (default: `5m`)
- `TMPEMAIL_ALLOWED_ORIGINS` - Comma-separated CORS origins (default: `http://localhost:5173,http://localhost:3000`)
- `TMPEMAIL_STORAGE_QUOTA` - Max storage per email address in bytes (default: `52428800` = 50MB, 0 = unlimited)

### Email Service (in `email-service/` directory)
```bash
make build       # Build for current platform
make run         # Build and run locally
make build-linux # Build for Linux deployment
make build-all   # Build for all platforms
make deps        # Install/update dependencies
make gen-certs   # Generate self-signed TLS certificate for development
```

**Environment Variables:**
- `TMPEMAIL_SMTP_PORT` - SMTP port (default: `2525`)
- `TMPEMAIL_SMTP_HOST` - SMTP host (default: `0.0.0.0`)
- `TMPEMAIL_HEALTH_PORT` - Health check HTTP port (default: `8081`)
- `TMPEMAIL_STORAGE_PATH` - Email storage (default: `./mail`)
- `TMPEMAIL_API_URL` - API Service URL (default: `http://localhost:8080`)
- `TMPEMAIL_MAX_EMAIL_SIZE` - Max email size in bytes (default: `20971520` = 20MB)
- `TMPEMAIL_TLS_ENABLED` - Enable STARTTLS support (default: `false`)
- `TMPEMAIL_TLS_CERT_PATH` - Path to TLS certificate file (default: `./certs/smtp.crt`)
- `TMPEMAIL_TLS_KEY_PATH` - Path to TLS private key file (default: `./certs/smtp.key`)
- `TMPEMAIL_VALIDATE_SPF` - Enable SPF validation (default: `false`)
- `TMPEMAIL_VALIDATE_DKIM` - Enable DKIM signature verification (default: `false`)
- `TMPEMAIL_VALIDATE_DMARC` - Enable DMARC policy checking (default: `false`)
- `TMPEMAIL_AUTH_POLICY` - Policy for failed validation: `none` (log only) or `reject` (default: `none`)

**Health Check Endpoints** (on TMPEMAIL_HEALTH_PORT):
- `GET /health` - Liveness check (returns ok if server is running)
- `GET /readiness` - Readiness check (verifies SMTP server ready + API connectivity)

### Frontend (in `frontend/` directory)
```bash
npm install      # Install dependencies
npm run dev      # Start development server (port 5173)
npm run build    # Build for production
npm run preview  # Preview production build
```

**Environment Variables** (create `.env` file):
- `VITE_API_URL` - API base URL (default: `http://localhost:8080`)
- `VITE_WS_URL` - WebSocket URL (default: `ws://localhost:8080`)

## Local Development Setup

1. **Start API Service:**
```bash
cd api
make run
# Server starts on port 8080
```

2. **Start Email Service:**
```bash
cd email-service
make run
# SMTP server starts on port 2525
```

3. **Start Frontend:**
```bash
cd frontend
npm run dev
# Dev server starts on port 5173
```

4. **Test Email Flow:**
   - Open `http://localhost:5173` in browser
   - Copy the displayed temporary email address
   - Send test email using any SMTP client to `localhost:2525`
   - Email should appear in the frontend in real-time

## Testing Emails Locally

**Using telnet:**
```bash
telnet localhost 2525
HELO localhost
MAIL FROM: <test@example.com>
RCPT TO: <your-generated-address@tmpemail.xyz>
DATA
Subject: Test Email
From: test@example.com

This is a test email body.
.
QUIT
```

**Using swaks:**
```bash
 swaks --to bright-dolphin-873008@tmpemail.xyz \
      --from test@example.com \
      --server localhost:2525 \
      --body "Test email body" \
      --header "Subject: Test from swaks"
```

## TLS/STARTTLS Setup

The Email Service supports STARTTLS for encrypted email transmission.

**Generate self-signed certificate (for development):**
```bash
cd email-service
make gen-certs
```

**Enable TLS:**
```bash
TMPEMAIL_TLS_ENABLED=true make run
```

**Test STARTTLS:**
```bash
# Using openssl
openssl s_client -starttls smtp -connect localhost:2525

# Using swaks with TLS
swaks --to bright-dolphin-873008@tmpemail.xyz \
      --from test@example.com \
      --server localhost:2525 \
      --tls \
      --body "Test email body" \
      --header "Subject: Test from swaks with STARTTLS"

The --tls flag tells swaks to use STARTTLS. For self-signed certificates, you may need to add --tls-verify to skip certificate verification:
swaks --to bright-dolphin-873008@tmpemail.xyz \
      --from test@example.com \
      --server localhost:2525 \
      --tls \
      --tls-verify \
      --body "Test email body" \
      --header "Subject: Test from swaks with STARTTLS"
```

**Note:** Self-signed certificates work for development and internal use. For production with Gmail/Yahoo/Outlook delivery, use a valid certificate from Let's Encrypt or another CA.

## Email Authentication (SPF/DKIM/DMARC)

The Email Service supports validation of incoming emails using SPF, DKIM, and DMARC.

| Protocol | What It Validates |
|----------|-------------------|
| **SPF** | Sender's IP is authorized by the domain's DNS |
| **DKIM** | Email signature matches the domain's public key |
| **DMARC** | Combines SPF + DKIM results with domain policy |

**Enable validation:**
```bash
TMPEMAIL_VALIDATE_SPF=true \
TMPEMAIL_VALIDATE_DKIM=true \
TMPEMAIL_VALIDATE_DMARC=true \
TMPEMAIL_AUTH_POLICY=none \
make run
```

**Policies:**
- `none` - Log validation results but accept all emails (default)
- `reject` - Reject emails that fail validation

**Note:** For local development, most emails will fail SPF validation since they're not sent from authorized servers. Use `TMPEMAIL_AUTH_POLICY=none` during development.

## Key Technical Details

- **Router**: go-chi/chi v5 for clean, composable routing with path parameters
- **IDs**: Using ULID (github.com/oklog/ulid) instead of UUID for sortable IDs
- **Database**: SQLite with WAL mode, foreign key constraints, using sqlx for type-safe queries
- **WebSocket**: gorilla/websocket with room-based broadcasting (one room per email address)
- **Security**: HTML sanitization (bluemonday), tiered rate limiting, CORS, request ID tracking
- **Email Parsing**: Full MIME multipart support with attachment handling
- **Cleanup**: Background job with configurable interval (default 5 minutes)
- **Expiration**: Default 24 hours, configurable via environment
- **File Storage**: SHA256-based filenames to prevent collisions
- **API Versioning**: All endpoints versioned under `/api/v1/` with legacy support
- **Health Checks**: Liveness (`/health`) and readiness (`/readiness`) endpoints for orchestration
- **Logging**: Structured JSON logging with slog

## Dependencies

### API Service
| Package | Purpose |
|---------|---------|
| `github.com/go-chi/chi/v5` | HTTP router with middleware support |
| `github.com/gorilla/websocket` | WebSocket implementation |
| `github.com/jmoiron/sqlx` | SQL extensions for Go |
| `github.com/mattn/go-sqlite3` | SQLite driver (CGO) |
| `github.com/oklog/ulid/v2` | ULID generation |
| `github.com/microcosm-cc/bluemonday` | HTML sanitization |

### Email Service
| Package | Purpose |
|---------|---------|
| `github.com/emersion/go-smtp` | SMTP server |
| `github.com/jhillyerd/enmime` | Robust MIME email parsing |
| `github.com/oklog/ulid/v2` | ULID generation |
| `github.com/emersion/go-msgauth` | DKIM/DMARC validation |
| `blitiri.com.ar/go/spf` | SPF validation |

### Frontend
| Package | Purpose |
|---------|---------|
| `axios` | HTTP client |
| `dompurify` | HTML sanitization |

## Deployment

Deployment is handled separately via Dibra - DO NOT TOUCH `dibra.yaml`

## Project Structure
```
tmpemail/
├── api/                    # API Service (Go)
│   ├── main.go             # Entry point, router setup
│   ├── go.mod              # Go module definition
│   ├── Makefile            # Build commands
│   ├── config/
│   │   └── config.go       # Environment configuration
│   ├── database/
│   │   ├── db.go           # SQLx database operations
│   │   └── schema.sql      # Embedded schema
│   ├── models/
│   │   └── models.go       # Data structures, ULID, address generator
│   ├── handlers/
│   │   ├── address_handler.go   # Generate endpoint
│   │   ├── email_handler.go     # Email & attachment endpoints
│   │   ├── health_handler.go    # Health checks
│   │   └── internal_handler.go  # Internal API for Email Service
│   ├── websocket/
│   │   ├── hub.go          # Room-based broadcasting
│   │   ├── handler.go      # WS upgrade handler
│   │   └── client.go       # Client connection
│   ├── middleware/
│   │   ├── ratelimit.go    # Rate limiter
│   │   ├── cors.go         # CORS handler
│   │   └── requestid.go    # Request ID tracking
│   └── cleanup/
│       └── cleanup.go      # Background cleanup job
├── email-service/          # Email Service (Go)
│   ├── main.go             # SMTP server entry point
│   ├── go.mod
│   ├── Makefile
│   ├── config/
│   │   └── config.go
│   ├── storage/
│   │   └── storage.go      # Filesystem operations
│   └── client/
│       └── api_client.go   # HTTP client for API Service
├── frontend/               # Frontend (React + TypeScript)
│   ├── src/
│   │   ├── App.tsx
│   │   ├── components/
│   │   │   ├── EmailDisplay.tsx
│   │   │   ├── EmailList.tsx
│   │   │   └── EmailViewer.tsx
│   │   ├── hooks/
│   │   │   └── useWebSocket.ts
│   │   ├── services/
│   │   │   └── api.ts
│   │   └── utils/
│   │       └── localStorage.ts
│   ├── package.json
│   └── vite.config.ts
└── CLAUDE.md               # This file
```
