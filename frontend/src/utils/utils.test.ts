import { describe, expect, it } from 'vitest';
import { formatDuration } from './durationFormat';
import { formatPeriodRange, toDateTimeLocalInput } from './dateFormat';
import { GPS_ONLINE_SECONDS, healthFromResponse } from './userDeviceStatus';

describe('formatDuration', () => {
  it('formats zero seconds', () => {
    expect(formatDuration(0)).toContain('0');
  });

  it('formats hours and minutes', () => {
    expect(formatDuration(3661)).toMatch(/1 час/);
    expect(formatDuration(3661)).toMatch(/1 минута/);
  });

  it('clamps negative to zero seconds wording', () => {
    expect(formatDuration(-5)).toBe('0 секунд');
  });
});

describe('dateFormat', () => {
  it('formatPeriodRange empty → all time', () => {
    expect(formatPeriodRange('', '')).toBe('За всё время');
  });

  it('toDateTimeLocalInput formats minsk local', () => {
    const v = toDateTimeLocalInput('2026-07-01T12:30:00+03:00');
    expect(v).toBe('2026-07-01T12:30');
  });
});

describe('userDeviceStatus', () => {
  it('exposes GPS online threshold of 300s', () => {
    expect(GPS_ONLINE_SECONDS).toBe(300);
  });

  it('healthFromResponse maps API payload', () => {
    const h = healthFromResponse({
      last_report_at: '2026-07-01T10:00:00Z',
      healthy: true,
      app_version: '1.0.25',
      platform: 'android',
      issues: ['battery'],
      report: { ok: true },
    });
    expect(h.healthy).toBe(true);
    expect(h.appVersion).toBe('1.0.25');
    expect(h.issues).toEqual(['battery']);
  });
});
