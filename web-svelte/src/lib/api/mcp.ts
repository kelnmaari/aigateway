import { api } from './client';

export interface MCPServer {
	id: string;
	name: string;
	description?: string;
	category?: string;
	type: 'stdio' | 'sse' | 'websocket';
	command?: string;
	args?: string[];
	url?: string;
	env?: Record<string, string>;
	status: 'running' | 'stopped' | 'error';
	enabled: boolean;
	created_at: string;
	updated_at: string;
	error_message?: string;
	tools?: MCPTool[];
	resources?: MCPResource[];
	// Catalog fields
	tags?: string[];
	installation_guide?: string;
	github_url?: string;
	website_url?: string;
}

export interface MCPTool {
	name: string;
	description?: string;
	input_schema?: Record<string, unknown>;
}

export interface MCPResource {
	uri: string;
	name: string;
	description?: string;
	mime_type?: string;
}

export interface MCPServersResponse {
	servers: MCPServer[];
	total?: number;
}

export interface MCPCategoriesResponse {
	categories: string[];
}

export interface CreateMCPServerRequest {
	name: string;
	description?: string;
	category?: string;
	type: 'stdio' | 'sse' | 'websocket';
	command?: string;
	args?: string[];
	url?: string;
	env?: Record<string, string>;
	enabled?: boolean;
	tags?: string[];
	installation_guide?: string;
	github_url?: string;
	website_url?: string;
}

export interface ListServersOptions {
	limit?: number;
	offset?: number;
	search?: string;
	category?: string;
	sort_by?: 'created_at' | 'name' | 'category';
	sort_order?: 'asc' | 'desc';
	active_only?: boolean;
}

export const mcpApi = {
	// Catalog - Public Servers (read-only)
	getServers: (options: ListServersOptions = {}) => {
		const params = new URLSearchParams();
		if (options.limit) params.set('limit', String(options.limit));
		if (options.offset) params.set('offset', String(options.offset));
		if (options.search) params.set('search', options.search);
		if (options.category) params.set('category', options.category);
		if (options.sort_by) params.set('sort_by', options.sort_by);
		if (options.sort_order) params.set('sort_order', options.sort_order);
		if (options.active_only) params.set('active_only', 'true');
		const query = params.toString();
		return api.get<MCPServersResponse>(`/api/mcp/servers${query ? `?${query}` : ''}`);
	},

	getServer: (id: string) => api.get<MCPServer>(`/api/mcp/servers/${id}`),

	getCategories: () => api.get<MCPCategoriesResponse>('/api/mcp/categories'),

	// Admin Servers (requires admin)
	createServer: (data: CreateMCPServerRequest) => api.post<MCPServer>('/api/admin/mcp/servers', data),

	updateServer: (id: string, data: Partial<CreateMCPServerRequest>) =>
		api.put<MCPServer>(`/api/admin/mcp/servers/${id}`, data),

	deleteServer: (id: string) => api.delete(`/api/admin/mcp/servers/${id}`),

	// Control (admin)
	startServer: (id: string) => api.post(`/api/admin/mcp/servers/${id}/start`, {}),

	stopServer: (id: string) => api.post(`/api/admin/mcp/servers/${id}/stop`, {}),

	restartServer: (id: string) => api.post(`/api/admin/mcp/servers/${id}/restart`, {}),

	// Tools & Resources
	getTools: (id: string) => api.get<{ tools: MCPTool[] }>(`/api/mcp/servers/${id}/tools`),

	getResources: (id: string) =>
		api.get<{ resources: MCPResource[] }>(`/api/mcp/servers/${id}/resources`),

	// Test connection
	testConnection: (data: CreateMCPServerRequest) =>
		api.post<{ success: boolean; message?: string }>('/api/admin/mcp/test', data)
};
