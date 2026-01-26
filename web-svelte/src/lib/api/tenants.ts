import { api } from './client';

export interface Tenant {
	id: string;
	name: string;
	slug: string;
	description?: string;
	type: 'personal' | 'organization';
	status: 'active' | 'inactive';
	created_at: string;
	updated_at: string;
	owner_id: string;
	member_count?: number;
}

export interface TenantMember {
	id: string;
	user_id: string;
	tenant_id: string;
	role: 'owner' | 'admin' | 'member';
	username: string;
	email: string;
	full_name?: string;
	joined_at: string;
}

export interface TenantsResponse {
	tenants: Tenant[];
}

export interface TenantMembersResponse {
	members: TenantMember[];
}

export interface CreateTenantRequest {
	name: string;
	slug: string;
	description?: string;
	type?: 'personal' | 'organization';
}

export interface AddMemberRequest {
	user_id: string;
	role: 'admin' | 'member';
}

export interface UserSearchResult {
	id: string;
	username: string;
	email: string;
	full_name?: string;
}

export const tenantsApi = {
	// User's tenants
	getUserTenants: () => api.get<TenantsResponse>('/api/users/me/tenants'),

	// Tenant CRUD
	getTenant: (id: string) => api.get<Tenant>(`/api/tenants/${id}`),

	createTenant: (data: CreateTenantRequest) => api.post<Tenant>('/api/tenants', data),

	updateTenant: (id: string, data: Partial<CreateTenantRequest>) =>
		api.put<Tenant>(`/api/tenants/${id}`, data),

	deleteTenant: (id: string) => api.delete(`/api/tenants/${id}`),

	// Members
	getMembers: (tenantId: string) => api.get<TenantMembersResponse>(`/api/tenants/${tenantId}/members`),

	addMember: (tenantId: string, data: AddMemberRequest) =>
		api.post<TenantMember>(`/api/tenants/${tenantId}/members`, data),

	updateMemberRole: (tenantId: string, memberId: string, role: string) =>
		api.put(`/api/tenants/${tenantId}/members/${memberId}`, { role }),

	removeMember: (tenantId: string, memberId: string) =>
		api.delete(`/api/tenants/${tenantId}/members/${memberId}`),

	// User search for adding members
	searchUsers: (tenantId: string, query: string) =>
		api.get<{ user: UserSearchResult; already_member: boolean }>(
			`/api/tenants/${tenantId}/search-users?query=${encodeURIComponent(query)}`
		)
};

