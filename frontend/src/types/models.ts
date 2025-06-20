// Типы, соответствующие моделям Go
export interface User {
    id: number;
    name: string;
    created_at: string;
    updated_at: string;
}

export interface Checkpoint {
    id: number;
    name: string;
    latitude: number;
    longitude: number;
    radius: number;
    created_at: string;
    updated_at: string;
}

export interface Location {
    id: number;
    user_id: number;
    latitude: number;
    longitude: number;
    created_at: string;
    updated_at: string;
}

export interface Visit {
    id: number;
    user_id: number;
    checkpoint_id: number;
    start_at: string;
    end_at: string | null;
    duration: number;
}

export interface LocationEvent {
    user_id: number;
    checkpoint_id: number;
    latitude: number;
    longitude: number;
    occurred_at: string;
}
export interface User {
    id: number;
    name: string;
    is_admin: boolean;
    created_at: string;
    updated_at: string;
}

export interface AuthState {
    isAuthenticated: boolean;
    user: User | null;
    apiKey: string | null;
    loading: boolean;
    error: string | null;
}