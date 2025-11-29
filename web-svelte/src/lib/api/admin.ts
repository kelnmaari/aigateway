import { api } from './client';

// ==================== Stats ====================
export interface AdminStats {
	users: number;
	api_keys: number;
	models: number;
	requests_today: number;
	requests_total: number;
}

// ==================== Users ====================
export interface AdminUser {
	id: string;
	username: string;
	email: string;
	full_name?: string;
	is_admin: boolean;
	status: 'active' | 'disabled' | 'pending';
	auth_provider: string;
	created_at: string;
	updated_at: string;
	last_login_at?: string;
	roles?: string[];
	tenants?: Array<{ id: string; name: string; role: string }>;
}

export interface AdminUsersResponse {
	users: AdminUser[];
	total: number;
	page: number;
	per_page: number;
}

export interface CreateUserRequest {
	username: string;
	email: string;
	password: string;
	full_name?: string;
	is_admin?: boolean;
}

export interface UpdateUserRequest {
	email?: string;
	full_name?: string;
	is_admin?: boolean;
	status?: 'active' | 'disabled';
}

// ==================== Settings ====================
export interface Setting {
	key: string;
	value: string;
	category: string;
	description?: string;
	type: 'string' | 'number' | 'boolean' | 'json';
	storage: 'database' | 'yaml';
	requires_restart: boolean;
}

export interface SettingsResponse {
	settings: Setting[];
}

// ==================== Logs ====================
export interface LogEntry {
	timestamp: string;
	level: string;
	message: string;
	fields?: Record<string, unknown>;
}

export interface LogsResponse {
	logs: LogEntry[];
	total: number;
}

export interface AuditEntry {
	id: string;
	user_id?: string;
	username?: string;
	action: string;
	resource: string;
	resource_id?: string;
	ip_address: string;
	user_agent?: string;
	status: 'success' | 'failure';
	details?: Record<string, unknown>;
	created_at: string;
}

export interface AuditResponse {
	events: AuditEntry[];
	total: number;
}

// ==================== Performance ====================
export interface SystemMetrics {
	cpu_percent: number;
	memory_used: number;
	memory_total: number;
	goroutines: number;
	uptime: number;
	requests_per_second: number;
}

// ==================== Invitations ====================
export interface Invitation {
	id: string;
	email?: string;
	token: string;
	role: string;
	created_by: string;
	created_at: string;
	expires_at: string;
	used_at?: string;
	used_by?: string;
	status: 'pending' | 'used' | 'expired' | 'revoked';
}

export interface InvitationsResponse {
	invitations: Invitation[];
	total: number;
}

export interface CreateInvitationRequest {
	email?: string;
	role?: string;
	expires_in_days?: number;
}

// ==================== API ====================
export const adminApi = {
	// Stats
	getStats: () => api.get<AdminStats>('/api/admin/stats'),

	// Users
	getUsers: (page = 1, perPage = 20, search?: string) => {
		const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
		if (search) params.set('search', search);
		return api.get<AdminUsersResponse>(`/api/admin/users?${params}`);
	},

	getUser: (id: string) => api.get<AdminUser>(`/api/admin/users/${id}`),

	createUser: (data: CreateUserRequest) => api.post<AdminUser>('/api/admin/users', data),

	updateUser: (id: string, data: UpdateUserRequest) =>
		api.put<AdminUser>(`/api/admin/users/${id}`, data),

	deleteUser: (id: string) => api.delete(`/api/admin/users/${id}`),

	resetPassword: (id: string, newPassword: string) =>
		api.post(`/api/admin/users/${id}/reset-password`, { password: newPassword }),

	toggleUserStatus: (id: string, enabled: boolean) =>
		api.put(`/api/admin/users/${id}`, { status: enabled ? 'active' : 'disabled' }),

	// Settings
	getSettings: () => api.get<SettingsResponse>('/api/admin/settings'),

	updateSetting: (key: string, value: string) =>
		api.put(`/api/admin/settings/${key}`, { value }),

	// Logs
	getLogs: (file = 'app', level?: string, limit = 100) => {
		const params = new URLSearchParams({ file, limit: String(limit) });
		if (level) params.set('level', level);
		return api.get<LogsResponse>(`/api/admin/logs?${params}`);
	},

	getLogFiles: () => api.get<{ files: string[] }>('/api/admin/logs/files'),

	// Audit
	getAuditLogs: (page = 1, perPage = 50, filters?: { action?: string; user_id?: string }) => {
		const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
		if (filters?.action) params.set('action', filters.action);
		if (filters?.user_id) params.set('user_id', filters.user_id);
		return api.get<AuditResponse>(`/api/admin/audit?${params}`);
	},

	// Performance
	getMetrics: () => api.get<SystemMetrics>('/api/admin/metrics'),

	// Invitations
	getInvitations: () => api.get<InvitationsResponse>('/api/admin/invitations'),

	createInvitation: (data: CreateInvitationRequest) =>
		api.post<Invitation>('/api/admin/invitations', data),

	revokeInvitation: (id: string) => api.delete(`/api/admin/invitations/${id}`)
};

