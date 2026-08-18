# HomeGauge

Mortgage enablement platform — understand options in your market, check salary-account eligibility, estimate affordability, prepare documents, and get guided help. Not a bank. Not a property marketplace.

## Stack

- `frontend/` — Next.js + TypeScript + Tailwind
- `backend/` — Go (Gin) modular monolith
- PostgreSQL + Redis
- AI only for unstructured jobs (e.g. statement extraction via Gemini); rules/calculator for the rest + optional n8n webhooks
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
- Request advisor; advisors work assigned files (notes, working status); admin assigns, reports, and records top-level status
- Auth roles: CUSTOMER / ADVISOR / ADMIN / LENDER_USER
- Role-specific workspaces: `/app` (homebuyer), `/advisor` (assigned cases), `/admin` (console + case ops), `/lender` (pipeline)

## Adding a country

1. Insert a row in `countries` (code, currency, locale, `region_label`, `regions` JSON, `status=active`).
2. Seed lenders + `mortgage_products` with that `country_code`, plus rules/document requirements.
3. Products and assessments automatically filter by country; the UI country switcher picks it up.

## Setup

```bash
cp .env.example .env
createdb homegauge   # if needed
cd frontend && npm install && cd ..

make start     # API + web in the background
make status
make stop
```

First-time data (demo users and NG products):

```bash
make seed
```

If `next dev` fails with a corrupt SWC binary (`bus error` / “missing section headers”):

```bash
make fix-swc
make start
```

Other targets: `make restart`, `make logs`, `make build`. Run `make` for the full list.

- Web: http://localhost:3000  
- API health: http://localhost:8080/health  
- Homebuyer: http://localhost:3000/app  
- Advisor: http://localhost:3000/advisor  
- Admin: http://localhost:3000/admin  
- Lender: http://localhost:3000/lender  

`.local` demo emails are accepted on the sign-in form. Re-run `make seed` if a demo password was changed.

### Seed logins

| Email | Password | Role | Lands on |
|-------|----------|------|----------|
| admin@homegauge.local | ChangeMeAdmin1! | ADMIN | `/admin` |
| advisor@homegauge.local | ChangeMeAdvisor1! | ADVISOR | `/advisor` |
| lender@homegauge.local | ChangeMeLender1! | LENDER_USER | `/lender` |
| demo@homegauge.local | ChangeMeDemo1! | CUSTOMER | `/app` |

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
