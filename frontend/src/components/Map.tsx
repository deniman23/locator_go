import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
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
import { formatDateTime, formatPeriodRange } from '../utils/dateFormat';
import {
    compareMinskDateTimes,
    filterLocationsInPeriod,
    isValidMapLocation,
    locationTrackSortAtMs,
    minskDateTimeHoursBefore,
    minskDayBounds,
    STAY_PERIOD_LOOKBACK_HOURS,
} from '../utils/locationTrack';
import { MAP_TILE_ATTRIBUTION, MAP_TILE_MAX_ZOOM, MAP_TILE_URL } from '../utils/mapTiles';

type RoutePoint = Location & { isLast?: boolean };

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
    fitNonce,
}: {
    checkpoints: Checkpoint[];
    locations: Location[];
    extraLatLng: [number, number][];
    fitNonce: number;
}) => {
    const map = useMap();
    const geometryRef = useRef({ checkpoints, locations, extraLatLng });
    geometryRef.current = { checkpoints, locations, extraLatLng };

    useEffect(() => {
        if (fitNonce === 0) return;
        const { checkpoints: cps, locations: locs, extraLatLng: extra } = geometryRef.current;
        const pts: [number, number][] = [
            ...cps.map(p => [p.latitude, p.longitude] as [number, number]),
            ...locs.map(p => [p.latitude, p.longitude] as [number, number]),
            ...extra,
        ];
        if (!pts.length) return;
        const bounds = L.latLngBounds(pts);
        if (bounds.isValid()) {
            map.fitBounds(bounds, { padding: [50, 50], maxZoom: 16 });
        }
    }, [map, fitNonce]);
    return null;
};

const MapComponent: React.FC = () => {
    const { apiKey, user } = useAuth();
    const [searchParams, setSearchParams] = useSearchParams();
    const routeParamsApplied = useRef(false);
    /** Survives until getAll applies so deep-link uid is not lost if selection state is still []. */
    const pendingDeepLinkUserIdRef = useRef<number | null>(null);
    const lastDeepLinkKeyRef = useRef<string | null>(null);
    const day0 = useMemo(() => minskDayBounds(0), []);
    const [checkpoints, setCheckpoints] = useState<Checkpoint[]>([]);
    const [userLocations, setUserLocations] = useState<Location[]>([]);
    const [allUsers, setAllUsers] = useState<User[]>([]);
    const [selectedUserIds, setSelectedUserIds] = useState<number[]>([]);
    const [userSearch, setUserSearch] = useState('');
    const [loading, setLoading] = useState(true);
    const [sidebarOpen, setSidebarOpen] = useState(() => window.innerWidth > 768);
    const [fitNonce, setFitNonce] = useState(0);
    const [error, setError] = useState<string | null>(null);
    /** По умолчанию — календарный сегодняшний день (Europe/Minsk) */
    const [fromTime, setFromTime] = useState(day0.from);
    const [toTime, setToTime] = useState(day0.to);

    const [useRoadMatch, setUseRoadMatch] = useState(true);
    const [roadCoords, setRoadCoords] = useState<[number, number][] | null>(null);
    const [roadSegments, setRoadSegments] = useState<[number, number][][]>([]);
    const [roadMatchNote, setRoadMatchNote] = useState<string | null>(null);
    const [routeLoading, setRouteLoading] = useState(false);
    /** Увеличивается после «Показать трек», чтобы заново запросить линию OSRM */
    const [matchedRouteNonce, setMatchedRouteNonce] = useState(0);
    const [trackRequested, setTrackRequested] = useState(false);
    const [usersLoadError, setUsersLoadError] = useState<string | null>(null);

    const filtersRef = useRef({ fromTime: day0.from, toTime: day0.to });
    const apiKeyRef = useRef<string | null>(apiKey);
    const fetchGenerationRef = useRef(0);
    const fetchDataRef = useRef<(() => Promise<void>) | null>(null);

    useLayoutEffect(() => {
        filtersRef.current = { fromTime, toTime };
    }, [fromTime, toTime]);

    useEffect(() => {
        setTrackRequested(false);
    }, [fromTime, toTime]);

    useEffect(() => {
        apiKeyRef.current = apiKey;
    }, [apiKey]);

    useEffect(() => {
        if (!apiKey) return;
        userApi
            .getAll(apiKey)
            .then(users => {
                setUsersLoadError(null);
                setAllUsers(users);
                // Do not overwrite Visits→Map deep-link selection when getAll finishes later.
                setSelectedUserIds(prev => {
                    const pending = pendingDeepLinkUserIdRef.current;
                    if (pending != null && users.some(u => u.id === pending)) {
                        pendingDeepLinkUserIdRef.current = null;
                        return [pending];
                    }
                    if (routeParamsApplied.current) {
                        const kept = prev.filter(id => users.some(u => u.id === id));
                        return kept.length > 0 ? kept : prev;
                    }
                    return users.map(u => u.id);
                });
            })
            .catch(err => {
                console.error('Не удалось загрузить пользователей:', err);
                setUsersLoadError('Не удалось загрузить список сотрудников');
            });
    }, [apiKey]);

    const fetchData = useCallback(async () => {
        const currentApiKey = apiKeyRef.current;
        const currentFilters = filtersRef.current;
        const fetchGen = ++fetchGenerationRef.current;

        if (!currentApiKey) {
            setError('Ошибка авторизации: отсутствует API ключ');
            setLoading(false);
            return;
        }

        if (!currentFilters.fromTime || !currentFilters.toTime) {
            setError('Укажите период (начало и конец)');
            setLoading(false);
            return;
        }

        if (compareMinskDateTimes(currentFilters.fromTime, currentFilters.toTime) >= 0) {
            setError('Дата начала должна быть раньше даты окончания');
            setLoading(false);
            return;
        }

        try {
            setLoading(true);
            setError(null);

            const cpRes = await checkpointApi.getAll(currentApiKey);
            if (fetchGen !== fetchGenerationRef.current) return;
            setCheckpoints(cpRes.data);

            const fetchFrom = minskDateTimeHoursBefore(
                currentFilters.fromTime,
                STAY_PERIOD_LOOKBACK_HOURS,
            );
            const locRes = await locationApi.getBetween(
                fetchFrom,
                currentFilters.toTime,
                currentApiKey,
                { raw: true }
            );
            if (fetchGen !== fetchGenerationRef.current) return;

            const locs = filterLocationsInPeriod(
                locRes.data || [],
                fetchFrom,
                currentFilters.toTime
            );
            setUserLocations(locs);

            if (
                !localStorage.getItem('mapPosition') &&
                !localStorage.getItem('mapLoaded') &&
                (cpRes.data.length || (locRes.data && locRes.data.length))
            ) {
                setFitNonce(n => n + 1);
                localStorage.setItem('mapLoaded', 'true');
            }
        } catch (e: unknown) {
            if (fetchGen !== fetchGenerationRef.current) return;
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
            if (fetchGen === fetchGenerationRef.current) {
                setLoading(false);
            }
        }
    }, []);

    fetchDataRef.current = fetchData;

    useEffect(() => {
        fetchData();
    }, [fetchData]);

    useEffect(() => {
        const userIdStr = searchParams.get('user_id');
        const from = searchParams.get('from');
        const to = searchParams.get('to');
        if (!userIdStr || !from || !to) {
            lastDeepLinkKeyRef.current = null;
            return;
        }

        const linkKey = `${userIdStr}|${from}|${to}`;
        if (lastDeepLinkKeyRef.current === linkKey) return;

        const uid = parseInt(userIdStr, 10);
        if (Number.isNaN(uid)) return;

        lastDeepLinkKeyRef.current = linkKey;
        routeParamsApplied.current = true;
        pendingDeepLinkUserIdRef.current = uid;
        setSelectedUserIds([uid]);
        setFromTime(from);
        setToTime(to);
        filtersRef.current = { fromTime: from, toTime: to };
        localStorage.removeItem('mapPosition');
        setSearchParams({}, { replace: true });

        void (async () => {
            await fetchDataRef.current?.();
            setTrackRequested(true);
            setFitNonce(n => n + 1);
        })();
    }, [searchParams, setSearchParams]);

    useEffect(() => {
        if (selectedUserIds.length !== 1 && useRoadMatch) {
            setUseRoadMatch(false);
        }
    }, [selectedUserIds, useRoadMatch]);

    useEffect(() => {
        let cancelled = false;
        setRoadCoords(null);
        setRoadSegments([]);
        setRoadMatchNote(null);

        if (
            !trackRequested ||
            !useRoadMatch ||
            selectedUserIds.length !== 1 ||
            !fromTime ||
            !toTime ||
            !apiKey
        ) {
            return () => {
                cancelled = true;
            };
        }

        if (compareMinskDateTimes(fromTime, toTime) >= 0) {
            return () => {
                cancelled = true;
            };
        }

        const uid = selectedUserIds[0];
        (async () => {
            try {
                const res = await locationApi.getMatchedRoute(uid, fromTime, toTime, apiKey);
                if (!cancelled) {
                    const segments = (res.data.segments || [])
                        .map(seg => seg.filter(pt => pt.length >= 2))
                        .filter(seg => seg.length >= 2);
                    setRoadSegments(segments);
                    setRoadCoords(
                        segments.length > 0
                            ? segments.flat()
                            : res.data.coordinates || []
                    );
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
    }, [trackRequested, useRoadMatch, selectedUserIds, fromTime, toTime, apiKey, matchedRouteNonce]);

    const getSavedPosition = () => {
        const saved = localStorage.getItem('mapPosition');
        if (saved) {
            try {
                return JSON.parse(saved);
            } catch {
                return { lat: 53.9, lng: 27.57, zoom: 11 };
            }
        }
        return { lat: 53.9, lng: 27.57, zoom: 11 };
    };
    const initialPosition = getSavedPosition();

    const visibleLocations = useMemo(() => {
        let locs = userLocations;
        if (fromTime && toTime && compareMinskDateTimes(fromTime, toTime) < 0) {
            locs = filterLocationsInPeriod(locs, fromTime, toTime);
        } else {
            locs = locs.filter(isValidMapLocation);
        }
        return locs.filter(loc => selectedUserIds.includes(loc.user_id));
    }, [userLocations, selectedUserIds, fromTime, toTime]);

    /** Все GPS-точки за период по времени (как в базе, без скрытия «выбросов»). */
    const routeByUser = useMemo(() => {
        const m = new Map<number, RoutePoint[]>();
        for (const uid of selectedUserIds) {
            const locs = visibleLocations
                .filter(l => l.user_id === uid)
                .sort(
                    (a, b) =>
                        locationTrackSortAtMs(a) - locationTrackSortAtMs(b) || a.id - b.id,
                );
            if (!locs.length) {
                m.set(uid, []);
                continue;
            }
            const withLast = locs.map((loc, i) => ({
                ...loc,
                isLast: i === locs.length - 1,
            }));
            m.set(uid, withLast);
        }
        return m;
    }, [visibleLocations, selectedUserIds]);

    const routePoints = useMemo(
        () => [...routeByUser.values()].flat(),
        [routeByUser],
    );

    const gpsPolylinesByUser = useMemo(() => {
        const m = new Map<number, [number, number][]>();
        for (const [uid, locs] of routeByUser.entries()) {
            if (locs.length >= 2) {
                m.set(uid, locs.map(l => [l.latitude, l.longitude] as [number, number]));
            }
        }
        return m;
    }, [routeByUser]);

    const extraLatLngForBounds = useMemo(() => {
        const pts: [number, number][] = [];
        for (const line of gpsPolylinesByUser.values()) {
            pts.push(...line);
        }
        if (useRoadMatch && roadCoords) {
            pts.push(...roadCoords);
        }
        return pts;
    }, [gpsPolylinesByUser, useRoadMatch, roadCoords]);

    const filteredUsersList = useMemo(() => {
        const q = userSearch.trim().toLowerCase();
        if (!q) return allUsers;
        return allUsers.filter(u => u.name.toLowerCase().includes(q));
    }, [allUsers, userSearch]);

    const handleShowTrack = async () => {
        if (!fromTime || !toTime) {
            setError('Укажите период (начало и конец)');
            return;
        }
        if (compareMinskDateTimes(fromTime, toTime) >= 0) {
            setError('Дата начала должна быть раньше даты окончания');
            return;
        }
        setRouteLoading(true);
        setError(null);
        try {
            await fetchData();
            setTrackRequested(true);
            setFitNonce(n => n + 1);
            localStorage.removeItem('mapPosition');
            if (useRoadMatch && selectedUserIds.length === 1) {
                setMatchedRouteNonce(n => n + 1);
            }
        } finally {
            setRouteLoading(false);
        }
    };

    const handleFitMap = () => {
        setFitNonce(n => n + 1);
        localStorage.removeItem('mapPosition');
    };

    if (loading && !checkpoints.length && !userLocations.length) {
        return <div className="loading-message">Загрузка карты...</div>;
    }

    return (
        <div className="map-dashboard">
            <button
                type="button"
                className="map-filters-toggle"
                onClick={() => setSidebarOpen((open) => !open)}
            >
                {sidebarOpen ? 'Скрыть фильтры' : 'Фильтры'}
            </button>
            <aside className={sidebarOpen ? 'map-sidebar open' : 'map-sidebar'}>
                <div className="sidebar-content">
                    <div className="filter-section map-period-section">
                        <div className="section-header">
                            <h3>Период</h3>
                        </div>
                        <div className="time-inputs map-time-inputs-compact">
                            <div className="input-group">
                                <label htmlFor="map-from">С</label>
                                <input
                                    id="map-from"
                                    type="datetime-local"
                                    value={fromTime}
                                    onChange={e => setFromTime(e.target.value)}
                                    max={toTime || undefined}
                                />
                            </div>
                            <div className="input-group">
                                <label htmlFor="map-to">По</label>
                                <input
                                    id="map-to"
                                    type="datetime-local"
                                    value={toTime}
                                    onChange={e => setToTime(e.target.value)}
                                    min={fromTime || undefined}
                                />
                            </div>
                        </div>

                        {fromTime && toTime && (
                            <p className="map-period-summary">{formatPeriodRange(fromTime, toTime)}</p>
                        )}

                        {fromTime && toTime && compareMinskDateTimes(fromTime, toTime) >= 0 && (
                            <p className="map-inline-error" role="alert">
                                Укажите «с» раньше, чем «по».
                            </p>
                        )}

                        <button
                            type="button"
                            className="btn-primary-apply"
                            disabled={routeLoading}
                            onClick={() => void handleShowTrack()}
                        >
                            {routeLoading ? 'Загрузка…' : 'Показать трек'}
                        </button>
                        <button
                            type="button"
                            className="btn-secondary"
                            style={{ width: '100%', marginTop: 8 }}
                            onClick={handleFitMap}
                        >
                            Подогнать карту
                        </button>
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
                        {usersLoadError && (
                            <p className="map-inline-error" role="alert">{usersLoadError}</p>
                        )}
                        {!usersLoadError && allUsers.length === 0 && !loading && (
                            <div className="empty-state">
                                <p className="empty-state-title">Нет сотрудников</p>
                                <p className="empty-state-hint">Создайте пользователя в разделе «Пользователи».</p>
                            </div>
                        )}
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
                            <h3>Маршрут</h3>
                        </div>
                        <p className="map-route-hint">
                            Линия и точки — все GPS-записи за выбранный период по времени.
                            Крупная точка — последняя запись в периоде.
                            При одном сотруднике линия по дорогам строится через OSRM.
                        </p>
                        <label className="map-control-row">
                            <input
                                type="checkbox"
                                checked={useRoadMatch}
                                disabled={selectedUserIds.length !== 1}
                                onChange={e => setUseRoadMatch(e.target.checked)}
                            />
                            <span>Привязка к дорогам (один сотрудник)</span>
                        </label>
                        {roadMatchNote && <div className="map-road-note">{roadMatchNote}</div>}
                    </div>
                </div>
            </aside>

            <div className="map-main with-sidebar">
                {error && (
                    <div className="map-error-banner">
                        <span>{error}</span>
                        <button type="button" onClick={() => setError(null)}>
                            ✕
                        </button>
                    </div>
                )}

                {trackRequested && !routeLoading && routePoints.length === 0 && (
                    <div className="map-empty-overlay" role="status">
                        <p className="empty-state-title">Нет GPS-точек за период</p>
                        <p className="empty-state-hint">
                            Расширьте интервал, проверьте сотрудника или устройство.
                        </p>
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
                        locations={routePoints}
                        extraLatLng={extraLatLngForBounds}
                        fitNonce={fitNonce}
                    />

                    <TileLayer
                        url={MAP_TILE_URL}
                        attribution={MAP_TILE_ATTRIBUTION}
                        maxZoom={MAP_TILE_MAX_ZOOM}
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

                    {[...gpsPolylinesByUser.entries()].map(([uid, positions]) => (
                            <Polyline
                                key={`gps-${uid}`}
                                positions={positions}
                                pathOptions={{
                                    color: getUserColor(uid),
                                    weight: 4,
                                    opacity: useRoadMatch && roadCoords && roadCoords.length ? 0.35 : 0.9,
                                }}
                            />
                        ))}

                    {useRoadMatch &&
                        roadSegments.length > 0 &&
                        selectedUserIds.length === 1 &&
                        roadSegments.map((segment, index) => (
                            <Polyline
                                key={`road-match-${index}`}
                                positions={segment}
                                pathOptions={{
                                    color: getUserColor(selectedUserIds[0]),
                                    weight: 5,
                                    opacity: 0.9,
                                }}
                            />
                        ))}

                    {useRoadMatch &&
                        roadSegments.length === 0 &&
                        roadCoords &&
                        roadCoords.length > 1 &&
                        selectedUserIds.length === 1 && (
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

                    {routePoints.map((loc, index) => {
                            const color = getUserColor(loc.user_id);
                            const userName =
                                allUsers.find(u => u.id === loc.user_id)?.name || `ID: ${loc.user_id}`;
                            const radius = loc.isLast ? 9 : 5;
                            const markerKey = `${loc.user_id}-${loc.id}-${index}`;
                            return (
                                <CircleMarker
                                    key={markerKey}
                                    center={[loc.latitude, loc.longitude]}
                                    radius={radius}
                                    pathOptions={{
                                        color,
                                        fillColor: color,
                                        fillOpacity: loc.isLast ? 0.95 : 0.7,
                                        weight: loc.isLast ? 3 : 1,
                                    }}
                                >
                                    <Popup>
                                        <div className="popup-content">
                                            <strong>{userName}</strong>
                                            {loc.isLast && (
                                                <div className="popup-meta">Последняя точка за период</div>
                                            )}
                                            <div>{formatDateTime(loc.captured_at ?? loc.created_at)}</div>
                                            {loc.captured_at &&
                                                loc.captured_at !== loc.created_at && (
                                                    <div className="popup-meta">
                                                        На сервере: {formatDateTime(loc.created_at)}
                                                    </div>
                                                )}
                                            <div className="popup-coords">
                                                {loc.latitude.toFixed(6)}, {loc.longitude.toFixed(6)}
                                            </div>
                                            {loc.id > 0 && (
                                                <div className="popup-id">Запись #{loc.id}</div>
                                            )}
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
                            Точек маршрута: <strong>{routePoints.length}</strong>
                            {routePoints.length > 0 && (
                                <>
                                    {' '}
                                    · последняя #{routePoints[routePoints.length - 1].id}
                                </>
                            )}
                        </span>
                        <span className="status-item map-status-period">
                            {fromTime && toTime ? (
                                <>
                                    Период: <strong>{formatPeriodRange(fromTime, toTime)}</strong>
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
                            setFitNonce(n => n + 1);
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
