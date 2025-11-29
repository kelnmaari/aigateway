import { api } from './client';

export interface RAGSource {
	id: string;
	name: string;
	type: 'file' | 'url' | 'database' | 'api';
	status: 'active' | 'indexing' | 'error' | 'disabled';
	config: Record<string, unknown>;
	document_count: number;
	chunk_count: number;
	last_indexed_at?: string;
	created_at: string;
	updated_at: string;
	error_message?: string;
}

export interface RAGSourcesResponse {
	sources: RAGSource[];
	total: number;
}

export interface CreateRAGSourceRequest {
	name: string;
	type: 'file' | 'url' | 'database' | 'api';
	config: Record<string, unknown>;
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

	// Indexing
	reindexSource: (id: string) => api.post(`/api/rag/sources/${id}/reindex`, {}),

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

