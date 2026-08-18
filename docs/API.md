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

Products and lenders are scoped by `country_code`. Assessments store `country_code` and only evaluate products in that market. Product payloads include `interest_rate` (indicative/headline), optional `interest_rate_min` / `interest_rate_max` (a published band), and `interest_rate_type` (`fixed`, `variable`, or `negotiable`).

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
| GET | `/applications/me` | customer — `{ application, documents, notes }` (buyer-visible notes) |
| POST | `/applications/request-advisor` | customer |
| PATCH | `/applications/me/product` | customer `{ product_id }` |

## Admin

ADMIN role only.

| Method | Path |
|--------|------|
| GET | `/admin/overview` |
| GET | `/admin/users` |
| POST | `/admin/users` |
| PATCH | `/admin/users/:id` |
| DELETE | `/admin/users/:id` |
| GET | `/admin/lenders` |
| POST | `/admin/lenders` |
| GET | `/admin/products` |
| POST | `/admin/products` |
| PATCH | `/admin/products/:id` |
| DELETE | `/admin/products/:id` |
| GET | `/admin/advisors` |
| GET | `/admin/cases` (`?unassigned=1`, `?advisor_id=`, `?status=`) |
| GET | `/admin/cases/:id` |
| POST | `/admin/cases/:id/assign` |
| PATCH | `/admin/cases/:id/status` |
| GET | `/admin/reports/advisors` |
| GET | `/admin/reports/buyers` |
| GET | `/admin/approvals` |
| GET | `/admin/ping` |
| GET | `/admin/ai-status` |

Create body: `{ email, password, full_name, role, lender_id }`. Role is `CUSTOMER`, `ADVISOR`, `ADMIN`, or `LENDER_USER`. `lender_id` is required for `LENDER_USER`. Accounts are created verified so they can sign in immediately. `.local` demo emails are accepted.

Update body (all optional): `{ full_name, role, status, password, lender_id }`. Status is `active` or `disabled`. You cannot disable, demote, or delete your own account, and the last active admin cannot be removed.

Product create/update body: `{ lender_id, country_code, name, mortgage_type, description, interest_rate, interest_rate_min, interest_rate_max, interest_rate_type, min_loan_amount, max_loan_amount, min_income, min_equity_pct, max_tenor_years, max_age, fees, status, verification_status, source, source_url, sync_rules }`. `mortgage_type` is `nhf`, `mreif`, `commercial`, `scheme`, or `other`. `interest_rate` is the indicative/headline figure used in estimates. Optional `interest_rate_min` / `interest_rate_max` are a published band (set both or neither; indicative must sit inside). `interest_rate_type` is `fixed`, `variable`, or `negotiable`. A true band or `negotiable` is shown as a range, not a locked offer. Status is `active` or `inactive`. Verification is `verified`, `needs_verification`, or `expired`. Saving with `sync_rules` (default true) updates eligibility rules for income, age, equity, and loan size. New products also get a default document checklist. Rate is not an eligibility hard rule.

Lender create body: `{ name, country_code, description, website }`.

Case assignment body: `{ advisor_id }`. Target must be an active `ADVISOR`. Status body: `{ status, next_action_text }`. Admin may set any `application_status`. Overview also returns `unassigned_cases` and `ready_for_approval`.

Advisors handle day-to-day case work. Admin case ops are assignment, status (including terminal outcomes), advisor/homebuyer reports, and a later-expanding approvals queue (today: cases in `READY_FOR_SUBMISSION`).

## Advisor

ADVISOR role only. The queue is cases assigned to the signed-in advisor.

| Method | Path |
|--------|------|
| GET | `/advisor/cases` |
| GET | `/advisor/cases/:id` |
| PATCH | `/advisor/cases/:id/status` |
| PATCH | `/advisor/cases/:id/product` |
| POST | `/advisor/cases/:id/notes` |
| GET | `/advisor/cases/:id/suggestions` |
| POST | `/advisor/suggestions/:id/resolve` |
| GET | `/advisor/documents/:id/download-url` |
| POST | `/advisor/documents/:id/review` |

`GET /advisor/cases/:id` returns `{ case, notes, suggestions, documents, assessment }`. `documents` is the buyer’s checklist (same shape as `/documents/checklist` items). `assessment` is the linked eligibility assessment, or the buyer’s latest if `assessment_id` is unset.

Document review body: `{ decision, notes }`. Decisions: `accepted`, `rejected`, `requires_replacement`, `under_review`.

Advisor status is working-file only: `DOCUMENTS_PENDING`, `DOCUMENTS_UNDER_REVIEW`, `READY_FOR_SUBMISSION`, `SUBMITTED_TO_LENDER`, `LENDER_REVIEW`, `ADDITIONAL_INFORMATION_REQUIRED`. `APPROVED`, `REJECTED`, `COMPLETED`, and `CANCELLED` are admin.

Submitting to a lender requires a preferred product on the file. If that lender has a `LENDER_USER` account, the file appears in their portal. If they do not, the advisor records lender updates on the file (liaison).

Notes visibility: `internal` (advisor), `customer` (buyer journey), `lender` (lender portal + advisor).

## Lender

LENDER_USER role only. The user must be linked to a `lenders` row (`users.lender_id`).

| Method | Path |
|--------|------|
| GET | `/lender/me` |
| GET | `/lender/pipeline` |
| GET | `/lender/cases/:id` |
| PATCH | `/lender/cases/:id/status` |
| POST | `/lender/cases/:id/notes` |
| GET | `/lender/documents/:id/download-url` |

Pipeline is files whose preferred product belongs to that lender and status is `SUBMITTED_TO_LENDER`, `LENDER_REVIEW`, `ADDITIONAL_INFORMATION_REQUIRED`, or a recorded outcome. Lender status: `SUBMITTED_TO_LENDER`, `LENDER_REVIEW`, `ADDITIONAL_INFORMATION_REQUIRED` only.

## RBAC probes

- GET `/api/v1/admin/ping` — ADMIN
- GET `/api/v1/advisor/ping` — ADVISOR
- GET `/api/v1/lender/ping` — LENDER_USER
