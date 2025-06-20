import React, { useEffect, useState } from 'react';
import MapComponent from '../components/Map';
import type { Visit } from '../types/models';
import { visitApi } from '../services/api';

const Dashboard: React.FC = () => {
    const [activeVisits, setActiveVisits] = useState<Visit[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchActiveVisits = async () => {
            try {
                setLoading(true);
                // Получаем визиты пользователя с ID 1 (для примера)
                // В реальном приложении здесь можно было бы получать все активные визиты
                const response = await visitApi.getByUserId(1);
                // Фильтруем только активные визиты (где end_at == null)
                const active = response.data.filter(visit => visit.end_at === null);
                setActiveVisits(active);
            } catch (error) {
                console.error('Ошибка при загрузке активных визитов:', error);
            } finally {
                setLoading(false);
            }
        };

        fetchActiveVisits();

        // Обновляем данные каждые 30 секунд
        const interval = setInterval(fetchActiveVisits, 30000);

        return () => clearInterval(interval);
    }, []);

    return (
        <div className="dashboard">
            <h1>Панель мониторинга</h1>
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