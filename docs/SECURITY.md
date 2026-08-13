# Security

- Passwords: bcrypt
- Sessions: opaque IDs in Redis; httpOnly cookie
- RBAC: CUSTOMER / ADVISOR / ADMIN / LENDER_USER
- Documents (upcoming): private bucket, signed URLs, mime/size limits, audit trail
- Never log passwords, tokens, statement contents, or account numbers
- Secrets via environment variables
- Eligibility language never claims bank approval
- Legal disclaimer copy is admin-editable; requires legal review before production
