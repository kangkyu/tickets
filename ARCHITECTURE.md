# Tickets by UMA — Architecture

An e-ticketing platform for virtual events with Lightning Network payments via the UMA (Universal Money Address) protocol. Attendees pay in Bitcoin (satoshis) using UMA-compatible wallets.

**Production domain:** fanmeeting.org

---

## Table of Contents

1. [Project Structure](#project-structure)
2. [Backend](#backend)
3. [Frontend](#frontend)
4. [Payment Flow](#payment-flow)
5. [Infrastructure](#infrastructure)
6. [Local Development](#local-development)

---

## Project Structure

```
tickets-by-uma/
├── backend/            Go API server
├── frontend/           React SPA (Vite + Tailwind)
├── main.tf             Terraform infrastructure
├── terraform.tfvars    Environment-specific variables
└── .github/workflows/  CI/CD pipeline
```

---

## Backend

Go HTTP server using Gorilla Mux, PostgreSQL via sqlx, and the Lightspark SDK for Lightning payments.

### File Organization

```
backend/
├── main.go                     Entry point
├── config/config.go            Environment variable loading
├── server/server.go            Router setup, middleware, handler wiring
├── apphandlers/
│   ├── user_handlers.go        User CRUD, auth, NWC connection
│   ├── event_handlers.go       Event CRUD, admin operations
│   ├── ticket_handlers.go      Ticket purchase, validation, status
│   ├── payment_handlers.go     Payment webhooks, retry logic
│   └── lnurl_handlers.go       LNURL and UMA protocol endpoints
├── services/uma_service.go     UMA/Lightspark business logic
├── repositories/
│   ├── interfaces.go           Repository interface definitions
│   ├── user_repository.go
│   ├── event_repository.go
│   ├── ticket_repository.go
│   ├── payment_repository.go
│   └── nwc_connection_repository.go
├── middleware/auth.go           JWT auth, helpers
├── models/models.go            Domain models and request/response structs
└── db/
    ├── schema.sql              Full database schema
    ├── seed.sql                Seed data
    └── migrations/             dbmate migration files (11 total)
```

### API Endpoints

#### Authentication & Users

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/users` | Public | Register (email, name, password) |
| POST | `/api/users/login` | Public | Login, returns JWT |
| GET | `/api/users/{id}` | Public | Get user |
| GET | `/api/users/me` | Bearer | Get current user |
| PUT | `/api/users/{id}` | Bearer | Update user |
| DELETE | `/api/users/{id}` | Bearer | Delete user |

#### Events

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/events` | Public | List active events (paginated: `limit`, `offset`) |
| GET | `/api/events/{id}` | Public | Get event with UMA invoice and user ticket status |
| POST | `/api/admin/events` | Admin | Create event |
| PUT | `/api/admin/events/{id}` | Admin | Update event |
| DELETE | `/api/admin/events/{id}` | Admin | Delete event |

#### Tickets

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/tickets/purchase` | Public | Purchase ticket (event_id, user_id, uma_address) |
| GET | `/api/tickets/{id}/status` | Public | Check ticket payment status |
| POST | `/api/tickets/validate` | Public | Validate ticket code for event access |
| GET | `/api/users/{user_id}/tickets` | Bearer | List user's tickets |

#### Payments & Webhooks

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/webhooks/payment` | Public | Lightspark webhook (signature-verified) |
| POST | `/api/tickets/uma-callback` | Public | UMA payment callback |
| GET | `/api/payments/{invoice_id}/status` | Bearer | Check payment status |
| GET | `/api/admin/payments/pending` | Admin | List pending payments |
| POST | `/api/admin/payments/{id}/retry` | Admin | Retry failed payment |

#### NWC (Nostr Wallet Connect)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/users/me/nwc-connection` | Bearer | Store NWC connection URI |
| GET | `/api/users/me/nwc-connection` | Bearer | Check NWC connection status |

#### UMA Protocol & LNURL

| Method | Path | Description |
|--------|------|-------------|
| GET | `/.well-known/lnurlp/tickets` | LNURL-pay discovery |
| GET | `/api/lnurl/callback` | LNURL-pay callback (amount in millisats) |
| POST | `/uma/payreq/{ticket_id}` | UMA payreq callback from buyer's VASP |
| GET | `/.well-known/lnurlpubkey` | UMA signing/encryption cert chains |
| GET | `/.well-known/uma-configuration` | UMA version and request endpoint |

#### Admin Utilities

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/admin/events/{id}/uma-invoice` | Admin | Create event-level UMA invoice |
| GET | `/api/admin/node/balance` | Admin | Lightning node balance (sats) |
| GET | `/api/admin/status` | Admin | Verify admin access |
| GET | `/health` | Public | Health check with DB ping |

### Database Schema

**Users** — email (unique), name, password_hash (bcrypt), timestamps.

**Events** — title, description, start_time, end_time, capacity, price_sats, stream_url, is_active, timestamps.

**Tickets** — event_id (FK), user_id (FK), ticket_code (unique), payment_status (pending/paid/failed), invoice_id, uma_address, paid_at, timestamps.

**Payments** — ticket_id (FK), invoice_id (unique, bolt11), amount_sats, status (pending/paid/failed/expired), paid_at, timestamps.

**UMA Request Invoices** — event_id (FK, nullable), ticket_id (FK, nullable), invoice_id (unique), payment_hash, bolt11, amount_sats, status, uma_address, description, expires_at, timestamps.

**NWC Connections** — user_id (FK, unique), connection_uri (nostr+walletconnect://...), expires_at, timestamps.

Migrations managed by **dbmate** in `backend/db/migrations/`.

### UMA Service

Core business logic in `services/uma_service.go`:

| Method | Purpose |
|--------|---------|
| `CreateTicketInvoice` | Create per-ticket Lightning invoice via Lightspark |
| `SendUMARequest` | Push payment request to buyer's VASP via UMA Request protocol |
| `PayWithNWC` | Pay invoice using user's Nostr Wallet Connect URI |
| `SendPaymentToInvoice` | Pay invoice directly using Lightspark SDK |
| `HandleUMACallback` | Process payment status callbacks |
| `GetNodeBalance` | Query Lightning node balance |
| `ValidateUMAAddress` | Validate `$user@domain` format |
| `SimulateIncomingPayment` | Test mode: simulate payment via Lightspark |

### Middleware

- **JWT Auth** — HS256 tokens, 24-hour expiry, extracted from `Authorization: Bearer` header.
- **Admin Check** — Compares user email against `ADMIN_EMAILS` config list.
- **CORS** — Allows `http://localhost:3000` and `https://fanmeeting.org`. Credentials enabled.
- **Logging** — Logs method, path, status code, duration for all requests.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `PORT` | Server port (default: 8080) |
| `DATABASE_URL` | PostgreSQL connection string |
| `JWT_SECRET` | JWT signing secret |
| `ADMIN_EMAILS` | Comma-separated admin email addresses |
| `DOMAIN` | Application domain (e.g. fanmeeting.org) |
| `LIGHTSPARK_CLIENT_ID` | Lightspark API client ID |
| `LIGHTSPARK_CLIENT_SECRET` | Lightspark API secret |
| `LIGHTSPARK_NODE_ID` | Lightspark Lightning node ID |
| `LIGHTSPARK_NODE_PASSWORD` | Lightspark node signing key password |
| `LIGHTSPARK_WEBHOOK_SIGNING_KEY` | Webhook signature verification key |
| `UMA_SIGNING_PRIVKEY` | UMA signing private key (hex) |
| `UMA_SIGNING_CERT_CHAIN` | UMA signing certificate chain (PEM) |
| `UMA_ENCRYPTION_PRIVKEY` | UMA encryption private key (hex) |
| `UMA_ENCRYPTION_CERT_CHAIN` | UMA encryption certificate chain (PEM) |

### Key Dependencies

- `gorilla/mux` — HTTP routing
- `jmoiron/sqlx` + `lib/pq` — PostgreSQL access
- `golang-jwt/jwt/v5` — JWT authentication
- `lightsparkdev/go-sdk` — Lightning node integration
- `uma-universal-money-address/uma-go-sdk` — UMA protocol
- `untreu2/go-nwc` — Nostr Wallet Connect
- `golang.org/x/crypto/bcrypt` — Password hashing

---

## Frontend

React 18 SPA built with Vite, styled with Tailwind CSS. Uses TanStack React Query for server state and react-hook-form for forms.

### File Organization

```
frontend/src/
├── App.jsx                 Route definitions
├── main.jsx                Entry point (React 18 createRoot)
├── index.css               Tailwind imports and custom classes
├── components/
│   ├── Layout.jsx          App shell with responsive navbar
│   ├── Login.jsx           Login/register page
│   ├── ProtectedRoute.jsx  Auth guard
│   ├── AdminRoute.jsx      Admin guard
│   ├── EventList.jsx       Browse events with search
│   ├── EventDetails.jsx    Event detail page
│   ├── EventCard.jsx       Reusable event card
│   ├── TicketPurchase.jsx  Purchase flow with UMA wallet
│   ├── TicketList.jsx      User's tickets with status polling
│   ├── QRCodeDisplay.jsx   Ticket QR code modal
│   ├── OAuthCallback.jsx   UMA OAuth token exchange
│   ├── AdminDashboard.jsx  Admin panel
│   ├── CreateEvent.jsx     Create event form
│   └── EditEvent.jsx       Edit event form
├── contexts/
│   └── AuthContext.jsx     Auth state (user, token, isAdmin)
├── hooks/
│   ├── useEvents.js        Event queries and mutations
│   ├── useTickets.js       Ticket queries, purchase mutation, status polling
│   └── usePayments.js      Payment status polling with timeout
├── services/
│   └── api.js              Axios instance with auth interceptor
├── config/
│   └── api.js              API URL, poll intervals, timeouts
└── utils/
    ├── formatters.js       Date and price formatting
    └── validators.js       Zod schemas for form validation
```

### Routes

| Path | Component | Auth | Description |
|------|-----------|------|-------------|
| `/` | EventList | Public | Browse all events |
| `/events/:eventId` | EventDetails | Public | View event details |
| `/login` | Login | Public | Sign in or register |
| `/events/:eventId/purchase` | TicketPurchase | Protected | Purchase tickets |
| `/tickets` | TicketList | Protected | View purchased tickets |
| `/oauth/callback` | OAuthCallback | Protected | UMA wallet OAuth callback |
| `/admin` | AdminDashboard | Admin | Dashboard with stats |
| `/admin/events/new` | CreateEvent | Admin | Create new event |
| `/admin/events/:eventId/edit` | EditEvent | Admin | Edit existing event |

### State Management

**Auth Context** (`AuthContext.jsx`):
- Stores `user`, `token` (localStorage), `isAdmin`.
- Methods: `login`, `register`, `logout`, `deleteAccount`.
- Auto-checks token validity on mount.

**Server State** (React Query):
- Default stale time: 5 minutes.
- Refetch on window focus: disabled.
- Payment and ticket status hooks poll every 10 seconds.

**Forms** (react-hook-form + Zod):
- UMA address validation: `$username@domain.com` format with real-time feedback.
- Event forms: date, capacity, and price validation.

### Custom Hooks

| Hook | Purpose |
|------|---------|
| `useEvents()` | Fetch all events with filters |
| `useEvent(eventId)` | Fetch single event |
| `useEventMutation()` | Create event |
| `useUserTickets(userId)` | Fetch user's tickets |
| `useTicketPurchase()` | Purchase ticket mutation |
| `useTicketStatus(ticketId)` | Poll ticket status (10s interval) |
| `usePaymentStatus(invoiceId)` | Poll payment status (10s interval) |
| `usePaymentPolling(invoiceId, cb)` | Payment polling with 5-minute timeout |

### Key Dependencies

- `react` 18.2 + `react-router-dom` 6.8
- `@tanstack/react-query` 4.29 — server state
- `react-hook-form` 7.43 + `zod` 3.21 — form validation
- `axios` 1.4 — HTTP client
- `@uma-sdk/uma-auth-client` 0.0.10 — UMA wallet OAuth and NWC
- `tailwindcss` 3.3 — styling
- `lucide-react` — icons
- `date-fns` — date formatting

---

## Payment Flow

### Paid Events (price > 0)

```
1. User submits purchase form
   ├── POST /api/tickets/purchase { event_id, user_id, uma_address }
   └── Backend creates Ticket (pending) + Payment + UMA Invoice

2. Backend attempts payment (async goroutine)
   ├── Has NWC connection? → PayWithNWC(bolt11)
   └── No / failed → SendUMARequest to buyer's VASP

3. UMA Request flow (if NWC not used)
   ├── Backend discovers buyer's VASP from UMA address
   ├── Fetches VASP's .well-known/uma-configuration
   ├── POSTs UMA Invoice to VASP's uma_request_endpoint
   └── Buyer sees payment request in their wallet

4. Buyer's VASP pays
   ├── VASP calls POST /uma/payreq/{ticket_id}
   ├── Backend returns existing bolt11 in UMA response
   └── VASP pays the Lightning invoice

5. Payment confirmation
   ├── Lightspark fires PAYMENT_FINISHED webhook
   ├── Backend matches bolt11 → Payment record
   ├── Updates Payment status → "paid"
   └── Updates Ticket payment_status → "paid"

6. Frontend polls ticket status every 10s
   └── Displays "Confirmed" when paid
```

### Free Events (price = 0)

Ticket created immediately with `payment_status = "paid"`. No invoice or payment processing.

### NWC (Nostr Wallet Connect)

1. User clicks "Connect Wallet" (UMA Connect Button on purchase page).
2. OAuth redirect to UMA auth provider.
3. Callback at `/oauth/callback` exchanges code for NWC connection URI.
4. URI stored on backend via `POST /api/users/me/nwc-connection`.
5. On next purchase, backend uses NWC to pay the invoice automatically.

---

## Infrastructure

### AWS Architecture

```
                    ┌──────────────────┐
                    │   AWS Amplify    │
                    │  fanmeeting.org  │
                    │  (React SPA)     │
                    └──────────────────┘

                    ┌──────────────────┐
                    │  ALB (HTTPS)     │
                    │  api.fanmeeting  │
                    │  ACM certificate │
                    └────────┬─────────┘
                             │
┌────────────────────────────┼────────────────────────────┐
│  VPC 10.0.0.0/16           │                            │
│                            │                            │
│  Public Subnets            │   Private Subnets          │
│  10.0.1.0/24               │   10.0.10.0/24             │
│  10.0.2.0/24               │   10.0.11.0/24             │
│  ┌────────────┐            │   ┌─────────────────────┐  │
│  │ NAT Gateway│            │   │ ECS Fargate (2x)    │  │
│  └────────────┘            │   │ 0.25 vCPU / 512 MB  │  │
│                            │   │ Port 8080            │  │
│                            │   └──────────┬──────────┘  │
│                            │              │              │
│                            │   ┌──────────▼──────────┐  │
│                            │   │ RDS PostgreSQL 15    │  │
│                            │   │ db.t3.micro          │  │
│                            │   │ Encrypted, 7d backup │  │
│                            │   └─────────────────────┘  │
└─────────────────────────────────────────────────────────┘

External Services:
  Lightspark    — Lightning node, invoice creation, payments
  UMA SDK       — Invoice signing and encryption
  NWC           — Nostr Wallet Connect for automatic payments
```

### Terraform Resources (`main.tf`)

- **VPC** with public/private subnets, internet gateway, NAT gateway.
- **ECS Fargate** cluster, service (2 replicas), task definition.
- **ECR** repository for backend Docker images.
- **ALB** with HTTPS listener and ACM certificate for `api.fanmeeting.org`.
- **RDS PostgreSQL 15** in private subnet, encrypted, auto-scaling storage.
- **Amplify** app for frontend, connected to GitHub, custom domain.
- **CloudWatch** log group (7-day retention), container insights enabled.
- **SSM Parameter Store** for secrets (Lightspark credentials, UMA keys).

### Docker

Multi-stage build (`backend/Dockerfile`):
- **Builder**: `golang:1.24`, CGO enabled for Lightspark crypto.
- **Runtime**: `debian:bookworm-slim`, includes dbmate for migrations.
- Entrypoint (`docker-entrypoint.sh`): waits for PostgreSQL, runs migrations, starts server.

### CI/CD (`.github/workflows/backend-ci-cd.yml`)

Triggered on push to main/master or PR:

1. **Test** — Go 1.24, PostgreSQL service container, `go test ./...`.
2. **Build & Push** — Docker Buildx, tag with commit SHA + `latest`, push to ECR.
3. **Deploy** — `aws ecs update-service --force-new-deployment`, polls for stability (10 min timeout).

Frontend deploys automatically via Amplify on push to main.

---

## Local Development

### Backend

```bash
# Set up database
make test-setup       # Creates local PostgreSQL DB
make db-migrate       # Run migrations

# Run server
PORT=8080 DATABASE_URL="postgres://postgres:password@localhost:5432/tickets_uma?sslmode=disable" go run .

# Run tests
make test

# Build binary
make build
```

### Frontend

```bash
cd frontend
npm install
npm run dev           # Vite dev server on port 3000
npm run build         # Production build to dist/
npm run lint          # ESLint
```

### Frontend Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_BASE_URL` | `http://localhost:8080` | Backend API URL |
| `VITE_UMA_AUTH_APP_IDENTITY_PUBKEY` | — | Nostr hex public key for UMA OAuth |
| `VITE_UMA_AUTH_REDIRECT_URI` | — | OAuth redirect URI |
