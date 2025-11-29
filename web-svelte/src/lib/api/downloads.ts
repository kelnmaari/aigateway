import { api } from './client';

export interface Download {
	id: string;
	name: string;
	model_name: string;
	source: 'ollama' | 'huggingface' | 'custom';
	status: 'pending' | 'downloading' | 'completed' | 'failed' | 'cancelled';
	progress: number;
	total_size: number;
	downloaded_size: number;
	speed?: number;
	eta?: number;
	error_message?: string;
	started_at?: string;
	completed_at?: string;
	created_at: string;
}

export interface DownloadsResponse {
	downloads: Download[];
}

export interface StartDownloadRequest {
	model_name: string;
	source?: 'ollama' | 'huggingface' | 'custom';
	url?: string;
}

export interface HuggingFaceModel {
	id: string;
	author: string;
	model_name: string;
	downloads: number;
	likes: number;
	tags: string[];
	last_modified: string;
}

export interface HuggingFaceSearchResponse {
	models: HuggingFaceModel[];
	total: number;
}

export const downloadsApi = {
	// Downloads
	getDownloads: () => api.get<DownloadsResponse>('/api/downloads'),

	startDownload: (data: StartDownloadRequest) => api.post<Download>('/api/downloads', data),

	cancelDownload: (id: string) => api.post(`/api/downloads/${id}/cancel`, {}),

	retryDownload: (id: string) => api.post(`/api/downloads/${id}/retry`, {}),

	deleteDownload: (id: string) => api.delete(`/api/downloads/${id}`),

	// Hugging Face
	searchHuggingFace: (query: string, page = 1, limit = 20) => {
		const params = new URLSearchParams({
			q: query,
			page: String(page),
			limit: String(limit)
		});
		return api.get<HuggingFaceSearchResponse>(`/api/huggingface/search?${params}`);
	},

	// Ollama library
	getOllamaModels: () =>
		api.get<{ models: Array<{ name: string; description: string; tags: string[] }> }>(
			'/api/ollama/library'
		)
};

