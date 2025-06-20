import axios from 'axios';
import type {Checkpoint, Location, Visit, LocationEvent, User} from '../types/models';

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
        // Сделаем запрос к любому защищенному эндпоинту с API ключом, чтобы проверить его
        const response = await fetch('/api/users/', {
            headers: {
                'X-API-Key': apiKey
            }
        });

        if (!response.ok) {
            throw new Error('Неверный API ключ');
        }

        // Теперь получим информацию о текущем пользователе по его API ключу
        // Это можно сделать через отдельный эндпоинт или через первый элемент в списке
        const users = await response.json();
        // Для демонстрации предположим, что первый пользователь в списке - это текущий
        // В реальном приложении лучше добавить отдельный эндпоинт GET /api/users/me
        return users.find((user: User) => user.is_admin) || users[0];
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
    }
};