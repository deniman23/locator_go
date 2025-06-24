import React, { useEffect, useState } from 'react';
import type {Visit} from '../types/models';
import { visitApi } from '../services/api';
import { useAuth } from '../context/AuthContext'; // Добавляем импорт

const UserVisits: React.FC = () => {
    const [visits, setVisits] = useState<Visit[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [userId, setUserId] = useState('1'); // По умолчанию пользователь с ID 1

    // Получаем API ключ
    const { apiKey } = useAuth();

    const fetchVisits = async () => {
        if (!userId || isNaN(parseInt(userId))) {
            return;
        }

        // Проверяем наличие API ключа
        const localStorageKey = localStorage.getItem('apiKey');
        const keyToUse = apiKey || localStorageKey;

        if (!keyToUse) {
            setError('Отсутствует API ключ. Пожалуйста, войдите в систему.');
            setLoading(false);
            return;
        }

        try {
            setLoading(true);
            setError(null);

            // Передаем API ключ в запрос
            const response = await visitApi.getByUserId(parseInt(userId), keyToUse);

            setVisits(response.data);
        } catch (error) {
            console.error('Ошибка при загрузке визитов:', error);
            setError('Ошибка при загрузке визитов');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchVisits();
    }, [userId, apiKey]); // Добавляем apiKey в зависимости

    return (
        <div className="visits-page">
            <h1>История визитов</h1>

            {error && <div className="error-message">{error}</div>}

            <div className="user-selector">
                <label htmlFor="user-id">ID пользователя:</label>
                <input
                    type="text"
                    id="user-id"
                    value={userId}
                    onChange={(e) => setUserId(e.target.value)}
                    onBlur={fetchVisits}
                />
                <button onClick={fetchVisits}>Загрузить</button>
            </div>

            <div className="visits-list">
                {loading ? (
                    <p>Загрузка визитов...</p>
                ) : visits.length > 0 ? (
                    <table>
                        <thead>
                        <tr>
                            <th>ID</th>
                            <th>Чекпоинт</th>
                            <th>Начало</th>
                            <th>Окончание</th>
                            <th>Длительность (сек)</th>
                        </tr>
                        </thead>
                        <tbody>
                        {visits.map(visit => (
                            <tr key={visit.id}>
                                <td>{visit.id}</td>
                                <td>{visit.checkpoint_id}</td>
                                <td>{new Date(visit.start_at).toLocaleString()}</td>
                                <td>{visit.end_at ? new Date(visit.end_at).toLocaleString() : 'Активен'}</td>
                                <td>{visit.duration}</td>
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