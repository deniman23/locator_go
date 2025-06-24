// context/AuthContext.tsx
import React, { createContext, useContext, useEffect, useState } from 'react';
import type {AuthState} from '../types/models';
import { userApi } from '../services/api';

interface AuthContextType extends AuthState {
    login: (apiKey: string) => Promise<void>;
    logout: () => void;
    refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
};

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [state, setState] = useState<AuthState>({
        isAuthenticated: false,
        user: null,
        apiKey: null,
        loading: true,
        error: null
    });

    // Функция для обновления информации о текущем пользователе
    const refreshUser = async () => {
        if (!state.apiKey) return;

        try {
            setState(prev => ({ ...prev, loading: true }));
            const user = await userApi.getCurrentUser(state.apiKey);

            setState(prev => ({
                ...prev,
                user,
                loading: false
            }));

            // Проверяем, является ли пользователь администратором
            if (!user.is_admin) {
                logout();
                setState(prev => ({
                    ...prev,
                    error: 'Доступ запрещен: требуются права администратора'
                }));
            }

            console.log('Обновлена информация о пользователе:', user);
        } catch (error) {
            console.error('Ошибка обновления данных пользователя:', error);
            setState(prev => ({
                ...prev,
                loading: false,
                error: 'Ошибка получения данных пользователя'
            }));
        }
    };

    // Проверяем наличие сохраненного API ключа при загрузке
    useEffect(() => {
        const savedApiKey = sessionStorage.getItem('apiKey');
        if (savedApiKey) {
            login(savedApiKey).catch(() => {
                // Если ключ не валидный, очищаем sessionStorage
                sessionStorage.removeItem('apiKey');
                setState({
                    isAuthenticated: false,
                    user: null,
                    apiKey: null,
                    loading: false,
                    error: 'Сессия истекла. Пожалуйста, войдите снова.'
                });
            });
        } else {
            setState(prev => ({ ...prev, loading: false }));
        }
    }, []);

    const login = async (apiKey: string) => {
        try {
            setState(prev => ({ ...prev, loading: true, error: null }));

            // Аутентификация пользователя с API ключом
            const user = await userApi.authenticate(apiKey);

            // Проверяем, является ли пользователь администратором
            if (!user.is_admin) {
                setState({
                    isAuthenticated: false,
                    user: null,
                    apiKey: null,
                    loading: false,
                    error: 'Доступ запрещен: требуются права администратора'
                });
                throw new Error('Доступ запрещен: требуются права администратора');
            }

            // Сохраняем API ключ в sessionStorage для текущей сессии
            sessionStorage.setItem('apiKey', apiKey);

            setState({
                isAuthenticated: true,
                user,
                apiKey,
                loading: false,
                error: null
            });

            console.log('Пользователь успешно вошел:', user);
        } catch (error) {
            setState({
                isAuthenticated: false,
                user: null,
                apiKey: null,
                loading: false,
                error: error instanceof Error ? error.message : 'Ошибка авторизации'
            });
            throw error;
        }
    };

    const logout = () => {
        sessionStorage.removeItem('apiKey');
        setState({
            isAuthenticated: false,
            user: null,
            apiKey: null,
            loading: false,
            error: null
        });
    };

    return (
        <AuthContext.Provider value={{ ...state, login, logout, refreshUser }}>
            {children}
        </AuthContext.Provider>
    );
};