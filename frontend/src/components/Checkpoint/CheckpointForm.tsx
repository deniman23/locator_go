import React, { useState } from 'react';
import { checkpointApi } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import CheckpointLocationPicker from './CheckpointLocationPicker';

interface CheckpointFormProps {
    onSuccess?: () => void | Promise<void>;
}

const validateCoords = (lat: number, lng: number, radius: number): string | null => {
    if (isNaN(lat) || isNaN(lng) || isNaN(radius)) {
        return 'Укажите точку на карте и корректный радиус';
    }
    if (lat < -90 || lat > 90 || lng < -180 || lng > 180) {
        return 'Координаты вне допустимого диапазона';
    }
    if (radius <= 0) {
        return 'Радиус должен быть больше 0';
    }
    return null;
};

const CheckpointForm: React.FC<CheckpointFormProps> = ({ onSuccess }) => {
    const [name, setName] = useState('');
    const [latitude, setLatitude] = useState('');
    const [longitude, setLongitude] = useState('');
    const [radius, setRadius] = useState('100');
    const [loading, setLoading] = useState(false);
    const [message, setMessage] = useState('');
    const [error, setError] = useState<string | null>(null);

    const { apiKey } = useAuth();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!apiKey) {
            setError('Отсутствует API ключ. Пожалуйста, войдите в систему снова.');
            return;
        }

        try {
            setLoading(true);
            setMessage('');
            setError(null);

            const checkpointData = {
                name,
                latitude: parseFloat(latitude),
                longitude: parseFloat(longitude),
                radius: parseFloat(radius),
            };

            const validationError = validateCoords(
                checkpointData.latitude,
                checkpointData.longitude,
                checkpointData.radius,
            );
            if (validationError) {
                setError(validationError);
                return;
            }

            const response = await checkpointApi.create(checkpointData, apiKey);

            setMessage(`Чекпоинт "${response.data.name}" успешно создан!`);

            setName('');
            setLatitude('');
            setLongitude('');
            setRadius('100');

            if (onSuccess) {
                onSuccess();
            }
        } catch (err) {
            console.error('Ошибка при создании чекпоинта:', err);
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

                <CheckpointLocationPicker
                    latitude={latitude}
                    longitude={longitude}
                    radius={radius}
                    onChange={(lat, lng) => {
                        setLatitude(lat.toFixed(6));
                        setLongitude(lng.toFixed(6));
                    }}
                />

                <div>
                    <label htmlFor="latitude">Широта:</label>
                    <input
                        type="text"
                        id="latitude"
                        value={latitude}
                        onChange={(e) => setLatitude(e.target.value)}
                        placeholder="Например: 53.9023"
                        required
                        readOnly
                    />
                </div>

                <div>
                    <label htmlFor="longitude">Долгота:</label>
                    <input
                        type="text"
                        id="longitude"
                        value={longitude}
                        onChange={(e) => setLongitude(e.target.value)}
                        placeholder="Например: 27.5619"
                        required
                        readOnly
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

                <button type="submit" className="btn-primary" disabled={loading}>
                    {loading ? 'Создание...' : 'Создать чекпоинт'}
                </button>
            </form>
        </div>
    );
};

export default CheckpointForm;
