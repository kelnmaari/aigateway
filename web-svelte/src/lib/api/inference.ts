// Inference API types and client functions

export type Provider = 'vllm' | 'sglang' | 'tgi' | 'tei' | 'tensorrt-llm' | 'llama.cpp';
export type Format = 'hf' | 'gguf' | 'trt' | 'other';

// Model capabilities based on Continue.dev model roles
// https://docs.continue.dev/customize/model-roles/00-intro
export type Capability =
	// Chat model capabilities (text generation)
	| 'chat' // Chat conversations
	| 'autocomplete' // Code autocomplete suggestions
	| 'edit' // Generate code based on edit prompts
	| 'apply' // Apply edits to files
	| 'vision' // Vision/image understanding
	// Embedding model capabilities
	| 'embeddings' // Vector embeddings for semantic search
	| 'rerank' // Rerank vector search results
	// Additional
	| 'function-calling' // Tool/function calling
	| 'code-completion'; // Code completion (legacy)

// Capabilities implied by chat - if model can chat, it can also do these
export const CHAT_IMPLIED_CAPABILITIES: Capability[] = ['chat', 'autocomplete', 'edit', 'apply'];

// Capabilities implied by embeddings - embedding models can rerank
export const EMBEDDING_IMPLIED_CAPABILITIES: Capability[] = ['embeddings', 'rerank'];

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
	vllm_extra_args?: string;
	vllm_quantization?: string;
	vllm_dtype?: string;
	vllm_kv_cache_dtype?: string;
	vllm_max_num_seqs?: number;
	vllm_enforce_eager?: boolean;
	vllm_enable_prefix_caching?: boolean;
	vllm_enable_chunked_prefill?: boolean;
	vllm_swap_space?: number;
	vllm_enable_auto_tool_choice?: boolean;
	vllm_tool_call_parser?: string;
	vllm_chat_template?: string;

	// llama.cpp params
	llama_main_gpu?: number;
	llama_tensor_split?: string;
	llama_n_gpu_layers?: number;
	llama_ctx_size?: number;
	llama_n_parallel?: number;
	llama_flash_attn?: boolean;
	llama_jinja?: boolean;
	llama_cache_reuse?: number;
	llama_extra_args?: string;
	llama_batch_size?: number;
	llama_ubatch_size?: number;
	llama_cache_type_k?: string;
	llama_cache_type_v?: string;
	llama_mlock?: boolean;
	llama_chat_template?: string;

	// SGLang params
	sglang_tensor_parallel?: number;
	sglang_mem_fraction?: number;
	sglang_data_parallel?: number;
	sglang_context_len?: number;
	sglang_chunked_prefill?: boolean;
	sglang_quantization?: string;
	sglang_attention_backend?: string;
	sglang_tool_call_parser?: string;
	sglang_extra_args?: string;

	// TGI params
	tgi_num_shard?: number;
	tgi_max_concurrent_reqs?: number;
	tgi_max_input_len?: number;
	tgi_max_total_tokens?: number;
	tgi_quantize?: string;
	tgi_cuda_memory_fraction?: number;
	tgi_extra_args?: string;

	// TEI params
	tei_cpu_mode?: boolean;
	tei_max_batch_tokens?: number;
	tei_max_concurrent_reqs?: number;
	tei_pooling?: string;
	tei_dtype?: string;
	tei_extra_args?: string;
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
	vllm_extra_args?: string;
	vllm_quantization?: string;
	vllm_dtype?: string;
	vllm_kv_cache_dtype?: string;
	vllm_max_num_seqs?: number;
	vllm_enforce_eager?: boolean;
	vllm_enable_prefix_caching?: boolean;
	vllm_enable_chunked_prefill?: boolean;
	vllm_swap_space?: number;
	vllm_enable_auto_tool_choice?: boolean;
	vllm_tool_call_parser?: string;
	vllm_chat_template?: string;
	llama_main_gpu?: number;
	llama_n_gpu_layers?: number;
	llama_ctx_size?: number;
	llama_n_parallel?: number;
	llama_flash_attn?: boolean;
	llama_jinja?: boolean;
	llama_tensor_split?: string;
	llama_cache_reuse?: number;
	llama_extra_args?: string;
	llama_batch_size?: number;
	llama_ubatch_size?: number;
	llama_cache_type_k?: string;
	llama_cache_type_v?: string;
	llama_mlock?: boolean;
	llama_chat_template?: string;
	sglang_tensor_parallel?: number;
	sglang_mem_fraction?: number;
	sglang_data_parallel?: number;
	sglang_context_len?: number;
	sglang_chunked_prefill?: boolean;
	sglang_quantization?: string;
	sglang_attention_backend?: string;
	sglang_tool_call_parser?: string;
	sglang_extra_args?: string;
	tgi_num_shard?: number;
	tgi_max_concurrent_reqs?: number;
	tgi_max_input_len?: number;
	tgi_max_total_tokens?: number;
	tgi_quantize?: string;
	tgi_cuda_memory_fraction?: number;
	tgi_extra_args?: string;
	tei_cpu_mode?: boolean;
	tei_max_batch_tokens?: number;
	tei_max_concurrent_reqs?: number;
	tei_pooling?: string;
	tei_dtype?: string;
	tei_extra_args?: string;
}

export interface UpdateSavedRequest {
	capabilities?: Capability[];
	auto_start?: boolean;
	vllm_tensor_parallel?: number;
	vllm_max_model_len?: number;
	vllm_gpu_utilization?: number;
	vllm_extra_args?: string;
	vllm_quantization?: string;
	vllm_dtype?: string;
	vllm_kv_cache_dtype?: string;
	vllm_max_num_seqs?: number;
	vllm_enforce_eager?: boolean;
	vllm_enable_prefix_caching?: boolean;
	vllm_enable_chunked_prefill?: boolean;
	vllm_swap_space?: number;
	vllm_enable_auto_tool_choice?: boolean;
	vllm_tool_call_parser?: string;
	vllm_chat_template?: string;
	llama_main_gpu?: number;
	llama_n_gpu_layers?: number;
	llama_ctx_size?: number;
	llama_n_parallel?: number;
	llama_flash_attn?: boolean;
	llama_jinja?: boolean;
	llama_tensor_split?: string;
	llama_cache_reuse?: number;
	llama_extra_args?: string;
	llama_batch_size?: number;
	llama_ubatch_size?: number;
	llama_cache_type_k?: string;
	llama_cache_type_v?: string;
	llama_mlock?: boolean;
	llama_chat_template?: string;
	sglang_tensor_parallel?: number;
	sglang_mem_fraction?: number;
	sglang_data_parallel?: number;
	sglang_context_len?: number;
	sglang_chunked_prefill?: boolean;
	sglang_quantization?: string;
	sglang_attention_backend?: string;
	sglang_tool_call_parser?: string;
	sglang_extra_args?: string;
	tgi_num_shard?: number;
	tgi_max_concurrent_reqs?: number;
	tgi_max_input_len?: number;
	tgi_max_total_tokens?: number;
	tgi_quantize?: string;
	tgi_cuda_memory_fraction?: number;
	tgi_extra_args?: string;
	tei_cpu_mode?: boolean;
	tei_max_batch_tokens?: number;
	tei_max_concurrent_reqs?: number;
	tei_pooling?: string;
	tei_dtype?: string;
	tei_extra_args?: string;
	gpu_device?: string;
}

export interface CreateSavedRequest {
	alias: string;
	provider: Provider;
	format: Format;
	hf_repo?: string;
	hf_file?: string;
	hf_revision?: string;
	gguf_url?: string;
	capabilities?: Capability[];
	gpu_device?: string;
	auto_start?: boolean;
	vllm_tensor_parallel?: number;
	vllm_max_model_len?: number;
	vllm_gpu_utilization?: number;
	vllm_extra_args?: string;
	vllm_quantization?: string;
	vllm_dtype?: string;
	vllm_kv_cache_dtype?: string;
	vllm_max_num_seqs?: number;
	vllm_enforce_eager?: boolean;
	vllm_enable_prefix_caching?: boolean;
	vllm_enable_chunked_prefill?: boolean;
	vllm_swap_space?: number;
	vllm_enable_auto_tool_choice?: boolean;
	vllm_tool_call_parser?: string;
	vllm_chat_template?: string;
	llama_main_gpu?: number;
	llama_tensor_split?: string;
	llama_n_gpu_layers?: number;
	llama_ctx_size?: number;
	llama_n_parallel?: number;
	llama_flash_attn?: boolean;
	llama_jinja?: boolean;
	llama_cache_reuse?: number;
	llama_extra_args?: string;
	llama_batch_size?: number;
	llama_ubatch_size?: number;
	llama_cache_type_k?: string;
	llama_cache_type_v?: string;
	llama_mlock?: boolean;
	llama_chat_template?: string;
	sglang_tensor_parallel?: number;
	sglang_mem_fraction?: number;
	sglang_data_parallel?: number;
	sglang_context_len?: number;
	sglang_chunked_prefill?: boolean;
	sglang_quantization?: string;
	sglang_attention_backend?: string;
	sglang_tool_call_parser?: string;
	sglang_extra_args?: string;
	tgi_num_shard?: number;
	tgi_max_concurrent_reqs?: number;
	tgi_max_input_len?: number;
	tgi_max_total_tokens?: number;
	tgi_quantize?: string;
	tgi_cuda_memory_fraction?: number;
	tgi_extra_args?: string;
	tei_cpu_mode?: boolean;
	tei_max_batch_tokens?: number;
	tei_max_concurrent_reqs?: number;
	tei_pooling?: string;
	tei_dtype?: string;
	tei_extra_args?: string;
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
	vllm_extra_args?: string;
	vllm_quantization?: string;
	vllm_dtype?: string;
	vllm_kv_cache_dtype?: string;
	vllm_max_num_seqs?: number;
	vllm_enforce_eager?: boolean;
	vllm_enable_prefix_caching?: boolean;
	vllm_enable_chunked_prefill?: boolean;
	vllm_swap_space?: number;
	vllm_enable_auto_tool_choice?: boolean;
	vllm_tool_call_parser?: string;
	vllm_chat_template?: string;
	llama_main_gpu?: number;
	llama_tensor_split?: string;
	llama_n_gpu_layers?: number;
	llama_ctx_size?: number;
	llama_n_parallel?: number;
	llama_flash_attn?: boolean;
	llama_jinja?: boolean;
	llama_cache_reuse?: number;
	llama_extra_args?: string;
	llama_batch_size?: number;
	llama_ubatch_size?: number;
	llama_cache_type_k?: string;
	llama_cache_type_v?: string;
	llama_mlock?: boolean;
	llama_chat_template?: string;
	sglang_tensor_parallel?: number;
	sglang_mem_fraction?: number;
	sglang_data_parallel?: number;
	sglang_context_len?: number;
	sglang_chunked_prefill?: boolean;
	sglang_quantization?: string;
	sglang_attention_backend?: string;
	sglang_tool_call_parser?: string;
	sglang_extra_args?: string;
	tgi_num_shard?: number;
	tgi_max_concurrent_reqs?: number;
	tgi_max_input_len?: number;
	tgi_max_total_tokens?: number;
	tgi_quantize?: string;
	tgi_cuda_memory_fraction?: number;
	tgi_extra_args?: string;
	tei_cpu_mode?: boolean;
	tei_max_batch_tokens?: number;
	tei_max_concurrent_reqs?: number;
	tei_pooling?: string;
	tei_dtype?: string;
	tei_extra_args?: string;
}

import { api } from './client';

export const inferenceApi = {
	// Models
	listModels: () => api.get<ModelInfo[]>('/api/system/inference/models'),

	load: (req: LoadRequest) => api.post<{ message: string }>('/api/system/inference/load', req),

	prepare: (req: LoadRequest) => api.post<{ message: string }>('/api/system/inference/prepare', req),

	stop: (alias: string) => api.post<{ message: string }>(`/api/system/inference/stop?alias=${encodeURIComponent(alias)}`),

	restart: (alias: string) => api.post<{ alias: string; status: string; endpoint: string }>(`/api/system/inference/restart?alias=${encodeURIComponent(alias)}`),

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

	// Create saved model configuration directly (without loading)
	createSaved: (config: CreateSavedRequest) =>
		api.post<{ status: string; alias: string; auto_start: boolean }>('/api/system/inference/create-saved', config),

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

	// Refresh saved model (re-download missing/corrupted files)
	refreshSaved: (alias: string) =>
		api.post<RepoDownloadResponse>(`/api/system/inference/refresh-saved?alias=${encodeURIComponent(alias)}`),
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

