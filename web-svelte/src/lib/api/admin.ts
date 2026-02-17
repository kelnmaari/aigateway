import { api } from './client';

// ==================== Stats ====================
export interface AdminStats {
	total_users: number;
	total_tenants: number;
	total_api_keys: number;
	total_requests: number;
	loaded_at: string;
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
	oidc_subject?: string;
	oidc_issuer?: string;
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
	id: string;
	category: string;
	key: string;
	value: string;
	type: string;
	default_value?: string;
	description?: string;
	is_editable: boolean;
	is_required: boolean;
	is_migrated: boolean;
	requires_restart: boolean;
	validation_rule?: string;
	updated_at?: string;
	updated_by?: string;
}

export interface SettingsResponse {
	// Backend returns settings grouped by category
	settings: Record<string, Setting[]>;
}

// ==================== Logs ====================
export interface LogEntry {
	timestamp: string;
	level: string;
	message: string;
	raw?: string;
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
// Go runtime metrics
export interface AppMetrics {
	timestamp: string;
	heap_alloc_mb: number;
	heap_sys_mb: number;
	num_gc: number;
	num_goroutines: number;
	num_cpu: number;
	gc_pause_ms: number;
	alloc_rate_mb_s: number;
}

// System-level metrics (gopsutil)
export interface SystemInfo {
	cpu_percent: number;
	cpu_cores: number;
	memory_used: number;
	memory_total: number;
	memory_percent: number;
	disk_used: number;
	disk_total: number;
	disk_percent: number;
	uptime: number;
	app_uptime: number;
}

export interface SystemMetrics {
	status: string;
	system: SystemInfo;
	app?: AppMetrics | null;
	baseline?: AppMetrics | null;
}

// ==================== Invitations ====================
export interface Invitation {
	id: string;
	token: string;
	created_by_user_id: string;
	created_at: string;
	email?: string;
	expires_at?: string;
	max_uses: number;
	current_uses: number;
	used_at?: string;
	used_by_user_id?: string;
	revoked_at?: string;
	revoked_by_user_id?: string;
}

export interface InvitationsResponse {
	invitations: Invitation[];
	count: number;
}

export interface CreateInvitationResponse {
	invitation: Invitation;
	invitation_link: string;
}

export interface CreateInvitationRequest {
	email?: string;
	expires_at?: string;  // ISO date string
	max_uses?: number;
}

// ==================== RAG Stats ====================
export interface VectorStats {
	total_vectors: number;
	dimensions: number;
	index_type: string;
	collection_url?: string;
}

export interface RAGStats {
	enabled: boolean;
	provider: string;
	vector_stats?: VectorStats;
	health_status: string;
	loaded_at: string;
}

// ==================== GPU Metrics ====================
export interface GPUDevice {
	index: number;
	name: string;
	uuid: string;
	temperature_c: number;
	power_usage_w: number;
	power_limit_w: number;
	utilization_gpu_percent: number;
	utilization_memory_percent: number;
	memory_total_mb: number;
	memory_used_mb: number;
	memory_free_mb: number;
	memory_usage_percent: number;
	fan_speed_percent: number;
	clock_graphics_mhz: number;
	clock_memory_mhz: number;
}

export interface GPUMetrics {
	device_count: number;
	devices: GPUDevice[];
	total_memory_mb: number;
	used_memory_mb: number;
	memory_usage_percent: number;
}

export interface GPUMetricsResponse {
	enabled: boolean;
	message?: string;
	data?: GPUMetrics;
}

// ==================== API ====================
export const adminApi = {
	// Stats
	getStats: () => api.get<AdminStats>('/api/admin/summary'),

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
		enabled
			? api.patch(`/api/admin/users/${id}/enable`, {})
			: api.patch(`/api/admin/users/${id}/disable`, {}),

	// API Keys (admin)
	getAPIKeys: () => api.get<{ api_keys: Array<{ id: string; name: string; key_prefix: string; user_id?: string; username?: string; tenant_id?: string; tenant_name?: string; status: string; all_models: boolean; models?: string[]; created_at: string; last_used_at?: string }> }>('/api/admin/keys'),

	revokeAPIKey: (id: string) => api.patch(`/api/admin/keys/${id}/revoke`, {}),

	// Settings
	getSettings: () => api.get<SettingsResponse>('/api/admin/settings'),

	updateSetting: (id: string, value: string) =>
		api.put(`/api/admin/settings/${id}`, { value }),

	// Logs - API returns { entries: LogEntry[], total: number, returned: number }
	getLogs: async (filename: string) => {
		const response = await api.get<{ entries: LogEntry[]; total: number; returned: number }>(
			`/api/admin/logs/${filename}`
		);
		return { logs: response.entries || [], total: response.total || 0 };
	},

	getLogFiles: () =>
		api.get<{ files: Array<{ name: string; size: number; modified: string; is_current: boolean }> }>(
			'/api/admin/logs'
		),

	// Audit
	getAuditLogs: (page = 1, perPage = 50, filters?: { action?: string; user_id?: string }) => {
		const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
		if (filters?.action) params.set('action', filters.action);
		if (filters?.user_id) params.set('user_id', filters.user_id);
		return api.get<AuditResponse>(`/api/admin/audit?${params}`);
	},

	// Performance
	getMetrics: () => api.get<SystemMetrics>('/api/admin/performance/metrics'),

	// Invitations
	getInvitations: () => api.get<InvitationsResponse>('/api/admin/invitations'),

	createInvitation: (data: CreateInvitationRequest) =>
		api.post<CreateInvitationResponse>('/api/admin/invitations', data),

	revokeInvitation: (id: string) => api.delete(`/api/admin/invitations/${id}`),

	// RAG Stats (v3.2.0+)
	getRAGStats: () => api.get<RAGStats>('/api/admin/rag/stats'),

	// GPU Metrics
	getGPUMetrics: () => api.get<GPUMetricsResponse>('/api/gpu/metrics'),

	// Backend status (v3.3.x) - shows which inference backend is active
	getBackendStatus: () => api.get<BackendStatusResponse>('/api/system/backend'),

	// Docker images for inference providers
	getDockerImages: () => api.get<DockerImagesResponse>('/api/system/inference/docker-images'),
	pullDockerImage: (image: string) => api.post<{ message: string; image: string }>(`/api/system/inference/docker-images/pull?image=${encodeURIComponent(image)}`)
};

export interface BackendStatusResponse {
	backend: 'docker';
	ready: boolean;
	loaded_models?: number;
	running_models?: number;
	max_running_models?: number;
	docker_enabled?: boolean;
}

export interface DockerImageStatus {
	provider: string;
	image: string;
	exists: boolean;
	size?: string;
	pulling?: boolean;
}

export interface DockerImagesResponse {
	images: DockerImageStatus[];
	error?: string;
}

