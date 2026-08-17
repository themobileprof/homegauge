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

## Admin

ADMIN role only.

| Method | Path |
|--------|------|
| GET | `/admin/overview` |
| GET | `/admin/users` |
| POST | `/admin/users` |
| PATCH | `/admin/users/:id` |
| DELETE | `/admin/users/:id` |
| GET | `/admin/ping` |
| GET | `/admin/ai-status` |

Create body: `{ email, password, full_name, role }`. Role is `CUSTOMER`, `ADVISOR`, `ADMIN`, or `LENDER_USER`. Accounts are created verified so they can sign in immediately. `.local` demo emails are accepted.

Update body (all optional): `{ full_name, role, status, password }`. Status is `active` or `disabled`. You cannot disable, demote, or delete your own account, and the last active admin cannot be removed.

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
