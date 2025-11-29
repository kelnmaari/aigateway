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
	// Downloads (HuggingFace UI API)
	getDownloads: () => api.get<DownloadsResponse>('/api/ui/hf/downloads'),

	startDownload: (modelId: string) => api.post<Download>('/api/ui/hf/download', { model_id: modelId }),

	cancelDownload: (id: string) => api.post(`/api/ui/hf/downloads/${id}/cancel`, {}),

	pauseDownload: (id: string) => api.post(`/api/ui/hf/downloads/${id}/pause`, {}),

	getDownloadProgress: (id: string) => api.get<Download>(`/api/ui/hf/downloads/${id}/progress`),

	// Hugging Face search
	searchHuggingFace: (query: string) => {
		const params = new URLSearchParams({ q: query });
		return api.get<HuggingFaceSearchResponse>(`/api/ui/hf/search?${params}`);
	},

	getPopularModels: () => api.get<HuggingFaceSearchResponse>('/api/ui/hf/popular'),

	// Ollama pull (via yzma)
	pullOllamaModel: (model: string) => api.post('/v1/yzma/models/load', { name: model })
};

