import { api } from './client';

export interface RAGSource {
	id: string;
	name: string;
	description?: string;
	type: 'file' | 'url' | 'database' | 'api' | 'web';
	status: 'active' | 'indexing' | 'error' | 'disabled' | 'pending';
	config: Record<string, unknown>;
	document_count: number;
	chunk_count: number;
	token_count?: number;
	last_indexed_at?: string;
	created_at: string;
	updated_at: string;
	error_message?: string;
	shared?: boolean;
}

export interface RAGSourcesResponse {
	sources: RAGSource[];
	total: number;
}

export interface CreateRAGSourceRequest {
	name: string;
	description?: string;
	type: 'file' | 'url' | 'database' | 'api' | 'web';
	config: Record<string, unknown>;
	shared?: boolean;
}

export interface TestConnectionRequest {
	type: string;
	config: Record<string, unknown>;
}

export interface TestConnectionResponse {
	success: boolean;
	message: string;
}

export interface RAGDocument {
	id: string;
	source_id: string;
	title: string;
	content_preview: string;
	chunk_count: number;
	indexed_at: string;
	metadata?: Record<string, unknown>;
}

export interface RAGDocumentsResponse {
	documents: RAGDocument[];
	total: number;
}

export const ragApi = {
	// Sources
	getSources: () => api.get<RAGSourcesResponse>('/api/rag/sources'),

	getSource: (id: string) => api.get<RAGSource>(`/api/rag/sources/${id}`),

	createSource: (data: CreateRAGSourceRequest) => api.post<RAGSource>('/api/rag/sources', data),

	updateSource: (id: string, data: Partial<CreateRAGSourceRequest>) =>
		api.put<RAGSource>(`/api/rag/sources/${id}`, data),

	deleteSource: (id: string) => api.delete(`/api/rag/sources/${id}`),

	// Test connection before creating
	testConnection: (data: TestConnectionRequest) =>
		api.post<TestConnectionResponse>('/api/rag/test-connection', data),

	// Indexing
	reindexSource: (id: string) => api.post(`/api/rag/sources/${id}/reindex`, {}),

	// Sync (alias for reindex)
	syncSource: (id: string) => api.post(`/api/rag/sources/${id}/sync`, {}),

	// Documents
	getDocuments: (sourceId: string, page = 1, perPage = 20) => {
		const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
		return api.get<RAGDocumentsResponse>(`/api/rag/sources/${sourceId}/documents?${params}`);
	},

	deleteDocument: (sourceId: string, documentId: string) =>
		api.delete(`/api/rag/sources/${sourceId}/documents/${documentId}`),

	// Search
	search: (query: string, sourceIds?: string[], limit = 10) =>
		api.post<{ results: Array<{ document: RAGDocument; score: number; chunk: string }> }>(
			'/api/rag/search',
			{ query, source_ids: sourceIds, limit }
		)
};
