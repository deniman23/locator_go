import { deviceApi, locationApi } from '../services/api';

export const GPS_ONLINE_SECONDS = 300;

export type GpsStatus = 'online' | 'stale' | 'offline';

export type UserDeviceStatus = {
    gps: GpsStatus;
    ageSeconds?: number;
    healthy?: boolean;
    lastReportAt?: string;
    appVersion?: string;
    platform?: string;
    issues: string[];
    report?: Record<string, unknown>;
};

function isNewerTimestamp(next?: string, baseline?: string): boolean {
    if (!next) return false;
    if (!baseline) return true;
    return new Date(next).getTime() > new Date(baseline).getTime();
}

export function healthFromResponse(data: {
    last_report_at: string;
    app_version?: string;
    platform?: string;
    issues?: string[];
    healthy: boolean;
    report: Record<string, unknown>;
}): Omit<UserDeviceStatus, 'gps' | 'ageSeconds'> {
    return {
        healthy: data.healthy,
        lastReportAt: data.last_report_at,
        appVersion: data.app_version,
        platform: data.platform,
        issues: data.issues ?? [],
        report: data.report,
    };
}

export async function fetchUserDeviceStatus(
    userId: number,
    apiKey: string
): Promise<UserDeviceStatus> {
    let gps: GpsStatus = 'offline';
    let ageSeconds: number | undefined;

    try {
        const { data } = await locationApi.getByUserId(userId, apiKey);
        ageSeconds = data.age_seconds;
        gps =
            ageSeconds != null && ageSeconds <= GPS_ONLINE_SECONDS ? 'online' : 'stale';
    } catch {
        gps = 'offline';
    }

    const base: UserDeviceStatus = { gps, ageSeconds, issues: [] };

    try {
        const { data } = await deviceApi.getUserHealth(userId, apiKey);
        return { ...base, ...healthFromResponse(data) };
    } catch {
        return base;
    }
}

/** Пакетная загрузка статусов для списка пользователей (админка). */
export async function fetchAllUserDeviceStatuses(
    userIds: number[],
    apiKey: string
): Promise<Record<number, UserDeviceStatus>> {
    const out: Record<number, UserDeviceStatus> = {};
    for (const id of userIds) {
        out[id] = { gps: 'offline', issues: [] };
    }

    try {
        const { data } = await deviceApi.getAllDevicesStatus(apiKey);
        for (const [idStr, row] of Object.entries(data.users ?? {})) {
            const id = Number(idStr);
            if (!Number.isFinite(id)) continue;
            const age = row.age_seconds;
            let gps: GpsStatus = 'offline';
            if (age != null) {
                gps = age <= GPS_ONLINE_SECONDS ? 'online' : 'stale';
            } else if (row.gps === 'online' || row.gps === 'stale') {
                gps = row.gps;
            }
            out[id] = {
                gps,
                ageSeconds: age,
                healthy: row.healthy,
                lastReportAt: row.last_report_at,
                appVersion: row.app_version,
                platform: row.platform,
                issues: row.issues ?? [],
            };
        }
    } catch {
        // fallback: пустые статусы
    }

    return out;
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

/** Ждёт новый device report после health_check (или первый отчёт). */
export async function waitForFreshHealthReport(
    userId: number,
    apiKey: string,
    baselineReportAt?: string,
    opts?: { attempts?: number; intervalMs?: number }
): Promise<UserDeviceStatus | null> {
    const attempts = opts?.attempts ?? 30;
    const intervalMs = opts?.intervalMs ?? 2000;

    for (let i = 0; i < attempts; i++) {
        try {
            const { data } = await deviceApi.getUserHealth(userId, apiKey);
            if (isNewerTimestamp(data.last_report_at, baselineReportAt)) {
                const status = await fetchUserDeviceStatus(userId, apiKey);
                return {
                    ...status,
                    ...healthFromResponse(data),
                };
            }
        } catch {
            // отчёта ещё нет
        }
        await sleep(intervalMs);
    }
    return null;
}

/** Ждёт свежую GPS-точку после on-demand запроса. */
export async function waitForFreshLocation(
    userId: number,
    apiKey: string,
    baselineAgeSeconds?: number,
    opts?: { attempts?: number; intervalMs?: number }
): Promise<UserDeviceStatus | null> {
    const attempts = opts?.attempts ?? 20;
    const intervalMs = opts?.intervalMs ?? 2000;

    for (let i = 0; i < attempts; i++) {
        try {
            const { data } = await locationApi.getByUserId(userId, apiKey);
            const age = data.age_seconds ?? Number.MAX_SAFE_INTEGER;
            const improved =
                baselineAgeSeconds == null || age < baselineAgeSeconds - 2;
            if (improved) {
                return fetchUserDeviceStatus(userId, apiKey);
            }
        } catch {
            // нет локации
        }
        await sleep(intervalMs);
    }
    return null;
}
