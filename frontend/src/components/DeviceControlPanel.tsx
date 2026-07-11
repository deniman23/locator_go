import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { deviceApi } from '../services/api';
import type { User } from '../types/models';
import { formatDateTime } from '../utils/dateFormat';
import {
    fetchUserDeviceStatus,
    type UserDeviceStatus,
    waitForFreshHealthReport,
} from '../utils/userDeviceStatus';

type Props = {
    user: User;
    apiKey: string;
    initialStatus?: UserDeviceStatus;
    onClose: () => void;
    onStatusUpdate: (status: UserDeviceStatus) => void;
};

type DeviceConfigForm = {
    trackingPaused: boolean;
    locationIntervalSeconds: string;
    pollIntervalSeconds: string;
    healthReportIntervalSeconds: string;
    hiddenFromLauncher: boolean;
    adminPin: string;
    pushApiKey: string;
};

function readObj(report: Record<string, unknown> | undefined, key: string): Record<string, unknown> | undefined {
    const v = report?.[key];
    return v != null && typeof v === 'object' ? (v as Record<string, unknown>) : undefined;
}

function readStr(report: Record<string, unknown> | undefined, ...path: string[]): string {
    let cur: unknown = report;
    for (const key of path) {
        if (cur == null || typeof cur !== 'object') return '';
        cur = (cur as Record<string, unknown>)[key];
    }
    if (cur == null || cur === '') return '';
    return String(cur);
}

function readBool(report: Record<string, unknown> | undefined, ...path: string[]): boolean {
    let cur: unknown = report;
    for (const key of path) {
        if (cur == null || typeof cur !== 'object') return false;
        cur = (cur as Record<string, unknown>)[key];
    }
    return cur === true;
}

function readNum(report: Record<string, unknown> | undefined, ...path: string[]): number | undefined {
    const s = readStr(report, ...path);
    if (!s) return undefined;
    const n = Number(s);
    return Number.isFinite(n) ? n : undefined;
}

function formFromReport(report?: Record<string, unknown>): DeviceConfigForm {
    return {
        trackingPaused: readBool(report, 'config', 'tracking_paused'),
        locationIntervalSeconds: String(readNum(report, 'config', 'location_interval_seconds') ?? 300),
        pollIntervalSeconds: String(readNum(report, 'config', 'poll_interval_seconds') ?? 60),
        healthReportIntervalSeconds: String(
            readNum(report, 'config', 'health_report_interval_seconds') ?? 1200
        ),
        hiddenFromLauncher: readBool(report, 'corporate', 'hidden_from_launcher'),
        adminPin: '',
        pushApiKey: '',
    };
}

const ISSUE_LABELS: Record<string, string> = {
    no_api_key: 'Нет API-ключа',
    auth_failed: 'Ошибка авторизации',
    auth_not_checked: 'Авторизация не проверялась',
    location_permission_denied: 'Нет доступа к геолокации',
    location_permission_not_always: 'Геолокация только «при использовании»',
    location_services_disabled: 'Геолокация выключена в системе',
    background_stopped: 'Фоновый сервис остановлен',
    post_failed: 'Ошибка отправки координат',
    last_post_401: 'Сервер отклонил ключ (401)',
    offline_sync_pending: 'Ожидает синхронизации офлайн-очереди',
    post_stale: 'Давно не было отправки GPS',
    poll_stale: 'Давно не было опроса сервера',
    offline_queue_large: 'Большая офлайн-очередь',
    tracking_paused: 'Трекинг на паузе',
    app_update_ready: 'Обновление скачано, ждёт установки',
    app_update_failed: 'Ошибка обновления приложения',
};

const DeviceControlPanel: React.FC<Props> = ({
    user,
    apiKey,
    initialStatus,
    onClose,
    onStatusUpdate,
}) => {
    const [status, setStatus] = useState<UserDeviceStatus | undefined>(initialStatus);
    const [form, setForm] = useState<DeviceConfigForm>(() => formFromReport(initialStatus?.report));
    const [baselineForm, setBaselineForm] = useState<DeviceConfigForm>(() =>
        formFromReport(initialStatus?.report)
    );
    const [busy, setBusy] = useState(false);
    const [notice, setNotice] = useState<string | null>(null);
    const [showRaw, setShowRaw] = useState(false);

    const report = status?.report;

    const refreshStatus = useCallback(async () => {
        const next = await fetchUserDeviceStatus(user.id, apiKey);
        setStatus(next);
        onStatusUpdate(next);
        return next;
    }, [apiKey, onStatusUpdate, user.id]);

    useEffect(() => {
        void refreshStatus();
    }, [refreshStatus]);

    useEffect(() => {
        const next = formFromReport(status?.report);
        setForm(next);
        setBaselineForm(next);
    }, [status?.report]);

    const config = useMemo(() => readObj(report, 'config'), [report]);
    const corporate = useMemo(() => readObj(report, 'corporate'), [report]);
    const location = useMemo(() => readObj(report, 'location'), [report]);
    const battery = useMemo(() => readObj(report, 'battery'), [report]);
    const network = useMemo(() => readObj(report, 'network'), [report]);
    const auth = useMemo(() => readObj(report, 'auth'), [report]);
    const appUpdate = useMemo(() => readObj(report, 'app_update'), [report]);

    const handleEnableLocation = async () => {
        setBusy(true);
        setNotice('Отправка команды включения геолокации…');
        const baseline = status?.lastReportAt;
        try {
            const { data } = await deviceApi.enableLocation(user.id, apiKey);
            setNotice(data.note ?? 'Команда отправлена');
            const fresh = await waitForFreshHealthReport(user.id, apiKey, baseline, { attempts: 25 });
            if (fresh) {
                setStatus(fresh);
                onStatusUpdate(fresh);
                setNotice(
                    fresh.healthy
                        ? 'Геолокация включена, трекинг работает'
                        : 'Команда применена, но остались проблемы — см. список выше',
                );
            } else {
                await refreshStatus();
                setNotice('Команда в очереди; телефон должен выйти в сеть');
            }
        } catch (err) {
            setNotice(err instanceof Error ? err.message : 'Не удалось отправить команду');
        } finally {
            setBusy(false);
        }
    };

    const handleRefreshFromPhone = async () => {
        setBusy(true);
        setNotice('Запрос диагностики с телефона…');
        const baseline = status?.lastReportAt;
        try {
            await deviceApi.sendCommand(user.id, { type: 'health_check' }, apiKey);
            const fresh = await waitForFreshHealthReport(user.id, apiKey, baseline);
            if (fresh) {
                setStatus(fresh);
                onStatusUpdate(fresh);
                setNotice('Данные обновлены с телефона');
            } else {
                await refreshStatus();
                setNotice('Новый отчёт не получен — устройство офлайн?');
            }
        } catch (err) {
            setNotice(err instanceof Error ? err.message : 'Ошибка запроса');
        } finally {
            setBusy(false);
        }
    };

    const handleApply = async (e: React.FormEvent) => {
        e.preventDefault();
        setBusy(true);
        setNotice('Отправка настроек на устройство…');

        const body: Record<string, unknown> = {};

        if (form.trackingPaused !== baselineForm.trackingPaused) {
            body.tracking_paused = form.trackingPaused;
        }
        const locSec = Number(form.locationIntervalSeconds);
        if (
            form.locationIntervalSeconds !== baselineForm.locationIntervalSeconds &&
            Number.isFinite(locSec) &&
            locSec >= 30
        ) {
            body.location_interval_seconds = locSec;
        }
        const pollSec = Number(form.pollIntervalSeconds);
        if (
            form.pollIntervalSeconds !== baselineForm.pollIntervalSeconds &&
            Number.isFinite(pollSec) &&
            pollSec >= 5
        ) {
            body.poll_interval_seconds = pollSec;
        }
        const healthSec = Number(form.healthReportIntervalSeconds);
        if (
            form.healthReportIntervalSeconds !== baselineForm.healthReportIntervalSeconds &&
            Number.isFinite(healthSec) &&
            healthSec >= 60
        ) {
            body.health_report_interval_seconds = healthSec;
        }
        if (form.hiddenFromLauncher !== baselineForm.hiddenFromLauncher) {
            body.hidden_from_launcher = form.hiddenFromLauncher;
        }
        if (form.adminPin.trim()) {
            body.admin_pin = form.adminPin.trim();
        }
        if (form.pushApiKey.trim()) {
            body.api_key = form.pushApiKey.trim();
        }

        if (Object.keys(body).length === 0) {
            setNotice('Измените хотя бы одно поле или укажите новый PIN / ключ');
            setBusy(false);
            return;
        }

        const baselineReportAt = status?.lastReportAt;
        try {
            await deviceApi.pushDeviceConfig(user.id, body, apiKey);
            setNotice('Ожидание применения на телефоне…');
            const fresh = await waitForFreshHealthReport(user.id, apiKey, baselineReportAt, {
                attempts: 25,
            });
            if (fresh) {
                setStatus(fresh);
                onStatusUpdate(fresh);
                setForm((prev) => ({ ...prev, adminPin: '', pushApiKey: '' }));
                setNotice('Настройки применены');
            } else {
                await refreshStatus();
                setNotice('Команда отправлена; подтверждение ещё не получено');
            }
        } catch (err) {
            setNotice(err instanceof Error ? err.message : 'Не удалось отправить настройки');
        } finally {
            setBusy(false);
        }
    };

    return (
        <div className="qr-code-modal device-control-modal" role="dialog" aria-modal="true">
            <div className="device-control-container">
                <div className="device-control-header">
                    <h3>Устройство: {user.name}</h3>
                    <button className="close-button" type="button" onClick={onClose} disabled={busy}>
                        ×
                    </button>
                </div>

                {notice && <p className="device-control-notice">{notice}</p>}

                <div className="device-control-body">
                    <section className="device-control-section">
                        <div className="device-control-section-head">
                            <h4>Состояние</h4>
                            <button
                                type="button"
                                className="device-action-button"
                                onClick={() => void handleRefreshFromPhone()}
                                disabled={busy}
                            >
                                Обновить с телефона
                            </button>
                            <button
                                type="button"
                                className="device-action-button device-action-button--location"
                                onClick={() => void handleEnableLocation()}
                                disabled={busy}
                                title="Повторно выдать разрешения и включить GPS на телефоне (Device Owner)"
                            >
                                Вкл. GPS на телефоне
                            </button>
                        </div>
                        <dl className="device-control-dl">
                            <dt>Приложение</dt>
                            <dd>
                                {[status?.platform, status?.appVersion].filter(Boolean).join(' · ') || '—'}
                            </dd>
                            <dt>Модель</dt>
                            <dd>{readStr(report, 'device_model') || '—'}</dd>
                            <dt>Отчёт</dt>
                            <dd>
                                {status?.lastReportAt
                                    ? formatDateTime(status.lastReportAt)
                                    : 'Нет данных'}
                            </dd>
                            <dt>Статус</dt>
                            <dd>
                                {status?.healthy == null
                                    ? '—'
                                    : status.healthy
                                      ? 'OK'
                                      : `Проблемы (${status.issues.length})`}
                            </dd>
                            {status && status.issues.length > 0 && (
                                <>
                                    <dt>Проблемы</dt>
                                    <dd>
                                        <ul className="device-issues">
                                            {status.issues.map((issue) => (
                                                <li key={issue}>
                                                    {ISSUE_LABELS[issue] ?? issue}
                                                </li>
                                            ))}
                                        </ul>
                                    </dd>
                                </>
                            )}
                            <dt>Ключ на устройстве</dt>
                            <dd>
                                {config?.api_key_present
                                    ? `…${String(config.api_key_last4 ?? '????')}`
                                    : 'не задан'}
                            </dd>
                            <dt>PIN входа</dt>
                            <dd>{auth?.pin_configured ? 'установлен' : 'по умолчанию (2580)'}</dd>
                            <dt>Device Owner</dt>
                            <dd>{corporate?.device_owner ? 'да' : 'нет'}</dd>
                            <dt>Скрыто из лаунчера</dt>
                            <dd>{corporate?.hidden_from_launcher ? 'да' : 'нет'}</dd>
                            <dt>Геолокация</dt>
                            <dd>
                                {location?.system_enabled === false
                                    ? 'выключена в системе'
                                    : location?.permission === 'always'
                                      ? 'всегда'
                                      : location?.permission === 'while_in_use'
                                        ? 'при использовании'
                                        : location?.permission === 'denied'
                                          ? 'запрещена'
                                          : '—'}
                            </dd>
                            <dt>Трекинг</dt>
                            <dd>
                                {config?.tracking_paused
                                    ? 'на паузе'
                                    : location?.foreground_service_running
                                      ? 'активен'
                                      : 'остановлен'}
                            </dd>
                            <dt>Интервал GPS</dt>
                            <dd>
                                {config?.location_interval_seconds != null
                                    ? `${config.location_interval_seconds} с`
                                    : '—'}
                            </dd>
                            <dt>Батарея</dt>
                            <dd>
                                {battery?.level_percent != null
                                    ? `${battery.level_percent}%`
                                    : '—'}
                                {battery?.power_save_mode ? ' (энергосбережение)' : ''}
                            </dd>
                            <dt>Сеть</dt>
                            <dd>{String(network?.type ?? '—')}</dd>
                            <dt>Обновление</dt>
                            <dd>
                                {appUpdate?.state
                                    ? `${String(appUpdate.state)}${
                                          appUpdate.target_version
                                              ? ` → ${String(appUpdate.target_version)}`
                                              : ''
                                      }`
                                    : '—'}
                            </dd>
                        </dl>
                    </section>

                    <section className="device-control-section">
                        <h4>Удалённые настройки</h4>
                        <p className="device-control-hint">
                            Изменения отправляются на телефон командой config_update. Устройство должно
                            быть онлайн (опрос сервера).
                        </p>
                        <form className="device-control-form" onSubmit={handleApply}>
                            <label className="form-check device-control-check">
                                <input
                                    type="checkbox"
                                    checked={form.trackingPaused}
                                    onChange={(e) =>
                                        setForm((f) => ({ ...f, trackingPaused: e.target.checked }))
                                    }
                                    disabled={busy}
                                />
                                <span>Пауза трекинга GPS</span>
                            </label>

                            <div className="form-group">
                                <label htmlFor={`loc-int-${user.id}`}>Интервал GPS (сек, ≥30)</label>
                                <input
                                    id={`loc-int-${user.id}`}
                                    type="number"
                                    min={30}
                                    value={form.locationIntervalSeconds}
                                    onChange={(e) =>
                                        setForm((f) => ({
                                            ...f,
                                            locationIntervalSeconds: e.target.value,
                                        }))
                                    }
                                    disabled={busy}
                                />
                            </div>

                            <div className="form-group">
                                <label htmlFor={`poll-int-${user.id}`}>Интервал опроса (сек, ≥5)</label>
                                <input
                                    id={`poll-int-${user.id}`}
                                    type="number"
                                    min={5}
                                    value={form.pollIntervalSeconds}
                                    onChange={(e) =>
                                        setForm((f) => ({ ...f, pollIntervalSeconds: e.target.value }))
                                    }
                                    disabled={busy}
                                />
                            </div>

                            <div className="form-group">
                                <label htmlFor={`health-int-${user.id}`}>
                                    Интервал диагностики (сек, ≥60)
                                </label>
                                <input
                                    id={`health-int-${user.id}`}
                                    type="number"
                                    min={60}
                                    value={form.healthReportIntervalSeconds}
                                    onChange={(e) =>
                                        setForm((f) => ({
                                            ...f,
                                            healthReportIntervalSeconds: e.target.value,
                                        }))
                                    }
                                    disabled={busy}
                                />
                            </div>

                            <label className="form-check device-control-check">
                                <input
                                    type="checkbox"
                                    checked={form.hiddenFromLauncher}
                                    onChange={(e) =>
                                        setForm((f) => ({
                                            ...f,
                                            hiddenFromLauncher: e.target.checked,
                                        }))
                                    }
                                    disabled={busy || !corporate?.device_owner}
                                />
                                <span>
                                    Скрыть из лаунчера
                                    {!corporate?.device_owner && ' (нужен Device Owner)'}
                                </span>
                            </label>

                            <div className="form-group">
                                <label htmlFor={`admin-pin-${user.id}`}>Новый PIN входа (4–12 цифр)</label>
                                <input
                                    id={`admin-pin-${user.id}`}
                                    type="password"
                                    inputMode="numeric"
                                    autoComplete="off"
                                    placeholder="оставьте пустым, чтобы не менять"
                                    value={form.adminPin}
                                    onChange={(e) =>
                                        setForm((f) => ({ ...f, adminPin: e.target.value }))
                                    }
                                    disabled={busy}
                                />
                            </div>

                            <div className="form-group">
                                <label htmlFor={`api-key-${user.id}`}>Отправить API-ключ на телефон</label>
                                <input
                                    id={`api-key-${user.id}`}
                                    type="text"
                                    autoComplete="off"
                                    placeholder="вставьте ключ или используйте «Перегенерировать QR»"
                                    value={form.pushApiKey}
                                    onChange={(e) =>
                                        setForm((f) => ({ ...f, pushApiKey: e.target.value }))
                                    }
                                    disabled={busy}
                                />
                            </div>

                            <div className="device-control-actions">
                                <button type="submit" className="create-button" disabled={busy}>
                                    {busy ? 'Отправка…' : 'Применить на устройстве'}
                                </button>
                                <button
                                    type="button"
                                    className="button"
                                    onClick={() => setShowRaw((v) => !v)}
                                    disabled={busy}
                                >
                                    {showRaw ? 'Скрыть JSON' : 'Полный JSON'}
                                </button>
                            </div>
                        </form>

                        {showRaw && report && (
                            <pre className="device-report-raw">{JSON.stringify(report, null, 2)}</pre>
                        )}
                    </section>
                </div>
            </div>
        </div>
    );
};

export default DeviceControlPanel;
