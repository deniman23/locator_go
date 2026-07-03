/** Читаемая длительность из секунд (рус.) */
export function formatDuration(seconds: number): string {
    if (seconds < 0) return '0 секунд';

    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const remainingSeconds = Math.floor(seconds % 60);

    const parts: string[] = [];

    if (hours > 0) {
        parts.push(`${hours} ${getHourForm(hours)}`);
    }
    if (minutes > 0 || hours > 0) {
        parts.push(`${minutes} ${getMinuteForm(minutes)}`);
    }
    if (remainingSeconds > 0 || (hours === 0 && minutes === 0)) {
        parts.push(`${remainingSeconds} ${getSecondForm(remainingSeconds)}`);
    }

    return parts.join(' ');
}

function getHourForm(hours: number): string {
    if (hours >= 11 && hours <= 19) return 'часов';
    const lastDigit = hours % 10;
    if (lastDigit === 1) return 'час';
    if (lastDigit >= 2 && lastDigit <= 4) return 'часа';
    return 'часов';
}

function getMinuteForm(minutes: number): string {
    if (minutes >= 11 && minutes <= 19) return 'минут';
    const lastDigit = minutes % 10;
    if (lastDigit === 1) return 'минута';
    if (lastDigit >= 2 && lastDigit <= 4) return 'минуты';
    return 'минут';
}

function getSecondForm(seconds: number): string {
    if (seconds >= 11 && seconds <= 19) return 'секунд';
    const lastDigit = seconds % 10;
    if (lastDigit === 1) return 'секунда';
    if (lastDigit >= 2 && lastDigit <= 4) return 'секунды';
    return 'секунд';
}
