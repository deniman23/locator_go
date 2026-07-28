# Security Audit — Locator (AppSec, readonly)

**Дата:** 2026-07-28  
**Режим:** readonly health-check / threat-oriented (MVP depth)  
**Стек:** Go/Gin API, React admin, Android device connector  
**KB:** `docs/knowledge-base/INDEX.md` пуст — предыдущих security-находок нет  
**Метод:** CodeGraph (`codegraph_explore`) по auth/middleware/routes/device commands + readonly проверка compose/CI/Dockerfile/.gitignore (без вывода значений секретов)

---

## 1. Scope & method / threat model sketch

### Scope
- Authn/authz: `X-API-Key`, Basic vs Admin middleware, FE session
- Secrets в репозитории/образах
- IDOR / broken access control на location/users/device
- Device command abuse (config_update, app_update, wake)
- Публичные endpoints и static
- CORS / middleware gaps / rate limit
- PII (геолокация), supply chain / CI
- Docker network exposure, TLS

### Assets
| Asset | Почему критичен |
|-------|-----------------|
| API-ключи пользователей/админов | Полный доступ к API от имени субъекта |
| GPS/локации | PII / tracking |
| Device command queue | Удалённое управление устройством (PIN, URL, ключ, OTA) |
| DB / RabbitMQ credentials | Компрометация данных и шины событий |
| APK / OTA manifest | Supply-chain на устройствах |

### Trust boundaries
1. Internet → `:8080` backend, `:3000` frontend (HTTP)
2. Device connector → `/api/device/*` (API key любого пользователя)
3. Admin UI → admin API-key middleware
4. Backend → Postgres / RabbitMQ / OSRM
5. Public → `/healthz`, `/api/app/release/latest`, `/static/*`

### Top abuse cases
1. Украсть API-ключ через публичный QR PNG → impersonation / admin (если QR админа)
2. Скомпрометировать секреты из git/Dockerfile → DB takeover
3. Compromised admin key → произвольные device commands / OTA / config hijack
4. Auth DoS через O(n)×bcrypt на каждый запрос
5. MitM на HTTP → перехват ключей и координат

### Positive controls (кратко)
- API keys в БД через bcrypt; `json:"-"` на `ApiKey`
- Admin routes за `APIKeyAuthMiddleware`
- Device ack проверяет `cmd.UserID == currentUser`
- Location GET/POST: non-admin не может чужой `user_id`
- `allowedDeviceCommandTypes` whitelist
- `BuildConfigUpdatePayload` валидирует PIN/интервалы (для dedicated endpoint)
- Lockfiles есть (`go.sum`, `package-lock.json`)
- Postgres в compose на `127.0.0.1:5433`

---

## 2. Findings

### CRITICAL (≈ blocker)

#### SEC-C01 — Публичная раздача QR PNG с plaintext API-ключом
| | |
|--|--|
| **Severity** | critical |
| **Asset** | API keys / device & admin identity |
| **Evidence** | `backend/service/user_service.go` — `writeUserQRCode` пишет в QR JSON `user_id`, `api_key`, `api_base_url`; `backend/router/routes.go` — `router.Static("/static", "./static")` без auth; `ServeStaticQRCode` в `user_controller.go` не подключён к роутеру |
| **Exploitability** | Высокая: `GET /static/qrcode/{id}.png` → decode QR → полный ключ. Перебор id. **Живая проверка (повторный прогон):** `GET http://127.0.0.1:8080/static/qrcode/1.png` → **200 без auth**. |
| **Impact** | Impersonation пользователя; для admin-ключа — полный admin API + device commands |
| **Recommendation** | Убрать QR из публичного Static; отдавать только через auth (`/api/users/.../qr-code-file`). Не кодировать plaintext key в публично доступный файл; enrollment через одноразовый token. Перегенерировать все ключи после фикса. |
| **Owner** | BE + DevOps |

#### SEC-C02 — Секрет БД захардкожен в `backend/Dockerfile`
| | |
|--|--|
| **Severity** | critical |
| **Asset** | DB credentials |
| **Evidence** | `backend/Dockerfile` — `ENV` содержит `DB_PASSWORD=***REDACTED*** (значение **не** приводим) |
| **Exploitability** | Высокая: любой с доступом к репо/слоям образа |
| **Impact** | Доступ к Postgres, PII локаций, хеши ключей |
| **Recommendation** | Удалить секреты из Dockerfile; только runtime env/secrets. Ротация `DB_PASSWORD`. Проверить историю git/образов. |
| **Owner** | DevOps + BE |

#### SEC-C03 — `backend/.env` отслеживается git при наличии в `.gitignore`
| | |
|--|--|
| **Severity** | critical |
| **Asset** | DB password (non-placeholder), RabbitMQ, admin seed key names |
| **Evidence** | `git ls-files` → `backend/.env`; `.gitignore` содержит `backend/.env`, но файл уже в истории (`git log` показывает коммиты). Классификация: `DB_PASSWORD` = non-placeholder |
| **Exploitability** | Высокая при публичном/широком доступе к remote |
| **Impact** | Утечка prod/dev credentials; seed admin key если был не-placeholder |
| **Recommendation** | `git rm --cached backend/.env`; scrub history (BFG/filter-repo) если remote shared; ротация всех затронутых секретов; pre-commit secret scan |
| **Owner** | DevOps |

---

### HIGH

#### SEC-H01 — API и админка по HTTP без TLS; публичный IP в коде/доках
| | |
|--|--|
| **Severity** | high |
| **Evidence** | `docs/AGENT_PHONE_SETUP.md`, fallback `http://87.232.65.52:8080` в `device_controller.go` (wake/enable-location), scripts `publish_apk_release.sh` и др.; compose порты `8080:8080`, `3000:80` |
| **Exploitability** | Средняя–высокая на недоверенной сети |
| **Impact** | MitM: API keys, GPS, device reports |
| **Recommendation** | TLS termination (reverse proxy); убрать hardcoded IP; требовать `BASE_URL` https; HSTS |
| **Owner** | DevOps |

#### SEC-H02 — RabbitMQ AMQP и OSRM слушают все интерфейсы
| | |
|--|--|
| **Severity** | high |
| **Evidence** | `docker-compose.yml`: `5672:5672` (не `127.0.0.1`), `5000:5000`; Postgres уже `127.0.0.1:5433` |
| **Exploitability** | Высокая, если хост в интернете/LAN без FW |
| **Impact** | Inject/consume событий локаций; abuse routing |
| **Recommendation** | Bind `127.0.0.1:` или internal network only; сильные RMQ creds (не `guest`) |
| **Owner** | DevOps |

#### SEC-H03 — `AuthenticateUser`: GetAll + bcrypt на каждого пользователя
| | |
|--|--|
| **Severity** | high |
| **Evidence** | `backend/service/user_service.go` `AuthenticateUser` — `DAO.GetAll()` + цикл `bcrypt.CompareHashAndPassword` |
| **Exploitability** | Высокая (unauthenticated 401 path всё равно гоняет bcrypt×N) |
| **Impact** | CPU DoS; плохо масштабируется |
| **Recommendation** | Lookup по key-id/prefix или HMAC lookup table; rate limit на auth; не логировать шумно каждый fail |
| **Owner** | BE |

#### SEC-H04 — Raw admin commands обходят валидацию config_update
| | |
|--|--|
| **Severity** | high |
| **Evidence** | `PostAdminUserCommand` → `EnqueueCommand(userID, body.Type, body.Payload)` без `BuildConfigUpdatePayload`; dedicated `/device/config` валидирует. Payload может задать `api_base_url` / `api_key` / `admin_pin` произвольно |
| **Exploitability** | Требует admin key (но один скомпрометированный ключ = полный device hijack) |
| **Impact** | Перенаправить устройство на attacker API; сменить PIN; подменить credentials |
| **Recommendation** | Для `config_update`/`app_update` только через validated builders; raw endpoint запретить или жёстко schema-validate |
| **Owner** | BE |

#### SEC-H05 — Слабые дефолты compose/seed
| | |
|--|--|
| **Severity** | high |
| **Evidence** | `docker-compose.yml`: `DB_PASSWORD:-change_me`, `DEFAULT_ADMIN_API_KEY:-change_me`, `RABBITMQ_PASS:-guest`; `seed/seed.go` берёт `DEFAULT_ADMIN_*` из env |
| **Exploitability** | Высокая при деплое без `.env` |
| **Impact** | Тривиальный admin / DB / broker доступ |
| **Recommendation** | Fail-fast если дефолты; запрет `change_me`/`guest` в production (`GIN_MODE=release`) |
| **Owner** | DevOps + BE |

---

### MEDIUM

#### SEC-M01 — Нет rate limiting / lockout на API-key auth
| | |
|--|--|
| **Evidence** | `middleware/middleware.go` — только проверка ключа; CI без WAF/rate-limit |
| **Impact** | Усиливает SEC-H03; brute force (ограничен энтропией 32-byte key, но DoS реален) |
| **Recommendation** | Per-IP rate limit + slowdown на 401 |
| **Owner** | BE / DevOps |

#### SEC-M02 — API key в `sessionStorage` (XSS → session theft)
| | |
|--|--|
| **Evidence** | `frontend/src/context/AuthContext.tsx` — `sessionStorage.setItem('apiKey', …)` |
| **Impact** | При XSS — кража admin session (сейчас `dangerouslySetInnerHTML` не найден — риск ниже, но модель слабая) |
| **Recommendation** | HttpOnly cookie + CSRF; или short-lived session token вместо long-lived API key в JS |
| **Owner** | FE + BE |

#### SEC-M03 — OTA/APK и manifest публично доступны
| | |
|--|--|
| **Evidence** | `GET /api/app/release/latest` без auth; `static/releases/` через Static; sha256 в manifest |
| **Impact** | Разведка версий; если device не проверяет sha256 строго — supply chain (нужна проверка на клиенте) |
| **Recommendation** | Подтвердить обязательную проверку sha256 на Android; опционально подпись APK + pinned cert |
| **Owner** | BE + mobile |

#### SEC-M04 — Path traversal risk в `PostSyncReleaseManifest` (admin)
| | |
|--|--|
| **Evidence** | `filepath.Join(rc.ReleasesDir, body.Filename)` без `filepath.Base` / containment check |
| **Impact** | Admin читает произвольный файл как «APK» / ошибка path disclosure |
| **Recommendation** | `filepath.Base`, resolve + `strings.HasPrefix` внутри ReleasesDir |
| **Owner** | BE |

#### SEC-M05 — `api_base_url` валидация только `HasPrefix("http")`
| | |
|--|--|
| **Evidence** | `device_config_update.go` |
| **Impact** | Admin (или raw command) может задать http://attacker — device exfil |
| **Recommendation** | Allowlist host/scheme https; запрет IP literal в prod |
| **Owner** | BE |

#### SEC-M06 — Нет security headers (CSP, X-Frame-Options, HSTS)
| | |
|--|--|
| **Evidence** | `frontend/nginx.conf` — только Cache-Control; backend `gin.Default()` без security middleware |
| **Impact** | Clickjacking, ослабление XSS defense-in-depth |
| **Recommendation** | CSP, `X-Frame-Options: DENY`, `Referrer-Policy`, HSTS на TLS |
| **Owner** | FE / DevOps |

#### SEC-M07 — Контейнер backend без non-root USER
| | |
|--|--|
| **Evidence** | `backend/Dockerfile` final stage — нет `USER` |
| **Impact** | При RCE — root в контейнере |
| **Recommendation** | non-root user + drop caps |
| **Owner** | DevOps |

#### SEC-M08 — CI без security gates
| | |
|--|--|
| **Evidence** | `.github/workflows/tests.yml` — unit/integration/e2e; нет gosec/govulncheck/npm audit/trivy; `main.yml` — только echo deploy |
| **Impact** | Уязвимости deps/SAST не блокируют merge |
| **Recommendation** | govulncheck, `npm audit`, image scan, gitleaks |
| **Owner** | DevOps |

---

### LOW

#### SEC-L01 — Имя `BasicAuthMiddleware` вводит в заблуждение (это API-key, не HTTP Basic)
| **Evidence** | `middleware/middleware.go`, комментарии в `AGENTS.md` |
| **Recommendation** | Переименовать для ясности операций |

#### SEC-L02 — Verbose auth logging (ID/Name на успех/отказ)
| **Evidence** | middleware + `AuthenticateUser` log.Printf |
| **Recommendation** | Structured logs без лишней PII; correlation id |

#### SEC-L03 — `PublishEvent` позволяет admin публиковать произвольные LocationEvent
| **Evidence** | `event_controller.go`, route `/api/event/publish` |
| **Recommendation** | Ограничить поля / убрать ручной publish из prod API |

#### SEC-L04 — DB SSL mode default `disable`
| **Evidence** | `connection.go`, `.env.example` |
| **Recommendation** | `require`/`verify-full` если DB не на localhost network |

#### SEC-L05 — Regenerate QR возвращает plaintext `api_key` в JSON
| **Evidence** | `PostRegenerateUserQR` |
| **Note** | Ожидаемо для enrollment, но audit-логировать и не кэшировать |
| **Recommendation** | One-time reveal + audit trail |

---

### INFO

#### SEC-I01 — CORS middleware отсутствует
Для same-origin admin через nginx proxy это скорее плюс (нет `Access-Control-Allow-Origin: *`). Следить, чтобы не открыли CORS широко.

#### SEC-I02 — IDOR на location для non-admin закрыт
`GetLocation` / `PostLocation` проверяют `requestedUserID != currentUser.ID` без admin → 403. Admin-группа за admin middleware — ок для single-tenant admin model.

#### SEC-I03 — Device command ack проверяет владельца
`ErrDeviceCommandWrongUser` → 403. Хороший контроль.

#### SEC-I04 — Генерация ключей криптостойкая
`crypto/rand` 32 bytes + hex; bcrypt storage — хорошо (при условии SEC-C01 fix).

#### SEC-I05 — TRUSTED_PROXIES обрабатывается
`backend/main.go` — `SetTrustedProxies(nil)` по умолчанию (безопасно против IP spoof).

---

## 3. Prioritized remediation

### P0 (немедленно, blocker)
1. **SEC-C01** — закрыть `/static/qrcode/*`; ротация всех API keys; перевыпуск QR только под auth  
2. **SEC-C02 / SEC-C03** — убрать секреты из Dockerfile и git; ротация DB (и связанных) паролей; scrub history при необходимости  
3. **SEC-H05** — убедиться, что prod не на `change_me`/`guest`

### P1 (дни)
4. **SEC-H01** — TLS + убрать hardcoded IP fallbacks  
5. **SEC-H02** — bind RMQ/OSRM на localhost/internal  
6. **SEC-H03 + M01** — O(1) auth lookup + rate limit  
7. **SEC-H04 + M05** — валидация всех device config/OTA payloads  

### P2 (спринт)
8. Security headers, non-root container, path containment sync-manifest  
9. CI: govulncheck / npm audit / gitleaks / image scan  
10. Усиление session model (не long-lived key в JS)

---

## 4. Acceptance checks

- [ ] `GET /static/qrcode/1.png` → **401/404**, не PNG с декодируемым ключом  
- [ ] QR file только с валидным `X-API-Key` (admin/self)  
- [ ] `git ls-files` **не** содержит `backend/.env`; Dockerfile **без** `DB_PASSWORD=***REDACTED*** plaintext  
- [ ] После ротации: старые ключи/пароли не принимают auth / DB login  
- [ ] Compose: `5672` и `5000` не на `0.0.0.0` (или закрыты FW)  
- [ ] HTTPS на admin и device API; HTTP redirect/HSTS  
- [ ] `POST /api/admin/users/:id/commands` с `type=config_update` и `api_base_url=http://evil` → **400**  
- [ ] Нагрузочный тест: 100 invalid keys/s не кладёт CPU (rate limit / O(1) auth)  
- [ ] CI job: secret scan + govulncheck green на main  
- [ ] Заголовки: `X-Frame-Options` / CSP на admin UI  

---

## 5. Assumptions / out of scope

**Assumptions**
- Single-tenant: любой admin видит всех пользователей — by design  
- Android client проверяет sha256 OTA (не верифицировано в этом проходе глубоко)  
- Production ≈ хост из доков (`87.232.65.52`) без edge WAF  
- `.env` в корне gitignored и не tracked; риск именно `backend/.env` + Dockerfile  

**Out of scope**
- Активный пентест / эксплуатация / сканы портов prod  
- Полный Android APK reverse / root detection  
- Юридический DPIA / compliance certification  
- Исправление кода (readonly)  
- Запись в `docs/knowledge-base/` (делает оркестратор)  

**Residual risk after P0 only:** MitM без TLS, device hijack при утечке admin key, DoS auth, слабый session storage — остаются до P1/P2.

**Go / no-go (MVP health-check):** **NO-GO в prod** до закрытия SEC-C01–C03 и проверки, что дефолтные credentials не в проде.
