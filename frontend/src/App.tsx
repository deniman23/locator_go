import React, {useEffect} from 'react';
import { BrowserRouter as Router, Routes, Route, Link, Navigate } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import Checkpoints from './pages/Checkpoints';
import UserVisits from './pages/UserVisits';
import Login from './components/Login';
import UserManagement from './components/UserManagement';
import { AuthProvider, useAuth } from './context/AuthContext';
import './App.css';

// Компонент защищенного маршрута
const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const { isAuthenticated, loading } = useAuth();

    if (loading) {
        return <div className="loading-overlay">Загрузка...</div>;
    }

    if (!isAuthenticated) {
        return <Navigate to="/login" replace />;
    }

    return <>{children}</>;
};

// Компонент навигации с учетом состояния авторизации
const Navigation: React.FC = () => {
    const { isAuthenticated, user, logout, refreshUser } = useAuth();

    // Обновляем информацию о пользователе при монтировании компонента
    useEffect(() => {
        if (isAuthenticated) {
            refreshUser();
        }
    }, [isAuthenticated]); // Зависимость от isAuthenticated, чтобы не обновлять при каждом рендере

    if (!isAuthenticated) return null;

    return (
        <nav>
            <ul>
                <li>
                    <Link to="/">Главная</Link>
                </li>
                <li>
                    <Link to="/checkpoints">Чекпоинты</Link>
                </li>
                <li>
                    <Link to="/visits">История визитов</Link>
                </li>
                {user?.is_admin && (
                    <li>
                        <Link to="/users">Пользователи</Link>
                    </li>
                )}
            </ul>
            <div className="user-controls">
                <span className="user-info">{user?.name} ({user?.is_admin ? 'Админ' : 'Пользователь'})</span>
                <button onClick={logout} className="logout-button">Выйти</button>
            </div>
        </nav>
    );
};

// Основные маршруты приложения
const AppRoutes: React.FC = () => {
    const { isAuthenticated } = useAuth();

    return (
        <Routes>
            <Route path="/login" element={
                isAuthenticated ? <Navigate to="/" replace /> : <Login />
            } />

            <Route path="/" element={
                <ProtectedRoute>
                    <Dashboard />
                </ProtectedRoute>
            } />

            <Route path="/checkpoints" element={
                <ProtectedRoute>
                    <Checkpoints />
                </ProtectedRoute>
            } />

            <Route path="/visits" element={
                <ProtectedRoute>
                    <UserVisits />
                </ProtectedRoute>
            } />

            <Route path="/users" element={
                <ProtectedRoute>
                    <UserManagement />
                </ProtectedRoute>
            } />

            <Route path="*" element={<h1>Страница не найдена</h1>} />
        </Routes>
    );
};

const App: React.FC = () => {
    return (
        <AuthProvider>
            <Router>
                <div className="app">
                    <header>
                        <Navigation />
                    </header>

                    <main>
                        <AppRoutes />
                    </main>
                </div>
            </Router>
        </AuthProvider>
    );
};

export default App;