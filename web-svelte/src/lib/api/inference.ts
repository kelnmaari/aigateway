// Inference API types and client functions

export type Provider = 'vllm' | 'sglang' | 'tgi' | 'tei' | 'tensorrt-llm' | 'llama.cpp';
export type Format = 'hf' | 'gguf' | 'trt' | 'other';
export type Capability = 'chat' | 'embeddings' | 'vision';

export interface ModelInfo {
	alias: string;
	provider: Provider;
	format: Format;
	status: string;
	endpoint?: string;
	local_path?: string;
	last_used?: string;
	capabilities?: Capability[];
	pinned?: boolean;
	last_error?: string;
	container_id?: string;

	// Source info (for restart)
	hf_repo?: string;
	hf_file?: string;
	gguf_url?: string;

	// vLLM params
	vllm_tensor_parallel?: number;
	vllm_max_model_len?: number;
	vllm_gpu_utilization?: number;

	// llama.cpp params
	llama_main_gpu?: number;
	llama_tensor_split?: string;
	llama_n_gpu_layers?: number;

	// SGLang params
	sglang_tensor_parallel?: number;
	sglang_mem_fraction?: number;

	// TGI params
	tgi_num_shard?: number;
}

export interface ArtifactInfo {
	path: string;
	size: number;
	mod_time: string;
	root: string;
	format: Format;
}

export interface HealthResponse {
	alias: string;
	status: string;
	endpoint?: string;
	response_time_ms?: number;
	error?: string;
}

export interface LogsResponse {
	alias: string;
	logs: string;
	lines: number;
}

export interface MetricsResponse {
	alias: string;
	metrics: string;
	content_type: string;
}

export interface TRTEngine {
	model_id: string;
	engine_path: string;
	created_at: string;
	cuda_version: string;
	trt_version: string;
	driver_version: string;
	gpu_sm: string;
	size_bytes: number;
	compatible: boolean;
}

export interface GPUDevice {
	index: number;
	name: string;
	memory_mb: number;
	memory_free_mb: number;
}

export interface GPUListResponse {
	enabled: boolean;
	devices: GPUDevice[];
	error?: string;
}

export interface TRTConvertRequest {
	hf_model: string;
	max_batch_size?: number;
	max_input_len?: number;
	max_output_len?: number;
	dtype?: string;
}

export interface SavedModel {
	alias: string;
	provider: Provider;
	format: Format;
	capabilities?: Capability[];
	hf_repo?: string;
	hf_file?: string;
	gguf_url?: string;
	gpu_device?: string;
	auto_start: boolean;
	saved_at?: string;
	vllm_tensor_parallel?: number;
	vllm_max_model_len?: number;
	vllm_gpu_utilization?: number;
	llama_main_gpu?: number;
	llama_n_gpu_layers?: number;
	llama_tensor_split?: string;
	sglang_tensor_parallel?: number;
	sglang_mem_fraction?: number;
	tgi_num_shard?: number;
}

export interface UpdateSavedRequest {
	vllm_tensor_parallel?: number;
	vllm_max_model_len?: number;
	vllm_gpu_utilization?: number;
	llama_main_gpu?: number;
	llama_n_gpu_layers?: number;
	llama_tensor_split?: string;
	sglang_tensor_parallel?: number;
	sglang_mem_fraction?: number;
	tgi_num_shard?: number;
	gpu_device?: string;
}

export interface LoadRequest {
	alias: string;
	provider: Provider;
	format: Format;
	hf_repo?: string;
	hf_file?: string;
	hf_revision?: string;
	gguf_url?: string;
	capabilities?: Capability[];
	
	// GPU selection (e.g., "0", "1", "0,1" for specific GPU(s), empty = all)
	gpu_device?: string;
	
	// Provider-specific
	vllm_tensor_parallel?: number;
	vllm_max_model_len?: number;
	vllm_gpu_utilization?: number;
	llama_main_gpu?: number;
	llama_tensor_split?: string;
	llama_n_gpu_layers?: number;
	sglang_tensor_parallel?: number;
	sglang_mem_fraction?: number;
	tgi_num_shard?: number;
}

import { api } from './client';

export const inferenceApi = {
	// Models
	listModels: () => api.get<ModelInfo[]>('/api/system/inference/models'),
	
	load: (req: LoadRequest) => api.post<{ message: string }>('/api/system/inference/load', req),
	
	prepare: (req: LoadRequest) => api.post<{ message: string }>('/api/system/inference/prepare', req),
	
	stop: (alias: string) => api.post<{ message: string }>(`/api/system/inference/stop?alias=${encodeURIComponent(alias)}`),
	
	evict: (alias: string) => api.post<{ message: string }>(`/api/system/inference/evict?alias=${encodeURIComponent(alias)}`),
	
	pin: (alias: string) => api.post<{ message: string }>(`/api/system/inference/pin?alias=${encodeURIComponent(alias)}`),
	
	unpin: (alias: string) => api.post<{ message: string }>(`/api/system/inference/unpin?alias=${encodeURIComponent(alias)}`),
	
	deleteArtifacts: (alias: string) => api.post<{ message: string }>(`/api/system/inference/delete-artifacts?alias=${encodeURIComponent(alias)}`),

	// Health & Observability
	health: (alias: string) => api.get<HealthResponse>(`/api/system/inference/health?alias=${encodeURIComponent(alias)}`),
	
	logs: (alias: string, tail: number = 100) => 
		api.get<LogsResponse>(`/api/system/inference/logs?alias=${encodeURIComponent(alias)}&tail=${tail}`),
	
	metrics: (alias: string) => 
		api.get<MetricsResponse>(`/api/system/inference/metrics?alias=${encodeURIComponent(alias)}`),

	// Cache
	listCache: () => api.get<ArtifactInfo[]>('/api/system/inference/cache'),
	
	evictCache: (limitBytes: number) => 
		api.post<{ evicted: number }>(`/api/system/inference/evict-cache?limit_bytes=${limitBytes}`),
	
	clearCache: () => 
		api.post<{ message: string; freed_bytes: number }>('/api/system/inference/cache/clear'),

	// TRT Engines
	listTRTEngines: () => api.get<TRTEngine[]>('/api/system/inference/trt-engines'),
	
	convertTRT: (req: TRTConvertRequest) => 
		api.post<{ engine_path: string }>('/api/system/inference/convert-trt', req),
	
	deleteTRTEngine: (modelId: string) => 
		api.post<{ message: string }>(`/api/system/inference/delete-trt-engine?model_id=${encodeURIComponent(modelId)}`),

	// GPU list for device selection
	listGPUs: () => api.get<GPUListResponse>('/api/gpu/list'),

	// Saved models (persist between restarts)
	listSaved: () => api.get<SavedModel[]>('/api/system/inference/saved'),
	
	saveModel: (alias: string, autoStart: boolean = false) => 
		api.post<{ status: string }>('/api/system/inference/save', { alias, auto_start: autoStart }),
	
	deleteSaved: (alias: string) => 
		api.post<void>(`/api/system/inference/delete-saved?alias=${encodeURIComponent(alias)}`),
	
	setAutoStart: (alias: string, enabled: boolean) => 
		api.post<{ alias: string; auto_start: boolean }>(`/api/system/inference/auto-start?alias=${encodeURIComponent(alias)}&enabled=${enabled}`),
	
	updateSaved: (alias: string, params: UpdateSavedRequest) =>
		api.post<{ status: string; alias: string }>(`/api/system/inference/update-saved?alias=${encodeURIComponent(alias)}`, params),

	// Repository downloads (v3.3.x+) - download all model files locally
	// If filename is provided, downloads only that specific file
	downloadRepository: (modelId: string, filename?: string) => 
		api.post<RepoDownloadResponse>('/api/system/inference/download-repo', { 
			model_id: modelId,
			filename: filename || undefined
		}),
	
	listRepoDownloads: () => api.get<RepoDownload[]>('/api/system/inference/repo-downloads'),
	
	getRepoDownloadStatus: (modelId: string) => 
		api.post<RepoDownload>('/api/system/inference/repo-downloads/status', { model_id: modelId }),
	
	cancelRepoDownload: (modelId: string) =>
		api.post('/api/system/inference/repo-downloads/cancel', { model_id: modelId }),
	
	removeRepoDownload: (modelId: string) =>
		api.post('/api/system/inference/repo-downloads/remove', { model_id: modelId }),
};

// Repository download types
export interface RepoDownload {
	id: string;
	model_id: string;
	status: 'pending' | 'downloading' | 'completed' | 'failed' | 'cancelled';
	total_files: number;
	completed_files: number;
	failed_files: number;
	total_size: number;
	downloaded_size: number;
	progress: number;
	error?: string;
	local_path: string;
	started_at?: string;
	completed_at?: string;
}

export interface RepoDownloadResponse {
	message: string;
	model_id: string;
	download_id: string;
	total_files: number;
	total_size: number;
	local_path: string;
}

