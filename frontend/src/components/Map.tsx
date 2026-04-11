import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import {
    MapContainer,
    TileLayer,
    Circle,
    CircleMarker,
    Polyline,
    Popup,
    useMap,
    useMapEvents
} from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import L from 'leaflet';
import type { Checkpoint, Location, User } from '../types/models';
import { checkpointApi, locationApi, userApi } from '../services/api';
import { useAuth } from '../context/AuthContext';
import icon from 'leaflet/dist/images/marker-icon.png';
import iconShadow from 'leaflet/dist/images/marker-shadow.png';
import {
    mergeLocationsByProximityAnchorFirst,
    minskDayBounds,
    minskShiftBounds,
    sortLocationsByCreatedAsc
} from '../utils/locationTrack';

const MERGE_RADIUS_M = 5;

const DefaultIcon = L.icon({
    iconUrl: icon,
    shadowUrl: iconShadow,
    iconSize: [25, 41],
    iconAnchor: [12, 41],
});
L.Marker.prototype.options.icon = DefaultIcon;

const getUserColor = (userId: number): string => {
    const hue = (userId * 137.508) % 360;
    return `hsl(${hue}, 70%, 50%)`;
};

const MapPositionSaver = () => {
    const map = useMapEvents({
        moveend: () => {
            const center = map.getCenter();
            const zoom = map.getZoom();
            localStorage.setItem(
                'mapPosition',
                JSON.stringify({ lat: center.lat, lng: center.lng, zoom })
            );
        },
    });
    return null;
};

const MapUpdater = ({
    checkpoints,
    locations,
    extraLatLng,
    shouldFitBounds,
}: {
    checkpoints: Checkpoint[];
    locations: Location[];
    extraLatLng: [number, number][];
    shouldFitBounds: boolean;
}) => {
    const map = useMap();
    useEffect(() => {
        if (!shouldFitBounds) return;
        const pts: [number, number][] = [
            ...checkpoints.map(p => [p.latitude, p.longitude] as [number, number]),
            ...locations.map(p => [p.latitude, p.longitude] as [number, number]),
            ...extraLatLng,
        ];
        if (!pts.length) return;
        const bounds = L.latLngBounds(pts);
        if (bounds.isValid()) {
            map.fitBounds(bounds, { padding: [50, 50], maxZoom: 16 });
        }
    }, [map, checkpoints, locations, extraLatLng, shouldFitBounds]);
    return null;
};

type TrackMode = 'none' | 'polyline' | 'polyline_only';
type MarkerMode = 'all' | 'merged5m';

const MapComponent: React.FC = () => {
    const { apiKey, user } = useAuth();
    const day0 = useMemo(() => minskDayBounds(0), []);
    const [checkpoints, setCheckpoints] = useState<Checkpoint[]>([]);
    const [userLocations, setUserLocations] = useState<Location[]>([]);
    const [allUsers, setAllUsers] = useState<User[]>([]);
    const [selectedUserIds, setSelectedUserIds] = useState<number[]>([]);
    const [userSearch, setUserSearch] = useState('');
    const [loading, setLoading] = useState(true);
    const [shouldFitBounds, setShouldFitBounds] = useState(false);
    const [error, setError] = useState<string | null>(null);
    /** По умолчанию — календарный сегодняшний день (Europe/Minsk) */
    const [fromTime, setFromTime] = useState(day0.from);
    const [toTime, setToTime] = useState(day0.to);
    /** Смещение календарного дня относительно «сегодня» в Минске (кнопки −1 / +1 день) */
    const [minskDayOffset, setMinskDayOffset] = useState(0);
    /** false — даты менялись вручную, подпись «Свой период» */
    const [dayRangeFromPreset, setDayRangeFromPreset] = useState(true);
    const [sidebarOpen, setSidebarOpen] = useState(true);

    const [useRawLocations, setUseRawLocations] = useState(true);
    const [markerMode, setMarkerMode] = useState<MarkerMode>('all');
    const [trackMode, setTrackMode] = useState<TrackMode>('polyline');
    const [useRoadMatch, setUseRoadMatch] = useState(false);
    const [roadCoords, setRoadCoords] = useState<[number, number][] | null>(null);
    const [roadMatchNote, setRoadMatchNote] = useState<string | null>(null);
    const [autoRefresh, setAutoRefresh] = useState(false);
    const [routeLoading, setRouteLoading] = useState(false);
    /** Увеличивается по кнопке «Получить маршрут», чтобы заново запросить линию OSRM */
    const [matchedRouteNonce, setMatchedRouteNonce] = useState(0);

    const filtersRef = useRef({ fromTime: day0.from, toTime: day0.to });
    const apiKeyRef = useRef<string | null>(apiKey);
    const useRawRef = useRef(useRawLocations);
    const fetchDataRef = useRef<(() => Promise<void>) | null>(null);

    useLayoutEffect(() => {
        filtersRef.current = { fromTime, toTime };
    }, [fromTime, toTime]);

    useEffect(() => {
        apiKeyRef.current = apiKey;
    }, [apiKey]);

    useEffect(() => {
        useRawRef.current = useRawLocations;
    }, [useRawLocations]);

    useEffect(() => {
        if (!apiKey) return;
        userApi
            .getAll(apiKey)
            .then(users => {
                setAllUsers(users);
                setSelectedUserIds(users.map(u => u.id));
            })
            .catch(err => console.error('Не удалось загрузить пользователей:', err));
    }, [apiKey]);

    const fetchData = useCallback(async () => {
        const currentApiKey = apiKeyRef.current;
        const currentFilters = filtersRef.current;
        const raw = useRawRef.current;

        if (!currentApiKey) {
            setError('Ошибка авторизации: отсутствует API ключ');
            setLoading(false);
            return;
        }

        try {
            setLoading(true);
            setError(null);

            const cpRes = await checkpointApi.getAll(currentApiKey);
            setCheckpoints(cpRes.data);

            let locRes;
            if (currentFilters.fromTime && currentFilters.toTime) {
                const fromDate = new Date(currentFilters.fromTime);
                const toDate = new Date(currentFilters.toTime);
                if (fromDate >= toDate) {
                    setError('Дата начала должна быть раньше даты окончания');
                    setLoading(false);
                    return;
                }
                locRes = await locationApi.getBetween(
                    currentFilters.fromTime,
                    currentFilters.toTime,
                    currentApiKey,
                    { raw }
                );
            } else {
                locRes = await locationApi.getAll(currentApiKey, { raw });
            }

            setUserLocations(locRes.data || []);

            if (
                !localStorage.getItem('mapPosition') &&
                !localStorage.getItem('mapLoaded') &&
                (cpRes.data.length || (locRes.data && locRes.data.length))
            ) {
                setShouldFitBounds(true);
                localStorage.setItem('mapLoaded', 'true');
            }
        } catch (e: unknown) {
            console.error('[Map] Ошибка загрузки:', e);
            const message =
                typeof e === 'object' &&
                e !== null &&
                'response' in e &&
                typeof (e as { response?: { data?: { error?: string } } }).response?.data?.error ===
                    'string'
                    ? (e as { response?: { data?: { error?: string } } }).response!.data!.error!
                    : e instanceof Error
                      ? e.message
                      : 'Ошибка при загрузке данных';
            setError(message);
        } finally {
            setLoading(false);
        }
    }, []);

    fetchDataRef.current = fetchData;

    useEffect(() => {
        fetchData();
    }, [fetchData]);

    useEffect(() => {
        if (!autoRefresh) return;
        const interval = setInterval(() => {
            fetchDataRef.current?.();
        }, 10000);
        return () => clearInterval(interval);
    }, [autoRefresh]);

    useEffect(() => {
        if (selectedUserIds.length !== 1 && useRoadMatch) {
            setUseRoadMatch(false);
        }
    }, [selectedUserIds, useRoadMatch]);

    useEffect(() => {
        let cancelled = false;
        setRoadCoords(null);
        setRoadMatchNote(null);

        if (
            !useRoadMatch ||
            selectedUserIds.length !== 1 ||
            trackMode === 'none' ||
            !fromTime ||
            !toTime ||
            !apiKey
        ) {
            return () => {
                cancelled = true;
            };
        }

        const fromDate = new Date(fromTime);
        const toDate = new Date(toTime);
        if (fromDate >= toDate) {
            return () => {
                cancelled = true;
            };
        }

        const uid = selectedUserIds[0];
        (async () => {
            try {
                const res = await locationApi.getMatchedRoute(uid, fromTime, toTime, apiKey);
                if (!cancelled) {
                    setRoadCoords(res.data.coordinates || []);
                    setRoadMatchNote(null);
                }
            } catch (e: unknown) {
                if (cancelled) return;
                const msg =
                    typeof e === 'object' &&
                    e !== null &&
                    'response' in e &&
                    typeof (e as { response?: { data?: { error?: string } } }).response?.data?.error ===
                        'string'
                        ? (e as { response?: { data?: { error?: string } } }).response!.data!.error!
                        : e instanceof Error
                          ? e.message
                          : 'Ошибка маршрута по дорогам';
                setRoadCoords(null);
                setRoadMatchNote(msg);
            }
        })();

        return () => {
            cancelled = true;
        };
    }, [useRoadMatch, selectedUserIds, trackMode, fromTime, toTime, apiKey, matchedRouteNonce]);

    const getSavedPosition = () => {
        const saved = localStorage.getItem('mapPosition');
        if (saved) {
            try {
                return JSON.parse(saved);
            } catch {
                return { lat: 55.75, lng: 37.61, zoom: 10 };
            }
        }
        return { lat: 55.75, lng: 37.61, zoom: 10 };
    };
    const initialPosition = getSavedPosition();

    const visibleLocations = useMemo(
        () => userLocations.filter(loc => selectedUserIds.includes(loc.user_id)),
        [userLocations, selectedUserIds]
    );

    const displayMarkers = useMemo(() => {
        if (markerMode === 'all') return visibleLocations;
        const byUser = new Map<number, Location[]>();
        for (const loc of visibleLocations) {
            byUser.set(loc.user_id, [...(byUser.get(loc.user_id) || []), loc]);
        }
        const out: Location[] = [];
        for (const list of byUser.values()) {
            out.push(...mergeLocationsByProximityAnchorFirst(list, MERGE_RADIUS_M));
        }
        return out;
    }, [visibleLocations, markerMode]);

    const gpsPolylinesByUser = useMemo(() => {
        const m = new Map<number, [number, number][]>();
        for (const uid of selectedUserIds) {
            const locs = visibleLocations.filter(l => l.user_id === uid);
            const sorted = sortLocationsByCreatedAsc(locs);
            if (sorted.length >= 2) {
                m.set(uid, sorted.map(l => [l.latitude, l.longitude] as [number, number]));
            }
        }
        return m;
    }, [visibleLocations, selectedUserIds]);

    const extraLatLngForBounds = useMemo(() => {
        const pts: [number, number][] = [];
        if (trackMode !== 'none') {
            for (const line of gpsPolylinesByUser.values()) {
                pts.push(...line);
            }
        }
        if (useRoadMatch && roadCoords) {
            pts.push(...roadCoords);
        }
        return pts;
    }, [trackMode, gpsPolylinesByUser, useRoadMatch, roadCoords]);

    const showMarkers = trackMode !== 'polyline_only';
    const showGpsTrack = trackMode !== 'none';
    const filteredUsersList = useMemo(() => {
        const q = userSearch.trim().toLowerCase();
        if (!q) return allUsers;
        return allUsers.filter(u => u.name.toLowerCase().includes(q));
    }, [allUsers, userSearch]);

    const applyPeriod = () => {
        void fetchData();
    };

    /** Загрузка точек за период, показ трека и подгонка карты; при включённой привязке к дорогам — повторный запрос OSRM */
    const handleGetRoute = async () => {
        if (!fromTime || !toTime) {
            setError('Укажите период (начало и конец)');
            return;
        }
        if (new Date(fromTime) >= new Date(toTime)) {
            setError('Дата начала должна быть раньше даты окончания');
            return;
        }
        setRouteLoading(true);
        setError(null);
        try {
            await fetchData();
            setTrackMode(prev => (prev === 'none' ? 'polyline' : prev));
            setShouldFitBounds(true);
            if (useRoadMatch && selectedUserIds.length === 1) {
                setMatchedRouteNonce(n => n + 1);
            }
        } finally {
            setRouteLoading(false);
        }
    };

    const handleResetTimeFilter = () => {
        setMinskDayOffset(0);
        setDayRangeFromPreset(true);
        const { from, to } = minskDayBounds(0);
        setFromTime(from);
        setToTime(to);
        setTimeout(() => void fetchData(), 100);
    };

    const loadAllTime = () => {
        setFromTime('');
        setToTime('');
        setMinskDayOffset(0);
        setDayRangeFromPreset(true);
        setTimeout(() => void fetchData(), 100);
    };

    const goPrevCalendarDay = () => {
        setDayRangeFromPreset(true);
        if (!fromTime || !toTime) {
            setMinskDayOffset(-1);
            const b = minskDayBounds(-1);
            setFromTime(b.from);
            setToTime(b.to);
            setTimeout(() => void fetchData(), 0);
            return;
        }
        const next = minskDayOffset - 1;
        setMinskDayOffset(next);
        const b = minskDayBounds(next);
        setFromTime(b.from);
        setToTime(b.to);
        setTimeout(() => void fetchData(), 0);
    };

    const goNextCalendarDay = () => {
        setDayRangeFromPreset(true);
        if (!fromTime || !toTime) {
            setMinskDayOffset(1);
            const b = minskDayBounds(1);
            setFromTime(b.from);
            setToTime(b.to);
            setTimeout(() => void fetchData(), 0);
            return;
        }
        const next = minskDayOffset + 1;
        setMinskDayOffset(next);
        const b = minskDayBounds(next);
        setFromTime(b.from);
        setToTime(b.to);
        setTimeout(() => void fetchData(), 0);
    };

    const setLastHour = () => {
        setDayRangeFromPreset(false);
        const now = new Date();
        const hourAgo = new Date(now.getTime() - 60 * 60 * 1000);
        setFromTime(hourAgo.toISOString().slice(0, 16));
        setToTime(now.toISOString().slice(0, 16));
        setTimeout(() => void fetchData(), 100);
    };

    const setLast24Hours = () => {
        setDayRangeFromPreset(false);
        const now = new Date();
        const dayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000);
        setFromTime(dayAgo.toISOString().slice(0, 16));
        setToTime(now.toISOString().slice(0, 16));
        setTimeout(() => void fetchData(), 100);
    };

    const setTodayMinsk = () => {
        setMinskDayOffset(0);
        setDayRangeFromPreset(true);
        const { from, to } = minskDayBounds(0);
        setFromTime(from);
        setToTime(to);
        setTimeout(() => void fetchData(), 100);
    };

    const setYesterdayMinsk = () => {
        setMinskDayOffset(-1);
        setDayRangeFromPreset(true);
        const { from, to } = minskDayBounds(-1);
        setFromTime(from);
        setToTime(to);
        setTimeout(() => void fetchData(), 100);
    };

    const setShiftMinsk = () => {
        setMinskDayOffset(0);
        setDayRangeFromPreset(true);
        const { from, to } = minskShiftBounds(0);
        setFromTime(from);
        setToTime(to);
        setTimeout(() => void fetchData(), 100);
    };

    if (loading && !checkpoints.length && !userLocations.length) {
        return <div className="loading-message">Загрузка карты...</div>;
    }

    return (
        <div className="map-dashboard">
            <button
                className={`sidebar-toggle ${!sidebarOpen ? 'sidebar-closed' : ''}`}
                onClick={() => setSidebarOpen(!sidebarOpen)}
                title={sidebarOpen ? 'Скрыть панель' : 'Показать панель'}
            >
                {sidebarOpen ? '◀' : '▶'}
            </button>

            <aside className={`map-sidebar ${sidebarOpen ? 'open' : 'closed'}`}>
                <div className="sidebar-content">
                    <div className="filter-section">
                        <div className="section-header">
                            <h3>Период и маршрут</h3>
                        </div>
                        <p className="map-hint">
                            При открытии карты по умолчанию загружаются точки за <strong>сегодня</strong> (календарный
                            день Europe/Minsk). Кнопки «− день» / «+ день» сдвигают этот день. «За всё время» —
                            без ограничения по дате.
                        </p>
                        <div className="map-day-nav">
                            <button type="button" className="btn-day-nav" onClick={goPrevCalendarDay} title="Предыдущий день">
                                − день
                            </button>
                            <span className="map-day-label">
                                {!fromTime || !toTime
                                    ? 'Все даты'
                                    : !dayRangeFromPreset
                                      ? 'Свой период'
                                      : minskDayOffset === 0
                                        ? 'Сегодня'
                                        : minskDayOffset === -1
                                          ? 'Вчера'
                                          : `${minskDayOffset > 0 ? '+' : ''}${minskDayOffset} дн.`}
                            </span>
                            <button type="button" className="btn-day-nav" onClick={goNextCalendarDay} title="Следующий день">
                                + день
                            </button>
                        </div>
                        <div className="quick-filters map-quick-row">
                            <button type="button" className="btn-quick" onClick={setTodayMinsk}>
                                Сегодня
                            </button>
                            <button type="button" className="btn-quick" onClick={setYesterdayMinsk}>
                                Вчера
                            </button>
                            <button type="button" className="btn-quick" onClick={setShiftMinsk}>
                                Смена 8–20
                            </button>
                        </div>
                        <div className="quick-filters">
                            <button type="button" className="btn-quick" onClick={setLastHour}>
                                Последний час
                            </button>
                            <button type="button" className="btn-quick" onClick={setLast24Hours}>
                                Последние 24ч
                            </button>
                        </div>

                        <div className="time-inputs">
                            <div className="input-group">
                                <label>Начало периода</label>
                                <input
                                    type="datetime-local"
                                    value={fromTime}
                                    onChange={e => {
                                        setDayRangeFromPreset(false);
                                        setFromTime(e.target.value);
                                    }}
                                    max={toTime || undefined}
                                />
                            </div>
                            <div className="input-group">
                                <label>Конец периода</label>
                                <input
                                    type="datetime-local"
                                    value={toTime}
                                    onChange={e => {
                                        setDayRangeFromPreset(false);
                                        setToTime(e.target.value);
                                    }}
                                    min={fromTime || undefined}
                                />
                            </div>
                        </div>

                        <button type="button" className="btn-primary-apply" onClick={applyPeriod}>
                            Применить период
                        </button>

                        <button
                            type="button"
                            className="btn-get-route"
                            disabled={routeLoading}
                            onClick={() => void handleGetRoute()}
                        >
                            {routeLoading ? 'Загрузка…' : 'Получить маршрут'}
                        </button>
                        <p className="map-hint map-hint-tight">
                            Подтягивает точки за выбранный период, включает линию трека и подгоняет карту. Если отмечено
                            «По дорогам» и выбран один пользователь — дополнительно строится линия OSRM (нужен{' '}
                            <code>ROUTING_BASE_URL</code> на сервере).
                        </p>

                        {fromTime && toTime && new Date(fromTime) < new Date(toTime) && (
                            <div className="filter-status">
                                <span className="status-active">Период задан</span>
                            </div>
                        )}
                        {fromTime && toTime && new Date(fromTime) >= new Date(toTime) && (
                            <div className="filter-status">
                                <span className="status-error">Некорректный период</span>
                            </div>
                        )}

                        <div className="map-period-actions">
                            <button type="button" className="btn-reset" onClick={handleResetTimeFilter}>
                                Только сегодня
                            </button>
                            <button type="button" className="btn-reset" onClick={loadAllTime}>
                                За всё время
                            </button>
                        </div>
                    </div>

                    <div className="filter-section">
                        <div className="section-header">
                            <h3>Пользователи</h3>
                            <div className="section-controls">
                                <button
                                    type="button"
                                    className="btn-mini"
                                    onClick={() => setSelectedUserIds(allUsers.map(u => u.id))}
                                    title="Выбрать всех"
                                >
                                    Все
                                </button>
                                <button
                                    type="button"
                                    className="btn-mini"
                                    onClick={() => setSelectedUserIds([])}
                                    title="Снять выбор"
                                >
                                    Снять
                                </button>
                            </div>
                        </div>
                        <input
                            type="search"
                            className="map-user-search"
                            placeholder="Поиск по имени…"
                            value={userSearch}
                            onChange={e => setUserSearch(e.target.value)}
                        />
                        <div className="user-list">
                            {filteredUsersList.map(u => (
                                <label key={u.id} className="user-item">
                                    <input
                                        type="checkbox"
                                        checked={selectedUserIds.includes(u.id)}
                                        onChange={e => {
                                            if (e.target.checked) {
                                                setSelectedUserIds([...selectedUserIds, u.id]);
                                            } else {
                                                setSelectedUserIds(selectedUserIds.filter(id => id !== u.id));
                                            }
                                        }}
                                    />
                                    <span
                                        className="user-name"
                                        style={{
                                            color: getUserColor(u.id),
                                            fontWeight: selectedUserIds.includes(u.id) ? 'bold' : 'normal',
                                        }}
                                    >
                                        {u.name}
                                    </span>
                                </label>
                            ))}
                        </div>
                    </div>

                    <div className="filter-section">
                        <div className="section-header">
                            <h3>Данные и трек</h3>
                        </div>
                        <label className="map-control-row">
                            <input
                                type="checkbox"
                                checked={useRawLocations}
                                onChange={e => {
                                    const v = e.target.checked;
                                    useRawRef.current = v;
                                    setUseRawLocations(v);
                                    void fetchData();
                                }}
                            />
                            <span>Все точки (без серверного упрощения)</span>
                        </label>
                        <div className="map-control-block">
                            <span className="map-control-label">Маркеры</span>
                            <label className="map-control-row">
                                <input
                                    type="radio"
                                    name="markerMode"
                                    checked={markerMode === 'all'}
                                    onChange={() => setMarkerMode('all')}
                                />
                                <span>Все точки</span>
                            </label>
                            <label className="map-control-row">
                                <input
                                    type="radio"
                                    name="markerMode"
                                    checked={markerMode === 'merged5m'}
                                    onChange={() => setMarkerMode('merged5m')}
                                />
                                <span>Основные (≤{MERGE_RADIUS_M} м от якоря)</span>
                            </label>
                        </div>
                        <div className="map-control-block">
                            <span className="map-control-label">Линия трека</span>
                            <label className="map-control-row">
                                <input
                                    type="radio"
                                    name="trackMode"
                                    checked={trackMode === 'none'}
                                    onChange={() => setTrackMode('none')}
                                />
                                <span>Нет</span>
                            </label>
                            <label className="map-control-row">
                                <input
                                    type="radio"
                                    name="trackMode"
                                    checked={trackMode === 'polyline'}
                                    onChange={() => setTrackMode('polyline')}
                                />
                                <span>Точки + трек</span>
                            </label>
                            <label className="map-control-row">
                                <input
                                    type="radio"
                                    name="trackMode"
                                    checked={trackMode === 'polyline_only'}
                                    onChange={() => setTrackMode('polyline_only')}
                                />
                                <span>Только трек</span>
                            </label>
                        </div>
                        <label className="map-control-row">
                            <input
                                type="checkbox"
                                checked={useRoadMatch}
                                disabled={selectedUserIds.length !== 1}
                                onChange={e => setUseRoadMatch(e.target.checked)}
                            />
                            <span>По дорогам (OSRM), один пользователь</span>
                        </label>
                        {roadMatchNote && <div className="map-road-note">{roadMatchNote}</div>}
                        <label className="map-control-row">
                            <input
                                type="checkbox"
                                checked={autoRefresh}
                                onChange={e => setAutoRefresh(e.target.checked)}
                            />
                            <span>Обновлять каждые 10 с</span>
                        </label>
                    </div>
                </div>
            </aside>

            <div className={`map-main ${sidebarOpen ? 'with-sidebar' : 'full-width'}`}>
                {error && (
                    <div className="map-error-banner">
                        <span>{error}</span>
                        <button type="button" onClick={() => setError(null)}>
                            ✕
                        </button>
                    </div>
                )}

                <MapContainer
                    center={[initialPosition.lat, initialPosition.lng]}
                    zoom={initialPosition.zoom}
                    style={{ height: '100%', width: '100%' }}
                    scrollWheelZoom
                >
                    <MapPositionSaver />
                    <MapUpdater
                        checkpoints={checkpoints}
                        locations={showMarkers ? displayMarkers : []}
                        extraLatLng={extraLatLngForBounds}
                        shouldFitBounds={shouldFitBounds}
                    />

                    <TileLayer
                        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                        attribution="&copy; OpenStreetMap"
                    />

                    {checkpoints.map(cp => (
                        <Circle
                            key={cp.id}
                            center={[cp.latitude, cp.longitude]}
                            radius={cp.radius}
                            pathOptions={{
                                color: '#3b82f6',
                                fillColor: '#3b82f6',
                                fillOpacity: 0.2,
                                weight: 2,
                            }}
                        >
                            <Popup>
                                <div className="popup-content">
                                    <strong>{cp.name}</strong>
                                    <div>Радиус: {cp.radius} м</div>
                                    <div className="popup-id">ID: {cp.id}</div>
                                </div>
                            </Popup>
                        </Circle>
                    ))}

                    {showGpsTrack &&
                        [...gpsPolylinesByUser.entries()].map(([uid, positions]) => (
                            <Polyline
                                key={`gps-${uid}`}
                                positions={positions}
                                pathOptions={{
                                    color: getUserColor(uid),
                                    weight: 3,
                                    opacity: useRoadMatch && roadCoords && roadCoords.length ? 0.35 : 0.85,
                                }}
                            />
                        ))}

                    {useRoadMatch && roadCoords && roadCoords.length > 1 && selectedUserIds.length === 1 && (
                        <Polyline
                            key="road-match"
                            positions={roadCoords}
                            pathOptions={{
                                color: getUserColor(selectedUserIds[0]),
                                weight: 5,
                                opacity: 0.9,
                            }}
                        />
                    )}

                    {showMarkers &&
                        displayMarkers.map(loc => {
                            const color = getUserColor(loc.user_id);
                            const userName =
                                allUsers.find(u => u.id === loc.user_id)?.name || `ID: ${loc.user_id}`;
                            return (
                                <CircleMarker
                                    key={loc.id}
                                    center={[loc.latitude, loc.longitude]}
                                    radius={8}
                                    pathOptions={{
                                        color,
                                        fillColor: color,
                                        fillOpacity: 0.8,
                                        weight: 2,
                                    }}
                                >
                                    <Popup>
                                        <div className="popup-content">
                                            <strong>{userName}</strong>
                                            <div>{new Date(loc.created_at).toLocaleString()}</div>
                                            <div className="popup-coords">
                                                {loc.latitude.toFixed(6)}, {loc.longitude.toFixed(6)}
                                            </div>
                                        </div>
                                    </Popup>
                                </CircleMarker>
                            );
                        })}
                </MapContainer>

                <div className="map-status-bar">
                    <div className="status-info">
                        <span className="status-item">
                            <strong>{user?.name}</strong>
                            {user?.is_admin && <span className="admin-badge">Админ</span>}
                        </span>
                        <span className="status-item">
                            Чекпоинты: <strong>{checkpoints.length}</strong>
                        </span>
                        <span className="status-item">
                            На карте: <strong>{showMarkers ? displayMarkers.length : 0}</strong> / всего выбр.:{' '}
                            <strong>{visibleLocations.length}</strong> (загр. {userLocations.length})
                        </span>
                        <span className="status-item map-status-period">
                            {fromTime && toTime ? (
                                <>
                                    Период: <strong>{fromTime}</strong> — <strong>{toTime}</strong>
                                </>
                            ) : (
                                <strong>Период: за всё время</strong>
                            )}
                        </span>
                    </div>
                    <button
                        type="button"
                        className="btn-reset-map"
                        onClick={() => {
                            localStorage.removeItem('mapPosition');
                            localStorage.removeItem('mapLoaded');
                            setShouldFitBounds(true);
                        }}
                        title="Сбросить положение карты"
                    >
                        Сброс карты
                    </button>
                </div>
            </div>
        </div>
    );
};

export default MapComponent;
