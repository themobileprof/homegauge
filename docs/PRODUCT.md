# HomeGauge — Product

## Positioning

HomeGauge helps people understand mortgage options in their market, estimate affordability, assess likely eligibility (salary-account based), prepare documents, and get guided help through the application process.

**Not a bank. Not a lender. Not a property marketplace (V2).**

Tagline direction: *Understand your mortgage. Know what you may qualify for. Get help getting approved.*

Primary CTA: **Check My Mortgage Eligibility**  
Secondary CTA: **Compare Mortgage Options**

## Locked MVP decisions

| Area | Decision |
|------|----------|
| Brand | HomeGauge |
| Markets | Multi-country ready; **Nigeria active** first; other countries via `countries` + product seeds (`coming_soon` stubs for GH/KE) |
| Underwriting basis | Salary account only: recognizable recurring salary credit for **6 consecutive months** |
| Employment in eligibility engine | Salaried / civil servant (salary credit). Others: learn + calculator only |
| Salary variance tolerance | ±15% of median credit (admin-configurable) |
| Payday window | Last 7 calendar days of each month (admin-configurable) |
| Accounts | Single salary domicile account for automated review |
| Currency | Per-country (`countries.currency_code`); UI formats from selected market |
| Default ITI guardrail | 35% unless product/country rule overrides |
| Interest estimate | Reducing-balance monthly; always labelled estimate |
| AI runtime | Anthropic Claude primary; Gemini + DeepSeek optional for multi-model advisor review |
| Workflow orchestration | **n8n** for email/escalation webhooks (not core underwriting) |
| Automation level | `suggest_only` at launch; `auto_safe` admin-toggleable |
| Customer AI chat | Not in MVP |
| Infra | No Docker; native local + managed cloud services |
| Auth | Email/password, verification, reset; cookie sessions |
| Roles | CUSTOMER, ADVISOR, ADMIN, LENDER_USER |
| Seed products (NG) | NHF (scheme), Stanbic MREIF (lender-published), commercial indicative (`needs_verification`) |
| Approval language | Never claim bank approval from automated assessment |

## Admin-configurable (CRUD)

Lenders, mortgage products, eligibility rules, document requirements, readiness weights, salary-detection settings, disclaimers/legal copy, educational content, automation level, advisor assignment, verification status / last verified dates.

## Regulatory stance

Automated outcomes use bands: Likely eligible / Potentially eligible / May require additional review / Unlikely to qualify / More information required.

All public legal text is admin-editable and **requires legal review before production launch**.
