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

/**
 * Точки в пределах radiusMeters от якоря (первая точка «стояния») скрываются;
 * при выходе за радиус якорь переносится — координаты не усредняются.
 */
export function mergeLocationsByProximityAnchorFirst(
    locations: Location[],
    radiusMeters: number
): Location[] {
    const sorted = sortLocationsByCreatedAsc(locations)
        .map(loc => ({
            ...loc,
            latitude: Number(loc.latitude),
            longitude: Number(loc.longitude),
        }))
        .filter(loc => Number.isFinite(loc.latitude) && Number.isFinite(loc.longitude));
    if (sorted.length === 0) return [];
    const out: Location[] = [];
    let anchorIdx = 0;
    out.push(sorted[0]);
    for (let i = 1; i < sorted.length; i++) {
        const anchor = sorted[anchorIdx];
        const cur = sorted[i];
        if (
            haversineMeters(anchor.latitude, anchor.longitude, cur.latitude, cur.longitude) <=
            radiusMeters
        ) {
            continue;
        }
        anchorIdx = i;
        out.push(cur);
    }
    return out;
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
