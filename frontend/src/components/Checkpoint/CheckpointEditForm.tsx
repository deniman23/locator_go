import React, { useState, useEffect } from 'react';
import { checkpointApi } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import type { Checkpoint } from '../../types/models';

interface CheckpointEditFormProps {
    checkpoint: Checkpoint;
    onSuccess?: () => void | Promise<void>;
    onCancel?: () => void;
}

const CheckpointEditForm: React.FC<CheckpointEditFormProps> = ({
                                                                   checkpoint,
                                                                   onSuccess,
                                                                   onCancel
                                                               }) => {
    const [name, setName] = useState(checkpoint.name);
    const [latitude, setLatitude] = useState(checkpoint.latitude.toString());
    const [longitude, setLongitude] = useState(checkpoint.longitude.toString());
    const [radius, setRadius] = useState(checkpoint.radius.toString());
    const [loading, setLoading] = useState(false);
    const [message, setMessage] = useState('');
    const [error, setError] = useState<string | null>(null);

    // Получаем API ключ из контекста авторизации
    const { apiKey } = useAuth();

    // Обновляем состояние, если пропсы изменились
    useEffect(() => {
        setName(checkpoint.name);
        setLatitude(checkpoint.latitude.toString());
        setLongitude(checkpoint.longitude.toString());
        setRadius(checkpoint.radius.toString());
    }, [checkpoint]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        // Проверяем наличие API ключа
        if (!apiKey) {
            setError('Отсутствует API ключ. Пожалуйста, войдите в систему снова.');
            return;
        }

        try {
            setLoading(true);
            setMessage('');
            setError(null);

            // Преобразуем строковые значения в числа
            const checkpointData = {
                name,
                latitude: parseFloat(latitude),
                longitude: parseFloat(longitude),
                radius: parseFloat(radius)
            };

            // Проверяем валидность данных
            if (isNaN(checkpointData.latitude) || isNaN(checkpointData.longitude) || isNaN(checkpointData.radius)) {
                setError('Пожалуйста, введите корректные числовые значения');
                return;
            }

            // Отправляем запрос на обновление чекпоинта, передавая API ключ
            const response = await checkpointApi.update(checkpoint.id, checkpointData, apiKey);

            setMessage(`Чекпоинт "${response.data.name}" успешно обновлен!`);

            // Вызываем функцию обратного вызова для обновления списка
            if (onSuccess) {
                onSuccess();
            }
        } catch (error) {
            console.error('Ошибка при обновлении чекпоинта:', error);
            setError('Произошла ошибка при обновлении чекпоинта');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="checkpoint-form checkpoint-edit-form">
            <h2>Редактировать чекпоинт</h2>

            {message && <div className="alert">{message}</div>}
            {error && <div className="error-message">{error}</div>}

            <form onSubmit={handleSubmit}>
                <div>
                    <label htmlFor="edit-name">Название:</label>
                    <input
                        type="text"
                        id="edit-name"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        required
                    />
                </div>

                <div>
                    <label htmlFor="edit-latitude">Широта:</label>
                    <input
                        type="text"
                        id="edit-latitude"
                        value={latitude}
                        onChange={(e) => setLatitude(e.target.value)}
                        required
                    />
                </div>

                <div>
                    <label htmlFor="edit-longitude">Долгота:</label>
                    <input
                        type="text"
                        id="edit-longitude"
                        value={longitude}
                        onChange={(e) => setLongitude(e.target.value)}
                        required
                    />
                </div>

                <div>
                    <label htmlFor="edit-radius">Радиус (м):</label>
                    <input
                        type="text"
                        id="edit-radius"
                        value={radius}
                        onChange={(e) => setRadius(e.target.value)}
                        required
                    />
                </div>

                <div className="form-buttons">
                    <button type="submit" disabled={loading}>
                        {loading ? 'Сохранение...' : 'Сохранить'}
                    </button>
                    <button
                        type="button"
                        onClick={onCancel}
                        disabled={loading}
                        className="cancel-button"
                    >
                        Отмена
                    </button>
                </div>
            </form>
        </div>
    );
};

export default CheckpointEditForm;