import React, { useEffect, useState } from 'react';
import type {Checkpoint} from '../types/models';
import { checkpointApi } from '../services/api';
import CheckpointForm from '../components/CheckpointForm';

const Checkpoints: React.FC = () => {
    const [checkpoints, setCheckpoints] = useState<Checkpoint[]>([]);
    const [loading, setLoading] = useState(true);

    const fetchCheckpoints = async () => {
        try {
            setLoading(true);
            const response = await checkpointApi.getAll();
            setCheckpoints(response.data);
        } catch (error) {
            console.error('Ошибка при загрузке чекпоинтов:', error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchCheckpoints();
    }, []);

    return (
        <div className="checkpoints-page">
            <h1>Управление чекпоинтами</h1>

            <CheckpointForm onSuccess={fetchCheckpoints} />

            <div className="checkpoints-list">
                <h2>Существующие чекпоинты</h2>
                {loading ? (
                    <p>Загрузка чекпоинтов...</p>
                ) : checkpoints.length > 0 ? (
                    <table>
                        <thead>
                        <tr>
                            <th>ID</th>
                            <th>Название</th>
                            <th>Координаты</th>
                            <th>Радиус (м)</th>
                        </tr>
                        </thead>
                        <tbody>
                        {checkpoints.map(cp => (
                            <tr key={cp.id}>
                                <td>{cp.id}</td>
                                <td>{cp.name}</td>
                                <td>{cp.latitude.toFixed(6)}, {cp.longitude.toFixed(6)}</td>
                                <td>{cp.radius}</td>
                            </tr>
                        ))}
                        </tbody>
                    </table>
                ) : (
                    <p>Нет созданных чекпоинтов</p>
                )}
            </div>
        </div>
    );
};

export default Checkpoints;