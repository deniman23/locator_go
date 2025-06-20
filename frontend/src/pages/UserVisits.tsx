import React, { useEffect, useState } from 'react';
import type {Visit} from '../types/models';
import { visitApi } from '../services/api';

const UserVisits: React.FC = () => {
    const [visits, setVisits] = useState<Visit[]>([]);
    const [loading, setLoading] = useState(true);
    const [userId, setUserId] = useState('1'); // По умолчанию пользователь с ID 1

    const fetchVisits = async () => {
        if (!userId || isNaN(parseInt(userId))) {
            return;
        }

        try {
            setLoading(true);
            const response = await visitApi.getByUserId(parseInt(userId));
            setVisits(response.data);
        } catch (error) {
            console.error('Ошибка при загрузке визитов:', error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchVisits();
    }, [userId]);

    return (
        <div className="visits-page">
            <h1>История визитов</h1>

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