// components/Login.tsx
import React, { useState } from 'react';
import { useAuth } from '../context/AuthContext';

const Login: React.FC = () => {
    const [apiKey, setApiKey] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const { login, error } = useAuth();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!apiKey.trim()) return;

        setIsLoading(true);
        try {
            await login(apiKey);
        } catch (err) {
            // Ошибка уже обрабатывается в AuthContext
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <div className="login-container">
            <div className="login-form">
                <h2>Вход в систему</h2>
                <p>Пожалуйста, введите свой API ключ для доступа к системе.</p>

                {error && <div className="error-message">{error}</div>}

                <form onSubmit={handleSubmit}>
                    <div className="form-group">
                        <label htmlFor="apiKey">API Ключ</label>
                        <input
                            type="password"
                            id="apiKey"
                            value={apiKey}
                            onChange={(e) => setApiKey(e.target.value)}
                            disabled={isLoading}
                            placeholder="Введите ваш API ключ"
                        />
                    </div>

                    <button
                        type="submit"
                        className="login-button"
                        disabled={isLoading || !apiKey.trim()}
                    >
                        {isLoading ? 'Вход...' : 'Войти'}
                    </button>
                </form>
            </div>
        </div>
    );
};

export default Login;