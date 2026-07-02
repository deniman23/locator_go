// components/UserManagement.tsx
import React, { useCallback, useEffect, useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { deviceApi, locationApi, releaseApi, userApi } from '../services/api';
import type { User } from '../types/models';
import { formatDateTime } from '../utils/dateFormat';
import {
    fetchUserDeviceStatus,
    type GpsStatus,
    type UserDeviceStatus,
    waitForFreshHealthReport,
    waitForFreshLocation,
} from '../utils/userDeviceStatus';
import QRCodeDisplay from './QRCodeDisplay';

const STATUS_POLL_MS = 15_000;

function formatAge(seconds?: number): string {
    if (seconds == null) return '—';
    if (seconds < 60) return `${seconds} с`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)} мин`;
    if (seconds < 86_400) return `${Math.floor(seconds / 3600)} ч`;
    return `${Math.floor(seconds / 86_400)} д`;
}

function gpsLabel(status: GpsStatus): string {
    switch (status) {
        case 'online':
            return 'Онлайн';
        case 'stale':
            return 'Устарело';
        default:
            return 'Нет связи';
    }
}

function reportSummary(report: Record<string, unknown>): string[] {
    const lines: string[] = [];
    const add = (label: string, key: string) => {
        const v = report[key];
        if (v != null && v !== '') lines.push(`${label}: ${String(v)}`);
    };
    add('Батарея', 'battery_level');
    add('Зарядка', 'battery_charging');
    add('Сеть', 'network_type');
    add('GPS вкл.', 'gps_enabled');
    add('Фоновый режим', 'background_restricted');
    add('Разрешения', 'permissions_ok');
    return lines;
}

const UserManagement: React.FC = () => {
    const { apiKey, user: currentUser } = useAuth();
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const [showQRCode, setShowQRCode] = useState(false);
    const [selectedUser, setSelectedUser] = useState<User | null>(null);
    const [qrRefreshKey, setQrRefreshKey] = useState(0);
    const [regenerateResult, setRegenerateResult] = useState<{
        userId: number;
        userName: string;
        apiKey: string;
        pushedToDevice: boolean;
    } | null>(null);

    const [newUserName, setNewUserName] = useState('');
    const [newUserIsAdmin, setNewUserIsAdmin] = useState(false);
    const [creating, setCreating] = useState(false);
    const [createError, setCreateError] = useState<string | null>(null);

    const [deviceStatus, setDeviceStatus] = useState<Record<number, UserDeviceStatus>>({});
    const [statusLoading, setStatusLoading] = useState(false);
    const [actionUserId, setActionUserId] = useState<number | null>(null);
    const [actionNotice, setActionNotice] = useState<Record<number, string>>({});
    const [expandedReportUserId, setExpandedReportUserId] = useState<number | null>(null);

    const setNotice = (userId: number, text: string, clearMs = 8000) => {
        setActionNotice((prev) => ({ ...prev, [userId]: text }));
        window.setTimeout(() => {
            setActionNotice((prev) => {
                const next = { ...prev };
                delete next[userId];
                return next;
            });
        }, clearMs);
    };

    const fetchDeviceStatuses = useCallback(
        async (userList: User[]) => {
            if (!apiKey || userList.length === 0) return;

            setStatusLoading(true);
            const entries = await Promise.all(
                userList.map(async (user) => [
                    user.id,
                    await fetchUserDeviceStatus(user.id, apiKey),
                ] as const)
            );
            setDeviceStatus(Object.fromEntries(entries));
            setStatusLoading(false);
        },
        [apiKey]
    );

    const applyUserStatus = (userId: number, status: UserDeviceStatus) => {
        setDeviceStatus((prev) => ({ ...prev, [userId]: status }));
    };

    useEffect(() => {
        const fetchUsers = async () => {
            if (!apiKey) return;

            try {
                setLoading(true);
                const userList = await userApi.getAll(apiKey);
                setUsers(userList);
                setError(null);
                await fetchDeviceStatuses(userList);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Ошибка при загрузке пользователей');
            } finally {
                setLoading(false);
            }
        };

        fetchUsers();
    }, [apiKey, fetchDeviceStatuses]);

    useEffect(() => {
        if (!apiKey || users.length === 0) return;

        const timer = window.setInterval(() => {
            void fetchDeviceStatuses(users);
        }, STATUS_POLL_MS);

        return () => window.clearInterval(timer);
    }, [apiKey, users, fetchDeviceStatuses]);

    const handleCreateUser = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!apiKey || !newUserName.trim()) return;

        try {
            setCreating(true);
            setCreateError(null);

            const newUser = await userApi.create(newUserName, newUserIsAdmin, apiKey);
            const nextUsers = [...users, newUser];
            setUsers(nextUsers);
            void fetchDeviceStatuses(nextUsers);

            setNewUserName('');
            setNewUserIsAdmin(false);
        } catch (err) {
            setCreateError(err instanceof Error ? err.message : 'Ошибка при создании пользователя');
        } finally {
            setCreating(false);
        }
    };

    const handleShowUserQRCode = (user: User) => {
        setQrRefreshKey(Date.now());
        setSelectedUser(user);
        setShowQRCode(true);
    };

    const handleCloseQRCode = () => {
        setShowQRCode(false);
        setSelectedUser(null);
    };

    const handleRegenerateQR = async (user: User) => {
        if (!apiKey) return;

        const confirmed = window.confirm(
            `Перегенерировать QR для «${user.name}»?\n\n` +
                'Будет создан новый API-ключ. Старый перестанет работать.\n' +
                'На телефоне нужно отсканировать новый QR (или дождаться config_update, если устройство онлайн).'
        );
        if (!confirmed) return;

        setActionUserId(user.id);
        setNotice(user.id, 'Перегенерация QR…', 60_000);

        try {
            const result = await userApi.regenerateQR(user.id, apiKey);
            setUsers((prev) =>
                prev.map((u) => (u.id === user.id ? { ...u, qr_code: result.qr_code } : u))
            );
            setQrRefreshKey(Date.now());
            setRegenerateResult({
                userId: user.id,
                userName: user.name,
                apiKey: result.api_key,
                pushedToDevice: Boolean(result.config_command_id),
            });
            setNotice(
                user.id,
                result.config_command_id
                    ? 'QR обновлён, config_update отправлен на устройство'
                    : 'QR обновлён — отсканируйте новый код на телефоне'
            );
        } catch (err) {
            setNotice(
                user.id,
                err instanceof Error ? err.message : 'Не удалось перегенерировать QR'
            );
        } finally {
            setActionUserId(null);
        }
    };

    const handleShowRegeneratedQR = () => {
        if (!regenerateResult) return;
        const user = users.find((u) => u.id === regenerateResult.userId);
        if (user) {
            setSelectedUser(user);
            setShowQRCode(true);
        }
    };

    const handleRequestLocation = async (userId: number) => {
        if (!apiKey) return;
        const baselineAge = deviceStatus[userId]?.ageSeconds;
        setActionUserId(userId);
        setNotice(userId, 'Запрос координат…', 60_000);

        try {
            await locationApi.requestOnDemand(userId, apiKey);
            setNotice(userId, 'Ожидание ответа устройства…', 60_000);
            const fresh = await waitForFreshLocation(userId, apiKey, baselineAge);
            if (fresh) {
                applyUserStatus(userId, fresh);
                setNotice(userId, 'Координаты обновлены');
            } else {
                const current = await fetchUserDeviceStatus(userId, apiKey);
                applyUserStatus(userId, current);
                setNotice(userId, 'Устройство не ответило вовремя');
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Не удалось запросить координаты');
        } finally {
            setActionUserId(null);
        }
    };

    const handleHealthCheck = async (userId: number) => {
        if (!apiKey) return;
        const baselineReportAt = deviceStatus[userId]?.lastReportAt;
        setActionUserId(userId);
        setExpandedReportUserId(null);
        setNotice(userId, 'Отправка диагностики…', 90_000);

        try {
            await deviceApi.sendCommand(userId, { type: 'health_check' }, apiKey);
            setNotice(userId, 'Ожидание отчёта с телефона…', 90_000);
            const fresh = await waitForFreshHealthReport(userId, apiKey, baselineReportAt);
            if (fresh) {
                applyUserStatus(userId, fresh);
                setExpandedReportUserId(userId);
                setNotice(userId, 'Диагностика обновлена');
            } else {
                const current = await fetchUserDeviceStatus(userId, apiKey);
                applyUserStatus(userId, current);
                setNotice(userId, 'Новый отчёт не получен — проверьте, что приложение онлайн');
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Не удалось отправить проверку');
        } finally {
            setActionUserId(null);
        }
    };

    const handlePublishUpdate = async (userId: number) => {
        if (!apiKey) return;
        setActionUserId(userId);
        setNotice(userId, 'Отправка обновления…', 30_000);

        try {
            const { data } = await releaseApi.publishUpdate(userId, apiKey);
            const version =
                (data.payload?.version_name as string | undefined) ??
                (data.payload?.versionName as string | undefined);
            setNotice(
                userId,
                version ? `Обновление v${version} отправлено на устройство` : 'Команда обновления отправлена'
            );
        } catch (err) {
            const msg =
                err instanceof Error && 'response' in err
                    ? ((err as { response?: { data?: { error?: string } } }).response?.data?.error ??
                        err.message)
                    : 'Не удалось отправить обновление';
            setNotice(userId, msg);
        } finally {
            setActionUserId(null);
        }
    };

    if (currentUser && !currentUser.is_admin) {
        return (
            <div className="user-management">
                <h2>Доступ запрещен</h2>
                <p>У вас нет прав для доступа к этой странице.</p>
            </div>
        );
    }

    return (
        <div className="user-management">
            <h2>Управление пользователями</h2>

            {showQRCode && (
                <QRCodeDisplay
                    onClose={handleCloseQRCode}
                    userId={selectedUser?.id}
                    userName={selectedUser?.name}
                    refreshKey={qrRefreshKey}
                />
            )}

            {regenerateResult && (
                <div className="qr-code-modal">
                    <div className="qr-code-container">
                        <div className="qr-code-header">
                            <h3>Новый ключ: {regenerateResult.userName}</h3>
                            <button
                                className="close-button"
                                onClick={() => setRegenerateResult(null)}
                                type="button"
                            >
                                ×
                            </button>
                        </div>
                        <div className="qr-code-content">
                            <p>Сохраните ключ — он показывается один раз:</p>
                            <code className="regenerate-api-key">{regenerateResult.apiKey}</code>
                            <p className="qr-code-hint">
                                {regenerateResult.pushedToDevice
                                    ? 'На устройство отправлен config_update. Если телефон онлайн, настройки обновятся автоматически; иначе отсканируйте QR.'
                                    : 'Отсканируйте новый QR-код в приложении на телефоне.'}
                            </p>
                        </div>
                        <div className="qr-code-footer">
                            <button className="button" type="button" onClick={handleShowRegeneratedQR}>
                                Показать QR
                            </button>
                            <button
                                className="button"
                                type="button"
                                onClick={() => setRegenerateResult(null)}
                            >
                                Закрыть
                            </button>
                        </div>
                    </div>
                </div>
            )}

            <div className="create-user-form">
                <h3>Создать нового пользователя</h3>
                {createError && <div className="error-message">{createError}</div>}

                <form onSubmit={handleCreateUser}>
                    <div className="form-group">
                        <label htmlFor="userName">Имя пользователя</label>
                        <input
                            type="text"
                            id="userName"
                            value={newUserName}
                            onChange={(e) => setNewUserName(e.target.value)}
                            disabled={creating}
                            placeholder="Введите имя пользователя"
                            required
                        />
                    </div>

                    <div className="form-check">
                        <input
                            type="checkbox"
                            id="isAdmin"
                            checked={newUserIsAdmin}
                            onChange={(e) => setNewUserIsAdmin(e.target.checked)}
                            disabled={creating}
                        />
                        <label htmlFor="isAdmin">Администратор</label>
                    </div>

                    <button
                        type="submit"
                        className="create-button"
                        disabled={creating || !newUserName.trim()}
                    >
                        {creating ? 'Создание...' : 'Создать пользователя'}
                    </button>
                </form>
            </div>

            <div className="users-list">
                <h3>Список пользователей</h3>
                {statusLoading && users.length > 0 && (
                    <p className="status-refresh-hint">Обновление статусов устройств…</p>
                )}

                {loading ? (
                    <p>Загрузка пользователей...</p>
                ) : error ? (
                    <div className="error-message">{error}</div>
                ) : users.length === 0 ? (
                    <p>Пользователи не найдены.</p>
                ) : (
                    <table className="users-table">
                        <thead>
                        <tr>
                            <th>ID</th>
                            <th>Имя</th>
                            <th>Роль</th>
                            <th>GPS</th>
                            <th>Устройство</th>
                            <th>Дата создания</th>
                            <th>Действия</th>
                        </tr>
                        </thead>
                        <tbody>
                        {users.map((user) => {
                            const status = deviceStatus[user.id];
                            const busy = actionUserId === user.id;
                            const notice = actionNotice[user.id];
                            const summaryLines =
                                status?.report && expandedReportUserId === user.id
                                    ? reportSummary(status.report)
                                    : [];

                            return (
                                <tr key={user.id}>
                                    <td>{user.id}</td>
                                    <td>{user.name}</td>
                                    <td>{user.is_admin ? 'Администратор' : 'Пользователь'}</td>
                                    <td>
                                        {status ? (
                                            <span
                                                className={`device-badge device-badge--${status.gps}`}
                                            >
                                                    {gpsLabel(status.gps)}
                                                </span>
                                        ) : (
                                            '—'
                                        )}
                                        {status?.ageSeconds != null && (
                                            <div className="device-meta">
                                                {formatAge(status.ageSeconds)} назад
                                            </div>
                                        )}
                                    </td>
                                    <td>
                                        {notice && (
                                            <p className="device-action-notice">{notice}</p>
                                        )}
                                        {status?.healthy != null ? (
                                            <>
                                                    <span
                                                        className={`device-badge device-badge--${
                                                            status.healthy ? 'healthy' : 'unhealthy'
                                                        }`}
                                                    >
                                                        {status.healthy
                                                            ? 'OK'
                                                            : `Проблемы (${status.issues.length})`}
                                                    </span>
                                                {(status.appVersion || status.platform) && (
                                                    <div className="device-meta">
                                                        {[status.platform, status.appVersion]
                                                            .filter(Boolean)
                                                            .join(' · ')}
                                                    </div>
                                                )}
                                                {status.lastReportAt && (
                                                    <div className="device-meta">
                                                        отчёт:{' '}
                                                        {formatDateTime(status.lastReportAt)}
                                                    </div>
                                                )}
                                                {status.issues.length > 0 && (
                                                    <ul
                                                        className="device-issues"
                                                        title={status.issues.join('\n')}
                                                    >
                                                        {status.issues.map((issue) => (
                                                            <li key={issue}>{issue}</li>
                                                        ))}
                                                    </ul>
                                                )}
                                                {expandedReportUserId === user.id &&
                                                    status.report && (
                                                        <details
                                                            className="device-report-details"
                                                            open
                                                        >
                                                            <summary>
                                                                Подробности диагностики
                                                            </summary>
                                                            {summaryLines.length > 0 ? (
                                                                <ul className="device-report-summary">
                                                                    {summaryLines.map((line) => (
                                                                        <li key={line}>{line}</li>
                                                                    ))}
                                                                </ul>
                                                            ) : null}
                                                            <pre className="device-report-raw">
                                                                    {JSON.stringify(
                                                                        status.report,
                                                                        null,
                                                                        2
                                                                    )}
                                                                </pre>
                                                        </details>
                                                    )}
                                            </>
                                        ) : (
                                            <span className="device-meta">Нет отчёта</span>
                                        )}
                                    </td>
                                    <td>{formatDateTime(user.created_at)}</td>
                                    <td className="user-actions-cell">
                                        <div className="user-actions-inner">
                                            <button
                                                className="qr-code-button-small"
                                                onClick={() => handleShowUserQRCode(user)}
                                                disabled={busy}
                                            >
                                                QR-код
                                            </button>
                                            <button
                                                className="device-action-button"
                                                onClick={() => handleRegenerateQR(user)}
                                                disabled={busy}
                                                title="Новый API-ключ и QR (старый перестанет работать)"
                                            >
                                                Перегенерировать QR
                                            </button>
                                            <button
                                                className="device-action-button"
                                                onClick={() => handleRequestLocation(user.id)}
                                                disabled={busy}
                                                title="Запросить координаты с телефона"
                                            >
                                                GPS
                                            </button>
                                            <button
                                                className="device-action-button"
                                                onClick={() => handleHealthCheck(user.id)}
                                                disabled={busy}
                                                title="Запросить диагностику приложения"
                                            >
                                                Диагностика
                                            </button>
                                            <button
                                                className="device-action-button device-action-button--update"
                                                onClick={() => handlePublishUpdate(user.id)}
                                                disabled={busy}
                                                title="Отправить APK-обновление на устройство"
                                            >
                                                Обновление
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            );
                        })}
                        </tbody>
                    </table>
                )}
            </div>
        </div>
    );
};

export default UserManagement;
