import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Visit, Checkpoint, User } from '../types/models';
import { visitApi, checkpointApi, userApi } from '../services/api';
import { useAuth } from '../context/AuthContext';
import { formatDateTime, formatPeriodRange, toDateTimeLocalInput } from '../utils/dateFormat';
import { formatDuration } from '../utils/durationFormat';
import { compareMinskDateTimes, minskDayBounds, minskNowRange } from '../utils/locationTrack';

const defaultDayRange = () => minskDayBounds(0);

const isOutsideVisit = (visit: Visit): boolean =>
    visit.kind === 'outside' || visit.checkpoint_id === 0;

const getVisitCheckpointLabel = (
    visit: Visit,
    checkpointMap: Record<number, string>,
): string => {
    if (isOutsideVisit(visit)) {
        return 'Вне чекпоинтов';
    }
    return checkpointMap[visit.checkpoint_id] || String(visit.checkpoint_id);
};

const UserVisits: React.FC = () => {
    const navigate = useNavigate();
    const defaultRange = useMemo(() => defaultDayRange(), []);
    const [visits, setVisits] = useState<Visit[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const [filters, setFilters] = useState({
        id: '',
        user_id: '',
        checkpoint_id: '',
        from: defaultRange.from,
        to: defaultRange.to,
        showOutside: false,
    });

    // Состояние для сопоставления ID чекпоинта и его названия
    const [checkpointMap, setCheckpointMap] = useState<Record<number, string>>({});
    // Состояние для сопоставления ID пользователя и его имени
    const [userMap, setUserMap] = useState<Record<number, string>>({});

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

    // Загружаем пользователей и формируем словарь (ID -> имя)
    useEffect(() => {
        if (apiKey) {
            userApi.getAll(apiKey)
                .then((users: User[]) => {
                    const map: Record<number, string> = {};
                    users.forEach((user: User) => {
                        map[user.id] = user.name;
                    });
                    setUserMap(map);
                })
                .catch(err => {
                    console.error("Ошибка загрузки пользователей:", err);
                });
        }
    }, [apiKey]);

    // Функция для загрузки визитов с применением фильтров
    const fetchVisits = useCallback(async () => {
        if (!apiKey) {
            setError('Отсутствует API ключ. Пожалуйста, войдите в систему.');
            setLoading(false);
            return;
        }

        try {
            setLoading(true);
            setError(null);

            // Создаем параметры для запроса
            const params: {
                id?: number;
                user_id?: number;
                checkpoint_id?: number;
                from?: string;
                to?: string;
                include_outside?: boolean;
            } = {};

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

            if (filters.from && filters.to) {
                if (compareMinskDateTimes(filters.from, filters.to) >= 0) {
                    setError('Дата начала должна быть раньше даты окончания');
                    setLoading(false);
                    return;
                }
                params.from = filters.from;
                params.to = filters.to;
            }

            if (filters.showOutside) {
                if (!filters.user_id || isNaN(parseInt(filters.user_id))) {
                    setError('Для участков вне чекпоинтов укажите ID пользователя и период');
                    setLoading(false);
                    return;
                }
                if (!params.from || !params.to) {
                    setError('Для участков вне чекпоинтов укажите период (с — по)');
                    setLoading(false);
                    return;
                }
                params.include_outside = true;
            }

            const response = await visitApi.getWithFilters(params, apiKey);
            setVisits(response.data);
        } catch (error) {
            console.error('Ошибка при загрузке визитов:', error);
            setError('Ошибка при загрузке визитов');
        } finally {
            setLoading(false);
        }
    }, [apiKey, filters]);

    // Загружаем визиты при первом рендере
    useEffect(() => {
        fetchVisits();
    }, [fetchVisits]);

    const handleFilterChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const { name, value, type, checked } = e.target;
        setFilters(prev => ({
            ...prev,
            [name]: type === 'checkbox' ? checked : value,
        }));
    };

    const handleResetFilters = () => {
        const b = defaultDayRange();
        setFilters({
            id: '',
            user_id: '',
            checkpoint_id: '',
            from: b.from,
            to: b.to,
            showOutside: false,
        });
        setTimeout(fetchVisits, 0);
    };

    const visitMapRange = (visit: Visit): { from: string; to: string } => ({
        from: toDateTimeLocalInput(visit.start_at),
        to: visit.end_at ? toDateTimeLocalInput(visit.end_at) : minskNowRange(0).to,
    });

    const openVisitOnMap = (visit: Visit) => {
        const { from, to } = visitMapRange(visit);
        if (!from || !to || compareMinskDateTimes(from, to) >= 0) {
            setError('Не удалось определить интервал визита для карты');
            return;
        }
        const q = new URLSearchParams({
            user_id: String(visit.user_id),
            from,
            to,
        });
        navigate(`/?${q.toString()}`);
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

                <div className="filter-group visits-date-range">
                    <span className="visits-date-range-label">Период (Europe/Minsk)</span>
                    <div className="visits-date-inputs">
                        <div className="filter-group">
                            <label htmlFor="from">С</label>
                            <input
                                type="datetime-local"
                                id="from"
                                name="from"
                                value={filters.from}
                                onChange={handleFilterChange}
                                max={filters.to || undefined}
                            />
                        </div>
                        <div className="filter-group">
                            <label htmlFor="to">По</label>
                            <input
                                type="datetime-local"
                                id="to"
                                name="to"
                                value={filters.to}
                                onChange={handleFilterChange}
                                min={filters.from || undefined}
                            />
                        </div>
                    </div>
                    {filters.from && filters.to && (
                        <p className="visits-period-summary">
                            {formatPeriodRange(filters.from, filters.to)}
                        </p>
                    )}
                </div>

                <div className="filter-group filter-group-checkbox">
                    <label htmlFor="showOutside" className="filter-checkbox-label">
                        <input
                            type="checkbox"
                            id="showOutside"
                            name="showOutside"
                            checked={filters.showOutside}
                            onChange={handleFilterChange}
                        />
                        Показывать перемещения вне чекпоинтов
                    </label>
                    <p className="filter-hint">
                        Участки по GPS, когда пользователь не в зоне ни одного чекпоинта. Нужны ID пользователя и период.
                    </p>
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
                            <th>Карта</th>
                        </tr>
                        </thead>
                        <tbody>
                        {visits.map(visit => (
                            <tr
                                key={`${visit.kind ?? 'checkpoint'}-${visit.id}`}
                                className={isOutsideVisit(visit) ? 'visit-row-outside' : undefined}
                            >
                                <td>{isOutsideVisit(visit) ? '—' : visit.id}</td>
                                <td>{userMap[visit.user_id] || visit.user_id}</td>
                                <td>{getVisitCheckpointLabel(visit, checkpointMap)}</td>
                                <td>{formatDateTime(visit.start_at)}</td>
                                <td>{visit.end_at ? formatDateTime(visit.end_at) : 'Активен'}</td>
                                <td>
                                    {visit.end_at ? formatDuration(visit.duration) : 'В процессе'}
                                </td>
                                <td>
                                    <button
                                        type="button"
                                        className="visit-map-button"
                                        onClick={() => openVisitOnMap(visit)}
                                    >
                                        Маршрут на карте
                                    </button>
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