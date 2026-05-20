import type { Location } from '../types/models';

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

export function sortLocationsByCreatedAsc(locations: Location[]): Location[] {
    return [...locations].sort(
        (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
    );
}

export interface LocationCluster {
    points: Location[];
    /** Точка для маркера (центр скопления, время — первое в группе) */
    representative: Location;
    centerLat: number;
    centerLng: number;
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

/**
 * Группирует подряд идущие GPS-точки в радиусе radiusMeters (стоянка / дрожание сигнала).
 * Для отображения — одна метка на группу и линия только между группами.
 */
export function buildLocationClusters(locations: Location[], radiusMeters: number): LocationCluster[] {
    const sorted = normalizeLocations(locations);
    if (sorted.length === 0) return [];

    const clusters: LocationCluster[] = [];
    let current: Location[] = [sorted[0]];

    const flush = () => {
        if (current.length === 0) return;
        const { lat, lng } = clusterCenter(current);
        clusters.push({
            points: current,
            centerLat: lat,
            centerLng: lng,
            representative: {
                ...current[0],
                latitude: lat,
                longitude: lng,
            },
        });
        current = [];
    };

    for (let i = 1; i < sorted.length; i++) {
        const point = sorted[i];
        const { lat, lng } = clusterCenter(current);
        if (haversineMeters(lat, lng, point.latitude, point.longitude) <= radiusMeters) {
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
