# Implementation Plan: TmpEmail Service

## Overview
Build a temporary email service with 2 Go services (API + Email Service) and a React frontend. Users get a temporary email address and see incoming emails in real-time via WebSocket.

## Architecture Summary

### Services
1. **API Service** (`api/`) - Frontend-facing service & database owner
   - Generate temporary email addresses (format: `adjective-noun-number@tmpemail.xyz`). Where number postfix is random with 4 to 6 figures.
   - WebSocket server for real-time email delivery
   - REST endpoints for frontend
   - **SOLE OWNER** of SQLite database for email addresses and metadata
   - Reads email content from filesystem when needed
   - Background job for cleaning up expired addresses. The job should be a goroutine in API

2. **Email Service** (`email-service/`) - Internal email processing (stateless)
   - Receives emails from Postfix forwarding script
   - Validates incoming email addresses against API Service
   - Saves email content to filesystem only (no direct DB access)
   - Notifies API service via HTTP POST with full email data
   - Rejects emails for invalid/expired addresses
   - Supports attachments with 20Mb max size. Attachment filename is the email_filename+_attachment_file_name

3. **Frontend** (`frontend/`) - React + Vite in Typescript
   - Display temporary email address
   - WebSocket connection for real-time updates
   - Email viewer interface
   - LocalStorage for email address persistence but add the option to fetch a new address wich will replace the old.

### Communication Flow
```
Postfix → Email Service → (validate) → Filesystem + HTTP POST → API Service → SQLite DB + WebSocket → Frontend
                                                                      ↑
                                                                   (cleanup job)
```

**Key Design Decision:** API Service is the single database owner to avoid SQLite concurrent write issues. Email Service is stateless and communicates via HTTP.

## API Contracts (Define First!)

### REST Endpoints

**API Service External:**
- `GET /api/generate` → `{address: string, expires_at: string}` // expiry default is 24h
- `GET /api/emails/:address` → `{emails: [...]}`
- `GET /api/email/:address/:id` → `{body_html: string, body_text: string, from: string, subject: string,received_at: string, attachments: array[{filename: string, id: string}]}`
- `GET /api/emails/:address/:id/attachments` → `{files: [{filename: string, id: string}]}`

**API Service Internal:**
- `POST /internal/email/:address/store` → Receives full email data from Email Service
  - Body: `{to, from, subject, body_text, body_html, raw_email, file_path, timestamp, attachment_file_path}`
  - Returns: `{success: bool, message: string}`
- `GET /internal/email/:address/` → `{valid: bool, expired: bool}`

**Email Service:**
- This should be deprecated -> `POST /receive-email`

### WebSocket Protocol

**Connection:** `ws://localhost:8080/ws?address=happy-turtle-42@tmpemail.xyz`

**Message Format (Server → Client):**
```json
{
  "type": "new_email",
  "data": {
    "id": "uuid",
    "from": "sender@example.com",
    "subject": "Hello",
    "preview": "First 200 chars...",
    "received_at": "2024-01-01T12:00:00Z"
  }
}
```

## Implementation Steps

### Phase 1: Core Models & Email Address Generation

**File: `api/models/models.go`**
- EmailAddress struct with validation
- Email struct with MIME parsing helpers
- Address generator function (adjective-noun-number pattern)
- Expiration logic (default: 1 hour)

**File: `api/database/schema.sql`**
- `email_addresses` table: id (UUID PK), address (unique), created_at, expires_at
- `emails` table: id (UUID PK), to_address (FK), from_address, subject, body_preview, file_path, received_at
- `attachments` table: id (UUID PK), address (FK), filepath, size
- Indexes: address (unique), expires_at, to_address + received_at

**File: `api/database/db.go`**
- SQLite initialization with proper pragmas (WAL mode, foreign keys)
- Auto-run schema.sql if tables don't exist
- Query methods: InsertAddress, GetAddress, IsValidAddress, InsertEmail, GetEmailsByAddress
- Use sqlx package

### Phase 2: API Service - HTTP Handlers & Security

**File: `api/config/config.go`**
- Configuration struct (DB path, port, domain, filesystem path for emails, email_domain)
- Defaults: DB at `/var/lib/tmpemail/tmpemail.db`, port 8080, filesystem at `/var/mail/tmpemail/`
- Load from environment variables

**File: `api/middleware/ratelimit.go`**
- Rate limiting middleware (10 req/min per IP for `/api/generate`)
- 429 responses when exceeded
- Simple in-memory counter (upgrade to Redis later if needed)

**File: `api/middleware/cors.go`**
- CORS middleware for frontend
- Allow configured origins only

**File: `api/handlers/address_handler.go`**
- `POST /api/generate` - Generate new temporary email address
  - Apply rate limiting
  - Generate readable address (adjective-noun-number).  Where number postfix is random with 4 to 6 figures.
  - Save to database with 1-hour expiration
  - Return JSON with address and expires_at

**File: `api/handlers/email_handler.go`**
- `GET /api/emails/:address` - Get all emails for address (metadata only)
  - Validate address exists and not expired
  - Return list of emails (without full body)
- `GET /api/email/:address/:id` - Get full email content from filesystem
  - Read file from path stored in database
  - Parse and return HTML/text parts
  - Sanitize HTML to prevent XSS
- `GET /api/emails/:address/:id/attachments` Get list of attachments per email
  - `{files: [{filename: string, id: string}]}`

**File: `api/handlers/internal_handler.go`**
- `POST /internal/email/:address/store` - Store email from Email Service
  - Validate address exists in database
  - Check address not expired
  - Insert email metadata into database
  - Trigger WebSocket broadcast
  - Return success/error
- `GET /internal/email/:address/` - Validate address
  - Check if address exists and not expired
  - Used by Email Service before processing

**File: `api/main.go`** (refactor existing)
- Remove old `/receive-email` endpoint
- Wire up all handlers with proper routing
- Initialize database and config
- Start cleanup goroutine
- Start HTTP server with middleware chain

### Phase 3: WebSocket Infrastructure

**File: `api/websocket/hub.go`**
- WebSocket hub with room management
- Map of email addresses to connected clients
- Broadcast method: send to all clients subscribed to specific address
- Register/unregister clients
- Handle connection cleanup

**File: `api/websocket/handler.go`**
- WebSocket upgrade handler at `/ws?address=xxx`
- Extract address from query params
- Validate address exists and not expired
- Register connection to hub under that address
- Handle ping/pong for keepalive
- Close stale connections

**File: `api/websocket/client.go`**
- Client struct wrapping WebSocket connection
- Read/write pumps
- Message sending with timeout
- Cleanup on disconnect

### Phase 4: Email Service (Stateless)

**File: `email-service/main.go`**
- HTTP server listening on port 8081 (internal only)
- DEPRECATE `POST /receive-email` endpoint (from Postfix script)
- SMTP server Receive email from SMTP
  - Validate these packages for smtp server
    - https://github.com/albertito/chasquid
    - https://github.com/foxcpp/maddy
    - https://github.com/axllent/mailpit
    - https://github.com/mjl-/mox
- Receive attachments

- **Validate address:** Call API `/internal/email/:address/`
- If invalid/expired: log and return error
- If valid: Generate filename hash and save to filesystem
- Parse email for preview text (first 200 chars)
- Call API `/internal/email/:address/store` with full metadata
- Log all operations with slog
- Enforce max email size (10MB)

**File: `email-service/storage/storage.go`**
- Generate secure filename: `SHA256(timestamp + to_address + random_number).eml` Where number postfix is random with 4 to 6 figures.
- Save raw email to `/var/mail/tmpemail/{filename}`
- Atomic write (temp file + rename)
- Return file path for database storage

**File: `email-service/config/config.go`**
- Configuration: storage path, API service URL, max email size
- Defaults: storage `/var/mail/tmpemail/`, API `http://localhost:8080`
- smtp port

**File: `email-service/client/api_client.go`**
- HTTP client for calling API Service
- `ValidateAddress(address)` - GET `/internal/email/:address/`
- `StoreEmail(metadata)` - POST `/internal/email/:address/store`
- Retry logic with exponential backoff (max 3 attempts)
- Timeout: 5 seconds per request

### Phase 5: Cleanup Job & Background Tasks

**File: `api/cleanup/cleanup.go`**
- Background goroutine running every 5 minutes
- Query for expired email addresses (`expires_at < NOW()`)
- For each expired address:
  - Get all associated email file paths from database
  - Delete email files from filesystem
  - Delete attachements files from filesystem
  - Delete email records from database
  - Delete email_address record
- Log cleanup operations
- Handle partial failures gracefully

**File: `api/main.go`** (add to startup)
- Start cleanup goroutine: `go cleanup.Start(db, config)`
- Graceful shutdown handling (wait for cleanup to finish current cycle)

### Phase 6: Frontend - React + Vite + TypeScript

**Setup:**
- `cd` to project root, create `frontend/` directory
- Run: `npm create vite@latest frontend -- --template react-ts`
- Install dependencies: `npm install axios`

**File: `frontend/src/App.tsx`**
- Main application with email address state
- On mount: check localStorage for existing address
- If none or expired: call API to generate new address
- Save address + expiry to localStorage
- Display EmailDisplay and EmailList components

**File: `frontend/src/components/EmailDisplay.tsx`**
- Display temporary email address in large, copyable format
- Copy to clipboard button with feedback
- Show expiration countdown timer

**File: `frontend/src/components/EmailList.tsx`**
- Display list of received emails (from, subject, preview, time ago)
- Empty state: "Waiting for emails..."
- Click to view full email

**File: `frontend/src/components/EmailViewer.tsx`**
- Modal or panel showing full email content
- Fetch full content via API when opened
- Sanitize and render HTML safely (use DOMPurify)
- Show text fallback if no HTML

**File: `frontend/src/hooks/useWebSocket.ts`**
- Custom hook accepting email address
- Opens WebSocket connection to `ws://localhost:8080/ws?address={address}`
- Parses incoming JSON messages
- Auto-reconnect on disconnect (exponential backoff, max 30s)
- Returns: `{ messages, isConnected, error }`

**File: `frontend/src/services/api.ts`**
- Axios instance with baseURL `http://localhost:8080`
- `generateEmail()` → `POST /api/generate`
- `getEmails(address)` → `GET /api/emails/:address`
- `getEmailContent(id)` → `GET /api/email/:id/content`

**File: `frontend/src/utils/localStorage.ts`**
- `saveEmailAddress(address, expiresAt)`
- `getEmailAddress()` → returns address if not expired, else null
- `clearEmailAddress()`

### Phase 7: Build & Configuration Files

**Create: `email-service/Makefile`**
- Same structure as `api/Makefile`
- Binary name: `tmpemail-email-service`
- All platform build targets

**Create: `email-service/go.mod`**
- Module: `tmpemail_email_service`
- Go version: 1.24.3

**Update: `api/go.mod`**
- Add dependencies (will be added via `go get` during implementation)

## Critical Files to Create/Modify

**New Files (API Service):**
- `api/database/schema.sql` - Database schema
- `api/database/db.go` - Database initialization and queries
- `api/models/models.go` - Data structures and address generator
- `api/config/config.go` - Configuration management
- `api/middleware/ratelimit.go` - Rate limiting
- `api/middleware/cors.go` - CORS handling
- `api/handlers/address_handler.go` - Email address generation endpoint
- `api/handlers/email_handler.go` - Email retrieval endpoints
- `api/handlers/internal_handler.go` - Internal endpoints for Email Service
- `api/websocket/hub.go` - WebSocket hub
- `api/websocket/handler.go` - WebSocket HTTP handler
- `api/websocket/client.go` - WebSocket client wrapper
- `api/cleanup/cleanup.go` - Background cleanup job

**New Files (Email Service):**
- `email-service/main.go` - Email receiving HTTP server
- `email-service/storage/storage.go` - Filesystem storage
- `email-service/config/config.go` - Configuration
- `email-service/client/api_client.go` - API Service HTTP client
- `email-service/Makefile` - Build automation
- `email-service/go.mod` - Go module definition

**New Files (Frontend):**
- `frontend/` (entire Vite + React + TypeScript app)
- All component, hook, and utility files as listed in Phase 6

**Modified Files:**
- `api/main.go` - Complete refactor (remove old endpoint, wire new architecture)
- `api/go.mod` - Add WebSocket, SQLite, CORS dependencies
- `CLAUDE.md` - Update with new architecture details

## Dependencies to Add

**API Service (api/go.mod):**
```bash
go get github.com/gorilla/websocket    # WebSocket support
go get github.com/mattn/go-sqlite3     # SQLite driver (CGO required)
go get https://github.com/oklog/ulid        # ULID generation
go get github.com/microcosm-cc/bluemonday  # HTML sanitization
```

**Email Service (email-service/go.mod):**
```bash
go get https://github.com/oklog/ulid       # ULID generation
```
(No database driver needed - stateless service)

**Frontend (package.json):**
```bash
npm install axios dompurify
npm install -D @types/dompurify
```
- `axios` - HTTP client for API calls
- `dompurify` - HTML sanitization for email content
- React, React-DOM, and Vite come with template

## Verification Steps

### 1. API Service - Standalone Testing
```bash
# Start API service
cd api && make run

# Test email address generation
curl -X POST http://localhost:8080/api/generate
# Expected: {"address":"happy-turtle-42@tmpemail.xyz","expires_at":"2024-01-01T13:00:00Z"}

# Test getting emails (should be empty)
curl http://localhost:8080/api/emails/happy-turtle-42@tmpemail.xyz
# Expected: {"emails":[]}

# Test internal validation endpoint
curl http://localhost:8080/internal/validate/happy-turtle-42@tmpemail.xyz
# Expected: {"valid":true,"expired":false}

# Verify database created
ls -la /var/lib/tmpemail/tmpemail.db
sqlite3 /var/lib/tmpemail/tmpemail.db "SELECT * FROM email_addresses;"
```

### 2. Email Service - Standalone Testing
- let's skip this

### 3. WebSocket Testing
- let's skip this 
```bash
# Use wscat for WebSocket testing
npm install -g wscat

# Connect to WebSocket
wscat -c "ws://localhost:8080/ws?address=happy-turtle-42@tmpemail.xyz"

# In another terminal, send test email via Email Service
# Should see message appear in wscat connection:
# {"type":"new_email","data":{...}}
```

### 4. Integration Testing (Both Services Running)
- let's skip this
```bash
# Terminal 1: Start API service
cd api && make run

# Terminal 2: Start Email Service
cd email-service && make run

# Terminal 3: Generate address and send email
ADDRESS=$(curl -s -X POST http://localhost:8080/api/generate | jq -r '.address')
echo "Generated address: $ADDRESS"

# Send test email
curl -X POST http://localhost:8081/receive-email \
  -H "Content-Type: application/json" \
  -d "{\"to\":\"$ADDRESS\",\"from\":\"sender@test.com\",\"subject\":\"Integration Test\",\"raw_email\":\"Test body\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}"

# Verify email stored
curl http://localhost:8080/api/emails/$ADDRESS
# Should see the test email in response

# Get email content
EMAIL_ID=$(curl -s http://localhost:8080/api/emails/$ADDRESS | jq -r '.emails[0].id')
curl http://localhost:8080/api/email/$EMAIL_ID/content
```

### 5. Frontend Testing
```bash
# Start frontend dev server
cd frontend && npm run dev

# Open browser to http://localhost:5173
# Verify:
# - Email address displayed
# - Address saved to localStorage
# - WebSocket connection indicator shows "Connected"
# - Empty state shows "Waiting for emails..."

# Send test email to displayed address
# Verify email appears in real-time without refresh
```

### 6. Cleanup Job Testing
```bash
# Start API service and wait 5+ minutes
# Check logs for cleanup messages

# Or manually test:
# 1. Create email address with 1-minute expiry (modify code temporarily)
# 2. Send emails to that address
# 3. Wait 1+ minutes
# 4. Check that files are deleted and database cleaned up
```

### 7. Security Testing
```bash
# Test rate limiting
for i in {1..15}; do
  curl -X POST http://localhost:8080/api/generate
done
# Should get 429 Too Many Requests after 10 requests

# Test invalid address rejection
curl -X POST http://localhost:8081/receive-email \
  -d '{"to":"doesnotexist@tmpemail.xyz"}'
# Should return error

# Test expired address
# (Create address, wait for expiry, try to send email)
```

### 8. End-to-End with Real SMTP (Manual)
```bash
# After full deployment:
# 1. Open frontend in browser
# 2. Copy displayed email address
# 3. Send real email from Gmail/Outlook to that address
# 4. Verify email appears in frontend within seconds
```

## Configuration Reference

**Ports:**
- API Service: `8080` (external)
- Frontend Dev: `5173` (Vite default)

**Paths:**
- SQLite DB: `/var/lib/tmpemail/tmpemail.db`
- Email Storage: `/var/mail/tmpemail/`
- Email files: `/var/mail/tmpemail/{sha256_hash}.eml`

**Email Address Format:**
- Pattern: `adjective-noun-number@tmpemail.xyz`
- Example: `happy-turtle-42@tmpemail.xyz`
- Expiration: 1 hour from creation

**Environment Variables:**
- `TMPEMAIL_DB_PATH` - Database path (default: `/var/lib/tmpemail/tmpemail.db`)
- `TMPEMAIL_STORAGE_PATH` - Email storage path (default: `/var/mail/tmpemail/`)
- `TMPEMAIL_API_URL` - API Service URL for Email Service (default: `http://localhost:8080`)
- `TMPEMAIL_PORT` - API Service port (default: `8080`)

**Manual Test STARTTLS**
```
swaks --to short-sphinx-123035@tmpemail.xyz \
      --from test@example.com \
      --server tmpemail.xyz:25 \
      --tls \
      --body "Test email body" \
      --header "Subject: Test from swaks with STARTTLS"
```