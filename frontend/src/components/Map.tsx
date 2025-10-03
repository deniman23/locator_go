import React, { useEffect, useState, useRef } from 'react';
import axios from 'axios';
import {
    MapContainer,
    TileLayer,
    Circle,
    CircleMarker,
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

// Настройка дефолтной иконки Leaflet
const DefaultIcon = L.icon({
    iconUrl: icon,
    shadowUrl: iconShadow,
    iconSize: [25, 41],
    iconAnchor: [12, 41],
});
L.Marker.prototype.options.icon = DefaultIcon;

// Генерация цвета по user_id
const getUserColor = (userId: number): string => {
    const hue = (userId * 137.508) % 360;
    return `hsl(${hue}, 70%, 50%)`;
};

// Сохраняем позицию карты
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

// Авто-подгонка под границы
const MapUpdater = ({
                        checkpoints,
                        locations,
                        shouldFitBounds,
                    }: {
    checkpoints: Checkpoint[];
    locations: Location[];
    shouldFitBounds: boolean;
}) => {
    const map = useMap();
    useEffect(() => {
        if (!shouldFitBounds) return;
        const pts = [...checkpoints, ...locations];
        if (!pts.length) return;
        const bounds = L.latLngBounds(pts.map(p => [p.latitude, p.longitude] as [number, number]));
        if (bounds.isValid()) {
            map.fitBounds(bounds, { padding: [50, 50], maxZoom: 16 });
        }
    }, [map, checkpoints, locations, shouldFitBounds]);
    return null;
};

const MapComponent: React.FC = () => {
    const { apiKey, user } = useAuth();
    const [checkpoints, setCheckpoints] = useState<Checkpoint[]>([]);
    const [userLocations, setUserLocations] = useState<Location[]>([]);
    const [allUsers, setAllUsers] = useState<User[]>([]);
    const [selectedUserIds, setSelectedUserIds] = useState<number[]>([]);
    const [loading, setLoading] = useState(true);
    const [shouldFitBounds, setShouldFitBounds] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [fromTime, setFromTime] = useState<string>('');
    const [toTime, setToTime] = useState<string>('');
    const [sidebarOpen, setSidebarOpen] = useState(true);

    // Используем useRef для хранения актуальных значений
    const filtersRef = useRef({ fromTime: '', toTime: '' });
    const apiKeyRef = useRef<string | null>(apiKey);
    const fetchDataRef = useRef<(() => Promise<void>) | null>(null);

    // Обновляем refs при изменении
    useEffect(() => {
        filtersRef.current = { fromTime, toTime };
    }, [fromTime, toTime]);

    useEffect(() => {
        apiKeyRef.current = apiKey;
    }, [apiKey]);

    // Загрузка всех пользователей
    useEffect(() => {
        if (!apiKey) return;
        userApi.getAll(apiKey)
            .then(users => {
                setAllUsers(users);
                setSelectedUserIds(users.map(u => u.id));
            })
            .catch(err => console.error('Не удалось загрузить пользователей:', err));
    }, [apiKey]);

    // Функция загрузки данных
    const fetchData = async () => {
        const currentApiKey = apiKeyRef.current;
        const currentFilters = filtersRef.current;

        if (!currentApiKey) {
            setError('Ошибка авторизации: отсутствует API ключ');
            setLoading(false);
            return;
        }

        try {
            setLoading(true);
            setError(null);

            // Загрузка чекпоинтов
            const cpRes = await checkpointApi.getAll(currentApiKey);
            setCheckpoints(cpRes.data);

            // Загрузка локаций с учетом фильтров
            let locRes;
            if (currentFilters.fromTime && currentFilters.toTime) {
                const fromDate = new Date(currentFilters.fromTime);
                const toDate = new Date(currentFilters.toTime);

                if (fromDate >= toDate) {
                    setError('Дата начала должна быть раньше даты окончания');
                    setLoading(false);
                    return;
                }
                const fromParam = fromDate.toISOString().replace(/Z$/, '+03:00');
                const toParam   = toDate.toISOString().replace(/Z$/, '+03:00');

                locRes = await axios.get<Location[]>(
                    `/api/location/?from=${encodeURIComponent(fromParam)}&to=${encodeURIComponent(toParam)}`,
                    {
                        headers: {
                            'X-API-Key': currentApiKey,
                            'Content-Type': 'application/json'
                        }
                    }
                );
            } else {
                locRes = await locationApi.getAll(currentApiKey);
            }

            setUserLocations(locRes.data || []);

            // Первоначальная подгонка карты
            if (!localStorage.getItem('mapPosition') && !localStorage.getItem('mapLoaded') &&
                (cpRes.data.length || (locRes.data && locRes.data.length))) {
                setShouldFitBounds(true);
                localStorage.setItem('mapLoaded', 'true');
            }
        } catch (e: any) {
            console.error('[Map] Ошибка загрузки:', e);
            setError(e.response?.data?.error || e.message || 'Ошибка при загрузке данных');
        } finally {
            setLoading(false);
        }
    };

    // Сохраняем функцию в ref
    fetchDataRef.current = fetchData;

    // Первоначальная загрузка
    useEffect(() => {
        fetchData();
    }, []);

    // Автообновление каждые 10 секунд
    useEffect(() => {
        const interval = setInterval(() => {
            if (fetchDataRef.current) {
                fetchDataRef.current();
            }
        }, 10000);

        return () => clearInterval(interval);
    }, []);

    // Получаем сохранённую позицию
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

    // Отфильтрованные локации
    const visibleLocations = userLocations.filter(loc =>
        selectedUserIds.includes(loc.user_id)
    );

    // Обработчики фильтров
    const handleResetTimeFilter = () => {
        setFromTime('');
        setToTime('');
        setTimeout(() => fetchData(), 100);
    };

    const setLastHour = () => {
        const now = new Date();
        const hourAgo = new Date(now.getTime() - 60 * 60 * 1000);
        setFromTime(hourAgo.toISOString().slice(0, 16));
        setToTime(now.toISOString().slice(0, 16));
        setTimeout(() => fetchData(), 100);
    };

    const setLast24Hours = () => {
        const now = new Date();
        const dayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000);
        setFromTime(dayAgo.toISOString().slice(0, 16));
        setToTime(now.toISOString().slice(0, 16));
        setTimeout(() => fetchData(), 100);
    };

    if (loading && !checkpoints.length && !userLocations.length) {
        return <div className="loading-message">Загрузка карты...</div>;
    }

    return (
        <div className="map-dashboard">
            {/* Кнопка toggle для сайдбара */}
            <button
                className={`sidebar-toggle ${!sidebarOpen ? 'sidebar-closed' : ''}`}
                onClick={() => setSidebarOpen(!sidebarOpen)}
                title={sidebarOpen ? "Скрыть панель" : "Показать панель"}
            >
                {sidebarOpen ? '◀' : '▶'}
            </button>

            {/* Сайдбар с фильтрами */}
            <aside className={`map-sidebar ${sidebarOpen ? 'open' : 'closed'}`}>
                <div className="sidebar-content">
                    {/* Секция пользователей */}
                    <div className="filter-section">
                        <div className="section-header">
                            <h3>👥 Пользователи</h3>
                            <div className="section-controls">
                                <button
                                    className="btn-mini"
                                    onClick={() => setSelectedUserIds(allUsers.map(u => u.id))}
                                    title="Выбрать всех"
                                >
                                    ✓ Все
                                </button>
                                <button
                                    className="btn-mini"
                                    onClick={() => setSelectedUserIds([])}
                                    title="Снять выбор"
                                >
                                    ✗ Снять
                                </button>
                            </div>
                        </div>
                        <div className="user-list">
                            {allUsers.map(u => (
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
                                            fontWeight: selectedUserIds.includes(u.id) ? 'bold' : 'normal'
                                        }}
                                    >
                                        {u.name}
                                    </span>
                                </label>
                            ))}
                        </div>
                    </div>

                    {/* Секция времени */}
                    <div className="filter-section">
                        <div className="section-header">
                            <h3>🕐 Период времени</h3>
                        </div>

                        <div className="quick-filters">
                            <button
                                className="btn-quick"
                                onClick={setLastHour}
                            >
                                Последний час
                            </button>
                            <button
                                className="btn-quick"
                                onClick={setLast24Hours}
                            >
                                Последние 24ч
                            </button>
                        </div>
                        
                        <div className="time-inputs">
                            <div className="input-group">
                                <label>Начало периода:</label>
                                <input
                                    type="datetime-local"
                                    value={fromTime}
                                    onChange={e => setFromTime(e.target.value)}
                                    max={toTime || undefined}
                                />
                            </div>
                            <div className="input-group">
                                <label>Конец периода:</label>
                                <input
                                    type="datetime-local"
                                    value={toTime}
                                    onChange={e => setToTime(e.target.value)}
                                    min={fromTime || undefined}
                                />
                            </div>
                        </div>

                        {fromTime && toTime && (
                            <div className="filter-status">
                                {fromTime < toTime ? (
                                    <span className="status-active">
                                        ✓ Фильтр активен
                                    </span>
                                ) : (
                                    <span className="status-error">
                                        ⚠ Некорректный период
                                    </span>
                                )}
                            </div>
                        )}

                        {fromTime && toTime && (
                            <button
                                className="btn-reset"
                                onClick={handleResetTimeFilter}
                            >
                                Сбросить фильтр
                            </button>
                        )}
                    </div>
                </div>
            </aside>

            {/* Основная область с картой */}
            <div className={`map-main ${sidebarOpen ? 'with-sidebar' : 'full-width'}`}>
                {error && (
                    <div className="map-error-banner">
                        <span>⚠ {error}</span>
                        <button onClick={() => setError(null)}>✕</button>
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
                        locations={visibleLocations}
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
                                weight: 2
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

                    {visibleLocations.map(loc => {
                        const color = getUserColor(loc.user_id);
                        const userName = allUsers.find(u => u.id === loc.user_id)?.name || `ID: ${loc.user_id}`;
                        return (
                            <CircleMarker
                                key={loc.id}
                                center={[loc.latitude, loc.longitude]}
                                radius={8}
                                pathOptions={{
                                    color: color,
                                    fillColor: color,
                                    fillOpacity: 0.8,
                                    weight: 2
                                }}
                            >
                                <Popup>
                                    <div className="popup-content">
                                        <strong>{userName}</strong>
                                        <div>{new Date(loc.updated_at).toLocaleString()}</div>
                                        <div className="popup-coords">
                                            {loc.latitude.toFixed(6)}, {loc.longitude.toFixed(6)}
                                        </div>
                                    </div>
                                </Popup>
                            </CircleMarker>
                        );
                    })}
                </MapContainer>

                {/* Информационная панель внизу */}
                <div className="map-status-bar">
                    <div className="status-info">
                        <span className="status-item">
                            <strong>{user?.name}</strong>
                            {user?.is_admin && <span className="admin-badge">Админ</span>}
                        </span>
                        <span className="status-item">
                            📍 Чекпоинты: <strong>{checkpoints.length}</strong>
                        </span>
                        <span className="status-item">
                            👤 Локации: <strong>{visibleLocations.length}/{userLocations.length}</strong>
                        </span>
                        {fromTime && toTime && (
                            <span className="status-item">
                                📅 {new Date(fromTime).toLocaleDateString()} - {new Date(toTime).toLocaleDateString()}
                            </span>
                        )}
                    </div>
                    <button
                        className="btn-reset-map"
                        onClick={() => {
                            localStorage.removeItem('mapPosition');
                            localStorage.removeItem('mapLoaded');
                            setShouldFitBounds(true);
                        }}
                        title="Сбросить положение карты"
                    >
                        ⟲ Сброс карты
                    </button>
                </div>
            </div>
        </div>
    );
};

export default MapComponent;