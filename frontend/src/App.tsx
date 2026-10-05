import React, {useEffect} from 'react';
import { BrowserRouter as Router, Routes, Route, NavLink, Navigate, Link } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import Checkpoints from './pages/Checkpoints';
import UserVisits from './pages/UserVisits';
import Login from './components/Login';
import UserManagement from './components/UserManagement';
import { AuthProvider, useAuth } from './context/AuthContext';
import './App.css';

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

const Navigation: React.FC = () => {
    const { isAuthenticated, user, logout, refreshUser } = useAuth();

    useEffect(() => {
        if (isAuthenticated) {
            refreshUser();
        }
    }, [isAuthenticated, refreshUser]);

    if (!isAuthenticated) return null;

    return (
        <nav className="app-nav">
            <div className="nav-brand-links">
                <Link to="/" className="nav-wordmark">
                    Locator
                </Link>
                <ul>
                    <li>
                        <NavLink to="/" end>
                            Карта
                        </NavLink>
                    </li>
                    <li>
                        <NavLink to="/checkpoints">Чекпоинты</NavLink>
                    </li>
                    <li>
                        <NavLink to="/visits">История визитов</NavLink>
                    </li>
                    {user?.is_admin && (
                        <li>
                            <NavLink to="/users">Пользователи</NavLink>
                        </li>
                    )}
                </ul>
            </div>
            <div className="user-controls">
                <span className="user-info">{user?.name} ({user?.is_admin ? 'Админ' : 'Пользователь'})</span>
                <button onClick={logout} className="logout-button">Выйти</button>
            </div>
        </nav>
    );
};

const NotFound: React.FC = () => (
    <div className="not-found-page">
        <h1>Страница не найдена</h1>
        <p className="empty-state-hint">Проверьте адрес или вернитесь к карте.</p>
        <Link to="/" className="btn-primary">
            На карту
        </Link>
    </div>
);

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

            <Route path="*" element={
                <ProtectedRoute>
                    <NotFound />
                </ProtectedRoute>
            } />
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
