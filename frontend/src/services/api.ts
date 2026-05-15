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

export type LocationFetchOpts = {
    /** true — все точки из БД без серверного «значимых» фильтра */
    raw?: boolean;
};

// API для работы с локациями
export const locationApi = {
    getAll: (apiKey?: string, opts?: LocationFetchOpts) =>
        api.get<Location[]>('/location/', {
            ...withApiKey(apiKey),
            params: opts?.raw ? { raw: 'true' } : undefined
        }),

    getBetween: (from: string, to: string, apiKey?: string, opts?: LocationFetchOpts) =>
        api.get<Location[]>('/location/', {
            ...withApiKey(apiKey),
            params: {
                from,
                to,
                ...(opts?.raw ? { raw: 'true' } : {})
            }
        }),

    getByUserId: (userId: number, apiKey?: string, maxAgeSeconds?: number) =>
        api.get<Location>(`/location/single`, {
            ...withApiKey(apiKey),
            params: {
                user_id: userId,
                ...(maxAgeSeconds != null ? { max_age_seconds: maxAgeSeconds } : {})
            }
        }),

    /** Последняя позиция текущего пользователя (по API-ключу) */
    getCurrent: (apiKey?: string, maxAgeSeconds?: number) =>
        api.get<Location>(`/location/current`, {
            ...withApiKey(apiKey),
            params: maxAgeSeconds != null ? { max_age_seconds: maxAgeSeconds } : undefined
        }),

    createLocation: (
        location: {
            user_id?: number;
            latitude: number;
            longitude: number;
            request_id?: string;
            source?: 'periodic' | 'on_demand';
        },
        apiKey?: string
    ) => api.post<Location>('/location', location, withApiKey(apiKey)),

    requestOnDemand: (userId: number, apiKey?: string) =>
        api.post<{ request_id: string; status: string; user_id: number }>(
            '/location/request',
            { user_id: userId },
            withApiKey(apiKey)
        ),

    getRequestStatus: (requestId: string, apiKey?: string) =>
        api.get<{ request_id: string; user_id: number; status: string; created_at: string; completed_at?: string }>(
            `/location/request/${requestId}`,
            withApiKey(apiKey)
        ),

    /** OSRM match; нужен ROUTING_BASE_URL на сервере. Координаты [lat, lng] */
    getMatchedRoute: (userId: number, from: string, to: string, apiKey?: string) =>
        api.get<{ coordinates: [number, number][] }>('/location/match-route', {
            ...withApiKey(apiKey),
            params: { user_id: userId, from, to }
        })
};

export const deviceApi = {
    sendCommand: (
        userId: number,
        body: { type: string; payload?: Record<string, unknown> },
        apiKey?: string
    ) =>
        api.post<{ command_id: string; type: string; status: string; user_id: number; payload?: Record<string, unknown> }>(
            `/admin/users/${userId}/commands`,
            body,
            withApiKey(apiKey)
        ),

    getUserHealth: (userId: number, apiKey?: string) =>
        api.get<{
            user_id: number;
            last_report_at: string;
            app_version?: string;
            platform?: string;
            issues: string[];
            issue_count: number;
            healthy: boolean;
            report: Record<string, unknown>;
        }>(`/users/${userId}/health`, withApiKey(apiKey))
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
    // Получение визитов с возможностью фильтрации
    getWithFilters: (params: {
        id?: number,
        user_id?: number,
        checkpoint_id?: number
    }, apiKey?: string) => {
        // Создаем строку запроса из параметров
        const queryParams = new URLSearchParams();
        if (params.id) queryParams.append('id', params.id.toString());
        if (params.user_id) queryParams.append('user_id', params.user_id.toString());
        if (params.checkpoint_id) queryParams.append('checkpoint_id', params.checkpoint_id.toString());

        const queryString = queryParams.toString();
        const url = queryString ? `/visits/?${queryString}` : '/visits/';

        return api.get<Visit[]>(url, withApiKey(apiKey));
    },

    // Получение всех визитов (для админов)
    getAll: (apiKey?: string) =>
        api.get<Visit[]>('/visits/', withApiKey(apiKey)),

    /** Только активные визиты (end_at IS NULL) по всем пользователям */
    getActive: (apiKey?: string) =>
        api.get<Visit[]>('/visits/?active=true', withApiKey(apiKey)),

    // Получение визита по ID
    getById: (id: number, apiKey?: string) =>
        api.get<Visit[]>(`/visits/?id=${id}`, withApiKey(apiKey)),

    // Получение визитов пользователя
    getByUserId: (userId: number, apiKey?: string) =>
        api.get<Visit[]>(`/visits/?user_id=${userId}`, withApiKey(apiKey)),

    // Получение визитов для чекпоинта
    getByCheckpointId: (checkpointId: number, apiKey?: string) =>
        api.get<Visit[]>(`/visits/?checkpoint_id=${checkpointId}`, withApiKey(apiKey))
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