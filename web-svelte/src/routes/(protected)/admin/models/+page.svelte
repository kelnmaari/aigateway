<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		inferenceApi,
		type ModelInfo,
		type ArtifactInfo,
		type TRTEngine,
		type Provider,
		type Format,
		type Capability,
		type LoadRequest,
		type GPUDevice,
		type SavedModel,
		type RepoDownload,
		CHAT_IMPLIED_CAPABILITIES,
		EMBEDDING_IMPLIED_CAPABILITIES
	} from '$lib/api/inference';
	import { api } from '$lib/api/client';
	import { downloadsApi } from '$lib/api/downloads';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import { faMagnifyingGlass, faDownload, faArrowUpRightFromSquare, faSpinner, faFont, faBolt, faRocket, faFaceSmile, faHorse, faLink, faChartBar, faShuffle, faGear, faArrowsRotate, faPause, faPlay, faXmark, faCircleCheck, faTriangleExclamation, faCircleXmark, faHourglass, faFolder, faTrashCan, faLightbulb, faArrowRight, faClipboard, faBan, faArrowDown } from '@fortawesome/free-solid-svg-icons';
	import { faHeart } from '@fortawesome/free-regular-svg-icons';
	import * as m from '$lib/paraglide/messages';

	// GPU devices for selection
	let gpuDevices: GPUDevice[] = $state([]);
	let selectedGPUs: number[] = $state([]);

	const providers: Provider[] = ['vllm', 'sglang', 'tgi', 'tei', 'tensorrt-llm', 'llama.cpp'];
	const formats: Format[] = ['hf', 'gguf', 'trt', 'other'];
	// Capabilities based on Continue.dev model roles
	// Chat models: chat, autocomplete, edit, apply
	// Embedding models: embeddings, rerank
	const capabilities: Capability[] = [
		'chat',
		'autocomplete',
		'edit',
		'apply',
		'vision',
		'embeddings',
		'rerank',
		'function-calling'
	];

	let models: ModelInfo[] = $state([]);
	let savedModels: SavedModel[] = $state([]);
	let artifacts: ArtifactInfo[] = $state([]);
	let trtEngines: TRTEngine[] = $state([]);
	let busy = $state(false);
	let msg = $state('');
	let msgType = $state<'info' | 'error' | 'success'>('info');
	let activeTab = $state<'models' | 'cache' | 'trt' | 'hf' | 'downloads'>('models');

	// Repository downloads state
	let repoDownloads = $state<RepoDownload[]>([]);
	let repoDownloadsInterval: ReturnType<typeof setInterval> | null = null;

	// HuggingFace Browser state - compatible with both local and API types
	interface HFModel {
		id: string;
		author: string;
		model_name?: string; // From API
		modelId?: string; // Legacy
		downloads: number;
		likes: number;
		last_modified?: string; // From API
		lastModified?: string; // Legacy
		tags: string[];
		pipeline_tag?: string;
		has_gguf?: boolean;
		// Model size info (from safetensors or config)
		safetensors?: { total?: number; parameters?: { [key: string]: number } };
		config?: { num_parameters?: number };
	}

	// Model memory recommendations
	interface MemoryRecommendation {
		modelSizeGB: number;
		modelParams: string;
		quantization: string;
		totalGPUMemoryGB: number;
		recommendedMemFraction: number;
		recommendedTensorParallel: number;
		minGPUsNeeded: number;
		willFit: boolean;
		warning?: string;
	}
	let memoryRecommendation = $state<MemoryRecommendation | null>(null);
	type HFProviderFilter =
		| 'all'
		| 'vllm'
		| 'sglang'
		| 'tgi'
		| 'llama.cpp'
		| 'tei'
		| 'embedding'
		| 'rerank';
	const hfProviderFilters: {
		id: HFProviderFilter;
		label: string;
		description: string;
		icon: any;
	}[] = [
		{ id: 'all', label: 'All LLMs', description: 'Text generation models', icon: faFont },
		{ id: 'vllm', label: 'vLLM', description: 'For vLLM inference', icon: faBolt },
		{ id: 'sglang', label: 'SGLang', description: 'For SGLang inference', icon: faRocket },
		{ id: 'tgi', label: 'TGI', description: 'For Text Generation Inference', icon: faFaceSmile },
		{
			id: 'llama.cpp',
			label: 'llama.cpp (GGUF)',
			description: 'Quantized GGUF models',
			icon: faHorse
		},
		{
			id: 'tei',
			label: 'TEI',
			description: 'Text Embeddings Inference compatible',
			icon: faLink
		},
		{
			id: 'embedding',
			label: 'Embeddings',
			description: 'Feature extraction & embeddings',
			icon: faChartBar
		},
		{
			id: 'rerank',
			label: 'Rerank',
			description: 'Reranking & cross-encoder models',
			icon: faShuffle
		}
	];

	// Model size filters
	type HFSizeFilter = 'any' | 'tiny' | 'small' | 'medium' | 'large' | 'xl' | 'xxl';
	const hfSizeFilters: {
		id: HFSizeFilter;
		label: string;
		range: string;
		minB: number;
		maxB: number;
	}[] = [
		{ id: 'any', label: 'Any', range: 'All sizes', minB: 0, maxB: Infinity },
		{ id: 'tiny', label: '< 3B', range: 'Tiny', minB: 0, maxB: 3 },
		{ id: 'small', label: '3-7B', range: 'Small', minB: 3, maxB: 7 },
		{ id: 'medium', label: '7-14B', range: 'Medium', minB: 7, maxB: 14 },
		{ id: 'large', label: '14-30B', range: 'Large', minB: 14, maxB: 30 },
		{ id: 'xl', label: '30-70B', range: 'XL', minB: 30, maxB: 70 },
		{ id: 'xxl', label: '70B+', range: 'XXL', minB: 70, maxB: Infinity }
	];
	let hfSizeFilter = $state<HFSizeFilter>('any');

	let hfSearchQuery = $state('');
	let hfProviderFilter = $state<HFProviderFilter>('all');
	let hfSearchResults = $state<HFModel[]>([]);
	let hfPopularModels = $state<HFModel[]>([]);
	let hfSearching = $state(false);
	let hfSearchPerformed = $state(false); // Track if search was performed
	let hfSearchError = $state(''); // Track search error message
	let hfSelectedModel = $state<HFModel | null>(null);
	let hfModelFiles = $state<any[]>([]);
	let hfRepoDebounce: ReturnType<typeof setTimeout> | undefined;

	// Pagination state
	let hfCurrentPage = $state(1);
	let hfHasMore = $state(true);
	let hfLoadingMore = $state(false);
	const HF_PAGE_SIZE = 30;

	// Selected model for details panel
	let selectedModel = $state<ModelInfo | null>(null);
	let selectedLogs = $state('');
	let selectedMetrics = $state('');
	let selectedHealth = $state<{ status: string; response_time_ms?: number; error?: string } | null>(
		null
	);
	let logsInterval: ReturnType<typeof setInterval> | null = null;

	// Logs modal state
	let logsModalOpen = $state(false);
	let logsModalAlias = $state('');
	let logsModalContent = $state('');
	let logsModalLoading = $state(false);
	let logsModalInterval: ReturnType<typeof setInterval> | null = null;
	let logsModalAutoScroll = $state(true);
	let logsContainer: HTMLDivElement | null = $state(null);

	// Colorize log lines
	function colorizeLogs(logs: string): string {
		if (!logs) return '';

		// Use inline styles because Tailwind JIT doesn't see dynamically generated classes
		const colors = {
			gray: '#6b7280',
			red: '#ef4444',
			yellow: '#eab308',
			blue: '#60a5fa',
			green: '#4ade80',
			cyan: '#22d3ee',
			purple: '#a78bfa',
			orange: '#fb923c',
			teal: '#2dd4bf',
			pink: '#f472b6',
			lime: '#a3e635'
		};

		return logs
			.split('\n')
			.map((line) => {
				let html = escapeHtml(line);

				// Timestamps: time="..." or [2025-...] or 2025-01-01T...
				html = html.replace(
					/(time=&quot;[^&]*&quot;|^\[\d{4}-[^\]]+\]|\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[^\s]*)/g,
					`<span style="color:${colors.gray}">$1</span>`
				);

				// Log levels with colors
				html = html.replace(
					/\b(level=error|ERROR|ERRO|FATAL|CRITICAL)\b/gi,
					`<span style="color:${colors.red};font-weight:bold">$1</span>`
				);
				html = html.replace(
					/\b(level=warn|WARNING|WARN)\b/gi,
					`<span style="color:${colors.yellow};font-weight:bold">$1</span>`
				);
				html = html.replace(
					/\b(level=info|INFO)\b/gi,
					`<span style="color:${colors.blue}">$1</span>`
				);
				html = html.replace(
					/\b(level=debug|DEBUG)\b/gi,
					`<span style="color:${colors.gray}">$1</span>`
				);

				// Success messages
				html = html.replace(
					/\b(SUCCESS|OK|READY|LOADED|STARTED|COMPLETED|loaded|started)\b/g,
					`<span style="color:${colors.green};font-weight:bold">$1</span>`
				);

				// msg="..." content (escaped quotes)
				html = html.replace(
					/msg=&quot;([^&]*)&quot;/g,
					`msg=&quot;<span style="color:${colors.cyan}">$1</span>&quot;`
				);

				// Key=value pairs (highlight keys) - avoid already colored spans
				html = html.replace(/\b([a-z_]+)=([^\s<]+)/gi, (match, key, value) => {
					if (
						key === 'level' ||
						key === 'msg' ||
						key === 'time' ||
						key === 'style' ||
						key === 'color'
					)
						return match;
					return `<span style="color:${colors.purple}">${key}</span>=<span style="color:${colors.orange}">${value}</span>`;
				});

				// Numbers with units
				html = html.replace(
					/\b(\d+\.?\d*)(ms|s|MB|GB|KB|MiB|GiB|B|%)\b/g,
					`<span style="color:${colors.yellow}">$1$2</span>`
				);

				// File paths
				html = html.replace(
					/(\/[a-zA-Z0-9_.\/-]+)/g,
					`<span style="color:${colors.teal}">$1</span>`
				);

				// HTTP methods
				html = html.replace(
					/\b(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\b/g,
					`<span style="color:${colors.pink};font-weight:bold">$1</span>`
				);

				// HTTP status codes
				html = html.replace(/\b(2\d{2})\b/g, `<span style="color:${colors.green}">$1</span>`);
				html = html.replace(/\b(4\d{2})\b/g, `<span style="color:${colors.yellow}">$1</span>`);
				html = html.replace(/\b(5\d{2})\b/g, `<span style="color:${colors.red}">$1</span>`);

				return html;
			})
			.join('\n');
	}

	function escapeHtml(text: string): string {
		return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
	}

	// Edit saved model modal state
	let editingSavedModel = $state<SavedModel | null>(null);
	let editSavedForm = $state<{
		capabilities: Capability[];
		auto_start: boolean;
		vllm_tensor_parallel: number;
		vllm_max_model_len: number;
		vllm_gpu_utilization: number;
		vllm_extra_args: string;
		llama_main_gpu: number;
		llama_n_gpu_layers: number;
		llama_ctx_size: number;
		llama_n_parallel: number;
		llama_flash_attn: boolean;
		llama_jinja: boolean;
		llama_cache_reuse: number;
		llama_extra_args: string;
		llama_tensor_split: string;
		sglang_tensor_parallel: number;
		sglang_mem_fraction: number;
		tgi_num_shard: number;
		gpu_device: string;
	}>({
		capabilities: ['chat'],
		auto_start: false,
		vllm_tensor_parallel: 1,
		vllm_max_model_len: 0,
		vllm_gpu_utilization: 0.9,
		vllm_extra_args: '',
		llama_main_gpu: 0,
		llama_n_gpu_layers: -1,
		llama_ctx_size: 0,
		llama_n_parallel: 0,
		llama_flash_attn: false,
		llama_jinja: false,
		llama_cache_reuse: 0,
		llama_extra_args: '',
		llama_tensor_split: '',
		sglang_tensor_parallel: 1,
		sglang_mem_fraction: 0.9,
		tgi_num_shard: 1,
		gpu_device: ''
	});

	let form = $state<LoadRequest>({
		alias: '',
		provider: 'vllm',
		format: 'hf',
		hf_repo: '',
		hf_file: '',
		hf_revision: '',
		gguf_url: '',
		capabilities: ['chat'],
		gpu_device: '',
		vllm_tensor_parallel: 0,
		vllm_max_model_len: 0,
		vllm_gpu_utilization: 0.8,
		vllm_extra_args: '',
		llama_main_gpu: 0,
		llama_tensor_split: '',
		llama_n_gpu_layers: 0,
		llama_ctx_size: 0,
		llama_n_parallel: 0,
		llama_flash_attn: false,
		llama_jinja: false,
		llama_cache_reuse: 0,
		llama_extra_args: '',
		sglang_tensor_parallel: 0,
		sglang_mem_fraction: 0.8,
		tgi_num_shard: 1
	});

	// TRT conversion form
	let trtForm = $state({
		hf_model: '',
		max_batch_size: 8,
		max_input_len: 2048,
		max_output_len: 512,
		dtype: 'float16'
	});

	onMount(async () => {
		await refreshAll();
		await loadPopularHF();
		await loadGPUs();
		await loadRepoDownloads();
	});

	onDestroy(() => {
		if (logsInterval) clearInterval(logsInterval);
		if (repoDownloadsInterval) clearInterval(repoDownloadsInterval);
		if (logsModalInterval) clearInterval(logsModalInterval);
	});

	// Auto-refresh downloads when tab is active
	$effect(() => {
		if (activeTab === 'downloads') {
			loadRepoDownloads();
			repoDownloadsInterval = setInterval(loadRepoDownloads, 3000);
		} else if (repoDownloadsInterval) {
			clearInterval(repoDownloadsInterval);
			repoDownloadsInterval = null;
		}
	});

	async function refreshAll() {
		await Promise.all([loadModels(), loadSavedModels(), loadCache(), loadTRTEngines()]);
	}

	async function loadSavedModels() {
		try {
			savedModels = (await inferenceApi.listSaved()) || [];
		} catch (e: any) {
			console.error('Failed to load saved models:', e);
			savedModels = [];
		}
	}

	async function loadGPUs() {
		try {
			const resp = await inferenceApi.listGPUs();
			if (resp.enabled && resp.devices) {
				gpuDevices = resp.devices;
			}
		} catch {
			gpuDevices = [];
		}
	}

	function toggleGPU(index: number) {
		if (selectedGPUs.includes(index)) {
			selectedGPUs = selectedGPUs.filter((i) => i !== index);
		} else {
			selectedGPUs = [...selectedGPUs, index];
		}
		// Update form.gpu_device string
		form.gpu_device = selectedGPUs.sort((a, b) => a - b).join(',');

		// Auto-set tensor parallel based on selected GPU count
		updateTensorParallelForGPUs();

		// Update memory recommendations
		updateRecommendations();
	}

	// Auto-set tensor parallel based on selected GPU count (for multi-GPU providers)
	function updateTensorParallelForGPUs() {
		const gpuCount = selectedGPUs.length;
		if (gpuCount > 1) {
			// vLLM, SGLang, TGI support tensor parallelism / sharding
			if (form.provider === 'vllm') {
				form.vllm_tensor_parallel = gpuCount;
			} else if (form.provider === 'sglang') {
				form.sglang_tensor_parallel = gpuCount;
			} else if (form.provider === 'tgi') {
				form.tgi_num_shard = gpuCount;
			}
		} else if (gpuCount <= 1) {
			// Reset to single GPU / default
			form.vllm_tensor_parallel = 0;
			form.sglang_tensor_parallel = 0;
			form.tgi_num_shard = 1;
		}
	}

	// When provider changes, update tensor parallel to match selected GPUs
	function updateTensorParallelForProvider() {
		updateTensorParallelForGPUs();
	}

	// Calculate memory recommendations based on model size and available GPUs
	function calculateMemoryRecommendation(model: HFModel | null): MemoryRecommendation | null {
		if (!model || gpuDevices.length === 0) return null;

		// Try to get model parameters from various sources
		let paramCount = 0;
		let quantization = 'BF16'; // Default assumption

		// Check tags for quantization info
		const tags = model.tags || [];
		if (
			tags.some(
				(t) =>
					t.toLowerCase().includes('awq') ||
					t.toLowerCase().includes('int4') ||
					t.toLowerCase().includes('4bit')
			)
		) {
			quantization = 'INT4';
		} else if (
			tags.some(
				(t) =>
					t.toLowerCase().includes('gptq') ||
					t.toLowerCase().includes('int8') ||
					t.toLowerCase().includes('8bit')
			)
		) {
			quantization = 'INT8';
		} else if (tags.some((t) => t.toLowerCase().includes('gguf'))) {
			quantization = 'GGUF (varies)';
		}

		// Extract param count from model name (common patterns: 7b, 13b, 30b, 70b)
		const nameMatch = model.id.toLowerCase().match(/(\d+\.?\d*)b/);
		if (nameMatch) {
			paramCount = parseFloat(nameMatch[1]) * 1e9;
		}

		// If no params from name, try safetensors info
		if (paramCount === 0 && model.safetensors?.total) {
			paramCount = model.safetensors.total;
		}

		if (paramCount === 0) {
			return null; // Can't calculate without knowing model size
		}

		// Calculate model size in GB based on quantization
		let bytesPerParam = 2; // BF16/FP16
		if (quantization === 'INT8') bytesPerParam = 1;
		else if (quantization === 'INT4') bytesPerParam = 0.5;
		else if (quantization.includes('GGUF')) bytesPerParam = 0.6; // Approximate average for GGUF

		const modelSizeGB = (paramCount * bytesPerParam) / 1024 ** 3;

		// Calculate total available GPU memory (memory_mb is in megabytes)
		const selectedGPUMemoryGB =
			selectedGPUs.length > 0
				? selectedGPUs.reduce((sum, idx) => sum + (gpuDevices[idx]?.memory_mb || 0), 0) / 1024
				: gpuDevices.reduce((sum, gpu) => sum + (gpu.memory_mb || 0), 0) / 1024;

		const numGPUs = selectedGPUs.length > 0 ? selectedGPUs.length : gpuDevices.length;
		const perGPUMemory = selectedGPUMemoryGB / numGPUs;

		// KV cache and overhead estimation (rough: 10-20% of model size)
		const kvCacheGB = modelSizeGB * 0.15;
		const overheadGB = 2; // General overhead
		const totalNeededGB = modelSizeGB + kvCacheGB + overheadGB;

		// Will it fit?
		const willFit = totalNeededGB <= selectedGPUMemoryGB;

		// Calculate recommended mem_fraction
		let recommendedMemFraction =
			selectedGPUMemoryGB > 0 ? Math.min(0.95, totalNeededGB / selectedGPUMemoryGB + 0.05) : 0.9;
		recommendedMemFraction = Math.round(recommendedMemFraction * 100) / 100;

		// Calculate minimum GPUs needed
		const singleGPUMemory = gpuDevices[0]?.memory_mb ? gpuDevices[0].memory_mb / 1024 : 24;
		const minGPUsNeeded = Math.ceil(totalNeededGB / (singleGPUMemory * 0.9));

		// Format params for display
		const paramsInB = paramCount / 1e9;
		const modelParams =
			paramsInB >= 1 ? `${paramsInB.toFixed(1)}B` : `${(paramCount / 1e6).toFixed(0)}M`;

		let warning: string | undefined;
		if (!willFit) {
			warning = `Model needs ~${totalNeededGB.toFixed(1)}GB but only ${selectedGPUMemoryGB.toFixed(1)}GB available. Use ${minGPUsNeeded}+ GPUs or quantized version.`;
		} else if (recommendedMemFraction > 0.85) {
			warning = `Model will use most of GPU memory. Consider lower context length if OOM occurs.`;
		}

		return {
			modelSizeGB: Math.round(modelSizeGB * 10) / 10,
			modelParams,
			quantization,
			totalGPUMemoryGB: Math.round(selectedGPUMemoryGB * 10) / 10,
			recommendedMemFraction: willFit ? recommendedMemFraction : 0.9,
			recommendedTensorParallel: Math.max(1, numGPUs),
			minGPUsNeeded,
			willFit,
			warning
		};
	}

	// Update recommendations when model or GPU selection changes
	function updateRecommendations() {
		// Use hfSelectedModel if available, otherwise try to create pseudo-model from hf_repo
		let modelForCalc = hfSelectedModel;
		if (!modelForCalc && form.hf_repo) {
			// Create minimal model object from manual input for calculation
			modelForCalc = {
				id: form.hf_repo,
				author: '',
				downloads: 0,
				likes: 0,
				tags: [],
				safetensors: undefined
			} as HFModel;
		}

		memoryRecommendation = calculateMemoryRecommendation(modelForCalc);

		// Auto-apply recommendations if available
		if (memoryRecommendation) {
			if (form.provider === 'vllm') {
				form.vllm_gpu_utilization = memoryRecommendation.recommendedMemFraction;
			} else if (form.provider === 'sglang') {
				form.sglang_mem_fraction = memoryRecommendation.recommendedMemFraction;
			}
		}
	}

	// HuggingFace Browser functions
	async function searchHF() {
		hfSearching = true;
		hfSearchError = '';
		hfSearchPerformed = true;
		try {
			const res = await downloadsApi.searchHuggingFace(hfSearchQuery, hfProviderFilter);
			hfSearchResults = (res.models || []) as HFModel[];
			if (hfSearchResults.length === 0 && hfSearchQuery) {
				hfSearchError = `No models found for "${hfSearchQuery}"`;
			}
		} catch (e: unknown) {
			hfSearchError = (e as Error)?.message || 'Failed to search HuggingFace';
			hfSearchResults = [];
		} finally {
			hfSearching = false;
		}
	}

	async function loadPopularHF(reset: boolean = true) {
		if (reset) {
			hfCurrentPage = 1;
			hfPopularModels = [];
			hfHasMore = true;
			hfSearching = true;
		} else {
			hfLoadingMore = true;
		}
		try {
			const res = await downloadsApi.getPopularModels(
				hfProviderFilter,
				HF_PAGE_SIZE,
				hfCurrentPage
			);
			const newModels = (res.models || []) as HFModel[];
			if (reset) {
				hfPopularModels = newModels;
			} else {
				hfPopularModels = [...hfPopularModels, ...newModels];
			}
			hfHasMore = newModels.length >= HF_PAGE_SIZE;
		} catch (e: unknown) {
			console.error('Failed to load popular models:', e);
		} finally {
			hfSearching = false;
			hfLoadingMore = false;
		}
	}

	async function loadMoreModels() {
		if (hfLoadingMore || !hfHasMore) return;
		hfCurrentPage++;
		await loadPopularHF(false);
	}

	// Reload when provider filter changes
	$effect(() => {
		if (activeTab === 'hf') {
			hfSearchResults = [];
			loadPopularHF();
		}
	});

	// Watch provider filter changes
	$effect(() => {
		// Trigger reload when filter changes
		const _ = hfProviderFilter;
		if (activeTab === 'hf') {
			loadPopularHF();
		}
	});

	async function selectHFModel(m: HFModel) {
		hfSelectedModel = m;
		try {
			const res = await api.get<{ siblings?: any[] }>(`/api/huggingface/models/${m.id}`);
			// Filter files based on provider filter
			const allFiles = res.siblings || [];
			if (hfProviderFilter === 'llama.cpp') {
				// Show only GGUF files for llama.cpp
				hfModelFiles = allFiles.filter((f: any) => f.rfilename?.endsWith('.gguf'));
			} else {
				// For HF models (vLLM/SGLang/TGI) show safetensors, bin, and config files
				hfModelFiles = allFiles.filter(
					(f: any) =>
						f.rfilename?.endsWith('.safetensors') ||
						f.rfilename?.endsWith('.bin') ||
						f.rfilename === 'config.json' ||
						f.rfilename === 'tokenizer.json'
				);
			}
		} catch (e: any) {
			console.error('Failed to get model files:', e);
			hfModelFiles = [];
		}
	}

	function useHFModel(m: HFModel, file?: any) {
		// Pre-fill the load form with HF model data
		form.hf_repo = m.id;

		if (file?.rfilename?.endsWith('.gguf')) {
			// GGUF file selected
			form.hf_file = file.rfilename;
			form.format = 'gguf';
			form.provider = 'llama.cpp';
			form.capabilities = [...CHAT_IMPLIED_CAPABILITIES];
		} else {
			form.hf_file = '';
			form.format = 'hf';

			// Set provider based on filter selection
			switch (hfProviderFilter) {
				case 'llama.cpp':
					// If browsing GGUF but selected non-GGUF file, still suggest llama.cpp
					form.provider = 'llama.cpp';
					form.format = 'gguf';
					form.capabilities = [...CHAT_IMPLIED_CAPABILITIES];
					break;
				case 'vllm':
					form.provider = 'vllm';
					form.capabilities = [...CHAT_IMPLIED_CAPABILITIES];
					break;
				case 'sglang':
					form.provider = 'sglang';
					form.capabilities = [...CHAT_IMPLIED_CAPABILITIES];
					break;
				case 'tgi':
					form.provider = 'tgi';
					form.capabilities = [...CHAT_IMPLIED_CAPABILITIES];
					break;
				case 'tei':
					form.provider = 'tei';
					form.capabilities = [...EMBEDDING_IMPLIED_CAPABILITIES];
					break;
				case 'embedding':
					form.provider = 'tei'; // TEI (Text Embeddings Inference) is best for embeddings
					form.capabilities = [...EMBEDDING_IMPLIED_CAPABILITIES];
					break;
				case 'rerank':
					form.provider = 'tei'; // Reranking also runs on TEI
					form.capabilities = [...EMBEDDING_IMPLIED_CAPABILITIES];
					break;
				default:
					form.provider = 'vllm'; // Default
					form.capabilities = [...CHAT_IMPLIED_CAPABILITIES];
			}
		}

		form.alias =
			m.id
				.split('/')
				.pop()
				?.toLowerCase()
				.replace(/[^a-z0-9]/g, '-') || 'model';

		// Store selected model for recommendations
		hfSelectedModel = m;

		// Update memory recommendations
		updateRecommendations();

		// Switch to models tab
		activeTab = 'models';
		showMsg(`Selected ${m.id} for ${form.provider}. Configure and click Load.`, 'info');
	}

	// Extract model size in billions from model name or tags
	function extractModelSizeB(model: HFModel): number | null {
		// Try to extract from model ID (common patterns: 7b, 13b, 30b, 70b, 1.5b)
		const nameMatch = model.id.toLowerCase().match(/(\d+\.?\d*)b/);
		if (nameMatch) {
			return parseFloat(nameMatch[1]);
		}
		// Try tags
		for (const tag of model.tags || []) {
			const tagMatch = tag.toLowerCase().match(/(\d+\.?\d*)b/);
			if (tagMatch) {
				return parseFloat(tagMatch[1]);
			}
		}
		return null;
	}

	// Filter models by size
	function filterBySize(models: HFModel[]): HFModel[] {
		if (hfSizeFilter === 'any') return models;

		const filter = hfSizeFilters.find((f) => f.id === hfSizeFilter);
		if (!filter) return models;

		return models.filter((m) => {
			const sizeB = extractModelSizeB(m);
			if (sizeB === null) return true; // Include if can't determine size
			return sizeB >= filter.minB && sizeB < filter.maxB;
		});
	}

	// Get displayed models (with size filter applied)
	function getFilteredHFModels(models: HFModel[]): HFModel[] {
		return filterBySize(models);
	}

	function formatNumber(n: number): string {
		if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M';
		if (n >= 1e3) return (n / 1e3).toFixed(1) + 'K';
		return n.toString();
	}

	async function loadModels() {
		try {
			models = (await inferenceApi.listModels()) || [];
		} catch (e: any) {
			showMsg(e?.message || 'Не удалось получить список моделей', 'error');
			models = [];
		}
	}

	async function loadCache() {
		try {
			artifacts = (await inferenceApi.listCache()) || [];
		} catch (e: any) {
			console.error(e);
			artifacts = [];
		}
	}

	async function loadRepoDownloads() {
		try {
			repoDownloads = (await inferenceApi.listRepoDownloads()) || [];
		} catch (e: any) {
			console.error('Failed to load repo downloads:', e);
			repoDownloads = [];
		}
	}

	async function cancelRepoDownload(modelId: string) {
		if (!confirm(`Cancel download of ${modelId}?`)) return;
		try {
			await inferenceApi.cancelRepoDownload(modelId);
			await loadRepoDownloads();
		} catch (e: any) {
			console.error('Failed to cancel download:', e);
			alert('Failed to cancel download: ' + (e.message || e));
		}
	}

	async function removeRepoDownload(modelId: string) {
		try {
			await inferenceApi.removeRepoDownload(modelId);
			repoDownloads = repoDownloads.filter((d) => d.model_id !== modelId);
		} catch (e: any) {
			console.error('Failed to remove download:', e);
		}
	}

	async function loadTRTEngines() {
		try {
			trtEngines = (await inferenceApi.listTRTEngines()) || [];
		} catch (e: any) {
			console.error(e);
			trtEngines = [];
		}
	}

	function showMsg(text: string, type: 'info' | 'error' | 'success' = 'info') {
		msg = text;
		msgType = type;
		setTimeout(() => {
			msg = '';
		}, 5000);
	}

	async function submit(start: boolean) {
		busy = true;
		msg = '';
		try {
			const req: LoadRequest = {
				alias: form.alias.trim(),
				provider: form.provider,
				format: form.format,
				hf_repo: form.hf_repo?.trim(),
				hf_file: form.hf_file?.trim(),
				hf_revision: form.hf_revision?.trim(),
				gguf_url: form.gguf_url?.trim(),
				capabilities: form.capabilities,
				gpu_device: form.gpu_device?.trim(),
				vllm_tensor_parallel: form.vllm_tensor_parallel,
				vllm_max_model_len: form.vllm_max_model_len,
				vllm_gpu_utilization: form.vllm_gpu_utilization,
				vllm_extra_args: form.vllm_extra_args,
				llama_main_gpu: form.llama_main_gpu,
				llama_tensor_split: form.llama_tensor_split,
				llama_n_gpu_layers: form.llama_n_gpu_layers,
				llama_ctx_size: form.llama_ctx_size,
				llama_n_parallel: form.llama_n_parallel,
				llama_flash_attn: form.llama_flash_attn,
				llama_jinja: form.llama_jinja,
				llama_cache_reuse: form.llama_cache_reuse,
				llama_extra_args: form.llama_extra_args,
				sglang_tensor_parallel: form.sglang_tensor_parallel,
				sglang_mem_fraction: form.sglang_mem_fraction,
				tgi_num_shard: form.tgi_num_shard
			};
			if (start) {
				await inferenceApi.load(req);
				showMsg('Модель загружается...', 'success');
			} else {
				// Download repository locally instead of just preparing
				if (form.hf_repo) {
					// If specific file selected (e.g., GGUF), download only that file
					const filename = form.hf_file?.trim() || undefined;
					await inferenceApi.downloadRepository(form.hf_repo, filename);
					const msg = filename
						? `Скачивание файла ${filename} началось. Смотрите вкладку Downloads.`
						: 'Скачивание модели началось. Смотрите вкладку Downloads.';
					showMsg(msg, 'success');
					activeTab = 'downloads';
					await loadRepoDownloads();
				} else {
					showMsg('Укажите HF Repo для скачивания', 'error');
				}
			}
			await refreshAll();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка', 'error');
		} finally {
			busy = false;
		}
	}

	// Save config directly (without loading)
	async function saveConfig(andDownload: boolean = false) {
		busy = true;
		msg = '';
		try {
			const config = {
				alias: form.alias.trim(),
				provider: form.provider,
				format: form.format,
				hf_repo: form.hf_repo?.trim(),
				hf_file: form.hf_file?.trim(),
				hf_revision: form.hf_revision?.trim(),
				gguf_url: form.gguf_url?.trim(),
				capabilities: form.capabilities,
				gpu_device: form.gpu_device?.trim(),
				auto_start: false,
				vllm_tensor_parallel: form.vllm_tensor_parallel,
				vllm_max_model_len: form.vllm_max_model_len,
				vllm_gpu_utilization: form.vllm_gpu_utilization,
				vllm_extra_args: form.vllm_extra_args,
				llama_main_gpu: form.llama_main_gpu,
				llama_tensor_split: form.llama_tensor_split,
				llama_n_gpu_layers: form.llama_n_gpu_layers,
				llama_ctx_size: form.llama_ctx_size,
				llama_n_parallel: form.llama_n_parallel,
				llama_flash_attn: form.llama_flash_attn,
				llama_jinja: form.llama_jinja,
				llama_cache_reuse: form.llama_cache_reuse,
				llama_extra_args: form.llama_extra_args,
				sglang_tensor_parallel: form.sglang_tensor_parallel,
				sglang_mem_fraction: form.sglang_mem_fraction,
				tgi_num_shard: form.tgi_num_shard
			};

			await inferenceApi.createSaved(config);
			showMsg(`Конфигурация ${config.alias} сохранена`, 'success');

			if (andDownload && form.hf_repo) {
				const filename = form.hf_file?.trim() || undefined;
				await inferenceApi.downloadRepository(form.hf_repo, filename);
				showMsg(`Конфигурация сохранена и скачивание началось`, 'success');
				activeTab = 'downloads';
				await loadRepoDownloads();
			}

			await loadSavedModels();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка сохранения', 'error');
		} finally {
			busy = false;
		}
	}

	async function stopModel(alias: string) {
		try {
			await inferenceApi.stop(alias);
			showMsg(`Модель ${alias} остановлена`, 'success');
			await loadModels();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка остановки', 'error');
		}
	}

	async function restartModel(alias: string) {
		try {
			showMsg(`Перезапуск ${alias}...`, 'success');
			await inferenceApi.restart(alias);
			showMsg(`Модель ${alias} перезапущена`, 'success');
			await loadModels();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка перезапуска', 'error');
		}
	}

	// Logs modal functions
	async function openLogsModal(alias: string) {
		logsModalAlias = alias;
		logsModalContent = '';
		logsModalLoading = true;
		logsModalOpen = true;

		// Initial load
		await fetchLogsForModal();

		// Start auto-refresh every 2 seconds
		logsModalInterval = setInterval(fetchLogsForModal, 2000);
	}

	async function fetchLogsForModal() {
		try {
			const resp = await inferenceApi.logs(logsModalAlias, 500);
			logsModalContent = resp.logs || '';

			// Auto-scroll to bottom
			if (logsModalAutoScroll && logsContainer) {
				setTimeout(() => {
					if (logsContainer) {
						logsContainer.scrollTop = logsContainer.scrollHeight;
					}
				}, 50);
			}
		} catch (e: any) {
			logsModalContent = `Error loading logs: ${e?.message || 'Unknown error'}`;
		} finally {
			logsModalLoading = false;
		}
	}

	function closeLogsModal() {
		logsModalOpen = false;
		logsModalAlias = '';
		logsModalContent = '';
		if (logsModalInterval) {
			clearInterval(logsModalInterval);
			logsModalInterval = null;
		}
	}

	async function startModel(m: ModelInfo) {
		try {
			// Restart model with same parameters
			const req: LoadRequest = {
				alias: m.alias,
				provider: m.provider,
				format: m.format,
				capabilities: m.capabilities || ['chat'],
				hf_repo: m.hf_repo,
				hf_file: m.hf_file,
				gguf_url: m.gguf_url,
				gpu_device: '', // Use default GPUs
				vllm_tensor_parallel: m.vllm_tensor_parallel,
				vllm_max_model_len: m.vllm_max_model_len,
				vllm_gpu_utilization: m.vllm_gpu_utilization,
				vllm_extra_args: m.vllm_extra_args,
				llama_main_gpu: m.llama_main_gpu,
				llama_tensor_split: m.llama_tensor_split,
				llama_n_gpu_layers: m.llama_n_gpu_layers,
				llama_ctx_size: m.llama_ctx_size,
				llama_n_parallel: m.llama_n_parallel,
				llama_flash_attn: m.llama_flash_attn,
				sglang_tensor_parallel: m.sglang_tensor_parallel,
				sglang_mem_fraction: m.sglang_mem_fraction,
				tgi_num_shard: m.tgi_num_shard
			};
			await inferenceApi.load(req);
			showMsg(`Модель ${m.alias} запускается`, 'success');
			await loadModels();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка запуска', 'error');
		}
	}

	async function evictModel(alias: string) {
		try {
			await inferenceApi.evict(alias);
			showMsg(`Модель ${alias} выгружена`, 'success');
			await loadModels();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка выгрузки', 'error');
		}
	}

	async function togglePin(m: ModelInfo) {
		try {
			if (m.pinned) {
				await inferenceApi.unpin(m.alias);
			} else {
				await inferenceApi.pin(m.alias);
			}
			await loadModels();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка pin/unpin', 'error');
		}
	}

	async function saveModelConfig(alias: string, autoStart: boolean = false) {
		try {
			await inferenceApi.saveModel(alias, autoStart);
			showMsg(`Конфигурация ${alias} сохранена`, 'success');
			await loadSavedModels();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка сохранения', 'error');
		}
	}

	async function deleteSavedModel(alias: string) {
		try {
			await inferenceApi.deleteSaved(alias);
			showMsg(`Сохранённая модель ${alias} удалена`, 'success');
			await loadSavedModels();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка удаления', 'error');
		}
	}

	async function toggleAutoStart(saved: SavedModel) {
		try {
			await inferenceApi.setAutoStart(saved.alias, !saved.auto_start);
			await loadSavedModels();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка изменения auto-start', 'error');
		}
	}

	async function refreshSavedModel(saved: SavedModel) {
		if (!saved.hf_repo) {
			showMsg('Модель не имеет HuggingFace репозитория для обновления', 'error');
			return;
		}
		try {
			const result = await inferenceApi.refreshSaved(saved.alias);
			showMsg(
				`Обновление модели ${saved.alias} запущено (${result.total_files} файлов)`,
				'success'
			);
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка запуска обновления', 'error');
		}
	}

	function openEditSavedModal(saved: SavedModel) {
		editingSavedModel = saved;
		editSavedForm = {
			capabilities: saved.capabilities || ['chat'],
			auto_start: saved.auto_start || false,
			vllm_tensor_parallel: saved.vllm_tensor_parallel || 1,
			vllm_max_model_len: saved.vllm_max_model_len || 0,
			vllm_gpu_utilization: saved.vllm_gpu_utilization || 0.9,
			vllm_extra_args: saved.vllm_extra_args || '',
			llama_main_gpu: saved.llama_main_gpu || 0,
			llama_n_gpu_layers: saved.llama_n_gpu_layers ?? -1,
			llama_ctx_size: saved.llama_ctx_size || 0,
			llama_n_parallel: saved.llama_n_parallel || 0,
			llama_flash_attn: saved.llama_flash_attn || false,
			llama_jinja: saved.llama_jinja || false,
			llama_cache_reuse: saved.llama_cache_reuse || 0,
			llama_extra_args: saved.llama_extra_args || '',
			llama_tensor_split: saved.llama_tensor_split || '',
			sglang_tensor_parallel: saved.sglang_tensor_parallel || 1,
			sglang_mem_fraction: saved.sglang_mem_fraction || 0.9,
			tgi_num_shard: saved.tgi_num_shard || 1,
			gpu_device: saved.gpu_device || ''
		};
	}

	function toggleEditCapability(cap: Capability) {
		const caps = editSavedForm.capabilities || [];
		if (caps.includes(cap)) {
			// When removing, also remove implied capabilities
			let toRemove: Capability[] = [cap];
			if (cap === 'chat') {
				toRemove = CHAT_IMPLIED_CAPABILITIES;
			} else if (cap === 'embeddings') {
				toRemove = EMBEDDING_IMPLIED_CAPABILITIES;
			}
			editSavedForm.capabilities = caps.filter((c) => !toRemove.includes(c));
		} else {
			// When adding, also add implied capabilities
			let toAdd: Capability[] = [cap];
			if (cap === 'chat') {
				toAdd = CHAT_IMPLIED_CAPABILITIES;
			} else if (cap === 'embeddings') {
				toAdd = EMBEDDING_IMPLIED_CAPABILITIES;
			}
			editSavedForm.capabilities = [...new Set([...caps, ...toAdd])];
		}
	}

	async function saveEditedModel() {
		if (!editingSavedModel) return;
		try {
			await inferenceApi.updateSaved(editingSavedModel.alias, editSavedForm);
			showMsg(`Model ${editingSavedModel.alias} updated`, 'success');
			editingSavedModel = null;
			await loadSavedModels();
		} catch (e: any) {
			showMsg(e?.message || 'Failed to update model', 'error');
		}
	}

	async function loadSavedModelConfig(saved: SavedModel) {
		try {
			const req: LoadRequest = {
				alias: saved.alias,
				provider: saved.provider,
				format: saved.format,
				capabilities: saved.capabilities || ['chat'],
				hf_repo: saved.hf_repo,
				hf_file: saved.hf_file,
				gguf_url: saved.gguf_url,
				gpu_device: saved.gpu_device || '',
				vllm_tensor_parallel: saved.vllm_tensor_parallel,
				vllm_max_model_len: saved.vllm_max_model_len,
				vllm_gpu_utilization: saved.vllm_gpu_utilization,
				vllm_extra_args: saved.vllm_extra_args,
				llama_main_gpu: saved.llama_main_gpu,
				llama_n_gpu_layers: saved.llama_n_gpu_layers,
				llama_ctx_size: saved.llama_ctx_size,
				llama_n_parallel: saved.llama_n_parallel,
				llama_flash_attn: saved.llama_flash_attn,
				sglang_tensor_parallel: saved.sglang_tensor_parallel,
				sglang_mem_fraction: saved.sglang_mem_fraction,
				tgi_num_shard: saved.tgi_num_shard
			};
			await inferenceApi.load(req);
			showMsg(`Модель ${saved.alias} запускается`, 'success');
			await loadModels();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка загрузки', 'error');
		}
	}

	async function deleteModelArtifacts(alias: string) {
		if (!confirm(m.confirm_delete_model({ name: alias }))) return;
		try {
			await inferenceApi.deleteArtifacts(alias);
			showMsg(`Артефакты ${alias} удалены`, 'success');
			await refreshAll();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка удаления', 'error');
		}
	}

	async function selectModel(m: ModelInfo) {
		selectedModel = m;
		selectedLogs = '';
		selectedMetrics = '';
		selectedHealth = null;

		// Load health, logs, metrics
		await Promise.all([fetchHealth(m.alias), fetchLogs(m.alias), fetchMetrics(m.alias)]);

		// Auto-refresh logs every 5s
		if (logsInterval) clearInterval(logsInterval);
		logsInterval = setInterval(() => fetchLogs(m.alias), 5000);
	}

	async function fetchHealth(alias: string) {
		try {
			const h = await inferenceApi.health(alias);
			selectedHealth = { status: h.status, response_time_ms: h.response_time_ms, error: h.error };
		} catch (e: any) {
			selectedHealth = { status: 'error', error: e?.message };
		}
	}

	async function fetchLogs(alias: string) {
		try {
			const l = await inferenceApi.logs(alias, 200);
			selectedLogs = l.logs || '';
		} catch {
			// ignore
		}
	}

	async function fetchMetrics(alias: string) {
		try {
			const m = await inferenceApi.metrics(alias);
			selectedMetrics = m.metrics || '';
		} catch {
			selectedMetrics = '';
		}
	}

	function closeDetails() {
		selectedModel = null;
		if (logsInterval) {
			clearInterval(logsInterval);
			logsInterval = null;
		}
	}

	async function evictCacheToLimit() {
		const input = document.getElementById('evictLimitMB') as HTMLInputElement;
		const mb = parseFloat(input?.value || '0');
		if (mb <= 0) return;
		const bytes = Math.floor(mb * 1024 * 1024);
		try {
			await inferenceApi.evictCache(bytes);
			showMsg('Cache evicted', 'success');
			await loadCache();
		} catch (e: any) {
			showMsg(e?.message || 'Error evicting cache', 'error');
		}
	}

	async function clearAllCache() {
		if (!confirm(m.confirm_clear_cache({ size: formatSize(totalCacheSize) }))) {
			return;
		}
		busy = true;
		try {
			const res = await inferenceApi.clearCache();
			showMsg(`Cache cleared! Freed ${formatSize(res.freed_bytes)}`, 'success');
			await loadCache();
		} catch (e: any) {
			showMsg(e?.message || 'Error clearing cache', 'error');
		} finally {
			busy = false;
		}
	}

	async function convertToTRT() {
		busy = true;
		try {
			await inferenceApi.convertTRT({
				hf_model: trtForm.hf_model.trim(),
				max_batch_size: trtForm.max_batch_size,
				max_input_len: trtForm.max_input_len,
				max_output_len: trtForm.max_output_len,
				dtype: trtForm.dtype
			});
			showMsg('TRT конверсия запущена', 'success');
			await loadTRTEngines();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка конверсии', 'error');
		} finally {
			busy = false;
		}
	}

	async function deleteTRTEngine(modelId: string) {
		if (!confirm(m.confirm_delete_trt({ name: modelId }))) return;
		try {
			await inferenceApi.deleteTRTEngine(modelId);
			showMsg('TRT engine удалён', 'success');
			await loadTRTEngines();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка удаления', 'error');
		}
	}

	function formatSize(bytes: number) {
		if (!bytes) return '0 B';
		const gb = bytes / 1_073_741_824;
		if (gb >= 1) return `${gb.toFixed(2)} GB`;
		const mb = bytes / 1_048_576;
		if (mb >= 1) return `${mb.toFixed(1)} MB`;
		const kb = bytes / 1024;
		if (kb >= 1) return `${kb.toFixed(0)} KB`;
		return `${bytes} B`;
	}

	function formatDate(str?: string) {
		if (!str) return '—';
		return new Date(str).toLocaleString('ru-RU');
	}

	function statusColor(status: string) {
		switch (status.toLowerCase()) {
			case 'running':
				return 'text-green-600 dark:text-green-400';
			case 'starting':
			case 'downloading':
			case 'preparing':
				return 'text-yellow-600 dark:text-yellow-400';
			case 'stopped':
			case 'idle':
				return 'text-gray-500';
			case 'error':
			case 'failed':
				return 'text-red-600 dark:text-red-400';
			default:
				return 'text-gray-600';
		}
	}

	function toggleCapability(cap: Capability) {
		const caps = form.capabilities || [];
		if (caps.includes(cap)) {
			// When removing, also remove implied capabilities
			let toRemove: Capability[] = [cap];
			if (cap === 'chat') {
				toRemove = CHAT_IMPLIED_CAPABILITIES;
			} else if (cap === 'embeddings') {
				toRemove = EMBEDDING_IMPLIED_CAPABILITIES;
			}
			form.capabilities = caps.filter((c) => !toRemove.includes(c));
		} else {
			// When adding, also add implied capabilities
			let toAdd: Capability[] = [cap];
			if (cap === 'chat') {
				toAdd = CHAT_IMPLIED_CAPABILITIES;
			} else if (cap === 'embeddings') {
				toAdd = EMBEDDING_IMPLIED_CAPABILITIES;
			}
			// Merge without duplicates
			form.capabilities = [...new Set([...caps, ...toAdd])];
		}
	}

	const totalCacheSize = $derived((artifacts || []).reduce((sum, a) => sum + (a.size || 0), 0));
</script>

<svelte:head>
	<title>Inference Models</title>
</svelte:head>

<div class="flex flex-col gap-4 p-4">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">{m.admin_models_title()}</h1>
			<p class="text-muted-foreground text-sm">{m.admin_models_subtitle()}</p>
		</div>
		<button
			class="bg-primary text-primary-foreground hover:bg-primary/90 rounded-md px-4 py-2"
			onclick={refreshAll}
		>
			Refresh
		</button>
	</div>

	<!-- Message -->
	{#if msg}
		<div
			class={`rounded-md border px-4 py-3 text-sm ${msgType === 'error' ? 'border-red-200 bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300' : msgType === 'success' ? 'border-green-200 bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-300' : 'bg-muted/50'}`}
		>
			{msg}
		</div>
	{/if}

	<!-- Tabs -->
	<div class="flex gap-2 border-b">
		<button
			class={`-mb-px px-4 py-2 ${activeTab === 'models' ? 'border-primary border-b-2 font-semibold' : 'text-muted-foreground'}`}
			onclick={() => (activeTab = 'models')}
		>
			Models ({(models || []).length})
		</button>
		<button
			class={`-mb-px px-4 py-2 ${activeTab === 'hf' ? 'border-primary border-b-2 font-semibold' : 'text-muted-foreground'}`}
			onclick={() => (activeTab = 'hf')}
		>
			<FontAwesomeIcon icon={faFaceSmile} class="inline h-4 w-4" /> HuggingFace
		</button>
		<button
			class={`-mb-px px-4 py-2 ${activeTab === 'cache' ? 'border-primary border-b-2 font-semibold' : 'text-muted-foreground'}`}
			onclick={() => (activeTab = 'cache')}
		>
			Cache ({formatSize(totalCacheSize)})
		</button>
		<button
			class={`-mb-px px-4 py-2 ${activeTab === 'trt' ? 'border-primary border-b-2 font-semibold' : 'text-muted-foreground'}`}
			onclick={() => (activeTab = 'trt')}
		>
			TRT Engines ({(trtEngines || []).length})
		</button>
		<button
			class={`-mb-px px-4 py-2 ${activeTab === 'downloads' ? 'border-primary border-b-2 font-semibold' : 'text-muted-foreground'}`}
			onclick={() => (activeTab = 'downloads')}
		>
			📥 Downloads ({repoDownloads.filter((d) => d.status === 'downloading').length || ''})
		</button>
	</div>

	<!-- Models Tab -->
	{#if activeTab === 'models'}
		<div class="grid gap-4 lg:grid-cols-[1fr_400px]">
			<!-- Left: Model list + Load form -->
			<div class="flex flex-col gap-4">
				<!-- Load Form -->
				<div class="bg-card space-y-4 rounded-lg border p-4">
					<h2 class="text-lg font-semibold">Load Model</h2>
					<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
						<label class="flex flex-col gap-1 text-sm">
							<span class="font-medium">Alias</span>
							<input
								class="bg-background rounded border px-3 py-2"
								bind:value={form.alias}
								placeholder="llama-3-8b"
							/>
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span class="font-medium">Provider</span>
							<select
								class="bg-background rounded border px-3 py-2"
								bind:value={form.provider}
								onchange={updateTensorParallelForProvider}
							>
								{#each providers as p}<option value={p}>{p}</option>{/each}
							</select>
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span class="font-medium">Format</span>
							<select class="bg-background rounded border px-3 py-2" bind:value={form.format}>
								{#each formats as f}<option value={f}>{f}</option>{/each}
							</select>
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span class="font-medium">HF Repo</span>
							<input
								class="bg-background rounded border px-3 py-2"
								bind:value={form.hf_repo}
								placeholder="meta-llama/Llama-3.1-8B-Instruct"
								oninput={() => {
									// Debounce recommendation update
									clearTimeout(hfRepoDebounce);
									hfRepoDebounce = setTimeout(() => updateRecommendations(), 500);
								}}
							/>
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span class="font-medium">HF File (GGUF)</span>
							<input
								class="bg-background rounded border px-3 py-2"
								bind:value={form.hf_file}
								placeholder="model-Q4_K_M.gguf"
							/>
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span class="font-medium">GGUF URL</span>
							<input
								class="bg-background rounded border px-3 py-2"
								bind:value={form.gguf_url}
								placeholder="https://..."
							/>
						</label>
					</div>

					<!-- GPU Device Selection -->
					<div class="flex flex-col gap-2 text-sm">
						<span class="font-medium">GPU Device</span>
						{#if gpuDevices.length > 0}
							<div class="flex flex-wrap gap-2">
								{#each gpuDevices as gpu}
									<button
										type="button"
										class="flex flex-col items-start rounded border px-3 py-2 text-sm {selectedGPUs.includes(
											gpu.index
										)
											? 'bg-primary text-primary-foreground border-primary'
											: 'hover:bg-muted'}"
										onclick={() => toggleGPU(gpu.index)}
									>
										<span class="font-medium">GPU {gpu.index}: {gpu.name}</span>
										<span class="text-xs opacity-75"
											>{Math.round(gpu.memory_mb / 1024)} GB ({Math.round(
												gpu.memory_free_mb / 1024
											)} GB free)</span
										>
									</button>
								{/each}
							</div>
							<span class="text-muted-foreground text-xs">
								{selectedGPUs.length === 0
									? 'No GPU selected = use all GPUs'
									: `Selected: GPU ${selectedGPUs.join(', ')}`}
							</span>
						{:else}
							<input
								class="bg-background rounded border px-3 py-2"
								bind:value={form.gpu_device}
								placeholder="0, 1, or 0,1 (empty = all GPUs)"
							/>
							<span class="text-muted-foreground text-xs"
								>GPU list unavailable. Enter index manually.</span
							>
						{/if}
					</div>

					<!-- Capabilities -->
					<div class="flex flex-col gap-2 text-sm">
						<span class="font-medium">Capabilities</span>
						<div class="flex gap-2">
							{#each capabilities as c}
								<button
									type="button"
									class={`rounded border px-3 py-1 text-sm ${(form.capabilities || []).includes(c) ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'}`}
									onclick={() => toggleCapability(c)}
								>
									{c}
								</button>
							{/each}
						</div>
					</div>

					<!-- Memory Recommendations -->
					{#if memoryRecommendation}
						<div
							class="rounded-lg border p-3 {memoryRecommendation.willFit
								? 'border-green-500/30 bg-green-500/10'
								: 'border-yellow-500/30 bg-yellow-500/10'}"
						>
							<div class="flex items-start justify-between gap-4">
								<div class="flex-1">
									<div class="mb-1 text-sm font-medium">
										<FontAwesomeIcon icon={memoryRecommendation.willFit ? faCircleCheck : faTriangleExclamation} class="mr-1 h-3.5 w-3.5" /> Memory Recommendation
									</div>
									<div class="text-muted-foreground grid grid-cols-2 gap-2 text-xs sm:grid-cols-4">
										<div>
											<span class="text-foreground font-medium"
												>{memoryRecommendation.modelParams}</span
											>
											<span class="block">{memoryRecommendation.quantization}</span>
										</div>
										<div>
											<span class="text-foreground font-medium"
												>~{memoryRecommendation.modelSizeGB} GB</span
											>
											<span class="block">Model Size</span>
										</div>
										<div>
											<span class="text-foreground font-medium"
												>{memoryRecommendation.totalGPUMemoryGB} GB</span
											>
											<span class="block">GPU Available</span>
										</div>
										<div>
											<span class="text-foreground font-medium"
												>{memoryRecommendation.recommendedMemFraction}</span
											>
											<span class="block">Rec. Mem Frac</span>
										</div>
									</div>
									{#if memoryRecommendation.warning}
										<p class="mt-2 text-xs text-yellow-600 dark:text-yellow-400">
											{memoryRecommendation.warning}
										</p>
									{/if}
								</div>
								<button
									type="button"
									class="bg-primary/10 hover:bg-primary/20 text-primary rounded px-2 py-1 text-xs"
									onclick={() => {
										if (memoryRecommendation) {
											if (form.provider === 'vllm') {
												form.vllm_gpu_utilization = memoryRecommendation.recommendedMemFraction;
											} else if (form.provider === 'sglang') {
												form.sglang_mem_fraction = memoryRecommendation.recommendedMemFraction;
											}
										}
									}}
								>
									Apply
								</button>
							</div>
						</div>
					{/if}

					<!-- Provider-specific params -->
					{#if form.provider === 'vllm'}
						<div class="grid gap-3 border-t pt-2 sm:grid-cols-3">
							<label class="flex flex-col gap-1 text-sm">
								<span>Tensor Parallel</span>
								<input
									type="number"
									min="0"
									class="bg-background rounded border px-3 py-2"
									bind:value={form.vllm_tensor_parallel}
								/>
							</label>
							<label class="flex flex-col gap-1 text-sm">
								<span>Max Model Len</span>
								<input
									type="number"
									min="0"
									class="bg-background rounded border px-3 py-2"
									bind:value={form.vllm_max_model_len}
								/>
							</label>
							<label class="flex flex-col gap-1 text-sm">
								<span>GPU Utilization</span>
								<input
									type="number"
									step="0.01"
									min="0"
									max="1"
									class="bg-background rounded border px-3 py-2"
									bind:value={form.vllm_gpu_utilization}
								/>
							</label>
						</div>
						<div class="pt-2">
							<label class="flex flex-col gap-1 text-sm">
								<span title="Extra command-line args passed to vLLM (space-separated). E.g. --enable-auto-tool-choice --tool-call-parser hermes">extra_args</span>
								<input
									class="bg-background rounded border px-3 py-2"
									bind:value={form.vllm_extra_args}
									placeholder="--enable-auto-tool-choice --tool-call-parser hermes"
								/>
								<span class="text-muted-foreground text-xs">Additional CLI flags for vLLM (e.g. tool calling support)</span>
							</label>
						</div>
					{:else if form.provider === 'llama.cpp'}
						<div class="grid gap-3 border-t pt-2 sm:grid-cols-2 lg:grid-cols-4">
							<label class="flex flex-col gap-1 text-sm">
								<span>n_gpu_layers</span>
								<input
									type="number"
									min="0"
									class="bg-background rounded border px-3 py-2"
									bind:value={form.llama_n_gpu_layers}
								/>
							</label>
							<label class="flex flex-col gap-1 text-sm">
								<span title="Context size (0 = default 2048)">ctx_size</span>
								<input
									type="number"
									min="0"
									class="bg-background rounded border px-3 py-2"
									bind:value={form.llama_ctx_size}
									placeholder="32768"
								/>
							</label>
							<label class="flex flex-col gap-1 text-sm">
								<span title="Parallel request slots (0 = auto)">n_parallel</span>
								<input
									type="number"
									min="0"
									class="bg-background rounded border px-3 py-2"
									bind:value={form.llama_n_parallel}
									placeholder="4"
								/>
							</label>
							<label class="flex items-center gap-2 pt-5 text-sm">
								<input type="checkbox" class="h-4 w-4" bind:checked={form.llama_flash_attn} />
								<span title="Enable Flash Attention for faster inference">Flash Attn</span>
							</label>
							<label class="flex items-center gap-2 pt-5 text-sm">
								<input type="checkbox" class="h-4 w-4" bind:checked={form.llama_jinja} />
								<span title="Enable Jinja template processing for chat templates">Jinja</span>
							</label>
						</div>
						<div class="grid gap-3 pt-2 sm:grid-cols-2 lg:grid-cols-3">
							<label class="flex flex-col gap-1 text-sm">
								<span>main_gpu</span>
								<input
									type="number"
									min="0"
									class="bg-background rounded border px-3 py-2"
									bind:value={form.llama_main_gpu}
								/>
							</label>
							<label class="flex flex-col gap-1 text-sm">
								<span>tensor_split</span>
								<input
									class="bg-background rounded border px-3 py-2"
									bind:value={form.llama_tensor_split}
									placeholder="0.5,0.5"
								/>
							</label>
						</div>
						<div class="grid gap-3 pt-2 sm:grid-cols-2">
							<label class="flex flex-col gap-1 text-sm">
								<span title="KV cache reuse size. 0 = llama.cpp default (256). -1 = disable (recommended for SWA models like GPT-OSS)">cache_reuse</span>
								<input
									type="number"
									min="-1"
									class="bg-background rounded border px-3 py-2"
									bind:value={form.llama_cache_reuse}
									placeholder="0"
								/>
								<span class="text-muted-foreground text-xs">0 = default, -1 = disable (SWA models)</span>
							</label>
							<label class="flex flex-col gap-1 text-sm">
								<span title="Extra command-line args passed to llama-server (space-separated)">extra_args</span>
								<input
									class="bg-background rounded border px-3 py-2"
									bind:value={form.llama_extra_args}
									placeholder="--no-mmap --verbose"
								/>
								<span class="text-muted-foreground text-xs">Additional CLI flags for llama-server</span>
							</label>
						</div>
					{:else if form.provider === 'sglang'}
						<div class="grid gap-3 border-t pt-2 sm:grid-cols-2">
							<label class="flex flex-col gap-1 text-sm">
								<span>Tensor Parallel</span>
								<input
									type="number"
									min="0"
									class="bg-background rounded border px-3 py-2"
									bind:value={form.sglang_tensor_parallel}
								/>
							</label>
							<label class="flex flex-col gap-1 text-sm">
								<span>Mem Fraction</span>
								<input
									type="number"
									step="0.01"
									min="0"
									max="1"
									class="bg-background rounded border px-3 py-2"
									bind:value={form.sglang_mem_fraction}
								/>
							</label>
						</div>
					{:else if form.provider === 'tgi'}
						<div class="grid gap-3 border-t pt-2 sm:grid-cols-1">
							<label class="flex flex-col gap-1 text-sm">
								<span>Num Shards</span>
								<input
									type="number"
									min="1"
									class="bg-background rounded border px-3 py-2"
									bind:value={form.tgi_num_shard}
								/>
							</label>
						</div>
					{/if}

					<div class="flex flex-wrap gap-2 pt-2">
						<button
							class="bg-primary text-primary-foreground rounded px-4 py-2 disabled:opacity-50"
							onclick={() => submit(true)}
							disabled={busy}
						>
							Load & Start
						</button>
						<button
							class="hover:bg-muted rounded border px-4 py-2 disabled:opacity-50"
							onclick={() => saveConfig(false)}
							disabled={busy}
						>
							Save Config
						</button>
						<button
							class="hover:bg-muted rounded border px-4 py-2 disabled:opacity-50"
							onclick={() => saveConfig(true)}
							disabled={busy}
						>
							Download & Save
						</button>
						<button
							class="hover:bg-muted text-muted-foreground rounded border px-4 py-2 disabled:opacity-50"
							onclick={() => submit(false)}
							disabled={busy}
						>
							Download Only
						</button>
					</div>
				</div>

				<!-- Saved Models (persisted configs) -->
				{#if savedModels.length > 0}
					<div class="bg-card overflow-hidden rounded-lg border">
						<div class="bg-muted/50 border-b px-4 py-3">
							<h2 class="font-semibold">
								Saved Models <span class="text-muted-foreground text-xs font-normal"
									>(click to load)</span
								>
							</h2>
						</div>
						<div class="divide-y">
							{#each savedModels as saved}
								{@const isRunning = models.some((m) => m.alias === saved.alias)}
								<div class="hover:bg-muted/30 flex items-center gap-4 px-4 py-3">
									<div class="min-w-0 flex-1">
										<div class="flex items-center gap-2">
											<span class="font-medium">{saved.alias}</span>
											{#if saved.auto_start}
												<span
													class="rounded bg-blue-100 px-1.5 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
													>auto-start</span
												>
											{/if}
											{#if isRunning}
												<span
													class="rounded bg-green-100 px-1.5 py-0.5 text-xs text-green-700 dark:bg-green-900/30 dark:text-green-300"
													>running</span
												>
											{/if}
										</div>
										<div class="text-muted-foreground mt-1 text-xs">
											{saved.provider} · {saved.format} · {saved.hf_repo ||
												saved.gguf_url ||
												'local'}
										</div>
									</div>
									<div class="flex flex-shrink-0 gap-1">
										{#if !isRunning}
											<button
												class="bg-primary/10 text-primary hover:bg-primary/20 rounded border px-2 py-1 text-xs"
												onclick={() => loadSavedModelConfig(saved)}>Load</button
											>
										{/if}
										<button
											class="hover:bg-muted rounded border px-2 py-1 text-xs"
											onclick={() => openEditSavedModal(saved)}
											title="Edit parameters"
										>
											<FontAwesomeIcon icon={faGear} class="h-3 w-3" />
										</button>
										{#if saved.hf_repo}
											<button
												class="hover:bg-muted rounded border px-2 py-1 text-xs text-blue-500 hover:bg-blue-500/10"
												onclick={() => refreshSavedModel(saved)}
												title="Update/re-download model files"
											>
												<FontAwesomeIcon icon={faArrowsRotate} class="h-3 w-3" />
											</button>
										{/if}
										<button
											class="hover:bg-muted rounded border px-2 py-1 text-xs"
											onclick={() => toggleAutoStart(saved)}
											title={saved.auto_start ? 'Disable auto-start' : 'Enable auto-start'}
										>
											<FontAwesomeIcon icon={saved.auto_start ? faPause : faPlay} class="h-3 w-3" />
										</button>
										<button
											class="rounded border px-2 py-1 text-xs text-red-500 hover:bg-red-500/10"
											onclick={() => deleteSavedModel(saved.alias)}><FontAwesomeIcon icon={faXmark} class="h-3 w-3" /></button
										>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Running Models List -->
				<div class="bg-card overflow-hidden rounded-lg border">
					<div class="bg-muted/50 flex items-center justify-between border-b px-4 py-3">
						<h2 class="font-semibold">Running Models</h2>
					</div>
					<div class="divide-y">
						{#if (models || []).length === 0}
							<div class="text-muted-foreground px-4 py-8 text-center">No models loaded</div>
						{:else}
							{#each models || [] as mdl}
								{@const isSaved = savedModels.some((s) => s.alias === mdl.alias)}
								<div
									class="hover:bg-muted/30 flex cursor-pointer items-start gap-4 px-4 py-3"
									role="button"
									tabindex="0"
									onclick={() => selectModel(mdl)}
									onkeydown={(e) => e.key === 'Enter' && selectModel(mdl)}
								>
									<div class="min-w-0 flex-1">
										<div class="flex items-center gap-2">
											<span class="font-semibold">{mdl.alias}</span>
											{#if mdl.pinned}
												<span
													class="rounded bg-yellow-100 px-1.5 py-0.5 text-xs text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300"
													>pinned</span
												>
											{/if}
											{#if isSaved}
												<span
													class="rounded bg-blue-100 px-1.5 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
													>saved</span
												>
											{/if}
											<span class={`text-sm font-medium ${statusColor(mdl.status)}`}
												>{mdl.status}</span
											>
										</div>
										<div class="text-muted-foreground mt-1 text-xs">
											{mdl.provider} · {mdl.format} · {(mdl.capabilities || []).join(', ') ||
												'chat'}
										</div>
										{#if mdl.endpoint}
											<div class="text-muted-foreground truncate text-xs">{mdl.endpoint}</div>
										{/if}
										{#if mdl.last_error}
											<div class="mt-1 text-xs text-red-500">{mdl.last_error}</div>
										{/if}
									</div>
									<div class="flex flex-shrink-0 gap-1">
										{#if mdl.status === 'running' || mdl.status === 'starting'}
											<button
												class="hover:bg-muted rounded border px-2 py-1 text-xs"
												onclick={(e) => {
													e.stopPropagation();
													restartModel(mdl.alias);
												}}
												title="Restart model container">Restart</button
											>
											<button
												class="hover:bg-muted rounded border px-2 py-1 text-xs"
												onclick={(e) => {
													e.stopPropagation();
													stopModel(mdl.alias);
												}}>Stop</button
											>
											<button
												class="hover:bg-muted rounded border px-2 py-1 text-xs"
												onclick={(e) => {
													e.stopPropagation();
													openLogsModal(mdl.alias);
												}}
												title={m.admin_models_logs_title()}>{m.admin_models_logs()}</button
											>
										{:else}
											<button
												class="rounded border bg-green-500/10 px-2 py-1 text-xs text-green-600 hover:bg-green-500/20"
												onclick={(e) => {
													e.stopPropagation();
													startModel(mdl);
												}}>Start</button
											>
										{/if}
										{#if !isSaved}
											<button
												class="rounded border bg-blue-500/10 px-2 py-1 text-xs text-blue-600 hover:bg-blue-500/20"
												onclick={(e) => {
													e.stopPropagation();
													saveModelConfig(mdl.alias);
												}}
												title="Save config for restart">Save</button
											>
										{/if}
										<button
											class="hover:bg-muted rounded border px-2 py-1 text-xs"
											onclick={(e) => {
												e.stopPropagation();
												evictModel(mdl.alias);
											}}>Evict</button
										>
										<button
											class="hover:bg-muted rounded border px-2 py-1 text-xs"
											onclick={(e) => {
												e.stopPropagation();
												togglePin(mdl);
											}}
										>
											{mdl.pinned ? 'Unpin' : 'Pin'}
										</button>
									</div>
								</div>
							{/each}
						{/if}
					</div>
				</div>
			</div>

			<!-- Right: Details Panel -->
			<div class="bg-card sticky top-4 h-fit rounded-lg border">
				{#if selectedModel}
					<div class="bg-muted/50 flex items-center justify-between border-b px-4 py-3">
						<h2 class="font-semibold">{selectedModel.alias}</h2>
						<button class="text-muted-foreground hover:text-foreground" onclick={closeDetails}
							><FontAwesomeIcon icon={faXmark} class="h-4 w-4" /></button
						>
					</div>
					<div class="space-y-4 p-4">
						<!-- Health -->
						<div>
							<h3 class="mb-2 text-sm font-medium">Health</h3>
							{#if selectedHealth}
								<div class="flex items-center gap-2">
									<span
										class={`h-2 w-2 rounded-full ${selectedHealth.status === 'healthy' ? 'bg-green-500' : selectedHealth.status === 'unhealthy' ? 'bg-red-500' : 'bg-yellow-500'}`}
									></span>
									<span class="text-sm">{selectedHealth.status}</span>
									{#if selectedHealth.response_time_ms}
										<span class="text-muted-foreground text-xs"
											>({selectedHealth.response_time_ms}ms)</span
										>
									{/if}
								</div>
								{#if selectedHealth.error}
									<div class="mt-1 text-xs text-red-500">{selectedHealth.error}</div>
								{/if}
							{:else}
								<div class="text-muted-foreground text-sm">Loading...</div>
							{/if}
						</div>

						<!-- Info -->
						<div>
							<h3 class="mb-2 text-sm font-medium">Details</h3>
							<dl class="space-y-1 text-sm">
								<div class="flex justify-between">
									<dt class="text-muted-foreground">Provider</dt>
									<dd>{selectedModel.provider}</dd>
								</div>
								<div class="flex justify-between">
									<dt class="text-muted-foreground">Format</dt>
									<dd>{selectedModel.format}</dd>
								</div>
								<div class="flex justify-between">
									<dt class="text-muted-foreground">Status</dt>
									<dd class={statusColor(selectedModel.status)}>{selectedModel.status}</dd>
								</div>
								{#if selectedModel.endpoint}
									<div class="flex justify-between">
										<dt class="text-muted-foreground">Endpoint</dt>
										<dd class="max-w-[200px] truncate">{selectedModel.endpoint}</dd>
									</div>
								{/if}
								{#if selectedModel.last_used}
									<div class="flex justify-between">
										<dt class="text-muted-foreground">Last Used</dt>
										<dd>{formatDate(selectedModel.last_used)}</dd>
									</div>
								{/if}
							</dl>
						</div>

						<!-- Logs -->
						<div>
							<h3 class="mb-2 text-sm font-medium">Logs (last 200 lines)</h3>
							<pre
								class="bg-muted/50 max-h-48 overflow-auto rounded p-2 text-xs whitespace-pre-wrap">{selectedLogs ||
									'No logs'}</pre>
						</div>

						<!-- Metrics -->
						{#if selectedMetrics}
							<div>
								<h3 class="mb-2 text-sm font-medium">Metrics</h3>
								<pre
									class="bg-muted/50 max-h-32 overflow-auto rounded p-2 text-xs whitespace-pre-wrap">{selectedMetrics}</pre>
							</div>
						{/if}

						<!-- Actions -->
						<div class="flex gap-2 border-t pt-2">
							<button
								class="hover:bg-muted rounded border px-3 py-1.5 text-sm"
								onclick={() => deleteModelArtifacts(selectedModel!.alias)}
							>
								Delete Artifacts
							</button>
							<button
								class="hover:bg-muted rounded border px-3 py-1.5 text-sm"
								onclick={() => fetchLogs(selectedModel!.alias)}
							>
								Refresh Logs
							</button>
						</div>
					</div>
				{:else}
					<div class="text-muted-foreground p-8 text-center">Select a model to view details</div>
				{/if}
			</div>
		</div>
	{/if}

	<!-- HuggingFace Browser Tab -->
	{#if activeTab === 'hf'}
		<div class="grid gap-4 lg:grid-cols-[1fr_400px]">
			<!-- Left: Search and results -->
			<div class="flex flex-col gap-4">
				<!-- Provider filter selector -->
				<div class="flex flex-wrap gap-2">
					{#each hfProviderFilters as pf}
						<button
							class="flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm transition-colors {hfProviderFilter ===
							pf.id
								? 'bg-primary text-primary-foreground border-primary'
								: 'hover:bg-muted'}"
							onclick={() => {
								hfProviderFilter = pf.id;
								hfSearchQuery = '';
								hfSearchResults = [];
								hfSearchPerformed = false;
								hfSearchError = '';
							}}
							title={pf.description}
						>
							<span><FontAwesomeIcon icon={pf.icon} class="h-4 w-4" /></span>
							{pf.label}
						</button>
					{/each}
				</div>

				<!-- Size filter selector -->
				<div class="flex flex-wrap items-center gap-2">
					<span class="text-muted-foreground text-xs">Size:</span>
					{#each hfSizeFilters as sf}
						<button
							class="rounded border px-2 py-1 text-xs transition-colors {hfSizeFilter === sf.id
								? 'bg-primary text-primary-foreground border-primary'
								: 'hover:bg-muted'}"
							onclick={() => {
								hfSizeFilter = sf.id;
							}}
							title={sf.range}
						>
							{sf.label}
						</button>
					{/each}
				</div>

				<!-- Search -->
				<div class="flex gap-2">
					<div class="relative flex-1">
						<Search
							class="text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2"
						/>
						<input
							type="text"
							class="bg-background w-full rounded-md border py-2 pr-4 pl-10"
							placeholder={m.placeholder_search_models()}
							bind:value={hfSearchQuery}
							onkeydown={(e) => e.key === 'Enter' && searchHF()}
						/>
					</div>
					<button
						class="bg-primary text-primary-foreground hover:bg-primary/90 flex items-center gap-2 rounded-md px-4 py-2 disabled:opacity-50"
						onclick={searchHF}
						disabled={hfSearching}
					>
						{#if hfSearching}
							<Loader2 class="h-4 w-4 animate-spin" />
						{/if}
						Search
					</button>
				</div>

				<!-- Provider hint -->
				<div class="text-muted-foreground bg-muted/50 rounded px-3 py-2 text-xs">
					{#if hfProviderFilter === 'llama.cpp'}
						<FontAwesomeIcon icon={faHorse} class="mr-1 h-3 w-3" /> GGUF quantized models for <strong>llama.cpp</strong> — efficient CPU/GPU inference
					{:else if hfProviderFilter === 'vllm'}
						<FontAwesomeIcon icon={faBolt} class="mr-1 h-3 w-3" /> Models for <strong>vLLM</strong> — high-throughput production serving
					{:else if hfProviderFilter === 'sglang'}
						<FontAwesomeIcon icon={faRocket} class="mr-1 h-3 w-3" /> Models for <strong>SGLang</strong> — fast inference with RadixAttention
					{:else if hfProviderFilter === 'tgi'}
						<FontAwesomeIcon icon={faFaceSmile} class="mr-1 h-3 w-3" /> Models for <strong>TGI</strong> — HuggingFace Text Generation Inference
					{:else if hfProviderFilter === 'tei'}
						<FontAwesomeIcon icon={faLink} class="mr-1 h-3 w-3" /> Models for <strong>TEI</strong> — Text Embeddings Inference (embeddings & reranking)
					{:else if hfProviderFilter === 'embedding'}
						<FontAwesomeIcon icon={faChartBar} class="mr-1 h-3 w-3" /> Embedding models for <strong>vLLM/SGLang</strong> — feature extraction & RAG
					{:else if hfProviderFilter === 'rerank'}
						<FontAwesomeIcon icon={faShuffle} class="mr-1 h-3 w-3" /> Reranking models for <strong>TEI</strong> — cross-encoder reranking
					{:else}
						<FontAwesomeIcon icon={faLightbulb} class="mr-1 h-3 w-3" /> Filter by provider to find compatible models
					{/if}
				</div>

				<!-- Results -->
				<div class="bg-card rounded-lg border">
					<div class="bg-muted/50 border-b px-4 py-3">
						<h2 class="font-semibold">
							{#if hfSearchResults.length > 0}
								{@const filtered = getFilteredHFModels(hfSearchResults)}
								Search Results ({filtered.length}{hfSizeFilter !== 'any'
									? ` of ${hfSearchResults.length}`
									: ''})
							{:else}
								{@const filtered = getFilteredHFModels(hfPopularModels)}
								Popular {hfProviderFilters.find((c) => c.id === hfProviderFilter)?.label || ''} Models
								({filtered.length}{hfSizeFilter !== 'any' ? ` of ${hfPopularModels.length}` : ''})
							{/if}
						</h2>
					</div>
					<div class="max-h-[600px] divide-y overflow-y-auto">
						{#each getFilteredHFModels(hfSearchResults.length > 0 ? hfSearchResults : hfPopularModels) as hfm}
							<div
								class="hover:bg-muted/30 flex cursor-pointer items-start gap-3 px-4 py-3 {hfSelectedModel?.id ===
								hfm.id
									? 'bg-primary/10'
									: ''}"
								role="button"
								tabindex="0"
								onclick={() => selectHFModel(hfm)}
								onkeydown={(e) => e.key === 'Enter' && selectHFModel(hfm)}
							>
								<div class="min-w-0 flex-1">
									<div class="truncate font-medium">{hfm.id}</div>
									<div class="text-muted-foreground mt-1 flex flex-wrap gap-2 text-xs">
										<span><FontAwesomeIcon icon={faArrowDown} class="mr-0.5 h-3 w-3" /> {formatNumber(hfm.downloads || 0)}</span>
										<span><FontAwesomeIcon icon={faHeart} class="mr-0.5 h-3 w-3" /> {formatNumber(hfm.likes || 0)}</span>
										{#if extractModelSizeB(hfm)}
											<span
												class="rounded bg-blue-500/20 px-1.5 py-0.5 text-xs font-medium text-blue-600 dark:text-blue-400"
												>{extractModelSizeB(hfm)}B</span
											>
										{/if}
										{#if hfm.pipeline_tag}
											<span class="bg-muted rounded px-1.5 py-0.5 text-xs">{hfm.pipeline_tag}</span>
										{/if}
									</div>
									{#if hfm.tags?.length}
										<div class="mt-1 flex flex-wrap gap-1">
											{#each hfm.tags.slice(0, 5) as tag}
												<span class="bg-muted/50 rounded px-1.5 py-0.5 text-xs">{tag}</span>
											{/each}
										</div>
									{/if}
								</div>
								<a
									href="https://huggingface.co/{hfm.id}"
									target="_blank"
									class="hover:bg-muted rounded p-1"
									onclick={(e) => e.stopPropagation()}
								>
									<ExternalLink class="h-4 w-4" />
								</a>
							</div>
						{:else}
							<div class="px-4 py-8 text-center text-muted-foreground">
								{#if hfSearching}
									<Loader2 class="h-6 w-6 animate-spin mx-auto mb-2" />
									Searching...
								{:else if hfSearchError}
									<div class="text-amber-500">
										<FontAwesomeIcon icon={faTriangleExclamation} class="mr-1 h-3.5 w-3.5" /> {hfSearchError}
									</div>
									<div class="text-xs mt-2">
										Try a different search term or check model name spelling
									</div>
								{:else if hfSearchPerformed && hfSearchResults.length === 0}
									<div>No models found</div>
									<div class="text-xs mt-2">Try a different search term</div>
								{:else}
									Search for models or wait for popular models to load
								{/if}
							</div>
						{/each}

						<!-- Load More button -->
						{#if hfSearchResults.length === 0 && hfHasMore && hfPopularModels.length > 0}
							<div class="border-t px-4 py-3">
								<button
									class="bg-muted hover:bg-muted/80 flex w-full items-center justify-center gap-2 rounded px-4 py-2 text-sm font-medium disabled:opacity-50"
									onclick={loadMoreModels}
									disabled={hfLoadingMore}
								>
									{#if hfLoadingMore}
										<Loader2 class="h-4 w-4 animate-spin" />
										{m.common_loading()}
									{:else}
										{m.common_loadMore()}
									{/if}
								</button>
							</div>
						{/if}
					</div>
				</div>
			</div>

			<!-- Right: Selected model details -->
			<div class="bg-card rounded-lg border">
				<div class="bg-muted/50 border-b px-4 py-3">
					<h2 class="font-semibold">Model Details</h2>
				</div>
				{#if hfSelectedModel}
					<div class="space-y-4 p-4">
						<div>
							<div class="text-muted-foreground text-sm">Model ID</div>
							<div class="font-mono text-sm break-all">{hfSelectedModel.id}</div>
						</div>
						<div>
							<div class="text-muted-foreground text-sm">Author</div>
							<div>{hfSelectedModel.author || hfSelectedModel.id.split('/')[0]}</div>
						</div>
						<div class="flex gap-4">
							<div>
								<div class="text-muted-foreground text-sm">Downloads</div>
								<div>{formatNumber(hfSelectedModel.downloads || 0)}</div>
							</div>
							<div>
								<div class="text-muted-foreground text-sm">Likes</div>
								<div>{formatNumber(hfSelectedModel.likes || 0)}</div>
							</div>
						</div>

						<!-- Files (GGUF/safetensors) -->
						{#if hfModelFiles.length > 0}
							<div>
								<div class="text-muted-foreground mb-2 text-sm">Available Files</div>
								<div class="max-h-64 space-y-1 overflow-y-auto">
									{#each hfModelFiles as f}
										<div
											class="bg-muted/30 flex items-center justify-between gap-2 rounded p-2 text-sm"
										>
											<span class="flex-1 truncate" title={f.rfilename}>{f.rfilename}</span>
											<div class="flex items-center gap-1">
												{#if f.size}
													<span class="text-muted-foreground text-xs">{formatSize(f.size)}</span>
												{/if}
												<button
													class="bg-primary text-primary-foreground hover:bg-primary/90 rounded px-2 py-1 text-xs"
													onclick={() => useHFModel(hfSelectedModel!, f)}
												>
													Use
												</button>
											</div>
										</div>
									{/each}
								</div>
							</div>
						{:else if hfProviderFilter === 'llama.cpp'}
							<div class="rounded border border-yellow-500/30 bg-yellow-500/10 p-3 text-sm">
								<div class="mb-1 font-medium text-yellow-600 dark:text-yellow-400">
									No GGUF files found
								</div>
								<div class="text-muted-foreground text-xs">
									This model doesn't contain GGUF quantized files. Use <strong>vLLM</strong> or
									<strong>SGLang</strong> provider instead, or find a GGUF quantized version (e.g., from
									TheBloke).
								</div>
							</div>
						{/if}

						<!-- Recommended provider -->
						<div class="text-muted-foreground bg-muted/30 rounded px-3 py-2 text-xs">
							{#if hfProviderFilter === 'llama.cpp'}
								Recommended: <strong>llama.cpp</strong>
							{:else if hfProviderFilter === 'tgi' || hfSelectedModel.tags?.includes('text-generation-inference')}
								Recommended: <strong>TGI</strong>
							{:else if hfProviderFilter === 'sglang'}
								Recommended: <strong>SGLang</strong>
							{:else if hfProviderFilter === 'tei' || hfProviderFilter === 'rerank'}
								Recommended: <strong>TEI</strong>
							{:else if hfProviderFilter === 'embedding'}
								Recommended: <strong>SGLang</strong> or <strong>vLLM</strong>
							{:else}
								Recommended: <strong>vLLM</strong> or <strong>SGLang</strong>
							{/if}
						</div>

						<div class="flex gap-2 pt-2">
							<button
								class="bg-primary text-primary-foreground hover:bg-primary/90 flex-1 rounded px-4 py-2"
								onclick={() => useHFModel(hfSelectedModel!)}
							>
								Use This Model
							</button>
							<a
								href="https://huggingface.co/{hfSelectedModel.id}"
								target="_blank"
								class="hover:bg-muted flex items-center gap-2 rounded border px-4 py-2"
							>
								<ExternalLink class="h-4 w-4" />
								View on HF
							</a>
						</div>
					</div>
				{:else}
					<div class="text-muted-foreground p-8 text-center">Select a model to view details</div>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Cache Tab -->
	{#if activeTab === 'cache'}
		<div class="bg-card rounded-lg border">
			<div class="bg-muted/50 flex flex-wrap items-center justify-between gap-2 border-b px-4 py-3">
				<h2 class="font-semibold">Cache Artifacts ({formatSize(totalCacheSize)})</h2>
				<div class="flex items-center gap-2">
					<input
						id="evictLimitMB"
						type="number"
						class="bg-background w-24 rounded border px-3 py-1.5 text-sm"
						placeholder="MB"
					/>
					<button
						class="hover:bg-muted rounded border px-3 py-1.5 text-sm"
						onclick={evictCacheToLimit}
					>
						Evict to Limit
					</button>
					<button
						class="rounded border border-red-500 px-3 py-1.5 text-sm text-red-500 hover:bg-red-500/10 disabled:opacity-50"
						onclick={clearAllCache}
						disabled={busy || (artifacts || []).length === 0}
					>
						<FontAwesomeIcon icon={faTrashCan} class="mr-1 h-3 w-3" /> Clear All
					</button>
					<button class="hover:bg-muted rounded border px-3 py-1.5 text-sm" onclick={loadCache}>
						Refresh
					</button>
				</div>
			</div>
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead class="bg-muted/30">
						<tr>
							<th class="px-4 py-2 text-left">Path</th>
							<th class="px-4 py-2 text-left">Size</th>
							<th class="px-4 py-2 text-left">Format</th>
							<th class="px-4 py-2 text-left">Root</th>
							<th class="px-4 py-2 text-left">Modified</th>
						</tr>
					</thead>
					<tbody class="divide-y">
						{#if (artifacts || []).length === 0}
							<tr
								><td class="text-muted-foreground px-4 py-8 text-center" colspan="5"
									>No cache artifacts</td
								></tr
							>
						{:else}
							{#each artifacts || [] as a}
								<tr class="hover:bg-muted/30">
									<td class="max-w-md px-4 py-2 break-all">{a.path}</td>
									<td class="px-4 py-2">{formatSize(a.size)}</td>
									<td class="px-4 py-2">{a.format}</td>
									<td class="text-muted-foreground px-4 py-2 text-xs">{a.root}</td>
									<td class="px-4 py-2">{formatDate(a.mod_time)}</td>
								</tr>
							{/each}
						{/if}
					</tbody>
				</table>
			</div>
		</div>
	{/if}

	<!-- TRT Engines Tab -->
	{#if activeTab === 'trt'}
		<div class="grid gap-4 lg:grid-cols-2">
			<!-- Convert Form -->
			<div class="bg-card space-y-4 rounded-lg border p-4">
				<h2 class="text-lg font-semibold">Convert to TensorRT</h2>
				<div class="grid gap-3">
					<label class="flex flex-col gap-1 text-sm">
						<span class="font-medium">HF Model ID</span>
						<input
							class="bg-background rounded border px-3 py-2"
							bind:value={trtForm.hf_model}
							placeholder="meta-llama/Llama-3.1-8B-Instruct"
						/>
					</label>
					<div class="grid grid-cols-2 gap-3">
						<label class="flex flex-col gap-1 text-sm">
							<span>Max Batch Size</span>
							<input
								type="number"
								min="1"
								class="bg-background rounded border px-3 py-2"
								bind:value={trtForm.max_batch_size}
							/>
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span>Max Input Len</span>
							<input
								type="number"
								min="1"
								class="bg-background rounded border px-3 py-2"
								bind:value={trtForm.max_input_len}
							/>
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span>Max Output Len</span>
							<input
								type="number"
								min="1"
								class="bg-background rounded border px-3 py-2"
								bind:value={trtForm.max_output_len}
							/>
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span>Dtype</span>
							<select class="bg-background rounded border px-3 py-2" bind:value={trtForm.dtype}>
								<option value="float16">float16</option>
								<option value="bfloat16">bfloat16</option>
								<option value="float32">float32</option>
							</select>
						</label>
					</div>
				</div>
				<button
					class="bg-primary text-primary-foreground rounded px-4 py-2 disabled:opacity-50"
					onclick={convertToTRT}
					disabled={busy}
				>
					Start Conversion
				</button>
			</div>

			<!-- Engines List -->
			<div class="bg-card rounded-lg border">
				<div class="bg-muted/50 border-b px-4 py-3">
					<h2 class="font-semibold">Cached TRT Engines</h2>
				</div>
				<div class="divide-y">
					{#if (trtEngines || []).length === 0}
						<div class="text-muted-foreground px-4 py-8 text-center">No TRT engines</div>
					{:else}
						{#each trtEngines || [] as e}
							<div class="px-4 py-3">
								<div class="flex items-start justify-between">
									<div>
										<div class="font-semibold">{e.model_id}</div>
										<div class="text-muted-foreground mt-1 text-xs">
											{formatSize(e.size_bytes)} · CUDA {e.cuda_version} · TRT {e.trt_version} · SM {e.gpu_sm}
										</div>
										<div class="text-muted-foreground text-xs">
											Created: {formatDate(e.created_at)}
										</div>
									</div>
									<div class="flex items-center gap-2">
										<span
											class={`rounded px-1.5 py-0.5 text-xs ${e.compatible ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300' : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'}`}
										>
											{e.compatible ? 'compatible' : 'needs reconvert'}
										</span>
										<button
											class="hover:bg-muted rounded border px-2 py-1 text-xs"
											onclick={() => deleteTRTEngine(e.model_id)}
										>
											Delete
										</button>
									</div>
								</div>
							</div>
						{/each}
					{/if}
				</div>
			</div>
		</div>
	{/if}

	<!-- Downloads Tab -->
	{#if activeTab === 'downloads'}
		<div class="bg-card rounded-lg border">
			<div class="bg-muted/50 flex items-center justify-between border-b px-4 py-3">
				<h2 class="font-semibold">Repository Downloads</h2>
				<button class="hover:bg-muted rounded border px-3 py-1 text-sm" onclick={loadRepoDownloads}>
					Refresh
				</button>
			</div>
			<div class="divide-y">
				{#if repoDownloads.length === 0}
					<div class="text-muted-foreground px-4 py-8 text-center">
						<p>No active downloads</p>
						<p class="mt-2 text-sm">
							Use "Download Local" button in the Models tab to start downloading a model.
						</p>
					</div>
				{:else}
					{#each repoDownloads as dl}
						<div class="px-4 py-4">
							<div class="mb-2 flex items-start justify-between">
								<div>
									<div class="font-semibold">{dl.model_id}</div>
									<div class="text-muted-foreground mt-1 text-xs">
										{dl.completed_files}/{dl.total_files} files · {formatSize(
											dl.downloaded_size
										)}/{formatSize(dl.total_size)}
									</div>
									{#if dl.local_path}
										<div class="text-muted-foreground mt-1 text-xs"><FontAwesomeIcon icon={faFolder} class="mr-0.5 h-3 w-3" /> {dl.local_path}</div>
									{/if}
								</div>
								<div class="flex items-center gap-2">
									{#if dl.status === 'downloading'}
										<button
											class="rounded border border-red-300 px-2 py-1 text-xs text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
											onclick={() => cancelRepoDownload(dl.model_id)}
											title="Cancel download"
										>
											<FontAwesomeIcon icon={faXmark} class="mr-0.5 h-3 w-3" /> Cancel
										</button>
									{:else if dl.status === 'completed' || dl.status === 'failed' || dl.status === 'cancelled'}
										<button
											class="rounded border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50 dark:hover:bg-gray-900/20"
											onclick={() => removeRepoDownload(dl.model_id)}
											title="Remove from list"
										>
											<FontAwesomeIcon icon={faXmark} class="h-3 w-3" />
										</button>
									{/if}
									<span
										class={`rounded px-2 py-1 text-xs font-medium ${
											dl.status === 'completed'
												? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
												: dl.status === 'downloading'
													? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
													: dl.status === 'failed'
														? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
														: dl.status === 'cancelled'
															? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300'
															: 'bg-gray-100 text-gray-700 dark:bg-gray-900/30 dark:text-gray-300'
										}`}
									>
										{#if dl.status === 'downloading'}
											<FontAwesomeIcon icon={faHourglass} class="mr-0.5 h-3 w-3" />
										{:else if dl.status === 'completed'}
											<FontAwesomeIcon icon={faCircleCheck} class="mr-0.5 h-3 w-3" />
										{:else if dl.status === 'failed'}
											<FontAwesomeIcon icon={faCircleXmark} class="mr-0.5 h-3 w-3" />
										{:else if dl.status === 'cancelled'}
											<FontAwesomeIcon icon={faBan} class="mr-0.5 h-3 w-3" />
										{/if}
										{dl.status}
									</span>
								</div>
							</div>
							<!-- Progress bar -->
							{#if dl.status === 'downloading'}
								<div class="bg-muted mt-2 h-2 w-full rounded-full">
									<div
										class="bg-primary h-2 rounded-full transition-all"
										style="width: {dl.progress}%"
									></div>
								</div>
								<div class="text-muted-foreground mt-1 text-right text-xs">
									{dl.progress.toFixed(1)}%
								</div>
							{/if}
							{#if dl.error}
								<div class="mt-2 text-xs text-red-500">{dl.error}</div>
							{/if}
						</div>
					{/each}
				{/if}
			</div>
		</div>
	{/if}
</div>

<!-- Edit Saved Model Modal -->
{#if editingSavedModel}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		tabindex="-1"
		role="dialog"
		aria-modal="true"
		onkeydown={(e) => e.key === 'Escape' && (editingSavedModel = null)}
	>
		<div class="bg-background w-full max-w-lg rounded-lg p-6 shadow-xl">
			<div class="mb-4 flex items-center justify-between">
				<h3 class="text-lg font-semibold">Edit: {editingSavedModel.alias}</h3>
				<button
					class="text-muted-foreground hover:text-foreground"
					onclick={() => (editingSavedModel = null)}><FontAwesomeIcon icon={faXmark} class="h-4 w-4" /></button
				>
			</div>

			<div class="text-muted-foreground mb-4 text-sm">
				Provider: <span class="text-foreground font-medium">{editingSavedModel.provider}</span>
			</div>

			<div class="max-h-[60vh] space-y-4 overflow-y-auto">
				<!-- Capabilities -->
				<div>
					<span class="mb-2 block text-sm font-medium">Capabilities</span>
					<div class="flex flex-wrap gap-2" role="group" aria-label="Capabilities">
						{#each capabilities as cap}
							<button
								type="button"
								class={`rounded border px-3 py-1 text-sm ${(editSavedForm.capabilities || []).includes(cap) ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'}`}
								onclick={() => toggleEditCapability(cap)}
							>
								{cap}
							</button>
						{/each}
					</div>
					<p class="text-muted-foreground mt-1 text-xs">
						Click chat → adds autocomplete, edit, apply. Click embeddings → adds rerank.
					</p>
				</div>

				<!-- Auto-start -->
				<div class="flex items-center gap-2">
					<input
						id="edit-auto-start"
						type="checkbox"
						class="rounded border"
						bind:checked={editSavedForm.auto_start}
					/>
					<label for="edit-auto-start" class="text-sm font-medium">Auto-start on server boot</label>
				</div>

				<!-- Common GPU Device -->
				<div>
					<label for="edit-gpu-device" class="mb-1 block text-sm font-medium">GPU Device</label>
					<input
						id="edit-gpu-device"
						type="text"
						class="bg-background w-full rounded border px-3 py-2"
						placeholder="e.g., 0 or 0,1"
						bind:value={editSavedForm.gpu_device}
					/>
					<p class="text-muted-foreground mt-1 text-xs">Comma-separated GPU indices</p>
				</div>

				<!-- Provider-specific settings -->
				{#if editingSavedModel.provider === 'vllm'}
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="edit-vllm-tp" class="mb-1 block text-sm font-medium"
								>Tensor Parallel</label
							>
							<input
								id="edit-vllm-tp"
								type="number"
								min="1"
								class="bg-background w-full rounded border px-3 py-2"
								bind:value={editSavedForm.vllm_tensor_parallel}
							/>
						</div>
						<div>
							<label for="edit-vllm-len" class="mb-1 block text-sm font-medium"
								>Max Model Length</label
							>
							<input
								id="edit-vllm-len"
								type="number"
								min="0"
								class="bg-background w-full rounded border px-3 py-2"
								placeholder="0 = auto"
								bind:value={editSavedForm.vllm_max_model_len}
							/>
						</div>
					</div>
					<div>
						<label for="edit-vllm-util" class="mb-1 block text-sm font-medium"
							>GPU Memory Utilization</label
						>
						<input
							id="edit-vllm-util"
							type="number"
							step="0.05"
							min="0.1"
							max="1.0"
							class="bg-background w-full rounded border px-3 py-2"
							bind:value={editSavedForm.vllm_gpu_utilization}
						/>
						<p class="text-muted-foreground mt-1 text-xs">0.1 - 1.0 (default: 0.9)</p>
					</div>
					<div>
						<label for="edit-vllm-extra" class="mb-1 block text-sm font-medium"
							>Extra Args</label
						>
						<input
							id="edit-vllm-extra"
							type="text"
							class="bg-background w-full rounded border px-3 py-2"
							placeholder="--enable-auto-tool-choice --tool-call-parser hermes"
							bind:value={editSavedForm.vllm_extra_args}
						/>
						<p class="text-muted-foreground mt-1 text-xs">Additional CLI flags for vLLM (e.g. tool calling support)</p>
					</div>
				{:else if editingSavedModel.provider === 'sglang'}
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="edit-sglang-tp" class="mb-1 block text-sm font-medium"
								>Tensor Parallel</label
							>
							<input
								id="edit-sglang-tp"
								type="number"
								min="1"
								class="bg-background w-full rounded border px-3 py-2"
								bind:value={editSavedForm.sglang_tensor_parallel}
							/>
						</div>
						<div>
							<label for="edit-sglang-mem" class="mb-1 block text-sm font-medium"
								>Memory Fraction</label
							>
							<input
								id="edit-sglang-mem"
								type="number"
								step="0.05"
								min="0.1"
								max="1.0"
								class="bg-background w-full rounded border px-3 py-2"
								bind:value={editSavedForm.sglang_mem_fraction}
							/>
						</div>
					</div>
				{:else if editingSavedModel.provider === 'tgi'}
					<div>
						<label for="edit-tgi-shards" class="mb-1 block text-sm font-medium">Num Shards</label>
						<input
							id="edit-tgi-shards"
							type="number"
							min="1"
							class="bg-background w-full rounded border px-3 py-2"
							bind:value={editSavedForm.tgi_num_shard}
						/>
						<p class="text-muted-foreground mt-1 text-xs">Number of GPUs to shard across</p>
					</div>
				{:else if editingSavedModel.provider === 'llama.cpp'}
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="edit-llama-main" class="mb-1 block text-sm font-medium">Main GPU</label>
							<input
								id="edit-llama-main"
								type="number"
								min="0"
								class="bg-background w-full rounded border px-3 py-2"
								bind:value={editSavedForm.llama_main_gpu}
							/>
						</div>
						<div>
							<label for="edit-llama-layers" class="mb-1 block text-sm font-medium"
								>N GPU Layers</label
							>
							<input
								id="edit-llama-layers"
								type="number"
								min="-1"
								class="bg-background w-full rounded border px-3 py-2"
								bind:value={editSavedForm.llama_n_gpu_layers}
							/>
							<p class="text-muted-foreground mt-1 text-xs">-1 = all layers</p>
						</div>
						<div>
							<label for="edit-llama-ctx" class="mb-1 block text-sm font-medium">Context Size</label
							>
							<input
								id="edit-llama-ctx"
								type="number"
								min="0"
								class="bg-background w-full rounded border px-3 py-2"
								placeholder="32768"
								bind:value={editSavedForm.llama_ctx_size}
							/>
							<p class="text-muted-foreground mt-1 text-xs">0 = default (2048)</p>
						</div>
						<div>
							<label for="edit-llama-parallel" class="mb-1 block text-sm font-medium"
								>N Parallel</label
							>
							<input
								id="edit-llama-parallel"
								type="number"
								min="0"
								class="bg-background w-full rounded border px-3 py-2"
								placeholder="4"
								bind:value={editSavedForm.llama_n_parallel}
							/>
							<p class="text-muted-foreground mt-1 text-xs">Concurrent request slots (0 = auto)</p>
						</div>
					</div>
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="edit-llama-flash" class="mb-1 block text-sm font-medium"
								>Flash Attention</label
							>
							<label class="flex items-center gap-2">
								<input
									id="edit-llama-flash"
									type="checkbox"
									class="h-4 w-4"
									bind:checked={editSavedForm.llama_flash_attn}
								/>
								<span class="text-muted-foreground text-sm">Enable for faster inference</span>
							</label>
						</div>
						<div>
							<label for="edit-llama-jinja" class="mb-1 block text-sm font-medium"
								>Jinja Templates</label
							>
							<label class="flex items-center gap-2">
								<input
									id="edit-llama-jinja"
									type="checkbox"
									class="h-4 w-4"
									bind:checked={editSavedForm.llama_jinja}
								/>
								<span class="text-muted-foreground text-sm">Enable Jinja template processing</span>
							</label>
						</div>
						<div>
							<label for="edit-llama-split" class="mb-1 block text-sm font-medium"
								>Tensor Split</label
							>
							<input
								id="edit-llama-split"
								type="text"
								class="bg-background w-full rounded border px-3 py-2"
								placeholder="e.g., 0.5,0.5"
								bind:value={editSavedForm.llama_tensor_split}
							/>
							<p class="text-muted-foreground mt-1 text-xs">
								Comma-separated split ratios for multi-GPU
							</p>
						</div>
					</div>
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="edit-llama-cache-reuse" class="mb-1 block text-sm font-medium"
								>Cache Reuse</label
							>
							<input
								id="edit-llama-cache-reuse"
								type="number"
								min="-1"
								class="bg-background w-full rounded border px-3 py-2"
								placeholder="0"
								bind:value={editSavedForm.llama_cache_reuse}
							/>
							<p class="text-muted-foreground mt-1 text-xs">
								0 = default (256), -1 = disable (for SWA models)
							</p>
						</div>
						<div>
							<label for="edit-llama-extra-args" class="mb-1 block text-sm font-medium"
								>Extra Args</label
							>
							<input
								id="edit-llama-extra-args"
								type="text"
								class="bg-background w-full rounded border px-3 py-2"
								placeholder="--no-mmap --verbose"
								bind:value={editSavedForm.llama_extra_args}
							/>
							<p class="text-muted-foreground mt-1 text-xs">
								Additional CLI flags for llama-server
							</p>
						</div>
					</div>
				{/if}
			</div>

			<div class="mt-6 flex justify-end gap-2 border-t pt-4">
				<button
					class="hover:bg-muted rounded border px-4 py-2"
					onclick={() => (editingSavedModel = null)}
				>
					Cancel
				</button>
				<button
					class="bg-primary text-primary-foreground hover:bg-primary/90 rounded px-4 py-2"
					onclick={saveEditedModel}
				>
					Save Changes
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Logs Modal (70-80% of screen) -->
{#if logsModalOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
		onclick={closeLogsModal}
		onkeydown={(e) => e.key === 'Escape' && closeLogsModal()}
		tabindex="-1"
		role="dialog"
		aria-modal="true"
	>
		<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
		<div
			class="bg-card flex flex-col rounded-lg border shadow-2xl"
			style="width: 80vw; height: 80vh; max-width: 1600px;"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
			role="document"
		>
			<!-- Header -->
			<div class="bg-muted/50 flex flex-shrink-0 items-center justify-between border-b px-4 py-3">
				<div class="flex items-center gap-3">
					<h2 class="text-lg font-semibold">{m.admin_models_logs_title()}: {logsModalAlias}</h2>
					{#if logsModalLoading}
						<Loader2 class="text-muted-foreground h-4 w-4 animate-spin" />
					{/if}
					<span class="text-muted-foreground bg-muted rounded px-2 py-0.5 text-xs"
						>{m.admin_models_logs_auto_refresh()}</span
					>
				</div>
				<div class="flex items-center gap-3">
					<label class="text-muted-foreground flex items-center gap-2 text-sm">
						<input type="checkbox" class="h-4 w-4" bind:checked={logsModalAutoScroll} />
						{m.admin_models_logs_auto_scroll()}
					</label>
					<button
						class="hover:bg-muted rounded border px-3 py-1 text-sm"
						onclick={fetchLogsForModal}
						title={m.common_refresh()}
					>
						{m.common_refresh()}
					</button>
					<button
						class="text-muted-foreground hover:text-foreground hover:bg-muted flex h-8 w-8 items-center justify-center rounded text-xl font-bold"
						onclick={closeLogsModal}
						title="Close"
					>
						<FontAwesomeIcon icon={faXmark} class="h-4 w-4" />
					</button>
				</div>
			</div>

			<!-- Logs Content -->
			<div
				class="flex-1 overflow-auto bg-black/95 p-4 font-mono text-sm text-gray-300"
				bind:this={logsContainer}
			>
				{#if logsModalContent}
					<pre class="leading-relaxed break-words whitespace-pre-wrap">{@html colorizeLogs(
							logsModalContent
						)}</pre>
				{:else if logsModalLoading}
					<div class="text-muted-foreground">{m.admin_models_logs_loading()}</div>
				{:else}
					<div class="text-muted-foreground">{m.admin_models_logs_no_logs()}</div>
				{/if}
			</div>
		</div>
	</div>
{/if}
