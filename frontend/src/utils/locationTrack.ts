import type { Location } from '../types/models';

/** ~90 км/ч — выше считаем GPS-выбросом */
const TRACK_MAX_SPEED_MPS = 25;
/** Окно пакетной отправки офлайн-очереди */
const TRACK_BATCH_WINDOW_MS = 10_000;
/** Макс. скачок при пакетной отправке (м) */
const TRACK_BATCH_MAX_JUMP_M = 250;
/** Короткий интервал между точками */
const TRACK_SHORT_GAP_MS = 2 * 60_000;
/** Макс. скачок за короткий интервал (м) */
const TRACK_SHORT_GAP_MAX_JUMP_M = 500;
/** Макс. скачок за разумный интервал (после backfill офлайн-очереди) */
const TRACK_ABSOLUTE_MAX_JUMP_M = 1500;
const TRACK_ABSOLUTE_MAX_JUMP_MS = 45 * 60_000;
/** Типичный интервал periodic (~5 мин): скачок >1500 м — подозрительно (смотрим по факту) */
const TRACK_PERIODIC_MIN_GAP_MS = 3.5 * 60_000;
const TRACK_PERIODIC_MAX_GAP_MS = 9 * 60_000;
const TRACK_PERIODIC_MAX_JUMP_M = 1500;
/** «Островок»: ушёл далеко и вернулся к той же точке (кэш сети, выброс) */
const ISLAND_RETURN_RADIUS_M = 130;
const ISLAND_MIN_JUMP_M = 200;
const MAX_ISLAND_SPAN_MS = 30 * 60_000;

/** Расстояние между двумя точками на сфере, метры */
export function haversineMeters(lat1: number, lon1: number, lat2: number, lon2: number): number {
    const R = 6371000;
    const toRad = (x: number) => (x * Math.PI) / 180;
    const dLat = toRad(lat2 - lat1);
    const dLon = toRad(lon2 - lon1);
    const a =
        Math.sin(dLat / 2) ** 2 +
        Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLon / 2) ** 2;
    return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

function locationTimeMs(loc: Location): number {
    return locationEffectiveAtMs(loc);
}

/** Точка невозможна относительно предыдущей принятой. */
export function isTrackOutlierFromPrev(prev: Location, curr: Location): boolean {
    const dist = haversineMeters(prev.latitude, prev.longitude, curr.latitude, curr.longitude);
    const tPrev = locationTimeMs(prev);
    const tCurr = locationTimeMs(curr);
    if (!Number.isFinite(tPrev) || !Number.isFinite(tCurr)) return false;
    const dtMs = tCurr - tPrev;
    if (dtMs < 0) return true;
    if (dtMs <= TRACK_BATCH_WINDOW_MS) return dist > TRACK_BATCH_MAX_JUMP_M;
    if (
        dtMs >= TRACK_PERIODIC_MIN_GAP_MS &&
        dtMs <= TRACK_PERIODIC_MAX_GAP_MS &&
        dist > TRACK_PERIODIC_MAX_JUMP_M
    ) {
        return true;
    }
    if (dtMs > 0 && dtMs <= TRACK_ABSOLUTE_MAX_JUMP_MS && dist > TRACK_ABSOLUTE_MAX_JUMP_M) return true;
    if (dtMs <= TRACK_SHORT_GAP_MS) return dist > TRACK_SHORT_GAP_MAX_JUMP_M;
    if (dtMs > 0) return dist / (dtMs / 1000) > TRACK_MAX_SPEED_MPS;
    return dist > TRACK_BATCH_MAX_JUMP_M;
}

/**
 * Убирает GPS-выбросы (офлайн-очередь, сетевая геолокация без A-GPS).
 * Оставляет физически возможный трек.
 */
export function filterTrackOutliers(locations: Location[]): Location[] {
    const sorted = normalizeLocations(locations);
    if (sorted.length <= 1) return sorted;
    const deduped = filterGpsIslands(sorted);
    const out: Location[] = [deduped[0]];
    for (let i = 1; i < deduped.length; i++) {
        const curr = deduped[i];
        const prev = out[out.length - 1];
        const baseline = outlierBaseline(out);
        if (isTrackOutlierFromPrev(prev, curr)) {
            const returningToBaseline =
                baseline != null &&
                haversineMeters(baseline.latitude, baseline.longitude, curr.latitude, curr.longitude) <=
                    ISLAND_RETURN_RADIUS_M;
            if (
                !returningToBaseline &&
                (!baseline ||
                    haversineMeters(baseline.latitude, baseline.longitude, curr.latitude, curr.longitude) >
                        TRACK_BATCH_MAX_JUMP_M)
            ) {
                continue;
            }
        } else if (baseline && isTrackOutlierFromPrev(baseline, curr)) {
            continue;
        }
        out.push(curr);
    }
    return out;
}

/** Последняя надёжная точка в уже принятом треке (пропуск цепочки выбросов). */
function outlierBaseline(kept: Location[]): Location | null {
    if (kept.length === 0) return null;
    let prev = kept[kept.length - 1];
    for (let n = 0; n < 8; n++) {
        const idx = kept.lastIndexOf(prev);
        if (idx <= 0) break;
        const grand = kept[idx - 1];
        if (isTrackOutlierFromPrev(grand, prev)) {
            prev = grand;
            continue;
        }
        break;
    }
    return prev;
}

/** Одна точка между двумя близкими: телепорт туда-обратно (#6012). */
function isSandwichOutlier(anchor: Location, mid: Location, next: Location): boolean {
    const dAnchorMid = haversineMeters(anchor.latitude, anchor.longitude, mid.latitude, mid.longitude);
    const dMidNext = haversineMeters(mid.latitude, mid.longitude, next.latitude, next.longitude);
    const dAnchorNext = haversineMeters(anchor.latitude, anchor.longitude, next.latitude, next.longitude);
    if (dAnchorMid < ISLAND_MIN_JUMP_M || dMidNext < ISLAND_MIN_JUMP_M) return false;
    return dAnchorNext <= ISLAND_RETURN_RADIUS_M;
}

/**
 * Убирает короткие «островки» GPS: несколько точек далеко от якоря, затем возврат
 * (ночной кэш сети, как #6002–#6003 с возвратом на #6004).
 */
export function filterGpsIslands(locations: Location[]): Location[] {
    if (locations.length < 3) return locations;
    const out: Location[] = [locations[0]];
    let i = 1;
    while (i < locations.length) {
        const anchor = out[out.length - 1];
        if (i + 1 < locations.length && isSandwichOutlier(anchor, locations[i], locations[i + 1])) {
            i += 1;
            continue;
        }
        let j = i;
        while (j < locations.length) {
            const d = haversineMeters(
                anchor.latitude,
                anchor.longitude,
                locations[j].latitude,
                locations[j].longitude,
            );
            if (d <= ISLAND_RETURN_RADIUS_M) break;
            j += 1;
        }
        if (j > i && j < locations.length) {
            const spanMs = locationTimeMs(locations[j]) - locationTimeMs(locations[i]);
            let allFar = spanMs <= MAX_ISLAND_SPAN_MS;
            if (allFar) {
                for (let k = i; k < j; k++) {
                    const dk = haversineMeters(
                        anchor.latitude,
                        anchor.longitude,
                        locations[k].latitude,
                        locations[k].longitude,
                    );
                    if (dk < ISLAND_MIN_JUMP_M) {
                        allFar = false;
                        break;
                    }
                }
            }
            if (allFar) {
                i = j;
                continue;
            }
        }
        out.push(locations[i]);
        i += 1;
    }
    return out;
}

/** Точки для линии трека: без выбросов, стоянки — одна точка на группу. */
export function buildCleanTrackPolyline(
    locations: Location[],
    stationaryRadiusM: number = STAY_CLUSTER_RADIUS_M,
): [number, number][] {
    const cleaned = filterTrackOutliers(locations);
    if (cleaned.length === 0) return [];
    if (cleaned.length === 1) return [[cleaned[0].latitude, cleaned[0].longitude]];
    return buildLocationClusters(cleaned, stationaryRadiusM).map(
        c => [c.centerLat, c.centerLng] as [number, number]
    );
}

export function sortLocationsByCreatedAsc(locations: Location[]): Location[] {
    return [...locations].sort(
        (a, b) => locationEffectiveAtMs(a) - locationEffectiveAtMs(b)
    );
}

/** Радиус стоянки на карте: GPS-дрейф дома/офиса */
export const STAY_CLUSTER_RADIUS_M = 100;
/** Разрыв во времени — новая стоянка, даже если координаты те же (уехал и вернулся) */
export const STAY_MAX_GAP_MS = 45 * 60 * 1000;
/** Минимальная длительность, чтобы показать как «стоянку», а не точку проезда */
export const MIN_STAY_DURATION_SEC = 5 * 60;
/** Сколько часов подгружать до начала периода, чтобы стоянка не обрезалась в 00:00 */
export const STAY_PERIOD_LOOKBACK_HOURS = 48;

export interface LocationCluster {
    points: Location[];
    representative: Location;
    centerLat: number;
    centerLng: number;
    fromMs: number;
    toMs: number;
}

function normalizeLocations(locations: Location[]): Location[] {
    return sortLocationsByCreatedAsc(locations)
        .map(loc => ({
            ...loc,
            latitude: Number(loc.latitude),
            longitude: Number(loc.longitude),
        }))
        .filter(loc => Number.isFinite(loc.latitude) && Number.isFinite(loc.longitude));
}

function clusterCenter(points: Location[]): { lat: number; lng: number } {
    const lat = points.reduce((s, p) => s + p.latitude, 0) / points.length;
    const lng = points.reduce((s, p) => s + p.longitude, 0) / points.length;
    return { lat, lng };
}

function isNearCluster(point: Location, cluster: Location[], radiusMeters: number): boolean {
    return cluster.some(
        p => haversineMeters(p.latitude, p.longitude, point.latitude, point.longitude) <= radiusMeters,
    );
}

export function clusterDurationMs(cluster: LocationCluster): number {
    return Math.max(0, cluster.toMs - cluster.fromMs);
}

export function clusterDurationSeconds(cluster: LocationCluster): number {
    return Math.round(clusterDurationMs(cluster) / 1000);
}

function formatMinskDateTimeLocalInput(ms: number): string {
    const p = new Intl.DateTimeFormat('sv-SE', {
        timeZone: 'Europe/Minsk',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
    })
        .formatToParts(new Date(ms))
        .reduce<Record<string, string>>((acc, x) => {
            if (x.type !== 'literal') acc[x.type] = x.value;
            return acc;
        }, {});
    return `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}`;
}

/** datetime-local «с» для API: на N часов раньше (Europe/Minsk). */
export function minskDateTimeHoursBefore(value: string, hours: number): string {
    return formatMinskDateTimeLocalInput(minskLocalToMs(value) - hours * 3_600_000);
}

export function clusterOverlapsPeriod(cluster: LocationCluster, from: string, to: string): boolean {
    const fromMs = minskLocalToMs(from);
    const toMs = periodEndInclusiveMs(to);
    if (!Number.isFinite(fromMs) || !Number.isFinite(toMs)) return true;
    return cluster.toMs >= fromMs && cluster.fromMs <= toMs;
}

export function clusterStartedBeforePeriod(cluster: LocationCluster, from: string): boolean {
    const fromMs = minskLocalToMs(from);
    return Number.isFinite(fromMs) && cluster.fromMs < fromMs;
}

/** Стоянка (не одиночная точка проезда) */
export function isSignificantStay(cluster: LocationCluster): boolean {
    return cluster.points.length >= 2 || clusterDurationSeconds(cluster) >= MIN_STAY_DURATION_SEC;
}

function buildClusterMeta(points: Location[]): Omit<LocationCluster, 'points'> {
    const { lat, lng } = clusterCenter(points);
    const fromMs = locationEffectiveAtMs(points[0]);
    const toMs = locationEffectiveAtMs(points[points.length - 1]);
    return {
        centerLat: lat,
        centerLng: lng,
        fromMs,
        toMs,
        representative: {
            ...points[0],
            latitude: lat,
            longitude: lng,
        },
    };
}

/**
 * Группирует GPS-точки в стоянки: любая точка кластера в радиусе radiusMeters,
 * разрыв по времени > STAY_MAX_GAP_MS начинает новую стоянку.
 */
export function buildLocationClusters(
    locations: Location[],
    radiusMeters: number = STAY_CLUSTER_RADIUS_M,
): LocationCluster[] {
    const sorted = normalizeLocations(locations);
    if (sorted.length === 0) return [];

    const clusters: LocationCluster[] = [];
    let current: Location[] = [sorted[0]];

    const flush = () => {
        if (current.length === 0) return;
        clusters.push({ points: current, ...buildClusterMeta(current) });
        current = [];
    };

    for (let i = 1; i < sorted.length; i++) {
        const point = sorted[i];
        const last = current[current.length - 1];
        const gapMs = locationEffectiveAtMs(point) - locationEffectiveAtMs(last);

        if (gapMs > STAY_MAX_GAP_MS) {
            flush();
            current = [point];
            continue;
        }

        if (isNearCluster(point, current, radiusMeters)) {
            current.push(point);
        } else {
            flush();
            current = [point];
        }
    }
    flush();
    return clusters;
}

/** Упрощённый список: по одной точке на каждую группу стоянки. */
export function mergeLocationsByProximityAnchorFirst(
    locations: Location[],
    radiusMeters: number
): Location[] {
    return buildLocationClusters(locations, radiusMeters).map(c => c.representative);
}

/** Календарная дата YYYY-MM-DD в часовом поясе Europe/Minsk */
export function calendarDateInMinsk(baseDate: Date, dayOffset: number): string {
    const shifted = new Date(baseDate.getTime() + dayOffset * 86400000);
    return shifted.toLocaleDateString('sv-SE', { timeZone: 'Europe/Minsk' });
}

/**
 * Границы календарного дня в Europe/Minsk без смещения в строке —
 * бэкенд парсит такой формат как локальное время Минска.
 */
export function minskDayBounds(dayOffset: number): { from: string; to: string } {
    const d = calendarDateInMinsk(new Date(), dayOffset);
    return {
        from: `${d}T00:00`,
        to: `${d}T23:59`,
    };
}

/** Типовая «смена» в тот же календарный день (Минск) */
export function minskShiftBounds(dayOffset: number): { from: string; to: string } {
    const d = calendarDateInMinsk(new Date(), dayOffset);
    return {
        from: `${d}T08:00`,
        to: `${d}T20:00`,
    };
}

/** Europe/Minsk — постоянный UTC+3 */
const MINSK_OFFSET = '+03:00';

/** Парсит YYYY-MM-DDTHH:mm как локальное время Минска (мс с эпохи). */
export function minskLocalToMs(value: string): number {
    if (!value) return NaN;
    if (/[Z+]/.test(value)) return new Date(value).getTime();
    return new Date(`${value}:00${MINSK_OFFSET}`).getTime();
}

export function compareMinskDateTimes(from: string, to: string): number {
    return minskLocalToMs(from) - minskLocalToMs(to);
}

/** Конец интервала с учётом полной минуты (как на бэкенде). */
export function periodEndInclusiveMs(to: string): number {
    const ms = minskLocalToMs(to);
    if (!Number.isFinite(ms)) return NaN;
    if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(to)) {
        return ms + 59_999;
    }
    return ms;
}

export function locationCreatedAtMs(value: string | undefined): number {
    if (!value) return NaN;
    return minskLocalToMs(value);
}

/** Время для трека и периода: captured_at или created_at. */
export function locationEffectiveAtMs(loc: Location): number {
    const captured = loc.captured_at ? locationCreatedAtMs(loc.captured_at) : NaN;
    if (Number.isFinite(captured)) return captured;
    return locationCreatedAtMs(loc.created_at);
}

const MIN_VALID_LOCATION_MS = Date.UTC(2020, 0, 1);

/** Точка с валидными координатами и нормальной меткой времени (не «архив»/синтетика). */
export function isValidMapLocation(loc: Location): boolean {
    const lat = Number(loc.latitude);
    const lng = Number(loc.longitude);
    if (!Number.isFinite(lat) || !Number.isFinite(lng)) return false;
    const t = locationEffectiveAtMs(loc);
    if (!Number.isFinite(t) || t < MIN_VALID_LOCATION_MS) return false;
    // Синтетические «значимые» точки с сервера без id
    if (loc.id === 0) return false;
    return true;
}

/** Оставляет только точки, попадающие в интервал (Europe/Minsk). */
export function filterLocationsInPeriod(
    locations: Location[],
    from: string,
    to: string
): Location[] {
    if (!from || !to) return locations.filter(isValidMapLocation);
    const fromMs = minskLocalToMs(from);
    const toMs = periodEndInclusiveMs(to);
    if (!Number.isFinite(fromMs) || !Number.isFinite(toMs)) {
        return locations.filter(isValidMapLocation);
    }
    return locations.filter(loc => {
        if (!isValidMapLocation(loc)) return false;
        const t = locationEffectiveAtMs(loc);
        return t >= fromMs && t <= toMs;
    });
}

/** «Сейчас» и N часов назад в формате API (Europe/Minsk). */
export function minskNowRange(hoursBack: number): { from: string; to: string } {
    const now = Date.now();
    const fromMs = now - hoursBack * 3600000;
    const fmt = (ms: number) => {
        const p = new Intl.DateTimeFormat('sv-SE', {
            timeZone: 'Europe/Minsk',
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            hour12: false,
        })
            .formatToParts(new Date(ms))
            .reduce<Record<string, string>>((acc, x) => {
                if (x.type !== 'literal') acc[x.type] = x.value;
                return acc;
            }, {});
        return `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}`;
    };
    return { from: fmt(fromMs), to: fmt(now) };
}
