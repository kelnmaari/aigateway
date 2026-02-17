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

	// Hugging Face search (uses /api/ui/huggingface/ with provider filter support)
	// provider filter: 'all' | 'vllm' | 'sglang' | 'tgi' | 'llama.cpp' | 'embedding'
	searchHuggingFace: (query: string, provider: string = 'all') => {
		const params = new URLSearchParams();
		if (query) {
			params.set('q', query);
		}
		if (provider && provider !== 'all') {
			params.set('provider', provider);
		}
		return api.get<HuggingFaceSearchResponse>(`/api/ui/huggingface/search?${params}`);
	},

	getPopularModels: (provider: string = 'all', limit: number = 30, page: number = 1) => {
		const params = new URLSearchParams();
		if (provider && provider !== 'all') {
			params.set('provider', provider);
		}
		params.set('limit', limit.toString());
		params.set('page', page.toString());
		return api.get<HuggingFaceSearchResponse>(`/api/ui/huggingface/popular?${params}`);
	}
};

