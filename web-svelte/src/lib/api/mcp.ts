import { api } from './client';

export interface MCPServer {
	id: string;
	name: string;
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
}

export interface CreateMCPServerRequest {
	name: string;
	type: 'stdio' | 'sse' | 'websocket';
	command?: string;
	args?: string[];
	url?: string;
	env?: Record<string, string>;
	enabled?: boolean;
}

export const mcpApi = {
	// Servers
	getServers: () => api.get<MCPServersResponse>('/api/mcp/servers'),

	getServer: (id: string) => api.get<MCPServer>(`/api/mcp/servers/${id}`),

	createServer: (data: CreateMCPServerRequest) => api.post<MCPServer>('/api/mcp/servers', data),

	updateServer: (id: string, data: Partial<CreateMCPServerRequest>) =>
		api.put<MCPServer>(`/api/mcp/servers/${id}`, data),

	deleteServer: (id: string) => api.delete(`/api/mcp/servers/${id}`),

	// Control
	startServer: (id: string) => api.post(`/api/mcp/servers/${id}/start`, {}),

	stopServer: (id: string) => api.post(`/api/mcp/servers/${id}/stop`, {}),

	restartServer: (id: string) => api.post(`/api/mcp/servers/${id}/restart`, {}),

	// Tools & Resources
	getTools: (id: string) => api.get<{ tools: MCPTool[] }>(`/api/mcp/servers/${id}/tools`),

	getResources: (id: string) =>
		api.get<{ resources: MCPResource[] }>(`/api/mcp/servers/${id}/resources`),

	// Test connection
	testConnection: (data: CreateMCPServerRequest) =>
		api.post<{ success: boolean; message?: string }>('/api/mcp/test', data)
};

