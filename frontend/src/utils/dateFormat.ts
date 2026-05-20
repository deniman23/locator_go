const DISPLAY_LOCALE = 'ru-RU';
const MINSK_TZ = 'Europe/Minsk';

function toDate(value: string | Date): Date {
    if (typeof value !== 'string') return value;
    // Строки API без смещения — локальное время Europe/Minsk (UTC+3)
    if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) {
        return new Date(`${value}:00+03:00`);
    }
    return new Date(value);
}

/** Дата и время для отображения пользователю (Europe/Minsk). */
export function formatDateTime(
    value: string | Date | null | undefined,
    options?: Intl.DateTimeFormatOptions
): string {
    if (value == null || value === '') return '—';
    const date = toDate(value);
    if (Number.isNaN(date.getTime())) return String(value);
    return new Intl.DateTimeFormat(DISPLAY_LOCALE, {
        timeZone: MINSK_TZ,
        day: 'numeric',
        month: 'long',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        ...options,
    }).format(date);
}

/** Только дата без времени. */
export function formatDate(value: string | Date | null | undefined): string {
    if (value == null || value === '') return '—';
    const date = toDate(value);
    if (Number.isNaN(date.getTime())) return String(value);
    return new Intl.DateTimeFormat(DISPLAY_LOCALE, {
        timeZone: MINSK_TZ,
        day: 'numeric',
        month: 'long',
        year: 'numeric',
    }).format(date);
}

/** Диапазон периода для подписей и статус-бара. */
export function formatPeriodRange(from: string, to: string): string {
    if (!from || !to) return 'За всё время';
    return `${formatDateTime(from)} — ${formatDateTime(to)}`;
}

/** ISO / API → значение для input[type=datetime-local] (Europe/Minsk). */
export function toDateTimeLocalInput(value: string | Date | null | undefined): string {
    if (value == null || value === '') return '';
    const date = toDate(value);
    if (Number.isNaN(date.getTime())) return '';
    const p = new Intl.DateTimeFormat('sv-SE', {
        timeZone: MINSK_TZ,
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
    })
        .formatToParts(date)
        .reduce<Record<string, string>>((acc, x) => {
            if (x.type !== 'literal') acc[x.type] = x.value;
            return acc;
        }, {});
    return `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}`;
}
