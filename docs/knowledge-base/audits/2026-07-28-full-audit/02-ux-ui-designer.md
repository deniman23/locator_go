# UX/UI Audit — Locator Admin (MVP health-check)

## 1. Scope & method

**Scope:** admin web UI (`frontend/`): login → dashboard/map → checkpoints → visits → users/devices; map track UX; empty/error/loading; visual consistency; mobile vs desktop; implementability.

**Out of scope for this domain:** Android connector UI, backend correctness (except where it shapes admin journeys), production ops.

**Method (read-only):** CodeGraph + targeted reads of `frontend/src/{App.tsx,App.css,pages/*,components/*}`, `package.json`, `e2e/smoke.spec.ts`. KB (`docs/knowledge-base/INDEX.md`) пуст — предыдущих UX-отчётов нет.

**Primary personas:** оператор/админ (desktop): мониторинг треков, визиты, геозоны, здоровье устройств и QR.

**Success metric (MVP):** за ≤2–3 клика увидеть трек сотрудника за период; создать/править чекпоинт без ошибок координат; понять статус устройства и действие; не потерять данные на irreversible QR.

---

## 2. Findings (ranked)

### Blocker

_Нет абсолютных блокеров для desktop-админки при знакомом API-ключе: основные сценарии (карта, визиты→карта, пользователи/устройства) реализованы и связаны._

---

### High

#### H1. Карта на узких экранах практически непригодна
**Evidence:** `frontend/src/components/Map.tsx` — сайдбар всегда `className="map-sidebar open"`, без toggle/collapse. `App.css` `@media (max-width: 768px|480px)` только сужает ширину сайдбара (280px / 100% max 320px), не убирает его. `.map-container` — `height: calc(100vh - 250px)`. Навигация в `App.tsx` — горизонтальный `ul` без hamburger.
**UX impact:** на телефоне/планшете карта перекрыта фильтрами; nav и широкие таблицы визитов/пользователей требуют горизонтального скролла; admin «в поле» неработоспособен.
**Recommendation:** collapsible sidebar (по умолчанию закрыт &lt;768px) + FAB/кнопка «Фильтры»; sticky compact period bar; `Nav` → drawer; таблицы → card list на mobile. Токены: `--sidebar-width`, `--map-chrome-h`.

#### H2. Фильтры истории визитов — сырые ID, не имена
**Evidence:** `frontend/src/pages/UserVisits.tsx` — inputs `user_id`, `checkpoint_id`, `id` с placeholder «ID …», при этом `userMap`/`checkpointMap` уже загружаются для таблицы. Outside-сегменты требуют ID пользователя + период (copy в hint).
**UX impact:** ежедневный JTBD «визиты Иванова в школе» ломается на вспоминании числовых ID; ошибки фильтра → пустой список («Нет данных о визитах») без отличия «нет данных» vs «плохой фильтр».
**Recommendation:** `<select>` / combobox по имени (данные уже есть); опционально оставить advanced ID; empty state: «Нет визитов за период — сбросьте фильтры / проверьте пользователя».

#### H3. Создание/редактирование чекпоинта без map picker
**Evidence:** `CheckpointForm.tsx` / `CheckpointEditForm.tsx` — text inputs lat/lng/radius; список в `Checkpoints.tsx` — таблица без превью зоны. На карте зоны рисуются (`Map.tsx` `Circle` + Popup), но CRUD отвязан от карты.
**UX impact:** опечатка в координатах → неверная геозона → ложные визиты; радиус без визуальной обратной связи; нет delete в UI (только edit) — оператор не понимает lifecycle.
**Recommendation:** MVP: «Указать на карте» (клик → lat/lng + preview circle + drag radius). Валидация: lat ∈ [-90,90], lng ∈ [-180,180], radius &gt; 0 с inline error (сейчас NaN иногда в `.alert`, не в `.error-message` — `CheckpointForm.tsx` L44–46).

#### H4. Модалки QR / Device / regenerate — слабая a11y и закрытие
**Evidence:** `DeviceControlPanel.tsx` — `role="dialog" aria-modal="true"`, без Escape/focus trap/return focus. `QRCodeDisplay.tsx` — без `aria-modal`, без Escape. `UserManagement.tsx` regenerate — `window.confirm` + модалка с показом API-ключа один раз; Escape только у rename (`onKeyDown`).
**UX impact:** irreversible QR-regen и device commands легко «потерять» фокус/кликнуть мимо; screen reader / keyboard ops рискованны; accidental dismiss неясен.
**Recommendation:** единый `Modal` pattern: Escape, focus trap, backdrop click = cancel (не для показа ключа), `aria-labelledby`, первичный CTA справа; для regenerate — явный «Скопировать ключ» + checkbox «ключ сохранён».

#### H5. Design system заявлен, фактически — ad-hoc CSS + мёртвые map/UI deps
**Evidence:** `package.json` — `@mui/material`, `mapbox-gl`, `react-map-gl`, `@react-google-maps/api`; в `frontend/src` **нет** импортов MUI/Mapbox/Google. UI на custom `App.css` + Leaflet/`react-leaflet`. AGENTS.md описывает «React 19, MUI… Leaflet/Mapbox».
**UX impact:** нет единых токенов/компонентов → расхождение кнопок, таблиц, модалок; риск «ещё один стиль» при доработках; bundle bloat косвенно бьёт по perf карты.
**Recommendation:** зафиксировать решение: **Leaflet + custom tokens** (или реально внедрить MUI). Убрать неиспользуемые map libs из зависимости FE (координация с devops). Документировать tokens: color, space 4/8/16, radius, focus ring.

---

### Medium

#### M1. Две кнопки периода на карте путают модель обновления
**Evidence:** `Map.tsx` — «Применить интервал» (`applyPeriod`) и «Обновить маршрут» (`handleGetRoute`); обе тянут данные/fitBounds; road match отдельно по nonce.
**UX impact:** оператор не понимает, какую нажать после смены дат/пользователя; частичное обновление vs полный refetch.
**Recommendation:** одна primary «Показать трек»; secondary «Подогнать карту»; road-match как toggle с авто-refresh при одном user.

#### M2. Пустой трек на карте без empty-state
**Evidence:** при `routePoints.length === 0` карта OSM + чекпоинты; status bar считает «Точек маршрута: 0»; нет banner/CTA. Ошибки пользователей при `getAll` — только `console.error` (L148).
**UX impact:** «карта сломана» vs «нет GPS за период» неразличимы; сбой загрузки users молча оставляет пустой список.
**Recommendation:** overlay: «Нет GPS-точек за период» + «Расширить день / проверить устройство»; surface error загрузки users в sidebar.

#### M3. Default center карты — Москва, домен времени — Europe/Minsk
**Evidence:** `Map.tsx` `getSavedPosition` fallback `{ lat: 55.75, lng: 37.61, zoom: 10 }`; period utils — Minsk (`locationTrack.ts`, copy «Europe/Minsk» в visits).
**UX impact:** первый заход без `mapPosition` — чужой регион до fitBounds; когнитивный разрыв продукт/карта.
**Recommendation:** default ≈ Минск (напр. 53.9, 27.57) или fit по checkpoints сразу; убрать зависимость от `mapLoaded` localStorage для first-run fit.

#### M4. Навигация без активного раздела; 404 — голый текст
**Evidence:** `App.tsx` — `Link`, не `NavLink`; `path="*"` → `<h1>Страница не найдена</h1>` без layout/CTA.
**UX impact:** слабая ориентация в IA; deep-link опечатки ведут в тупик.
**Recommendation:** `NavLink` + `aria-current`; 404 с ссылкой на «Главная» внутри app chrome.

#### M5. Dashboard: активные визиты не связаны с картой
**Evidence:** `Dashboard.tsx` — список poll 30s; клика нет. Deep-link есть только из `UserVisits.openVisitOnMap`.
**UX impact:** «кто сейчас в зоне» → ручной поиск на карте.
**Recommendation:** клик по визиту → `/?user_id&from&to` (как visits) или highlight user на карте.

#### M6. Несогласованность loading/error/empty
**Evidence:** loading — текст «Загрузка…» / overlay (`App.tsx`); errors — `.error-message` или banner карты; часть catch только `console.error` (Dashboard checkpoints/users). `VisitsList.tsx` — отдельный компонент с сырым `user_id` и duration в секундах — **нигде не импортируется**; `ProtectedRoute.tsx` дублирует inline в `App.tsx`.
**UX impact:** разный UX при сбоях; мёртвый код путает FE при правках списков.
**Recommendation:** shared `PageState` (loading | error | empty | ready); удалить/использовать `VisitsList`; один `ProtectedRoute`.

#### M7. Focus: `outline: none` на inputs
**Evidence:** `App.css` L33–38 — `outline: none` + blue box-shadow; кнопки/ссылки без системного focus-visible паттерна.
**UX impact:** keyboard navigation слабее стандарта WCAG 2.4.7.
**Recommendation:** глобальный `:focus-visible` ring для interactive controls.

#### M8. Плотность User Management / device actions
**Evidence:** `UserManagement.tsx` — таблица с QR, regenerate, wake, enable location, OTA, device panel; notices через timeout; status poll.
**UX impact:** на desktop мощно (плюс), на mobile и для новичка — cognitive overload; destructive рядом с informative.
**Recommendation:** row primary «Устройство» → panel; secondary в overflow menu; цветовая иерархия healthy/warn/error (уже есть labels — усилить visual chips).

---

### Low

#### L1. Слабый product/brand chrome
**Evidence:** header `#2c3e50`, system font stack (`App.css`), нет логотипа/product name в hero nav — только ссылки.
**UX impact:** «ещё одна админка»; ок для internal MVP, плохо для handoff/trust.
**Recommendation:** wordmark «Locator» в header; не раздувать marketing.

#### L2. CSS-остатки (`.map-stays-*`) без UI
**Evidence:** `App.css` ~693 — stays empty styles; в текущем `Map.tsx` stays-секции нет.
**UX impact:** шум для FE; ложное ожидание feature.
**Recommendation:** удалить orphan CSS или вернуть stays UX осознанно.

#### L3. E2E покрывает только smoke body
**Evidence:** `frontend/e2e/smoke.spec.ts` — goto `/`, status &lt;500.
**UX impact:** регрессии map/visits/login не ловятся автоматически.
**Recommendation:** Playwright: login stub, map period apply, visits→map deep-link (с FE/QA).

---

### Info (положительное / контекст)

#### I1. Сильные паттерны уже есть
- Deep-link Visits → Map (`UserVisits.tsx` → query `user_id/from/to`) с защитой от race (`pendingDeepLinkUserIdRef`) — хороший cross-screen flow.
- Road match OSRM при одном сотруднике + GPS polyline + last-point emphasis — понятная модель трека.
- Device health: русские `ISSUE_LABELS`, async wait for health report, confirm на QR regenerate — business rules видимы.
- Auth: только admin (`AuthContext`) — продукт явно admin console; sessionStorage key; adopt key после self-regenerate.

#### I2. IA простая и адекватная MVP
4 раздела: Главная (карта+активные), Чекпоинты, Визиты, Пользователи (admin). One job в целом соблюдён, кроме перегруза Users.

#### I3. Leaflet — правильный выбор для текущего UX
Mapbox/Google в deps не используются; OSM tiles + Circle/Polyline/CircleMarker достаточны для MVP.

---

## 3. Prioritized remediation

| Priority | Item | Effort (MVP) | Outcome |
|----------|------|--------------|---------|
| P0 | H1 Mobile map: collapse sidebar + nav drawer | M | Admin usable on phone |
| P0 | H2 Visits filters: name selects + better empty | S | Faster daily visit lookup |
| P1 | H3 Checkpoint map picker + validation | M | Fewer bad geofences |
| P1 | H4 Shared modal a11y + copy key | S | Safer device/QR ops |
| P1 | H5 Lock design system (tokens / drop dead deps) | S–M | Consistent FE velocity |
| P2 | M1 Unify map apply CTA | S | Less map confusion |
| P2 | M2 Empty track overlay + user load errors | S | Trust in «no data» |
| P2 | M3 Default map center Minsk / fit | XS | First-run orientation |
| P2 | M5 Active visit → map | S | Monitoring loop closed |
| P3 | M4/M6/M7 NavLink, 404, PageState, focus-visible | S | Polish + a11y |
| P3 | L3 Playwright critical paths | M | Regression safety |

**MVP cut:** P0+P1 без полного MUI-migration; tokens в CSS variables поверх текущего `App.css`.

---

## 4. Acceptance checks

1. **Mobile map (&lt;768px):** сайдбар закрыт по умолчанию; карта ≥60% viewport; фильтры открываются/закрываются одной кнопкой; nav не горизонтально обрезается.
2. **Visits:** выбрать пользователя и чекпоинт по имени → таблица; empty при нуле результатов объясняет причину; «Маршрут на карте» открывает Главную с одним user и периодом визита.
3. **Checkpoint:** создать точку кликом по карте, увидеть preview radius, сохранить; edit показывает ту же preview; invalid radius блокирует submit с error, не success-alert.
4. **Modals:** QR/Device/regenerate закрываются Escape; Tab циклится внутри; после regenerate ключ копируется кнопкой; focus возвращается на trigger.
5. **Map empty:** период без точек → явный empty overlay, не «пустая Москва».
6. **Desktop regression:** multi-user colors, road match one-user, fitBounds, status bar counts — без регресса.
7. **A11y smoke:** keyboard login → nav → map period fields → apply; visible focus на controls.
8. **Design lock:** либо нет неиспользуемых map/UI deps в runtime bundle story, либо MUI реально в layout (документировано).

---

## 5. Assumptions / out of scope

- Primary use — **desktop admin**; mobile — secondary but must not be broken for field ops.
- Non-admin web UI не в scope продукта (login отвергает non-admin) — «роль Пользователь» в chrome фактически недостижима.
- Android app UX не аудировался.
- Визуальный редизайн «с нуля» / marketing landing — out of scope.
- Backend geofence/visit correctness — только как влияние на empty/false-negative UX.
- Live browser/Playwright exploratory session в этом прогоне не выполнялась — выводы по коду и структуре UI; runtime visuals (contrast на реальных tiles) желательно подтвердить FE/QA.

**Координация:** `frontend-developer` (sidebar, selects, modal, picker), `business-analyst` (нужен ли checkpoint delete / stays feature), `qa-tester` (acceptance выше), `security-engineer` (показ API-ключа в UI — уже есть; a11y не заменяет secret handling).