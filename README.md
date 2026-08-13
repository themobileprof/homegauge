# HomeGauge

Nigerian mortgage enablement platform — understand options, check salary-account eligibility, estimate affordability, prepare documents, and get guided help. Not a bank. Not a property marketplace.

## Stack

- `frontend/` — Next.js + TypeScript + Tailwind
- `backend/` — Go (Gin) modular monolith
- PostgreSQL + Redis
- Claude (AI concierge) + optional n8n webhooks
- No Docker required

## Prerequisites

- Go 1.24+ (toolchain may auto-fetch 1.25 for some deps)
- Node 20+
- PostgreSQL 14+ (peer/auth via Unix socket works with the default `DATABASE_URL`)
- Redis 7+

## Setup

```bash
cp .env.example .env
createdb homegauge   # if needed

# API (from repo root so .env loads)
go -C backend build -o /tmp/homegauge-api ./cmd/api
go -C backend build -o /tmp/homegauge-seed ./cmd/seed
/tmp/homegauge-api
/tmp/homegauge-seed

# Web
cd frontend && npm install && npm run dev
```

- Web: http://localhost:3000  
- API health: http://localhost:8080/health  

### Seed logins

| Email | Password | Role |
|-------|----------|------|
| admin@homegauge.local | ChangeMeAdmin1! | ADMIN |
| advisor@homegauge.local | ChangeMeAdvisor1! | ADVISOR |
| demo@homegauge.local | ChangeMeDemo1! | CUSTOMER |

## Tests

```bash
cd backend && go test ./...
```

## Docs

- [Product](docs/PRODUCT.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Database](docs/DATABASE.md)
- [API](docs/API.md)
- [Security](docs/SECURITY.md)
- [Deployment](docs/DEPLOYMENT.md)
