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

## RBAC probes

- `GET /api/v1/admin/ping` — ADMIN
- `GET /api/v1/advisor/ping` — ADVISOR or ADMIN

Further modules (lenders, assessments, documents, applications, admin CRUD) land in later phases.
