// context/AuthContext.tsx
import React, { createContext, useContext, useEffect, useState } from 'react';
import type {AuthState} from '../types/models';
import { userApi } from '../services/api';

interface AuthContextType extends AuthState {
    login: (apiKey: string) => Promise<void>;
    logout: () => void;
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

    // Проверяем наличие сохраненного API ключа при загрузке
    useEffect(() => {
        const savedApiKey = localStorage.getItem('apiKey');
        if (savedApiKey) {
            login(savedApiKey).catch(() => {
                // Если ключ не валидный, очищаем localStorage
                localStorage.removeItem('apiKey');
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

            // Сохраняем API ключ в localStorage для последующих сессий
            localStorage.setItem('apiKey', apiKey);

            setState({
                isAuthenticated: true,
                user,
                apiKey,
                loading: false,
                error: null
            });
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
        localStorage.removeItem('apiKey');
        setState({
            isAuthenticated: false,
            user: null,
            apiKey: null,
            loading: false,
            error: null
        });
    };

    return (
        <AuthContext.Provider value={{ ...state, login, logout }}>
            {children}
        </AuthContext.Provider>
    );
};