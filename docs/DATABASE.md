# Database

PostgreSQL schema lives in `backend/migrations/`.

## Apply

Migrations run automatically on API startup via golang-migrate.

Manual:

```bash
cd backend
migrate -path migrations -database "$DATABASE_URL" up
```

## Core entities

users, user_profiles, employment_profiles, financial_profiles, auth_tokens, lenders, mortgage_products, mortgage_rules, mortgage_product_documents, document_types, eligibility_assessments, eligibility_results, readiness_scores, salary_account_analyses, documents, document_reviews, mortgage_applications, application_events, advisor_assignments, advisor_notes, concierge_suggestions, notifications, content_items, legal_disclaimers, platform_settings, audit_logs, analytics_events

## Seed

```bash
go run ./cmd/seed
```

Seeds document types, platform settings, disclaimers, admin/advisor/demo users, and Nigeria mortgage products (NHF, Stanbic MREIF, commercial indicative).
