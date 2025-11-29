import { api } from './client';

export interface APIKey {
	id: string;
	name: string;
	key_prefix: string;
	description?: string;
	models: string[];
	all_models: boolean;
	rate_limit_rpm?: number;
	rate_limit_rph?: number;
	status: 'active' | 'disabled' | 'revoked';
	created_at: string;
	last_used_at?: string;
	expires_at?: string;
	tenant_id?: string;
}

export interface APIKeysResponse {
	api_keys: APIKey[];
	total: number;
}

export interface CreateAPIKeyRequest {
	name: string;
	description?: string;
	models?: string[];
	all_models?: boolean;
	rate_limit_rpm?: number;
	rate_limit_rph?: number;
	expires_at?: string;
}

export interface CreateAPIKeyResponse {
	api_key: APIKey;
	key: string; // Full key shown only once
}

export interface Tenant {
	id: string;
	name: string;
	slug: string;
	type: 'personal' | 'organization';
	status: 'active' | 'inactive';
	member_count?: number;
	role?: string;
}

export interface TenantsResponse {
	tenants: Tenant[];
}

export const apiKeysApi = {
	// Personal API Keys
	getPersonalKeys: () => api.get<APIKeysResponse>('/api/users/me/api-keys'),

	createPersonalKey: (data: CreateAPIKeyRequest) =>
		api.post<CreateAPIKeyResponse>('/api/users/me/api-keys', data),

	deletePersonalKey: (keyId: string) => api.delete(`/api/users/me/api-keys/${keyId}`),

	// Tenant API Keys
	getTenantKeys: (tenantId: string) => api.get<APIKeysResponse>(`/api/tenants/${tenantId}/api-keys`),

	createTenantKey: (tenantId: string, data: CreateAPIKeyRequest) =>
		api.post<CreateAPIKeyResponse>(`/api/tenants/${tenantId}/api-keys`, data),

	deleteTenantKey: (tenantId: string, keyId: string) =>
		api.delete(`/api/tenants/${tenantId}/api-keys/${keyId}`),

	// User Tenants
	getUserTenants: () => api.get<TenantsResponse>('/api/users/me/tenants')
};

