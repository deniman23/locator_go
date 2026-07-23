import { describe, expect, it } from 'vitest';
import {
  filterTrackOutliers,
  filterGpsIslands,
  haversineMeters,
  isSignificantStay,
  buildLocationClusters,
  minskDayBounds,
  locationEffectiveAtMs,
  type LocationCluster,
} from './locationTrack';
import type { Location } from '../types/models';

function loc(
  id: number,
  lat: number,
  lng: number,
  createdAt: string,
  capturedAt?: string,
): Location {
  return {
    id,
    user_id: 1,
    latitude: lat,
    longitude: lng,
    created_at: createdAt,
    updated_at: createdAt,
    captured_at: capturedAt,
  };
}

describe('haversineMeters', () => {
  it('returns ~0 for same point', () => {
    expect(haversineMeters(53.9, 27.5, 53.9, 27.5)).toBeLessThan(0.01);
  });

  it('is symmetric', () => {
    const a = haversineMeters(53.9, 27.5, 53.91, 27.51);
    const b = haversineMeters(53.91, 27.51, 53.9, 27.5);
    expect(Math.abs(a - b)).toBeLessThan(0.01);
  });
});

describe('filterTrackOutliers (Go parity)', () => {
  // Mirrors backend/service/track_filter_test.go TestFilterTrackOutliers_offlineBatchJumps
  it('drops offline batch teleport and keeps return home', () => {
    const home = loc(1, 53.88586, 27.51026, '2026-07-01T09:01:07Z', '2026-07-01T08:00:00Z');
    const jump = loc(2, 53.92684, 27.69516, '2026-07-01T09:01:07.130Z', '2026-07-01T08:05:00Z');
    const back = loc(3, 53.88585, 27.51024, '2026-07-01T09:01:07.260Z', '2026-07-01T08:10:00Z');

    const out = filterTrackOutliers([home, jump, back]);
    expect(out).toHaveLength(2);
    expect(out[1].id).toBe(3);
  });

  it('keeps normal drive points', () => {
    const a = loc(1, 53.9, 27.5, '2026-07-01T10:00:00Z');
    const b = loc(2, 53.905, 27.51, '2026-07-01T10:05:00Z');
    expect(filterTrackOutliers([a, b])).toHaveLength(2);
  });
});

describe('filterGpsIslands', () => {
  it('returns short lists unchanged', () => {
    const a = loc(1, 53.9, 27.5, '2026-07-01T10:00:00Z');
    const b = loc(2, 53.91, 27.51, '2026-07-01T10:01:00Z');
    expect(filterGpsIslands([a, b])).toHaveLength(2);
  });
});

describe('clusters / stays', () => {
  it('marks stay of 5+ minutes as significant', () => {
    const cluster: LocationCluster = {
      points: [],
      representative: loc(1, 53.9, 27.5, '2026-07-01T10:00:00Z'),
      centerLat: 53.9,
      centerLng: 27.5,
      fromMs: Date.parse('2026-07-01T10:00:00Z'),
      toMs: Date.parse('2026-07-01T10:10:00Z'),
    };
    expect(isSignificantStay(cluster)).toBe(true);
  });

  it('buildLocationClusters groups nearby points', () => {
    const points = [
      loc(1, 53.9, 27.5, '2026-07-01T10:00:00Z'),
      loc(2, 53.9001, 27.5001, '2026-07-01T10:05:00Z'),
      loc(3, 53.9002, 27.5002, '2026-07-01T10:10:00Z'),
    ];
    const clusters = buildLocationClusters(points);
    expect(clusters.length).toBeGreaterThanOrEqual(1);
  });
});

describe('minsk helpers', () => {
  it('minskDayBounds returns from before to', () => {
    const { from, to } = minskDayBounds(0);
    expect(from < to).toBe(true);
    expect(from).toMatch(/T/);
  });

  it('locationEffectiveAtMs prefers captured_at', () => {
    const l = loc(1, 53.9, 27.5, '2026-07-01T12:00:00Z', '2026-07-01T08:00:00Z');
    expect(locationEffectiveAtMs(l)).toBe(Date.parse('2026-07-01T08:00:00Z'));
  });
});
