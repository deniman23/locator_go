import React, { useState, useEffect } from 'react';
import { checkpointApi } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import type { Checkpoint } from '../../types/models';
import CheckpointLocationPicker from './CheckpointLocationPicker';

interface CheckpointEditFormProps {
    checkpoint: Checkpoint;
    onSuccess?: () => void | Promise<void>;
    onCancel?: () => void;
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

const CheckpointEditForm: React.FC<CheckpointEditFormProps> = ({
    checkpoint,
    onSuccess,
    onCancel,
}) => {
    const [name, setName] = useState(checkpoint.name);
    const [latitude, setLatitude] = useState(checkpoint.latitude.toString());
    const [longitude, setLongitude] = useState(checkpoint.longitude.toString());
    const [radius, setRadius] = useState(checkpoint.radius.toString());
    const [loading, setLoading] = useState(false);
    const [message, setMessage] = useState('');
    const [error, setError] = useState<string | null>(null);

    const { apiKey } = useAuth();

    useEffect(() => {
        setName(checkpoint.name);
        setLatitude(checkpoint.latitude.toString());
        setLongitude(checkpoint.longitude.toString());
        setRadius(checkpoint.radius.toString());
    }, [checkpoint]);

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

            const response = await checkpointApi.update(checkpoint.id, checkpointData, apiKey);

            setMessage(`Чекпоинт "${response.data.name}" успешно обновлен!`);

            if (onSuccess) {
                onSuccess();
            }
        } catch (err) {
            console.error('Ошибка при обновлении чекпоинта:', err);
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
                    <label htmlFor="edit-latitude">Широта:</label>
                    <input
                        type="text"
                        id="edit-latitude"
                        value={latitude}
                        onChange={(e) => setLatitude(e.target.value)}
                        required
                        readOnly
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
                        readOnly
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
                    <button type="submit" className="btn-primary" disabled={loading}>
                        {loading ? 'Сохранение...' : 'Сохранить'}
                    </button>
                    <button
                        type="button"
                        onClick={onCancel}
                        disabled={loading}
                        className="btn-secondary cancel-button"
                    >
                        Отмена
                    </button>
                </div>
            </form>
        </div>
    );
};

export default CheckpointEditForm;
