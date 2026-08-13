# Deployment

No Docker/Kubernetes for MVP.

## Suggested path

1. Managed PostgreSQL (Neon/Supabase/RDS)
2. Managed Redis (Upstash/ElastiCache)
3. S3-compatible storage (R2/S3) for documents
4. API on a VM or Fly/Render/Railway Go service
5. Frontend on Vercel or same host
6. n8n on a small VPS for notification webhooks
7. Claude API key in secrets store

## Checklist

- Rotate `SESSION_SECRET`
- Set `APP_ENV=production`, HTTPS, secure cookies
- Configure real mailer (replace log mailer)
- Verify mortgage product sources before marking `verified`
- Legal review of disclaimers
- Rate limiting and malware scanning before public document upload
