# SNAPSHOT — locator_go

**Captured (UTC):** 2026-07-28T11:59:44Z  
**Source:** init-project-memory + `docs/knowledge-base/audits/2026-07-28-full-audit/`

## Health (from last full audit)

| Domain | Status |
|--------|--------|
| BA / UX / FE | 🟡 |
| Backend / QA / DevOps / Security | 🔴 |
| Overall | 🔴 Production NO-GO until P0 AppSec/delivery |

## Top risks (no secrets)

- Public QR / static key exposure
- Secrets in image/git history risk
- Auth O(N)×bcrypt
- Deploy without CI gate; RMQ exposure; no TLS

## Memory bootstrap

- `docs/kb`, `docs/plans`, `docs/stats` created
- Existing audit KB retained (append-only)
