import React, { useEffect, useState } from 'react';
import MapComponent from '../components/Map';
import type { Visit } from '../types/models';
import { visitApi } from '../services/api';
import { useAuth } from '../context/AuthContext'; // Добавляем импорт

const Dashboard: React.FC = () => {
    const [activeVisits, setActiveVisits] = useState<Visit[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Получаем API ключ
    const { apiKey } = useAuth();

    useEffect(() => {
        const fetchActiveVisits = async () => {
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
                const response = await visitApi.getByUserId(1, keyToUse);

                // Фильтруем только активные визиты
                const active = response.data.filter(visit => visit.end_at === null);
                setActiveVisits(active);
            } catch (error) {
                console.error('Ошибка при загрузке активных визитов:', error);
                setError('Ошибка при загрузке активных визитов');
            } finally {
                setLoading(false);
            }
        };

        fetchActiveVisits();

        // Обновляем данные каждые 30 секунд
        const interval = setInterval(fetchActiveVisits, 30000);

        return () => clearInterval(interval);
    }, [apiKey]); // Добавляем apiKey в зависимости

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
                                Пользователь #{visit.user_id} на чекпоинте #{visit.checkpoint_id} с {new Date(visit.start_at).toLocaleTimeString()}
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