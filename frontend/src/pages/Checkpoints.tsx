import React, { useCallback, useEffect, useState } from 'react';
import type {Checkpoint} from '../types/models';
import { checkpointApi } from '../services/api';
import CheckpointForm from '../components/Checkpoint/CheckpointForm';
import CheckpointEditForm from '../components/Checkpoint/CheckpointEditForm';
import { useAuth } from '../context/AuthContext';

const Checkpoints: React.FC = () => {
    const [checkpoints, setCheckpoints] = useState<Checkpoint[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [editingCheckpoint, setEditingCheckpoint] = useState<Checkpoint | null>(null);

    // Получаем API ключ из контекста авторизации
    const { apiKey } = useAuth();

    const fetchCheckpoints = useCallback(async () => {
        if (!apiKey) {
            setError('Отсутствует API ключ. Пожалуйста, войдите в систему.');
            setLoading(false);
            return;
        }

        try {
            setLoading(true);
            setError(null);

            // Передаем API ключ в запрос
            const response = await checkpointApi.getAll(apiKey);
            setCheckpoints(response.data);
        } catch (error) {
            console.error('Ошибка при загрузке чекпоинтов:', error);
            setError('Ошибка при загрузке чекпоинтов');
        } finally {
            setLoading(false);
        }
    }, [apiKey]);

    useEffect(() => {
        fetchCheckpoints();
    }, [fetchCheckpoints]); // Перезагружаем при изменении API ключа

    const handleEdit = (checkpoint: Checkpoint) => {
        setEditingCheckpoint(checkpoint);
    };

    const handleCancelEdit = () => {
        setEditingCheckpoint(null);
    };

    const handleEditSuccess = () => {
        fetchCheckpoints();
        setEditingCheckpoint(null);
    };

    return (
        <div className="checkpoints-page">
            <h1>Управление чекпоинтами</h1>

            {editingCheckpoint ? (
                <CheckpointEditForm
                    checkpoint={editingCheckpoint}
                    onSuccess={handleEditSuccess}
                    onCancel={handleCancelEdit}
                />
            ) : (
                <CheckpointForm onSuccess={fetchCheckpoints} />
            )}

            {error && <div className="error-message">{error}</div>}

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
                            <th>Действия</th>
                        </tr>
                        </thead>
                        <tbody>
                        {checkpoints.map(cp => (
                            <tr key={cp.id}>
                                <td>{cp.id}</td>
                                <td>{cp.name}</td>
                                <td>{cp.latitude.toFixed(6)}, {cp.longitude.toFixed(6)}</td>
                                <td>{cp.radius}</td>
                                <td>
                                    <button
                                        onClick={() => handleEdit(cp)}
                                        className="edit-button"
                                    >
                                        Редактировать
                                    </button>
                                </td>
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