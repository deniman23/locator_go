// context/AuthContext.tsx
import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { AuthState } from '../types/models';
import { userApi } from '../services/api';

interface AuthContextType extends AuthState {
    login: (apiKey: string) => Promise<void>;
    logout: () => void;
    /** Silent profile refresh — does not toggle global `loading` (avoids remounting pages). */
    refreshUser: () => Promise<void>;
    /** Replace session key after QR regenerate (self) without full remount flash. */
    adoptApiKey: (apiKey: string) => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
};

const expiredSession = (error: string): AuthState => ({
    isAuthenticated: false,
    user: null,
    apiKey: null,
    loading: false,
    error,
});

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [state, setState] = useState<AuthState>({
        isAuthenticated: false,
        user: null,
        apiKey: null,
        loading: true,
        error: null,
    });

    const logout = useCallback(() => {
        sessionStorage.removeItem('apiKey');
        setState({
            isAuthenticated: false,
            user: null,
            apiKey: null,
            loading: false,
            error: null,
        });
    }, []);

    const refreshUser = useCallback(async () => {
        if (!state.apiKey) return;

        try {
            // Do not set loading:true — ProtectedRoute would unmount the whole page tree.
            const user = await userApi.getCurrentUser(state.apiKey);

            if (!user.is_admin) {
                sessionStorage.removeItem('apiKey');
                setState(
                    expiredSession('Доступ запрещен: требуются права администратора'),
                );
                return;
            }

            setState((prev) => ({
                ...prev,
                user,
                error: null,
            }));
        } catch (error) {
            console.error('Ошибка обновления данных пользователя:', error);
            sessionStorage.removeItem('apiKey');
            setState(expiredSession('Сессия истекла. Пожалуйста, войдите снова.'));
        }
    }, [state.apiKey]);

    const adoptApiKey = useCallback(async (newApiKey: string) => {
        sessionStorage.setItem('apiKey', newApiKey);
        try {
            const user = await userApi.getCurrentUser(newApiKey);
            if (!user.is_admin) {
                sessionStorage.removeItem('apiKey');
                setState(
                    expiredSession('Доступ запрещен: требуются права администратора'),
                );
                return;
            }
            setState({
                isAuthenticated: true,
                user,
                apiKey: newApiKey,
                loading: false,
                error: null,
            });
        } catch (error) {
            console.error('Ошибка принятия нового API-ключа:', error);
            sessionStorage.removeItem('apiKey');
            setState(expiredSession('Сессия истекла. Пожалуйста, войдите снова.'));
            throw error;
        }
    }, []);

    const login = useCallback(async (apiKey: string) => {
        try {
            setState((prev) => ({ ...prev, loading: true, error: null }));

            const user = await userApi.authenticate(apiKey);

            if (!user.is_admin) {
                setState(
                    expiredSession('Доступ запрещен: требуются права администратора'),
                );
                throw new Error('Доступ запрещен: требуются права администратора');
            }

            sessionStorage.setItem('apiKey', apiKey);

            setState({
                isAuthenticated: true,
                user,
                apiKey,
                loading: false,
                error: null,
            });
        } catch (error) {
            setState(
                expiredSession(
                    error instanceof Error ? error.message : 'Ошибка авторизации',
                ),
            );
            throw error;
        }
    }, []);

    useEffect(() => {
        const savedApiKey = sessionStorage.getItem('apiKey');
        if (savedApiKey) {
            login(savedApiKey).catch(() => {
                sessionStorage.removeItem('apiKey');
                setState(expiredSession('Сессия истекла. Пожалуйста, войдите снова.'));
            });
        } else {
            setState((prev) => ({ ...prev, loading: false }));
        }
    }, [login]);

    const contextValue = useMemo(
        () => ({ ...state, login, logout, refreshUser, adoptApiKey }),
        [state, login, logout, refreshUser, adoptApiKey],
    );

    return (
        <AuthContext.Provider value={contextValue}>
            {children}
        </AuthContext.Provider>
    );
};
