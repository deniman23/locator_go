// components/UserManagement.tsx
import React, { useEffect, useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { userApi } from '../services/api';
import type {User} from '../types/models';

const UserManagement: React.FC = () => {
    const { apiKey, user: currentUser } = useAuth();
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Состояние для формы создания пользователя
    const [newUserName, setNewUserName] = useState('');
    const [newUserIsAdmin, setNewUserIsAdmin] = useState(false);
    const [creating, setCreating] = useState(false);
    const [createError, setCreateError] = useState<string | null>(null);

    // Загрузка списка пользователей
    useEffect(() => {
        const fetchUsers = async () => {
            if (!apiKey) return;

            try {
                setLoading(true);
                const userList = await userApi.getAll(apiKey);
                setUsers(userList);
                setError(null);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Ошибка при загрузке пользователей');
            } finally {
                setLoading(false);
            }
        };

        fetchUsers();
    }, [apiKey]);

    // Обработчик создания нового пользователя
    const handleCreateUser = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!apiKey || !newUserName.trim()) return;

        try {
            setCreating(true);
            setCreateError(null);

            const newUser = await userApi.create(newUserName, newUserIsAdmin, apiKey);

            // Добавляем нового пользователя в список
            setUsers(prev => [...prev, newUser]);

            // Очищаем форму
            setNewUserName('');
            setNewUserIsAdmin(false);
        } catch (err) {
            setCreateError(err instanceof Error ? err.message : 'Ошибка при создании пользователя');
        } finally {
            setCreating(false);
        }
    };

    // Если текущий пользователь не админ, не показываем страницу управления
    if (currentUser && !currentUser.is_admin) {
        return (
            <div className="user-management">
                <h2>Доступ запрещен</h2>
                <p>У вас нет прав для доступа к этой странице.</p>
            </div>
        );
    }

    return (
        <div className="user-management">
            <h2>Управление пользователями</h2>

            {/* Форма создания пользователя */}
            <div className="create-user-form">
                <h3>Создать нового пользователя</h3>
                {createError && <div className="error-message">{createError}</div>}

                <form onSubmit={handleCreateUser}>
                    <div className="form-group">
                        <label htmlFor="userName">Имя пользователя</label>
                        <input
                            type="text"
                            id="userName"
                            value={newUserName}
                            onChange={(e) => setNewUserName(e.target.value)}
                            disabled={creating}
                            placeholder="Введите имя пользователя"
                            required
                        />
                    </div>

                    <div className="form-check">
                        <input
                            type="checkbox"
                            id="isAdmin"
                            checked={newUserIsAdmin}
                            onChange={(e) => setNewUserIsAdmin(e.target.checked)}
                            disabled={creating}
                        />
                        <label htmlFor="isAdmin">Администратор</label>
                    </div>

                    <button
                        type="submit"
                        className="create-button"
                        disabled={creating || !newUserName.trim()}
                    >
                        {creating ? 'Создание...' : 'Создать пользователя'}
                    </button>
                </form>
            </div>

            {/* Список пользователей */}
            <div className="users-list">
                <h3>Список пользователей</h3>

                {loading ? (
                    <p>Загрузка пользователей...</p>
                ) : error ? (
                    <div className="error-message">{error}</div>
                ) : users.length === 0 ? (
                    <p>Пользователи не найдены.</p>
                ) : (
                    <table className="users-table">
                        <thead>
                        <tr>
                            <th>ID</th>
                            <th>Имя</th>
                            <th>Роль</th>
                            <th>Дата создания</th>
                        </tr>
                        </thead>
                        <tbody>
                        {users.map(user => (
                            <tr key={user.id}>
                                <td>{user.id}</td>
                                <td>{user.name}</td>
                                <td>{user.is_admin ? 'Администратор' : 'Пользователь'}</td>
                                <td>{new Date(user.created_at).toLocaleString()}</td>
                            </tr>
                        ))}
                        </tbody>
                    </table>
                )}
            </div>
        </div>
    );
};

export default UserManagement;