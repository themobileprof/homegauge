# HomeGauge — Architecture

## Shape

Modular monolith:

- **Frontend:** Next.js (TypeScript, Tailwind) — `frontend/`
- **API:** Go + Gin — `backend/`
- **DB:** PostgreSQL
- **Cache/sessions:** Redis
- **Files:** S3-compatible private bucket (signed URLs only)
- **AI concierge:** Claude via API (structured JSON out)
- **Automation edge:** n8n webhooks for notifications/escalations

No Docker, no Kubernetes, no microservices for MVP.

## Backend modules

`auth` · `users` · `lenders` · `mortgages` · `eligibility` · `calculator` · `readiness` · `documents` · `applications` · `advisors` · `concierge` (AI) · `notifications` · `content` · `admin` · `audit` · `analytics`

## Request flow (eligibility)

1. Customer completes assessment + uploads 6-month salary statement  
2. Rules engine evaluates product rules (data-driven)  
3. Claude extracts salary credits → `salary_account_analyses`  
4. Combined outcome + readiness score persisted  
5. Concierge proposes next actions (`suggest_only` / `auto_safe`)  
6. n8n notified for email / advisor Slack (optional)

## Security highlights

RBAC on every sensitive route · private object storage · short-lived signed URLs · audit log · no secrets/PII in application logs · rate limits on auth/upload
