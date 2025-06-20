import React from 'react';
import type { Visit } from '../types/models';

interface VisitsListProps {
    visits: Visit[];
    loading: boolean;
    title?: string;
    emptyMessage?: string;
}

const VisitsList: React.FC<VisitsListProps> = ({
                                                   visits,
                                                   loading,
                                                   title = "Визиты",
                                                   emptyMessage = "Нет данных о визитах"
                                               }) => {
    return (
        <div className="visits-list-component">
            <h2>{title}</h2>
            {loading ? (
                <p>Загрузка данных...</p>
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
                            <td>{visit.checkpoint_id}</td>
                            <td>{new Date(visit.start_at).toLocaleString()}</td>
                            <td>{visit.end_at ? new Date(visit.end_at).toLocaleString() : 'Активен'}</td>
                            <td>{visit.duration ? `${visit.duration} сек.` : '-'}</td>
                        </tr>
                    ))}
                    </tbody>
                </table>
            ) : (
                <p className="empty-message">{emptyMessage}</p>
            )}
        </div>
    );
};

export default VisitsList;