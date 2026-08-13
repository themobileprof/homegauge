# API

Base URL: `http://localhost:8080/api/v1`  
Sessions: httpOnly cookie `homegauge_session` (credentials included).

## Health

`GET /health`

## Auth

| Method | Path | Auth |
|--------|------|------|
| POST | `/auth/register` | no |
| POST | `/auth/login` | no |
| POST | `/auth/logout` | cookie |
| GET | `/auth/me` | yes |
| POST | `/auth/verify-email` | no |
| POST | `/auth/forgot-password` | no |
| POST | `/auth/reset-password` | no |

## Calculator

`POST /calculator/affordability` — public estimate (reducing balance).

## Mortgages (public)

| Method | Path |
|--------|------|
| GET | `/countries` (`?include=coming_soon`) |
| GET | `/countries/:code` |
| GET | `/lenders?country=NG` |
| GET | `/mortgage-products?country=NG` |
| GET | `/mortgage-products/:id` |
| POST | `/mortgage-products/compare` |

Products and lenders are scoped by `country_code`. Assessments store `country_code` and only evaluate products in that market.

## Eligibility (authenticated)

| Method | Path |
|--------|------|
| POST | `/assessments` |
| GET | `/assessments/latest` |
| GET | `/assessments/:id` |
| PATCH | `/assessments/:id` |
| POST | `/assessments/:id/complete` |

Complete runs the rules engine + readiness score against active products. Outcomes never claim bank approval.

## Documents (authenticated)

| Method | Path |
|--------|------|
| GET | `/documents/checklist` |
| POST | `/documents/upload` (multipart) |
| GET | `/documents/:id/download-url` |
| GET | `/documents/file?token=` (short-lived signed; no session cookie) |

## Applications

| Method | Path | Auth |
|--------|------|------|
| GET | `/applications/me` | customer |
| POST | `/applications/request-advisor` | customer |

## Advisor

| Method | Path |
|--------|------|
| GET | `/advisor/cases` |
| GET | `/advisor/cases/:id` |
| PATCH | `/advisor/cases/:id/status` |
| POST | `/advisor/cases/:id/notes` |
| GET | `/advisor/cases/:id/suggestions` |
| POST | `/advisor/suggestions/:id/resolve` |
| POST | `/advisor/documents/:id/review` |

## RBAC probes

- `GET /api/v1/admin/ping` — ADMIN
- `GET /api/v1/advisor/ping` — ADVISOR or ADMIN
