import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import MapComponent from '../components/Map';
import { formatDateTime, toDateTimeLocalInput } from '../utils/dateFormat';
import type { Visit, Checkpoint, User } from '../types/models';
import { visitApi, checkpointApi, userApi } from '../services/api';
import { useAuth } from '../context/AuthContext';
import { compareMinskDateTimes, minskNowRange } from '../utils/locationTrack';

const POLL_MS = 30_000;

const isVisitActive = (visit: Visit) => visit.end_at == null || visit.end_at === '';

const Dashboard: React.FC = () => {
    const navigate = useNavigate();
    const [activeVisits, setActiveVisits] = useState<Visit[]>([]);
    const [checkpointMap, setCheckpointMap] = useState<Record<number, string>>({});
    const [userMap, setUserMap] = useState<Record<number, string>>({});
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [metaError, setMetaError] = useState<string | null>(null);

    const { apiKey } = useAuth();

    useEffect(() => {
        if (!apiKey) return;

        checkpointApi.getAll(apiKey)
            .then(response => {
                const map: Record<number, string> = {};
                response.data.forEach((cp: Checkpoint) => {
                    map[cp.id] = cp.name;
                });
                setCheckpointMap(map);
            })
            .catch(err => {
                console.error('Ошибка загрузки чекпоинтов:', err);
                setMetaError('Не удалось загрузить чекпоинты');
            });

        userApi.getAll(apiKey)
            .then((users: User[]) => {
                const map: Record<number, string> = {};
                users.forEach(u => {
                    map[u.id] = u.name;
                });
                setUserMap(map);
            })
            .catch(err => {
                console.error('Ошибка загрузки пользователей:', err);
                setMetaError('Не удалось загрузить сотрудников');
            });
    }, [apiKey]);

    useEffect(() => {
        const fetchActiveVisits = async () => {
            if (!apiKey) {
                setError('Отсутствует API ключ. Пожалуйста, войдите в систему.');
                setLoading(false);
                return;
            }

            try {
                setError(null);

                const response = await visitApi.getActive(apiKey);
                const active = response.data.filter(isVisitActive);
                setActiveVisits(active);
            } catch (err) {
                console.error('Ошибка при загрузке активных визитов:', err);
                setError('Ошибка при загрузке активных визитов');
            } finally {
                setLoading(false);
            }
        };

        void fetchActiveVisits();
        const poll = () => {
            if (document.visibilityState !== 'hidden') void fetchActiveVisits();
        };
        const interval = setInterval(poll, POLL_MS);
        return () => clearInterval(interval);
    }, [apiKey]);

    const openVisitOnMap = (visit: Visit) => {
        const from = toDateTimeLocalInput(visit.start_at);
        const to = visit.end_at ? toDateTimeLocalInput(visit.end_at) : minskNowRange(0).to;
        if (!from || !to || compareMinskDateTimes(from, to) >= 0) {
            setError('Не удалось определить интервал визита для карты');
            return;
        }
        const q = new URLSearchParams({
            user_id: String(visit.user_id),
            from,
            to,
            markers: 'all',
        });
        navigate(`/?${q.toString()}`);
    };

    return (
        <div className="dashboard">
            <h1>Панель мониторинга</h1>

            {error && <div className="error-message">{error}</div>}
            {metaError && <div className="error-message">{metaError}</div>}

            <div className="map-container">
                <MapComponent />
            </div>

            <div className="active-visits">
                <h2>Активные визиты</h2>
                {loading ? (
                    <p>Загрузка...</p>
                ) : activeVisits.length > 0 ? (
                    <ul>
                        {activeVisits.map(visit => (
                            <li key={visit.id} className="active-visit-item">
                                <button
                                    type="button"
                                    className="active-visit-button"
                                    onClick={() => openVisitOnMap(visit)}
                                    title="Открыть трек на карте"
                                >
                                    {userMap[visit.user_id] ?? `Пользователь #${visit.user_id}`}
                                    {' — '}
                                    {checkpointMap[visit.checkpoint_id] ?? `чекпоинт #${visit.checkpoint_id}`}
                                    {', с '}
                                    {formatDateTime(visit.start_at)}
                                </button>
                            </li>
                        ))}
                    </ul>
                ) : (
                    <div className="empty-state">
                        <p className="empty-state-title">Нет активных визитов</p>
                        <p className="empty-state-hint">Когда сотрудник войдёт в зону чекпоинта, визит появится здесь.</p>
                    </div>
                )}
            </div>
        </div>
    );
};

export default Dashboard;
