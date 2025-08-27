import React, { useEffect, useState } from 'react';
import type { Visit, Checkpoint } from '../types/models';
import { visitApi, checkpointApi } from '../services/api';
import { useAuth } from '../context/AuthContext';

// Функция для форматирования длительности из секунд в читаемый формат
const formatDuration = (seconds: number): string => {
    if (seconds < 0) return "0 секунд";

    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const remainingSeconds = Math.floor(seconds % 60);

    const parts = [];

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
};

// Вспомогательные функции для правильных окончаний в русском языке
const getHourForm = (hours: number): string => {
    if (hours >= 11 && hours <= 19) return 'часов';
    const lastDigit = hours % 10;
    if (lastDigit === 1) return 'час';
    if (lastDigit >= 2 && lastDigit <= 4) return 'часа';
    return 'часов';
};

const getMinuteForm = (minutes: number): string => {
    if (minutes >= 11 && minutes <= 19) return 'минут';
    const lastDigit = minutes % 10;
    if (lastDigit === 1) return 'минута';
    if (lastDigit >= 2 && lastDigit <= 4) return 'минуты';
    return 'минут';
};

const getSecondForm = (seconds: number): string => {
    if (seconds >= 11 && seconds <= 19) return 'секунд';
    const lastDigit = seconds % 10;
    if (lastDigit === 1) return 'секунда';
    if (lastDigit >= 2 && lastDigit <= 4) return 'секунды';
    return 'секунд';
};

const UserVisits: React.FC = () => {
    const [visits, setVisits] = useState<Visit[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Состояние для фильтров
    const [filters, setFilters] = useState({
        id: '',
        user_id: '',
        checkpoint_id: ''
    });

    // Состояние для сопоставления ID чекпоинта и его названия
    const [checkpointMap, setCheckpointMap] = useState<Record<number, string>>({});

    // Получаем API ключ из контекста авторизации
    const { apiKey } = useAuth();

    // Загружаем чекпоинты и формируем словарь (ID -> название)
    useEffect(() => {
        if (apiKey) {
            checkpointApi.getAll(apiKey)
                .then(response => {
                    const map: Record<number, string> = {};
                    response.data.forEach((checkpoint: Checkpoint) => {
                        map[checkpoint.id] = checkpoint.name;
                    });
                    setCheckpointMap(map);
                })
                .catch(err => {
                    console.error("Ошибка загрузки чекпоинтов:", err);
                });
        }
    }, [apiKey]);

    // Функция для загрузки визитов с применением фильтров
    const fetchVisits = async () => {
        if (!apiKey) {
            setError('Отсутствует API ключ. Пожалуйста, войдите в систему.');
            setLoading(false);
            return;
        }

        try {
            setLoading(true);
            setError(null);

            // Создаем параметры для запроса
            const params: Record<string, number> = {};

            // Добавляем только непустые параметры
            if (filters.id && !isNaN(parseInt(filters.id))) {
                params.id = parseInt(filters.id);
            }

            if (filters.user_id && !isNaN(parseInt(filters.user_id))) {
                params.user_id = parseInt(filters.user_id);
            }

            if (filters.checkpoint_id && !isNaN(parseInt(filters.checkpoint_id))) {
                params.checkpoint_id = parseInt(filters.checkpoint_id);
            }

            // Выполняем запрос с фильтрами
            const response = await visitApi.getWithFilters(params, apiKey);
            setVisits(response.data);
        } catch (error) {
            console.error('Ошибка при загрузке визитов:', error);
            setError('Ошибка при загрузке визитов');
        } finally {
            setLoading(false);
        }
    };

    // Загружаем визиты при первом рендере
    useEffect(() => {
        fetchVisits();
    }, [apiKey]);

    // Обработчик изменения значений фильтров
    const handleFilterChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const { name, value } = e.target;
        setFilters(prev => ({
            ...prev,
            [name]: value
        }));
    };

    // Обработчик сброса фильтров
    const handleResetFilters = () => {
        setFilters({
            id: '',
            user_id: '',
            checkpoint_id: ''
        });
        // После сброса фильтров загружаем все визиты
        setTimeout(fetchVisits, 0);
    };

    return (
        <div className="visits-page">
            <h1>История визитов</h1>

            {error && <div className="error-message">{error}</div>}

            <div className="filters">
                <h3>Фильтры</h3>
                <div className="filter-group">
                    <label htmlFor="id">ID визита:</label>
                    <input
                        type="text"
                        id="id"
                        name="id"
                        value={filters.id}
                        onChange={handleFilterChange}
                        placeholder="ID визита"
                    />
                </div>

                <div className="filter-group">
                    <label htmlFor="user_id">ID пользователя:</label>
                    <input
                        type="text"
                        id="user_id"
                        name="user_id"
                        value={filters.user_id}
                        onChange={handleFilterChange}
                        placeholder="ID пользователя"
                    />
                </div>

                <div className="filter-group">
                    <label htmlFor="checkpoint_id">ID чекпоинта:</label>
                    <input
                        type="text"
                        id="checkpoint_id"
                        name="checkpoint_id"
                        value={filters.checkpoint_id}
                        onChange={handleFilterChange}
                        placeholder="ID чекпоинта"
                    />
                </div>

                <div className="filter-buttons">
                    <button onClick={fetchVisits} className="filter-button">
                        Применить фильтры
                    </button>
                    <button onClick={handleResetFilters} className="reset-button">
                        Сбросить фильтры
                    </button>
                </div>
            </div>

            <div className="visits-list">
                <h2>Результаты</h2>
                {loading ? (
                    <p>Загрузка визитов...</p>
                ) : visits.length > 0 ? (
                    <table>
                        <thead>
                        <tr>
                            <th>ID</th>
                            <th>Пользователь</th>
                            <th>Чекпоинт</th>
                            <th>Начало</th>
                            <th>Окончание</th>
                            <th>Длительность</th>
                        </tr>
                        </thead>
                        <tbody>
                        {visits.map(visit => (
                            <tr key={visit.id}>
                                <td>{visit.id}</td>
                                <td>{visit.user_id}</td>
                                {/* Используем сопоставление для получения названия чекпоинта */}
                                <td>{checkpointMap[visit.checkpoint_id] || visit.checkpoint_id}</td>
                                <td>{new Date(visit.start_at).toLocaleString()}</td>
                                <td>{visit.end_at ? new Date(visit.end_at).toLocaleString() : 'Активен'}</td>
                                <td>
                                    {visit.end_at ? formatDuration(visit.duration) : 'В процессе'}
                                </td>
                            </tr>
                        ))}
                        </tbody>
                    </table>
                ) : (
                    <p>Нет данных о визитах</p>
                )}
            </div>
        </div>
    );
};

export default UserVisits;