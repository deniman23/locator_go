import React, { useEffect, useState } from 'react';
import MapComponent from '../components/Map';
import type { Visit, Checkpoint, User } from '../types/models';
import { visitApi, checkpointApi, userApi } from '../services/api';
import { useAuth } from '../context/AuthContext';

const POLL_MS = 10000;

const isVisitActive = (visit: Visit) => visit.end_at == null || visit.end_at === '';

const Dashboard: React.FC = () => {
    const [activeVisits, setActiveVisits] = useState<Visit[]>([]);
    const [checkpointMap, setCheckpointMap] = useState<Record<number, string>>({});
    const [userMap, setUserMap] = useState<Record<number, string>>({});
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

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
            .catch(err => console.error('Ошибка загрузки чекпоинтов:', err));

        userApi.getAll(apiKey)
            .then((users: User[]) => {
                const map: Record<number, string> = {};
                users.forEach(u => {
                    map[u.id] = u.name;
                });
                setUserMap(map);
            })
            .catch(err => console.error('Ошибка загрузки пользователей:', err));
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
        const interval = setInterval(() => void fetchActiveVisits(), POLL_MS);
        return () => clearInterval(interval);
    }, [apiKey]);

    return (
        <div className="dashboard">
            <h1>Панель мониторинга</h1>

            {error && <div className="error-message">{error}</div>}

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
                            <li key={visit.id}>
                                {userMap[visit.user_id] ?? `Пользователь #${visit.user_id}`}
                                {' — '}
                                {checkpointMap[visit.checkpoint_id] ?? `чекпоинт #${visit.checkpoint_id}`}
                                {', с '}
                                {new Date(visit.start_at).toLocaleString()}
                            </li>
                        ))}
                    </ul>
                ) : (
                    <p>Нет активных визитов</p>
                )}
            </div>
        </div>
    );
};

export default Dashboard;
