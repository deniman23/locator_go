import React, { useEffect, useState } from 'react';
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

    // Загрузка всех пользователей для фильтра по пользователям
    useEffect(() => {
        if (!apiKey) return;
        userApi.getAll(apiKey)
            .then(users => {
                setAllUsers(users);
                setSelectedUserIds(users.map(u => u.id));
            })
            .catch(err => console.error('Не удалось загрузить пользователей:', err));
    }, [apiKey]);

    // Функция загрузки данных с сервера:
    const fetchData = async () => {
        if (!apiKey) {
            setError('Ошибка авторизации: отсутствует API ключ');
            setLoading(false);
            return;
        }
        try {
            setLoading(true);
            setError(null);

            // Загрузка чекпоинтов без изменений
            const cpRes = await checkpointApi.getAll(apiKey);
            setCheckpoints(cpRes.data);

            // Если заданы фильтры по времени, добавляем параметры запроса
            let locRes;
            if (fromTime && toTime) {
                // Для удобства можно добавить "Z", если сервер ожидает время в UTC
                const fromParam = encodeURIComponent(fromTime + 'Z');
                const toParam = encodeURIComponent(toTime + 'Z');
                locRes = await axios.get<Location[]>(`/api/location?from=${fromParam}&to=${toParam}`, {
                    headers: {
                        'X-API-Key': apiKey,
                        'Content-Type': 'application/json'
                    }
                });
            } else {
                locRes = await locationApi.getAll(apiKey);
            }
            setUserLocations(locRes.data);

            if (!localStorage.getItem('mapPosition') && (cpRes.data.length || locRes.data.length)) {
                setShouldFitBounds(true);
            }
        } catch (e: any) {
            setError(e.message || 'Ошибка при загрузке данных');
        } finally {
            setLoading(false);
        }
    };

    // Первоначальная загрузка данных и автообновление каждые 10 сек.
    useEffect(() => {
        fetchData();
        const interval = setInterval(fetchData, 10000);
        return () => clearInterval(interval);
    }, [apiKey]);

    // Получаем сохранённую позицию или дефолтные координаты
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

    // Отфильтрованные локации по чекбоксам пользователей
    const visibleLocations = userLocations.filter(loc =>
        selectedUserIds.includes(loc.user_id)
    );

    if (loading && !checkpoints.length && !userLocations.length) {
        return <div className="loading-message">Загрузка карты...</div>;
    }

    return (
        <div className="map-with-filters">
            {/* Панель фильтров */}
            <aside className="user-filters">
                <h4>Фильтр по пользователям</h4>
                <button
                    onClick={() => setSelectedUserIds(allUsers.map(u => u.id))}
                    title="Выбрать всех"
                >
                    Выбрать всех
                </button>
                <button
                    onClick={() => setSelectedUserIds([])}
                    title="Снять всё"
                >
                    Снять все
                </button>
                <ul>
                    {allUsers.map(u => (
                        <li key={u.id}>
                            <label>
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
                                <span style={{ color: getUserColor(u.id) }}>{u.name}</span>
                            </label>
                        </li>
                    ))}
                </ul>

                {/* Фильтр по времени */}
                <div className="time-filter" style={{ marginTop: '1rem' }}>
                    <h4>Фильтр по времени</h4>
                    <div>
                        <label>
                            От:&nbsp;
                            <input
                                type="datetime-local"
                                value={fromTime}
                                onChange={e => setFromTime(e.target.value)}
                            />
                        </label>
                    </div>
                    <div>
                        <label>
                            До:&nbsp;
                            <input
                                type="datetime-local"
                                value={toTime}
                                onChange={e => setToTime(e.target.value)}
                            />
                        </label>
                    </div>
                    <div style={{ marginTop: '0.5rem' }}>
                        <button onClick={fetchData}>Применить временной фильтр</button>
                        <button
                            style={{ marginLeft: '0.5rem' }}
                            onClick={() => {
                                setFromTime('');
                                setToTime('');
                                fetchData();
                            }}
                        >
                            Сбросить временной фильтр
                        </button>
                    </div>
                </div>
            </aside>

            {/* Карта */}
            <div className="map-container">
                {error && <div className="error-message map-error">{error}</div>}

                <MapContainer
                    center={[initialPosition.lat, initialPosition.lng]}
                    zoom={initialPosition.zoom}
                    style={{ height: '600px', width: '100%' }}
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

                    {/* Отображение чекпоинтов */}
                    {checkpoints.map(cp => (
                        <Circle
                            key={cp.id}
                            center={[cp.latitude, cp.longitude]}
                            radius={cp.radius}
                            pathOptions={{ color: 'blue', fillColor: 'blue', fillOpacity: 0.4, weight: 3 }}
                        >
                            <Popup>
                                <strong>{cp.name}</strong>
                                <br />
                                Радиус: {cp.radius} м
                                <br />
                                ID: {cp.id}
                            </Popup>
                        </Circle>
                    ))}

                    {/* Отображение локаций пользователей (отфильтрованных по выбранным пользователям) */}
                    {visibleLocations.map(loc => {
                        const color = getUserColor(loc.user_id);
                        return (
                            <CircleMarker
                                key={loc.id}
                                center={[loc.latitude, loc.longitude]}
                                radius={8}
                                pathOptions={{ color, fillColor: color, fillOpacity: 0.8 }}
                            >
                                <Popup>
                                    <strong>Пользователь ID: {loc.user_id}</strong>
                                    <br />
                                    Обновлено: {new Date(loc.updated_at).toLocaleString()}
                                    <br />
                                    Координаты: {loc.latitude.toFixed(6)}, {loc.longitude.toFixed(6)}
                                </Popup>
                            </CircleMarker>
                        );
                    })}
                </MapContainer>

                <div className="map-info">
                    <p>
                        Пользователь: <strong>{user?.name}</strong> |
                        Чекпоинты: {checkpoints.length} |
                        Отображено локаций: {visibleLocations.length}/{userLocations.length}
                    </p>
                    <button
                        onClick={() => {
                            localStorage.removeItem('mapPosition');
                            setShouldFitBounds(true);
                            alert('Положение карты сброшено.');
                        }}
                        className="reset-button"
                    >
                        Сбросить положение
                    </button>
                </div>
            </div>
        </div>
    );
};

export default MapComponent;