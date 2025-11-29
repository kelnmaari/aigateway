import { api } from './client';

export interface LoginRequest {
	username: string;
	password: string;
	remember_me?: boolean;
}

export interface LoginResponse {
	access_token: string;
	refresh_token: string;
	expires_at: number;
	user: {
		id: string;
		username: string;
		email: string;
		full_name?: string;
		is_admin: boolean;
	};
}

export interface RegisterRequest {
	username: string;
	email: string;
	password: string;
	full_name?: string;
	invitation_token?: string;
}

export interface BootstrapRequest {
	bootstrap_token: string;
	username: string;
	email: string;
	password: string;
}

export interface InitStatus {
	initialized: boolean;
	has_users: boolean;
	bootstrap_required: boolean;
}

export const authApi = {
	login: (data: LoginRequest) => api.post<LoginResponse>('/api/auth/login', data, { skipAuth: true }),

	register: (data: RegisterRequest) =>
		api.post<LoginResponse>('/api/auth/register', data, { skipAuth: true }),

	bootstrap: (data: BootstrapRequest) =>
		api.post<LoginResponse>('/api/system/bootstrap', data, { skipAuth: true }),

	logout: () => api.post('/api/auth/logout'),

	refresh: (refreshToken: string) =>
		api.post<LoginResponse>('/api/auth/refresh', { refresh_token: refreshToken }, { skipAuth: true }),

	getInitStatus: () => api.get<InitStatus>('/api/system/init-status', { skipAuth: true }),

	getCurrentUser: () =>
		api.get<LoginResponse['user']>('/api/users/me'),

	validateInvitation: (token: string) =>
		api.get<{ valid: boolean; email?: string }>(`/api/invitations/${token}/validate`, {
			skipAuth: true
		})
};

