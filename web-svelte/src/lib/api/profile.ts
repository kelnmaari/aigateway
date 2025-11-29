import { api } from './client';

export interface UserProfile {
	id: string;
	username: string;
	email: string;
	full_name?: string;
	is_admin: boolean;
	status: 'active' | 'disabled';
	created_at: string;
	updated_at: string;
	last_login_at?: string;
	auth_provider?: string;
}

export interface UpdateProfileRequest {
	full_name?: string;
	email?: string;
}

export interface ChangePasswordRequest {
	current_password: string;
	new_password: string;
}

export interface Device {
	id: string;
	device_name: string;
	device_type: string;
	ip_address: string;
	user_agent?: string;
	last_active_at: string;
	created_at: string;
	is_current: boolean;
}

export interface DevicesResponse {
	devices: Device[];
}

export const profileApi = {
	// Profile
	getProfile: () => api.get<UserProfile>('/api/users/me'),

	updateProfile: (data: UpdateProfileRequest) => api.put<UserProfile>('/api/users/me', data),

	// Password
	changePassword: (data: ChangePasswordRequest) =>
		api.post('/api/users/me/password', data),

	// Devices / Sessions (API: /api/auth/devices)
	getDevices: () => api.get<DevicesResponse>('/api/auth/devices'),

	revokeDevice: (deviceId: string) => api.delete(`/api/auth/devices/${deviceId}`),

	revokeAllDevices: () => api.delete('/api/auth/devices')
};

