import React, { useState } from 'react';
import { checkpointApi } from '../services/api';
import { useAuth } from '../context/AuthContext'; // Импортируем хук useAuth

interface CheckpointFormProps {
    onSuccess?: () => void | Promise<void>;
}

const CheckpointForm: React.FC<CheckpointFormProps> = ({ onSuccess }) => {
    const [name, setName] = useState('');
    const [latitude, setLatitude] = useState('');
    const [longitude, setLongitude] = useState('');
    const [radius, setRadius] = useState('');
    const [loading, setLoading] = useState(false);
    const [message, setMessage] = useState('');
    const [error, setError] = useState<string | null>(null);

    // Получаем API ключ из контекста авторизации
    const { apiKey } = useAuth();

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
                setMessage('Пожалуйста, введите корректные числовые значения');
                return;
            }

            // Отправляем запрос на создание чекпоинта, передавая API ключ
            const response = await checkpointApi.create(checkpointData, apiKey);

            setMessage(`Чекпоинт "${response.data.name}" успешно создан!`);

            // Очищаем форму
            setName('');
            setLatitude('');
            setLongitude('');
            setRadius('');

            // Вызываем функцию обратного вызова для обновления списка
            if (onSuccess) {
                onSuccess();
            }
        } catch (error) {
            console.error('Ошибка при создании чекпоинта:', error);
            setError('Произошла ошибка при создании чекпоинта');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="checkpoint-form">
            <h2>Создать новый чекпоинт</h2>

            {message && <div className="alert">{message}</div>}
            {error && <div className="error-message">{error}</div>}

            <form onSubmit={handleSubmit}>
                <div>
                    <label htmlFor="name">Название:</label>
                    <input
                        type="text"
                        id="name"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        required
                    />
                </div>

                <div>
                    <label htmlFor="latitude">Широта:</label>
                    <input
                        type="text"
                        id="latitude"
                        value={latitude}
                        onChange={(e) => setLatitude(e.target.value)}
                        placeholder="Например: 55.7522"
                        required
                    />
                </div>

                <div>
                    <label htmlFor="longitude">Долгота:</label>
                    <input
                        type="text"
                        id="longitude"
                        value={longitude}
                        onChange={(e) => setLongitude(e.target.value)}
                        placeholder="Например: 37.6156"
                        required
                    />
                </div>

                <div>
                    <label htmlFor="radius">Радиус (м):</label>
                    <input
                        type="text"
                        id="radius"
                        value={radius}
                        onChange={(e) => setRadius(e.target.value)}
                        placeholder="Например: 100"
                        required
                    />
                </div>

                <button type="submit" disabled={loading}>
                    {loading ? 'Создание...' : 'Создать чекпоинт'}
                </button>
            </form>
        </div>
    );
};

export default CheckpointForm;