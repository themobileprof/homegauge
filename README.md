# HomeGauge

Mortgage enablement platform — understand options in your market, check salary-account eligibility, estimate affordability, prepare documents, and get guided help. Not a bank. Not a property marketplace.

## Stack

- `frontend/` — Next.js + TypeScript + Tailwind
- `backend/` — Go (Gin) modular monolith
- PostgreSQL + Redis
- AI by job (Claude concierge, Gemini documents, DeepSeek numerics) + optional n8n webhooks
- No Docker required

## Prerequisites

- Go 1.24+ (toolchain may auto-fetch 1.25 for some deps)
- Node 20+
- PostgreSQL 14+ (peer/auth via Unix socket works with the default `DATABASE_URL`)
- Redis 7+

## Current MVP capabilities

- Multi-country foundation (`countries` table; Nigeria active; Ghana/Kenya marked coming soon)
- Public site + calculator (currency follows selected country)
- Mortgage product browse/compare scoped by country
- Salary-first eligibility assessment + readiness score
- Personalized document checklist + private upload (signed download links)
- Request advisor + advisor case queue (status, notes, AI suggestion approve/reject)
- Auth roles: CUSTOMER / ADVISOR / ADMIN

## Adding a country

1. Insert a row in `countries` (code, currency, locale, `region_label`, `regions` JSON, `status=active`).
2. Seed lenders + `mortgage_products` with that `country_code`, plus rules/document requirements.
3. Products and assessments automatically filter by country; the UI country switcher picks it up.

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

If `next dev` fails with a corrupt SWC binary (`bus error` / “missing section headers”), restore from the vendored tarball (or re-download from a fast npm mirror):

```bash
cd frontend
tar -xzf .vendor/swc-linux-x64-gnu-15.5.23.tgz -C /tmp
rm -rf node_modules/@next/swc-linux-x64-gnu
mv /tmp/package node_modules/@next/swc-linux-x64-gnu
npm run dev
```

- Web: http://localhost:3000  
- API health: http://localhost:8080/health  
- Advisor UI: http://localhost:3000/advisor  

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
