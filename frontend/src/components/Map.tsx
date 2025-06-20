import React, { useEffect, useState } from 'react';
import { MapContainer, TileLayer, Circle, Marker, Popup, useMap, useMapEvents } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import type {Checkpoint, Location} from '../types/models';
import { checkpointApi, locationApi } from '../services/api';
import L from 'leaflet';
import { useAuth } from '../context/AuthContext';

// Исправление иконки маркера в Leaflet
import icon from 'leaflet/dist/images/marker-icon.png';
import iconShadow from 'leaflet/dist/images/marker-shadow.png';

// Исправляем проблему с маркерами в Leaflet
const DefaultIcon = L.icon({
    iconUrl: icon,
    shadowUrl: iconShadow,
    iconSize: [25, 41],
    iconAnchor: [12, 41]
});

L.Marker.prototype.options.icon = DefaultIcon;

// Компонент для сохранения позиции карты
const MapPositionSaver = () => {
    const map = useMapEvents({
        moveend: () => {
            // Сохраняем центр карты и уровень зума в localStorage
            const center = map.getCenter();
            const zoom = map.getZoom();
            localStorage.setItem('mapPosition', JSON.stringify({
                lat: center.lat,
                lng: center.lng,
                zoom: zoom
            }));
            console.log('Позиция карты сохранена:', center, zoom);
        }
    });

    return null;
};

// Компонент для автоматического обновления вида карты
const MapUpdater = ({
                        checkpoints,
                        locations,
                        shouldFitBounds
                    }: {
    checkpoints: Checkpoint[],
    locations: Location[],
    shouldFitBounds: boolean
}) => {
    const map = useMap();

    useEffect(() => {
        if (!shouldFitBounds) return;

        if (checkpoints.length === 0 && locations.length === 0) return;

        // Создаем границы для всех точек
        const bounds = L.latLngBounds([]);

        // Добавляем чекпоинты в границы
        checkpoints.forEach(cp => {
            bounds.extend([cp.latitude, cp.longitude]);
        });

        // Добавляем локации в границы
        locations.forEach(loc => {
            bounds.extend([loc.latitude, loc.longitude]);
        });

        if (!bounds.isValid()) return;

        // Устанавливаем вид карты с отступами
        map.fitBounds(bounds, {
            padding: [50, 50],
            maxZoom: 16
        });

        console.log('Карта отцентрирована по точкам');
    }, [map, checkpoints, locations, shouldFitBounds]);

    return null;
};

const MapComponent: React.FC = () => {
    const [checkpoints, setCheckpoints] = useState<Checkpoint[]>([]);
    const [userLocations, setUserLocations] = useState<Location[]>([]);
    const [loading, setLoading] = useState(true);
    const [shouldFitBounds, setShouldFitBounds] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Получаем API ключ из контекста авторизации
    const { apiKey, user } = useAuth();

    // Получаем сохраненную позицию из localStorage или используем значения по умолчанию
    const getSavedPosition = () => {
        const savedPosition = localStorage.getItem('mapPosition');
        if (savedPosition) {
            try {
                return JSON.parse(savedPosition);
            } catch (e) {
                console.error('Ошибка при чтении сохраненной позиции:', e);
            }
        }
        return { lat: 55.75, lng: 37.61, zoom: 10 }; // Москва по умолчанию
    };

    const initialPosition = getSavedPosition();

    useEffect(() => {
        const fetchData = async () => {
            if (!apiKey) {
                setError('Ошибка авторизации: отсутствует API ключ');
                setLoading(false);
                return;
            }

            try {
                setLoading(true);
                setError(null);
                console.log('Загрузка данных...');

                // Получаем все чекпоинты с использованием API ключа
                const checkpointsResponse = await checkpointApi.getAll(apiKey);
                console.log('Чекпоинты:', checkpointsResponse.data);
                setCheckpoints(checkpointsResponse.data);

                // Получаем все локации с использованием API ключа
                const locationsResponse = await locationApi.getAll(apiKey);
                console.log('Локации:', locationsResponse.data);
                setUserLocations(locationsResponse.data);

                // Если в localStorage нет сохраненной позиции и есть данные,
                // то устанавливаем флаг для подгонки карты под точки
                if (!localStorage.getItem('mapPosition') &&
                    (checkpointsResponse.data.length > 0 || locationsResponse.data.length > 0)) {
                    setShouldFitBounds(true);
                }
            } catch (error) {
                console.error('Ошибка при загрузке данных:', error);
                setError(error instanceof Error ? error.message : 'Ошибка при загрузке данных');
            } finally {
                setLoading(false);
            }
        };

        fetchData();

        // Устанавливаем интервал для обновления данных каждые 10 секунд
        const interval = setInterval(fetchData, 10000);

        return () => clearInterval(interval);
    }, [apiKey]); // Добавляем apiKey в массив зависимостей

    if (loading && userLocations.length === 0 && checkpoints.length === 0) {
        return <div className="loading-message">Загрузка карты...</div>;
    }

    return (
        <div className="map-container">
            {error && (
                <div className="error-message map-error">
                    {error}
                </div>
            )}

            <MapContainer
                center={[initialPosition.lat, initialPosition.lng]}
                zoom={initialPosition.zoom}
                style={{ height: '500px', width: '100%' }}
                scrollWheelZoom={true}
            >
                <MapPositionSaver />

                <MapUpdater
                    checkpoints={checkpoints}
                    locations={userLocations}
                    shouldFitBounds={shouldFitBounds}
                />

                <TileLayer
                    url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                    attribution="&copy; <a href='https://www.openstreetmap.org/copyright'>OpenStreetMap</a>"
                />

                {/* Отображаем чекпоинты как круги с более заметными стилями */}
                {checkpoints.map(checkpoint => (
                    <Circle
                        key={`checkpoint-${checkpoint.id}`}
                        center={[checkpoint.latitude, checkpoint.longitude]}
                        radius={checkpoint.radius}
                        pathOptions={{
                            color: 'blue',
                            fillColor: 'blue',
                            fillOpacity: 0.4,
                            weight: 3
                        }}
                    >
                        <Popup>
                            <div>
                                <strong>{checkpoint.name}</strong><br />
                                Радиус: {checkpoint.radius} м<br />
                                ID: {checkpoint.id}
                            </div>
                        </Popup>
                    </Circle>
                ))}

                {/* Маркеры для локаций пользователей */}
                {userLocations.map(location => (
                    <Marker
                        key={`location-${location.id}`}
                        position={[location.latitude, location.longitude]}
                    >
                        <Popup>
                            <div>
                                <strong>Пользователь ID: {location.user_id}</strong><br />
                                Обновлено: {new Date(location.updated_at).toLocaleString()}<br />
                                Координаты: {location.latitude.toFixed(6)}, {location.longitude.toFixed(6)}
                            </div>
                        </Popup>
                    </Marker>
                ))}
            </MapContainer>

            <div className="map-info">
                <p>
                    Пользователь: <strong>{user?.name}</strong> |
                    Чекпоинты: {checkpoints.length} |
                    Локации: {userLocations.length}
                </p>
                {checkpoints.length === 0 && userLocations.length === 0 && (
                    <p className="warning">Нет данных для отображения. Добавьте чекпоинты или локации.</p>
                )}
                <div className="map-controls">
                    <button
                        onClick={() => {
                            localStorage.removeItem('mapPosition');
                            setShouldFitBounds(true);
                            alert('Положение карты сброшено. Карта будет отцентрирована по всем точкам после обновления данных.');
                        }}
                        className="reset-button"
                    >
                        Сбросить положение карты
                    </button>
                </div>
            </div>
        </div>
    );
};

export default MapComponent;