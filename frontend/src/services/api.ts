import axios from 'axios';
import type {Checkpoint, Location, LocationEvent, User, Visit} from '../types/models';

const api = axios.create({
    baseURL: '/api',
    headers: {
        'Content-Type': 'application/json'
    }
});

// Функция для добавления API ключа в конфигурацию запроса
const withApiKey = (apiKey?: string) => {
    if (!apiKey) return {};

    return {
        headers: {
            'X-API-Key': apiKey
        }
    };
};

// API для работы с локациями
export const locationApi = {
    // Получение всех локаций с опциональным API ключом
    getAll: (apiKey?: string) => api.get<Location[]>('/location/', withApiKey(apiKey)),

    // Получение локации по ID пользователя
    getByUserId: (userId: number, apiKey?: string) =>
        api.get<Location>(`/location/single?user_id=${userId}`, withApiKey(apiKey)),

    // Создание новой локации
    createLocation: (location: { user_id: number, latitude: number, longitude: number }, apiKey?: string) =>
        api.post<Location>('/location/', location, withApiKey(apiKey))
};

// API для работы с чекпоинтами
export const checkpointApi = {
    // Получение всех чекпоинтов с опциональным API ключом
    getAll: (apiKey?: string) => api.get<Checkpoint[]>('/checkpoint/', withApiKey(apiKey)),

    // Создание нового чекпоинта
    create: (checkpoint: { name: string, latitude: number, longitude: number, radius: number }, apiKey?: string) =>
        api.post<Checkpoint>('/checkpoint/', checkpoint, withApiKey(apiKey)),

    // Обновление существующего чекпоинта
    update: (id: number, checkpoint: { name: string, latitude: number, longitude: number, radius: number }, apiKey?: string) =>
        api.put<Checkpoint>(`/checkpoint/${id}`, checkpoint, withApiKey(apiKey)),

    // Проверка, находится ли пользователь в чекпоинте
    checkUserInCheckpoint: (userId: number, checkpointId: number, apiKey?: string) =>
        api.get(`/checkpoint/check?user_id=${userId}&checkpoint_id=${checkpointId}`, withApiKey(apiKey))
};

// API для работы с визитами
export const visitApi = {
    // Получение визитов пользователя
    getByUserId: (userId: number, apiKey?: string) =>
        api.get<Visit[]>(`/visits/?user_id=${userId}`, withApiKey(apiKey))
};

// API для работы с событиями
export const eventApi = {
    // Публикация события
    publish: (event: LocationEvent, apiKey?: string) =>
        api.post('/event/publish', event, withApiKey(apiKey))
};

// API для работы с пользователями
export const userApi = {
    authenticate: async (apiKey: string): Promise<User> => {
        try {
            // Используем новый эндпоинт для получения текущего пользователя
            const response = await fetch('/api/users/me', {
                headers: {
                    'X-API-Key': apiKey
                }
            });

            if (!response.ok) {
                throw new Error('Неверный API ключ');
            }

            const userData = await response.json();

            // Преобразуем данные в формат User
            return {
                id: userData.id,
                name: userData.name,
                is_admin: userData.is_admin,
                created_at: userData.created_at,
                updated_at: userData.updated_at,
                qr_code: userData.qr_code
            };
            
        } catch (error) {
            console.error('Ошибка аутентификации:', error);
            throw new Error('Ошибка аутентификации');
        }
    },

    // Метод для обновления информации о текущем пользователе
    getCurrentUser: async (apiKey: string): Promise<User> => {
        try {
            const response = await fetch('/api/users/me', {
                headers: {
                    'X-API-Key': apiKey
                }
            });

            if (!response.ok) {
                throw new Error('Ошибка получения данных пользователя');
            }

            const userData = await response.json();

            // Преобразуем данные в формат User
            return {
                id: userData.id,
                name: userData.name,
                is_admin: userData.is_admin,
                created_at: userData.created_at,
                updated_at: userData.updated_at,
                qr_code: userData.qr_code
            };
        } catch (error) {
            console.error('Ошибка получения текущего пользователя:', error);
            throw new Error('Ошибка получения данных пользователя');
        }
    },

    getAll: async (apiKey: string): Promise<User[]> => {
        const response = await fetch('/api/users/', {
            headers: {
                'X-API-Key': apiKey
            }
        });

        if (!response.ok) {
            throw new Error('Ошибка при получении списка пользователей');
        }

        return response.json();
    },

    getById: async (id: number, apiKey: string): Promise<User> => {
        const response = await fetch(`/api/users/${id}`, {
            headers: {
                'X-API-Key': apiKey
            }
        });

        if (!response.ok) {
            throw new Error('Пользователь не найден');
        }

        return response.json();
    },

    create: async (name: string, isAdmin: boolean, apiKey: string): Promise<User> => {
        const response = await fetch('/api/users/', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-API-Key': apiKey
            },
            body: JSON.stringify({ name, is_admin: isAdmin })
        });

        if (!response.ok) {
            const data = await response.json();
            throw new Error(data.error || 'Ошибка при создания пользователя');
        }

        return response.json();
    },

    getQRCodeUrl: () => {
        // Здесь создаем URL с учетом текущего хоста и порта
        const baseUrl = window.location.origin;
        // API-ключ будет передан через заголовки автоматически
        return `${baseUrl}/api/users/qr-code-file`;
    },
    getQRCodeData: (apiKey: string) => {
        return api.get('/users/qr-code', withApiKey(apiKey));
    },
    getCurrentUserQRCodeFile: () => {
        return `/api/users/qr-code-file?t=${Date.now()}`;
    },

    // Получить QR-код конкретного пользователя в виде файла (для админов)
    getUserQRCodeFile: (userId: number) => {
        return `/api/users/${userId}/qr-code-file?t=${Date.now()}`;
    },

    // Получить данные QR-кода конкретного пользователя (для админов)
    getUserQRCodeData: (userId: number, apiKey: string) => {
        return api.get(`/users/${userId}/qr-code`, withApiKey(apiKey));
    }
};