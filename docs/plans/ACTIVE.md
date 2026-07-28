# ACTIVE — текущий фокус

**Updated (UTC):** 2026-07-28T12:15:00Z

## Now

- **P0 backend LIVE** (recreated) — QR public blocked; auth cache on; both devices back.
- Sec checklist: approach OK; wait for controlled **backend-only** recreate.
- См. [`../knowledge-base/audits/2026-07-28-full-audit/99-consolidated-report.md`](../knowledge-base/audits/2026-07-28-full-audit/99-consolidated-report.md)

### MUST NOT while devices online
- No API key rotation / RegenerateUserQR for live devices
- No delete `static/qrcode/*.png`
- No compose down / db wipe / full-stack recreate
- No commit of `.env` / secrets

### Post-deploy verified 2026-07-28T12:23:21Z
- healthz 200; static QR 403; releases 200; both devices poll/location OK

### After backend-only restart — verify
- `GET /static/qrcode/N.png` → 403/404 (not 200)
- `GET /api/users/:id/qr-code-file` + admin key → 200
- Device poll/location → 200; both devices back online
- `/healthz` ok

## Next

1. Ask user → rebuild/recreate **backend only** (short HTTP blip)
2. Later window: recreate rabbitmq for `127.0.0.1:5672` bind
3. Coordinated key rotation + git history scrub (deferred)

## Blocked / wait

- ~~User OK for backend recreate~~ done

### Reviewer APPROVE (residual)
- No critical blockers for backend-only recreate
- Residual: change_me compose default, BASE_URL localhost fallback if unset, bcrypt DoS on miss

### Analyst GO gates (2026-07-28)
- Backend-only: `docker compose up -d --build --no-deps backend`
- Preflight: real `DB_PASSWORD` + production `BASE_URL` in env
- Post: `/healthz` 200; `/static/qrcode/N.png` 403; auth QR 200; both devices poll again
- Same window NO-GO: rabbitmq recreate, key rotation, QR wipe

---

## Deploy caution — P0 secrets / bind changes (added 2026-07-28)

| Change | File | Takes effect after |
|--------|------|--------------------|
| Hardcoded `DB_PASSWORD` removed from image | `backend/Dockerfile` | Next `docker compose build backend` + controlled recreate |
| Public `/static/qrcode` blocked + auth cache | `routes.go`, `user_service.go` | Backend rebuild/recreate |
| RabbitMQ bound to `127.0.0.1:5672` | `docker-compose.yml` | rabbitmq recreate (maintenance window) |
| `backend/.env` removed from git index | git | next commit (file stays on disk) |

1. Confirm `DB_PASSWORD` set in `.env` — backend refuses to start without it.
2. Rebuild backend only; do NOT recreate `db`.
3. Key rotation deferred while devices online.
