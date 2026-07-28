# Аудит Frontend — Locator (React 19 + Vite + Leaflet)

**Дата:** 2026-07-28  
**Аудитор:** frontend-developer subagent  
**Коммит-состояние:** чистая сборка (`tsc -b && vite build` — 0 ошибок), lint — 0 ошибок (1 warning), тесты 16/16 passed

---

## 1. Scope & Method

**Охват:**
- `frontend/src/` — все 27 файлов исходного кода (компоненты, страницы, сервисы, утилиты, контекст)
- `frontend/package.json`, `tsconfig.app.json`, `eslint.config.js`, `vite.config.ts`
- `frontend/nginx.conf`, `frontend/Dockerfile`
- `frontend/e2e/smoke.spec.ts`

**Метод:**
1. Полное чтение всех source-файлов через Read
2. Запуск `npm run lint`, `npm run test:run`, `npm run build` — верификация живого состояния
3. Статический поиск паттернов (`sessionStorage`, `aria-`, `setTimeout`, `window.confirm`, импорты зависимостей)
4. Анализ бандла (490 KB unminified JS, gzip 153 KB)
5. Кросс-проверка API-контрактов `services/api.ts` с `types/models.ts`

**Фокус:** архитектура FE, роутинг, API-клиент, безопасность токена, доступность (a11y), производительность (карты/списки), управление состоянием, обработка ошибок, качество TypeScript, готовность к сборке.

---

## 2. Findings

### 🔴 HIGH

---

#### H-1 — API-ключ отображается открытым текстом в DOM после перегенерации QR

**Файл:** `src/components/UserManagement.tsx:529`

```tsx
<code className="regenerate-api-key">{regenerateResult.apiKey}</code>
```

**Описание:** После вызова `userApi.regenerateQR()` новый `api_key` сохраняется в стейте `regenerateResult` и рендерится в `<code>` без маскировки. Он виден в DOM-инспекторе браузера, пишется в React DevTools и потенциально попадает в логи расширений/мониторинга.

**Impact:** Секрет-уровня учётных данных (API-ключ — единственный фактор авторизации) доступен в plaintext в браузере дольше, чем необходимо. При наличии злоумышленника с доступом к скриншоту/расширению/session recording — полная компрометация аккаунта целевого пользователя.

**Рекомендация:**
- Показывать ключ через `<input type="password" readOnly value={...} />` с кнопкой «Показать/скрыть»
- Добавить кнопку «Скопировать» (`navigator.clipboard.writeText`) и авто-скрытие через 30–60 секунд
- Не хранить ключ в state дольше необходимого (сразу очищать после копирования)

---

### 🟠 MEDIUM

---

#### M-1 — Три неиспользуемых библиотеки карт в production-зависимостях (~280 KB gzip)

**Файл:** `frontend/package.json` (dependencies)

```
mapbox-gl       ^3.13.0   — не импортируется ни в одном src/*.tsx
react-map-gl    ^8.0.4    — не импортируется
@react-google-maps/api  ^2.20.6 — не импортируется
```

**Верификация:** `grep -rn "import.*mapbox\|react-map-gl\|@react-google" src/` — 0 совпадений в production-коде.

**Impact:** Бандл JS: 490 KB unmin / 153 KB gzip. Mapbox GL + react-map-gl + Google Maps API суммарно добавляют ~250–350 KB gzip к потенциальному размеру, если Vite не tree-shake их полностью (mapbox-gl имеет side-effects и плохо tree-shaking'ится). Кроме того, пакеты обновляются, расширяя attack surface. `@types/leaflet` числится в `dependencies` (должен быть в `devDependencies`).

**Рекомендация:**
- `npm uninstall mapbox-gl react-map-gl @react-google-maps/api`
- Переместить `@types/leaflet` в `devDependencies`
- Добавить `npm run build -- --analyze` (vite-bundle-analyzer) в CI для контроля роста бандла

---

#### M-2 — Дублирование fetch-логики чекпоинтов/пользователей в трёх местах без общего кэша

**Файлы:**
- `src/pages/Dashboard.tsx:24–42` — `checkpointApi.getAll` + `userApi.getAll`
- `src/pages/UserVisits.tsx:51–82` — то же самое
- `src/components/VisitsList.tsx:27–40` — `checkpointApi.getAll`
- `src/components/Map.tsx:129–148` — `userApi.getAll`

**Описание:** При одновременном открытии Dashboard и переходе на UserVisits происходит 4+ одинаковых запроса к `/api/checkpoint/` и `/api/users/`. Нет никакого кэширования (ни React Query, ни Context с кэшем).

**Impact:** Лишняя нагрузка на backend при N пользователях; видимые waterfall-задержки при каждой навигации; `VisitsList.tsx` является дублирующим компонентом (уже есть полноценный `UserVisits.tsx` с теми же данными).

**Рекомендация:**
- Вынести `checkpointMap` и `userMap` в Context или использовать React Query (`useQuery` с `staleTime: 60_000`)
- Рассмотреть удаление `VisitsList.tsx` (компонент не используется на роутах, только его логика дублирует `UserVisits`)
- Либо минимально: закэшировать результаты в `module-level` мемо при монтировании AuthProvider

---

#### M-3 — Отсутствует ErrorBoundary — необработанная ошибка в Map/UserManagement рушит всё приложение

**Файл:** `src/App.tsx` — нет ни одного `<ErrorBoundary>`

**Описание:** `Map.tsx` содержит сложную async-логику (OSRM-запросы, Leaflet-инициализация), `UserManagement.tsx` — fire-and-forget `window.setTimeout(async () => { ... })` (строка 317) без try/catch вокруг внешнего `setTimeout`. Любая необработанная ошибка в render-фазе убьёт всё дерево.

**Impact:** Белый экран для всего приложения при ошибке на странице карты или управления пользователями. Нет возможности восстановления без перезагрузки.

**Рекомендация:**
```tsx
// src/components/AppErrorBoundary.tsx
class AppErrorBoundary extends React.Component<...> {
  componentDidCatch(error, info) { /* Sentry.captureException */ }
  render() { return this.state.hasError ? <ErrorFallback /> : this.props.children }
}
// В App.tsx обернуть каждый Route или хотя бы <main>
```

---

#### M-4 — `window.setTimeout(async () => {...})` без cleanup в UserManagement — потенциальная утечка состояния

**Файл:** `src/components/UserManagement.tsx:317`

```tsx
window.setTimeout(async () => {
    const fresh = await waitForFreshHealthReport(userId, apiKey, baseline, { attempts: 15, intervalMs: 2000 });
    if (fresh) {
        applyUserStatus(userId, fresh);
        setNotice(userId, 'Устройство ответило...');
```

**Описание:** `waitForFreshHealthReport` может выполняться до 30 секунд (15 попыток × 2000 мс). Если пользователь уходит со страницы или компонент размонтируется за это время, `setNotice`/`applyUserStatus` вызовутся на уже размонтированном компоненте. React 18 не крашится от этого, но стейт применяется к мёртвому компоненту + возможны утечки в pollRequest-loop.

**Рекомендация:**
- Использовать `useRef` с флагом `mounted` или `AbortController`/cancellation token
- Аналогичный паттерн применить к `handleWakeDevice` и `handleEnableLocation`

---

#### M-5 — Нет глобального перехватчика 401/403 в Axios — сессия не сбрасывается при протухшем ключе

**Файл:** `src/services/api.ts:4–9`

```ts
const api = axios.create({
    baseURL: '/api',
    headers: { 'Content-Type': 'application/json' }
});
```

**Описание:** Axios-инстанс не имеет `interceptors.response`. Если API-ключ истечёт или будет отозван в runtime (например, после `regenerateQR` другого пользователя), все запросы начнут падать с 401, но UI продолжит показывать защищённые страницы. Только `refreshUser` в `AuthContext` реагирует на ошибку — но он вызывается только при монтировании навигации.

**Impact:** Пользователь видит бесконечные ошибки в консоли без понятного сообщения; нет автоматического redirect на `/login`.

**Рекомендация:**
```ts
api.interceptors.response.use(
  (r) => r,
  (err) => {
    if (err.response?.status === 401) {
      // вызвать logout из AuthContext через exported setter или event
    }
    return Promise.reject(err);
  }
);
```
Или использовать callback, зарегистрированный при инициализации AuthProvider.

---

#### M-6 — `handleResetFilters` в UserVisits использует `setTimeout(fetchVisits, 0)` для обхода stale closure

**Файл:** `src/pages/UserVisits.tsx:165–176`

```ts
const handleResetFilters = () => {
    const b = defaultDayRange();
    setFilters({ id: '', user_id: '', checkpoint_id: '', from: b.from, to: b.to, showOutside: false });
    setTimeout(fetchVisits, 0); // ← stale closure: fetchVisits захватывает старые фильтры
};
```

**Описание:** `setTimeout` используется как хак для того, чтобы `fetchVisits` (через `useCallback` с `[apiKey, filters]`) получил обновлённое состояние. Это ненадёжно: `fetchVisits` в closure по-прежнему захватит старые `filters` к моменту исполнения таймера. Правильное поведение срабатывает лишь случайно из-за того, что `useEffect([fetchVisits])` уже подхватит новый callback после ре-рендера.

**Impact:** Потенциальный баг при медленном рендере: "Сбросить фильтры" вернёт данные по старым фильтрам.

**Рекомендация:** Вынести reset в отдельный `useEffect` или передавать новые фильтры напрямую в функцию fetch:
```ts
const handleResetFilters = useCallback(() => {
    const b = defaultDayRange();
    const next = { id: '', user_id: '', checkpoint_id: '', from: b.from, to: b.to, showOutside: false };
    setFilters(next);
    void fetchVisitsWithFilters(next); // принимает filters как аргумент
}, [apiKey]);
```

---

#### M-7 — Дублирование компонента `ProtectedRoute` — определён дважды

**Файлы:**
- `src/App.tsx:12–24` — inline `ProtectedRoute` (используется)
- `src/components/ProtectedRoute.tsx` — отдельный файл с идентичной логикой (не импортируется нигде)

**Описание:** `ProtectedRoute.tsx` существует как отдельный файл, но `App.tsx` объявляет собственную inline-версию и использует только её. Файл `ProtectedRoute.tsx` — мёртвый код.

**Impact:** Путаница при рефакторинге; потенциальное расхождение логик при будущих изменениях.

**Рекомендация:** Удалить `src/components/ProtectedRoute.tsx`, либо наоборот — убрать inline-объявление из `App.tsx` и импортировать из компонента.

---

### 🟡 LOW

---

#### L-1 — A11y: модальные окна QRCodeDisplay и regenerateResult не имеют `role="dialog"` + `aria-labelledby`

**Файлы:**
- `src/components/QRCodeDisplay.tsx:79` — `<div className="qr-code-modal">` без `role`
- `src/components/UserManagement.tsx:515` — `<div className="qr-code-modal">` без `role`

В сравнении: `DeviceControlPanel.tsx:259` — единственный модал с `role="dialog" aria-modal="true"`.

**Impact:** Скринридеры не анонсируют открытие модала; фокус не перехватывается; клавиатурный escape не стандартизован.

**Рекомендация:** Добавить `role="dialog"`, `aria-modal="true"`, `aria-labelledby` на все `.qr-code-modal`. Реализовать focus trap (можно через `@mui/base/FocusTrap` или `focus-trap-react`).

---

#### L-2 — A11y: кнопки в таблице UserManagement без достаточного text/aria-label

**Файл:** `src/components/UserManagement.tsx:773–836`

```tsx
<button className="qr-code-button-small" onClick={...}>QR-код</button>
<button ... onClick={...}>GPS</button>
```

Кнопка «GPS» без `aria-label` не информативна для скринридера: непонятно, GPS какого пользователя.

**Рекомендация:** `aria-label={`Запросить координаты: ${user.name}`}` на каждой action-кнопке в строке таблицы.

---

#### L-3 — `CheckpointForm` и `CheckpointEditForm` используют `type="text"` для числовых полей (lat/lng/radius)

**Файлы:** `src/components/Checkpoint/CheckpointForm.tsx:97,107,119`, `CheckpointEditForm.tsx:102,112,122`

**Описание:** Широта, долгота и радиус принимаются как `type="text"`. Валидация выполняется только через `isNaN(parseFloat(...))` после сабмита. Нет `inputMode="decimal"`, нет `min`/`max` ограничений, нет inline-ошибок.

**Impact:** На мобильных устройствах нет числовой клавиатуры; возможно вводить невалидные координаты (например, lat > 90); ошибка отображается только после сабмита.

**Рекомендация:** `type="number" inputMode="decimal" step="any" min="-90" max="90"` для lat; `min="-180" max="180"` для lng; `min="1"` для radius. Добавить `pattern` для locale-независимого десятичного разделителя.

---

#### L-4 — Полинг Dashboard не очищается при размонтировании в strict-режиме

**Файл:** `src/pages/Dashboard.tsx:68–73`

```ts
const interval = setInterval(poll, POLL_MS);
return () => clearInterval(interval);
```

Cleanup корректен, но первый вызов `void fetchActiveVisits()` на строке 67 не отменяется. В React 18 Strict Mode двойной mount/unmount вызовет два параллельных fetch. Хотя функционально безвреден, это создаёт лишние запросы в dev-режиме.

**Рекомендация:** Добавить `AbortController` / generation counter (паттерн уже применён в `Map.tsx:154–118` — скопировать оттуда).

---

#### L-5 — `getSavedPosition()` в `Map.tsx` вызывается во время render, а не в `useMemo`/`useRef`

**Файл:** `src/components/Map.tsx:326–337`

```ts
const getSavedPosition = () => { ... }; // определён внутри компонента
const initialPosition = getSavedPosition(); // вызывается при каждом рендере
```

**Описание:** `localStorage.getItem('mapPosition')` вызывается на каждый рендер MapComponent. Это synchronous I/O в render-пути.

**Рекомендация:** `const initialPosition = useMemo(() => getSavedPosition(), [])` или вынести в `useRef` с инициализацией.

---

#### L-6 — `VisitsList.tsx` — мёртвый компонент, не подключён к роутингу

**Файл:** `src/components/VisitsList.tsx`

**Описание:** Компонент экспортируется, но не импортируется ни в одном активном роуте или компоненте (проверено grep по всему `src/`). Содержит дублирующую логику загрузки чекпоинтов и уступает по функциональности `UserVisits.tsx`.

**Рекомендация:** Удалить файл либо задокументировать планируемое использование.

---

#### L-7 — Nginx: отсутствуют security-headers (CSP, X-Frame-Options, X-Content-Type-Options)

**Файл:** `frontend/nginx.conf`

**Описание:** `nginx.conf` не содержит:
- `Content-Security-Policy`
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy`
- `Permissions-Policy`

**Impact:** Без CSP возможны XSS-атаки через инжектированный скрипт; без `X-Frame-Options` — clickjacking.

**Рекомендация:**
```nginx
add_header X-Frame-Options "DENY" always;
add_header X-Content-Type-Options "nosniff" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header Content-Security-Policy "default-src 'self'; img-src 'self' data: https://*.tile.openstreetmap.org; connect-src 'self'" always;
```

---

### ℹ️ INFO

---

#### I-1 — MUI установлен, но не используется ни в одном компоненте

**Файл:** `frontend/package.json` — `@mui/material ^7.1.1`, `@emotion/react`, `@emotion/styled`

**Описание:** В `src/` нет ни одного `import ... from '@mui/material'`. Весь UI построен на нативном HTML + custom CSS. MUI добавляет ~50 KB gzip потенциальной зависимости (при импорте).

**Impact:** Сейчас tree-shaking не включает MUI в бандл (подтверждено: бандл 490 KB без MUI). Но наличие пакета вводит в заблуждение разработчиков и расширяет потенциальный attack surface при обновлениях.

**Рекомендация:** Либо начать использовать MUI (перевести компоненты на MUI-примитивы), либо удалить из зависимостей.

---

#### I-2 — E2E-тесты: единственный smoke-тест проверяет только HTTP-статус

**Файл:** `frontend/e2e/smoke.spec.ts`

**Описание:** Тест проверяет лишь `res.status() < 500`. Нет ни одного сценария: логин, навигация по Dashboard, отображение карты, создание чекпоинта.

**Рекомендация:** Добавить минимум:
- `test('login flow')` — ввод API-ключа, редирект на Dashboard
- `test('checkpoint CRUD')` — создание/отображение чекпоинта
- `test('map loads')` — `expect(page.locator('.leaflet-container')).toBeVisible()`

---

#### I-3 — `AuthContext` экспортирует и компонент (`AuthProvider`), и хук (`useAuth`) из одного файла — предупреждение lint

**Файл:** `src/context/AuthContext.tsx:17`

```
warning  react-refresh/only-export-components
```

**Описание:** ESLint-предупреждение указывает на нарушение Fast Refresh: файл экспортирует и non-component (`useAuth`), и компонент. В dev-режиме изменение файла вызывает полный ремонт вместо hot-update.

**Рекомендация:** Вынести `useAuth` и типы в отдельный файл `src/context/useAuth.ts`, оставив в `AuthContext.tsx` только `AuthProvider`.

---

## 3. Prioritized Remediation

| Приоритет | ID | Действие | Усилие |
|---|---|---|---|
| 1 | H-1 | Маскировать API-ключ в regenerateResult-модале | S (1–2 ч) |
| 2 | M-1 | Удалить mapbox-gl / react-map-gl / @react-google-maps/api, @types/leaflet → devDeps | XS (30 мин) |
| 3 | M-5 | Добавить Axios interceptor 401→logout | S (1–2 ч) |
| 4 | M-3 | Добавить ErrorBoundary вокруг `<main>` | S (1–2 ч) |
| 5 | M-4 | Добавить cancellation flag в fire-and-forget setTimeout | S (2–3 ч) |
| 6 | L-7 | Добавить security headers в nginx.conf | XS (30 мин) |
| 7 | M-7 | Удалить дублирующий ProtectedRoute.tsx | XS (15 мин) |
| 8 | L-6 | Удалить или подключить VisitsList.tsx | XS (15 мин) |
| 9 | M-2 | Вынести checkpointMap/userMap в Context или React Query | M (4–6 ч) |
| 10 | M-6 | Починить handleResetFilters (stale closure) | S (1 ч) |
| 11 | I-3 | Разделить AuthContext на AuthProvider + useAuth | XS (30 мин) |
| 12 | L-1/L-2 | A11y: role/aria для модалов и кнопок таблицы | S (2–3 ч) |
| 13 | L-3 | Числовые поля чекпоинтов → type="number" + валидация | S (1–2 ч) |
| 14 | L-4/L-5 | AbortController в Dashboard polling; useMemo для initialPosition | XS (1 ч) |
| 15 | I-1 | Решить судьбу MUI (использовать или удалить) | — |
| 16 | I-2 | Расширить E2E smoke → login + checkpoint + map | M (3–5 ч) |

---

## 4. Acceptance Checks

После применения исправлений верифицировать:

1. **H-1:** `document.querySelector('.regenerate-api-key')` — элемент должен быть `input[type=password]`, значение не видно в скриншоте.
2. **M-1:** `npm run build` — размер JS-бандла уменьшился минимум на 50 KB gzip. `mapbox-gl` отсутствует в `node_modules/.package-lock.json`.
3. **M-5:** Имитировать 401 через DevTools → Network → Block request; приложение должно автоматически перенаправить на `/login`.
4. **M-3:** Бросить ошибку в `Map.tsx` render → ErrorBoundary должен показать fallback вместо белого экрана.
5. **M-4:** Навигация со страницы пользователей во время активного `waitForFreshHealthReport` → нет React warning «Can't perform a React state update on an unmounted component».
6. **L-7:** `curl -I http://localhost:3000` → ответ содержит `X-Frame-Options`, `X-Content-Type-Options`, `Content-Security-Policy`.
7. **M-7 / L-6:** `find src/ -name "ProtectedRoute.tsx" -o -name "VisitsList.tsx"` — файлы удалены (или VisitsList.tsx подключён к роуту).
8. **I-3:** `npm run lint` — 0 warnings.
9. **I-2:** `npm run test:e2e` → минимум 3 сценария passed.
10. **Общий:** `npm run build` — 0 ошибок TypeScript; `npm run test:run` — все тесты зелёные.

---

## 5. Assumptions / Out of Scope

**Допущения:**
- Анализ проводился статически; backend-эндпоинты не вызывались — контракты API проверены через сопоставление `types/models.ts` с `services/api.ts`.
- `mapbox-gl`/`react-map-gl` считаются неиспользуемыми на основании отсутствия импортов в `src/`; если они планируются — удалять не нужно.
- Оценки производительности бандла основаны на размерах gzip из `vite build` (153 KB) — профилирование в браузере (LCP, FID) не проводилось.
- Авторизация основана исключительно на `X-API-Key`; анализ не рассматривал переход на OAuth/JWT как in-scope.

**Out of scope:**
- Android-клиент (connector)
- Backend Go-сервисы
- Качество самих API-эндпоинтов (latency, pagination)
- Lighthouse/WebPageTest CWV-метрики в production
- HTTPS-конфигурация (upstream reverse proxy)
- Storybook / design system (не существует)


---

## Дополнение — повторный прогон ([FE](438f165c-80f3-4ed1-b017-6902ff1eb072))

Уникальные / усиленные пункты относительно основного FE-отчёта:

| Sev | Finding |
|-----|---------|
| medium | Нет **axios `timeout`** на инстансе `api` — зависшие запросы без abort. |
| medium | Возможный баг `if (params.id)` в `visitApi.getWithFilters` при `id === 0` (falsy). |
| medium | Нет **кластеризации** маркеров на карте — риск деградации при >~500 точек. |
| medium | Dashboard/UserVisits: справочники грузятся **последовательно**, нет `Promise.all`. |
| medium | `shouldFitBounds` может не сбрасываться после auto-fit (лишние fitBounds). |
| info | Повторно подтверждены: мёртвые map deps, ErrorBoundary, 401 interceptor, sessionStorage key, CSP/security headers; tsc/eslint/16 unit — green. |

Severity note повторного прогона: dead map deps помечены как blocker; в основном отчёте — medium (M-1) при том, что Vite tree-shake может не включать их в бандл, если нет импортов — всё равно удалять из `package.json`.
