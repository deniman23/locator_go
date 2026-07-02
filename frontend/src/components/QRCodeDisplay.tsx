// components/QRCodeDisplay.tsx
import React, { useEffect, useState } from 'react';
import { useAuth } from '../context/AuthContext';

interface QRCodeDisplayProps {
    onClose?: () => void;
    userId?: number;
    userName?: string;
    refreshKey?: number;
}

const QRCodeDisplay: React.FC<QRCodeDisplayProps> = ({ onClose, userId, userName, refreshKey }) => {
    const { apiKey } = useAuth();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [imageData, setImageData] = useState<string | null>(null);

    useEffect(() => {
        const fetchQRCode = async () => {
            if (!apiKey) {
                setError('Отсутствует API ключ');
                setLoading(false);
                return;
            }

            setImageData(null);

            try {
                setLoading(true);
                setError(null);

                const cacheToken = `${refreshKey ?? 0}-${Date.now()}`;
                const url = userId
                    ? `/api/users/${userId}/qr-code-file?t=${cacheToken}`
                    : `/api/users/qr-code-file?t=${cacheToken}`;

                const response = await fetch(url, {
                    headers: {
                        'X-API-Key': apiKey,
                        'Cache-Control': 'no-cache',
                        Pragma: 'no-cache',
                    },
                    cache: 'no-store',
                });

                if (!response.ok) {
                    throw new Error(`Ошибка ${response.status}: ${response.statusText}`);
                }

                // Получаем blob и конвертируем в Data URL
                const blob = await response.blob();
                const reader = new FileReader();
                reader.onloadend = () => {
                    setImageData(reader.result as string);
                    setLoading(false);
                };
                reader.onerror = () => {
                    setError('Ошибка чтения данных изображения');
                    setLoading(false);
                };
                reader.readAsDataURL(blob);

            } catch (err) {
                console.error('Ошибка загрузки QR-кода:', err);
                setError(err instanceof Error ? err.message : 'Ошибка загрузки QR-кода');
                setLoading(false);
            }
        };

        fetchQRCode();
    }, [userId, apiKey, refreshKey]);

    // Формируем заголовок окна
    const modalTitle = userId && userName
        ? `QR-код пользователя: ${userName}`
        : 'QR-код для аутентификации';

    return (
        <div className="qr-code-modal">
            <div className="qr-code-container">
                <div className="qr-code-header">
                    <h3>{modalTitle}</h3>
                    <button className="close-button" onClick={onClose}>×</button>
                </div>

                <div className="qr-code-content">
                    {error ? (
                        <div className="error-message">{error}</div>
                    ) : (
                        <>
                            <p>Отсканируйте этот QR-код с помощью мобильного приложения для аутентификации.</p>
                            <p className="qr-code-hint">
                                После перегенерации старый ключ перестаёт работать — на телефоне нужно отсканировать новый QR.
                            </p>
                            <p className="qr-code-hint">
                                Если картинка не обновилась — закройте окно и откройте QR снова (Ctrl+F5 на странице).
                            </p>
                            <div className="qr-image-container">
                                {loading && <div className="loading-indicator">Загрузка...</div>}
                                {imageData && (
                                    <img
                                        src={imageData}
                                        alt="QR-код для аутентификации"
                                        style={{ display: loading ? 'none' : 'block' }}
                                    />
                                )}
                            </div>
                        </>
                    )}
                </div>

                <div className="qr-code-footer">
                    <button className="button" onClick={onClose}>Закрыть</button>
                </div>
            </div>
        </div>
    );
};

export default QRCodeDisplay;