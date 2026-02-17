// Registry API types and client functions for multi-provider management

import { api } from './client';

// ==================== Provider Types ====================

export type ProviderType = 'vllm' | 'openai' | 'anthropic' | 'gemini' | 'deepseek' | 'custom';

export interface ModelProvider {
	id: string;
	name: string;
	provider_type: ProviderType;
	base_url: string;
	enabled: boolean;
	priority: number;
	config: Record<string, any>;
	health_status: string;
	last_health_check?: string;
	error_message?: string;
	created_at: string;
	updated_at: string;
}

export interface CreateProviderRequest {
	name: string;
	provider_type: ProviderType;
	base_url: string;
	api_key?: string;
	enabled?: boolean;
	priority?: number;
	config?: Record<string, any>;
}

export interface UpdateProviderRequest {
	name?: string;
	provider_type?: ProviderType;
	base_url?: string;
	api_key?: string;
	enabled?: boolean;
	priority?: number;
	config?: Record<string, any>;
}

// ==================== Model Registry Types ====================

export interface ModelRegistry {
	id: string;
	model_id: string;
	model_name: string;
	provider_id: string;
	capabilities: string[];
	requires_gpu: boolean;
	context_length?: number;
	status: string;
	health_status: string;
	created_at: string;
	updated_at: string;
}

// ==================== Response Types ====================

export interface ProvidersResponse {
	providers: ModelProvider[];
}

export interface ModelsRegistryResponse {
	models: ModelRegistry[];
}

export interface DiscoverModelsResponse {
	message: string;
	models_found: number;
	models: ModelRegistry[];
}

export interface HealthCheckResponse {
	status: string;
	message: string;
	response_time_ms?: number;
}

// ==================== API ====================

export const registryApi = {
	// Providers
	listProviders: () =>
		api.get<ProvidersResponse>('/api/admin/registry/providers'),

	getProvider: (id: string) =>
		api.get<ModelProvider>(`/api/admin/registry/providers/${id}`),

	createProvider: (data: CreateProviderRequest) =>
		api.post<ModelProvider>('/api/admin/registry/providers', data),

	updateProvider: (id: string, data: UpdateProviderRequest) =>
		api.put<ModelProvider>(`/api/admin/registry/providers/${id}`, data),

	deleteProvider: (id: string) =>
		api.delete(`/api/admin/registry/providers/${id}`),

	healthCheckProvider: (id: string) =>
		api.post<HealthCheckResponse>(`/api/admin/registry/providers/${id}/health`),

	discoverModels: (id: string) =>
		api.post<DiscoverModelsResponse>(`/api/admin/registry/providers/${id}/discover`),

	// Models
	listModels: () =>
		api.get<ModelsRegistryResponse>('/api/admin/registry/models'),
};
