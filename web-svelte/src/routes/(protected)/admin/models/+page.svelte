<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { inferenceApi, type ModelInfo, type ArtifactInfo, type TRTEngine, type Provider, type Format, type Capability, type LoadRequest, type GPUDevice, type SavedModel, type RepoDownload } from '$lib/api/inference';
	import { api } from '$lib/api/client';
	import { downloadsApi } from '$lib/api/downloads';
	import { Search, Download, ExternalLink, Loader2 } from 'lucide-svelte';
	
	// GPU devices for selection
	let gpuDevices: GPUDevice[] = $state([]);
	let selectedGPUs: number[] = $state([]);

	const providers: Provider[] = ['vllm', 'sglang', 'tgi', 'tei', 'tensorrt-llm', 'llama.cpp'];
	const formats: Format[] = ['hf', 'gguf', 'trt', 'other'];
	const capabilities: Capability[] = ['chat', 'embeddings', 'vision'];

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
	
	// HuggingFace Browser state
	interface HFModel {
		id: string;
		author: string;
		modelId: string;
		downloads: number;
		likes: number;
		lastModified: string;
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
	type HFProviderFilter = 'all' | 'vllm' | 'sglang' | 'tgi' | 'llama.cpp' | 'embedding';
	const hfProviderFilters: {id: HFProviderFilter; label: string; description: string; icon: string}[] = [
		{ id: 'all', label: 'All LLMs', description: 'Text generation models', icon: '🔤' },
		{ id: 'vllm', label: 'vLLM', description: 'For vLLM inference', icon: '⚡' },
		{ id: 'sglang', label: 'SGLang', description: 'For SGLang inference', icon: '🚀' },
		{ id: 'tgi', label: 'TGI', description: 'For Text Generation Inference', icon: '🤗' },
		{ id: 'llama.cpp', label: 'llama.cpp (GGUF)', description: 'Quantized GGUF models', icon: '🦙' },
		{ id: 'embedding', label: 'Embeddings', description: 'Feature extraction & embeddings', icon: '📊' },
	];
	
	// Model size filters
	type HFSizeFilter = 'any' | 'tiny' | 'small' | 'medium' | 'large' | 'xl' | 'xxl';
	const hfSizeFilters: {id: HFSizeFilter; label: string; range: string; minB: number; maxB: number}[] = [
		{ id: 'any', label: 'Any', range: 'All sizes', minB: 0, maxB: Infinity },
		{ id: 'tiny', label: '< 3B', range: 'Tiny', minB: 0, maxB: 3 },
		{ id: 'small', label: '3-7B', range: 'Small', minB: 3, maxB: 7 },
		{ id: 'medium', label: '7-14B', range: 'Medium', minB: 7, maxB: 14 },
		{ id: 'large', label: '14-30B', range: 'Large', minB: 14, maxB: 30 },
		{ id: 'xl', label: '30-70B', range: 'XL', minB: 30, maxB: 70 },
		{ id: 'xxl', label: '70B+', range: 'XXL', minB: 70, maxB: Infinity },
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
	
	// Selected model for details panel
	let selectedModel = $state<ModelInfo | null>(null);
	let selectedLogs = $state('');
	let selectedMetrics = $state('');
	let selectedHealth = $state<{status: string; response_time_ms?: number; error?: string} | null>(null);
	let logsInterval: ReturnType<typeof setInterval> | null = null;

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
		llama_main_gpu: 0,
		llama_tensor_split: '',
		llama_n_gpu_layers: 0,
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
			selectedGPUs = selectedGPUs.filter(i => i !== index);
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
		if (tags.some(t => t.toLowerCase().includes('awq') || t.toLowerCase().includes('int4') || t.toLowerCase().includes('4bit'))) {
			quantization = 'INT4';
		} else if (tags.some(t => t.toLowerCase().includes('gptq') || t.toLowerCase().includes('int8') || t.toLowerCase().includes('8bit'))) {
			quantization = 'INT8';
		} else if (tags.some(t => t.toLowerCase().includes('gguf'))) {
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
		
		const modelSizeGB = (paramCount * bytesPerParam) / (1024 ** 3);
		
		// Calculate total available GPU memory (memory_mb is in megabytes)
		const selectedGPUMemoryGB = selectedGPUs.length > 0 
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
		let recommendedMemFraction = selectedGPUMemoryGB > 0 
			? Math.min(0.95, (totalNeededGB / selectedGPUMemoryGB) + 0.05)
			: 0.9;
		recommendedMemFraction = Math.round(recommendedMemFraction * 100) / 100;
		
		// Calculate minimum GPUs needed
		const singleGPUMemory = gpuDevices[0]?.memory_mb ? gpuDevices[0].memory_mb / 1024 : 24;
		const minGPUsNeeded = Math.ceil(totalNeededGB / (singleGPUMemory * 0.9));
		
		// Format params for display
		const paramsInB = paramCount / 1e9;
		const modelParams = paramsInB >= 1 ? `${paramsInB.toFixed(1)}B` : `${(paramCount / 1e6).toFixed(0)}M`;
		
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
			hfSearchResults = res.models || [];
			if (hfSearchResults.length === 0 && hfSearchQuery) {
				hfSearchError = `No models found for "${hfSearchQuery}"`;
			}
		} catch (e: any) {
			hfSearchError = e?.message || 'Failed to search HuggingFace';
			hfSearchResults = [];
		} finally {
			hfSearching = false;
		}
	}
	
	async function loadPopularHF() {
		hfSearching = true;
		try {
			const res = await downloadsApi.getPopularModels(hfProviderFilter);
			hfPopularModels = res.models || [];
		} catch (e: any) {
			console.error('Failed to load popular models:', e);
		} finally {
			hfSearching = false;
		}
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
			const res = await api.get<{siblings?: any[]}>(`/api/huggingface/models/${m.id}`);
			// Filter files based on provider filter
			const allFiles = res.siblings || [];
			if (hfProviderFilter === 'llama.cpp') {
				// Show only GGUF files for llama.cpp
				hfModelFiles = allFiles.filter((f: any) => f.rfilename?.endsWith('.gguf'));
			} else {
				// For HF models (vLLM/SGLang/TGI) show safetensors, bin, and config files
				hfModelFiles = allFiles.filter((f: any) => 
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
			form.capabilities = ['chat'];
		} else {
			form.hf_file = '';
			form.format = 'hf';
			
			// Set provider based on filter selection
			switch (hfProviderFilter) {
				case 'llama.cpp':
					// If browsing GGUF but selected non-GGUF file, still suggest llama.cpp
					form.provider = 'llama.cpp';
					form.format = 'gguf';
					form.capabilities = ['chat'];
					break;
				case 'vllm':
					form.provider = 'vllm';
					form.capabilities = ['chat'];
					break;
				case 'sglang':
					form.provider = 'sglang';
					form.capabilities = ['chat'];
					break;
				case 'tgi':
					form.provider = 'tgi';
					form.capabilities = ['chat'];
					break;
				case 'embedding':
					form.provider = 'tei'; // TEI (Text Embeddings Inference) is best for embeddings
					form.capabilities = ['embeddings'];
					break;
				default:
					form.provider = 'vllm'; // Default
					form.capabilities = ['chat'];
			}
		}
		
		form.alias = m.id.split('/').pop()?.toLowerCase().replace(/[^a-z0-9]/g, '-') || 'model';
		
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
		
		const filter = hfSizeFilters.find(f => f.id === hfSizeFilter);
		if (!filter) return models;
		
		return models.filter(m => {
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
		setTimeout(() => { msg = ''; }, 5000);
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
				llama_main_gpu: form.llama_main_gpu,
				llama_tensor_split: form.llama_tensor_split,
				llama_n_gpu_layers: form.llama_n_gpu_layers,
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
					await inferenceApi.downloadRepository(form.hf_repo);
					showMsg('Скачивание модели началось. Смотрите вкладку Downloads.', 'success');
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

	async function stopModel(alias: string) {
		try {
			await inferenceApi.stop(alias);
			showMsg(`Модель ${alias} остановлена`, 'success');
			await loadModels();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка остановки', 'error');
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
				llama_main_gpu: m.llama_main_gpu,
				llama_tensor_split: m.llama_tensor_split,
				llama_n_gpu_layers: m.llama_n_gpu_layers,
				sglang_tensor_parallel: m.sglang_tensor_parallel,
				sglang_mem_fraction: m.sglang_mem_fraction,
				tgi_num_shard: m.tgi_num_shard,
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
				llama_main_gpu: saved.llama_main_gpu,
				llama_n_gpu_layers: saved.llama_n_gpu_layers,
				sglang_tensor_parallel: saved.sglang_tensor_parallel,
				sglang_mem_fraction: saved.sglang_mem_fraction,
				tgi_num_shard: saved.tgi_num_shard,
			};
			await inferenceApi.load(req);
			showMsg(`Модель ${saved.alias} запускается`, 'success');
			await loadModels();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка загрузки', 'error');
		}
	}

	async function deleteModelArtifacts(alias: string) {
		if (!confirm(`Удалить файлы модели ${alias}?`)) return;
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
		await Promise.all([
			fetchHealth(m.alias),
			fetchLogs(m.alias),
			fetchMetrics(m.alias)
		]);
		
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
		if (!confirm(`Are you sure you want to clear ALL cache (${formatSize(totalCacheSize)})?\n\nThis will delete all downloaded models and cannot be undone.`)) {
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
		if (!confirm(`Удалить TRT engine для ${modelId}?`)) return;
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
			case 'running': return 'text-green-600 dark:text-green-400';
			case 'starting': case 'downloading': case 'preparing': return 'text-yellow-600 dark:text-yellow-400';
			case 'stopped': case 'idle': return 'text-gray-500';
			case 'error': case 'failed': return 'text-red-600 dark:text-red-400';
			default: return 'text-gray-600';
		}
	}

	function toggleCapability(cap: Capability) {
		const caps = form.capabilities || [];
		if (caps.includes(cap)) {
			form.capabilities = caps.filter(c => c !== cap);
		} else {
			form.capabilities = [...caps, cap];
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
			<h1 class="text-2xl font-bold">Inference Models</h1>
			<p class="text-sm text-muted-foreground">Multi-provider: vLLM, SGLang, TGI, TensorRT-LLM, llama.cpp</p>
		</div>
		<button class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90" onclick={refreshAll}>
			Refresh
		</button>
	</div>

	<!-- Message -->
	{#if msg}
		<div class={`px-4 py-3 rounded-md border text-sm ${msgType === 'error' ? 'bg-red-50 dark:bg-red-900/20 border-red-200 text-red-700 dark:text-red-300' : msgType === 'success' ? 'bg-green-50 dark:bg-green-900/20 border-green-200 text-green-700 dark:text-green-300' : 'bg-muted/50'}`}>
			{msg}
		</div>
	{/if}

	<!-- Tabs -->
	<div class="flex gap-2 border-b">
		<button class={`px-4 py-2 -mb-px ${activeTab === 'models' ? 'border-b-2 border-primary font-semibold' : 'text-muted-foreground'}`} onclick={() => activeTab = 'models'}>
			Models ({(models || []).length})
		</button>
		<button class={`px-4 py-2 -mb-px ${activeTab === 'hf' ? 'border-b-2 border-primary font-semibold' : 'text-muted-foreground'}`} onclick={() => activeTab = 'hf'}>
			🤗 HuggingFace
		</button>
		<button class={`px-4 py-2 -mb-px ${activeTab === 'cache' ? 'border-b-2 border-primary font-semibold' : 'text-muted-foreground'}`} onclick={() => activeTab = 'cache'}>
			Cache ({formatSize(totalCacheSize)})
		</button>
		<button class={`px-4 py-2 -mb-px ${activeTab === 'trt' ? 'border-b-2 border-primary font-semibold' : 'text-muted-foreground'}`} onclick={() => activeTab = 'trt'}>
			TRT Engines ({(trtEngines || []).length})
		</button>
		<button class={`px-4 py-2 -mb-px ${activeTab === 'downloads' ? 'border-b-2 border-primary font-semibold' : 'text-muted-foreground'}`} onclick={() => activeTab = 'downloads'}>
			📥 Downloads ({repoDownloads.filter(d => d.status === 'downloading').length || ''})
		</button>
	</div>

	<!-- Models Tab -->
	{#if activeTab === 'models'}
		<div class="grid gap-4 lg:grid-cols-[1fr_400px]">
			<!-- Left: Model list + Load form -->
			<div class="flex flex-col gap-4">
				<!-- Load Form -->
				<div class="border rounded-lg p-4 space-y-4 bg-card">
					<h2 class="text-lg font-semibold">Load Model</h2>
					<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
						<label class="flex flex-col gap-1 text-sm">
							<span class="font-medium">Alias</span>
							<input class="border rounded px-3 py-2 bg-background" bind:value={form.alias} placeholder="llama-3-8b" />
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span class="font-medium">Provider</span>
							<select class="border rounded px-3 py-2 bg-background" bind:value={form.provider} onchange={updateTensorParallelForProvider}>
								{#each providers as p}<option value={p}>{p}</option>{/each}
							</select>
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span class="font-medium">Format</span>
							<select class="border rounded px-3 py-2 bg-background" bind:value={form.format}>
								{#each formats as f}<option value={f}>{f}</option>{/each}
							</select>
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span class="font-medium">HF Repo</span>
							<input 
								class="border rounded px-3 py-2 bg-background" 
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
							<input class="border rounded px-3 py-2 bg-background" bind:value={form.hf_file} placeholder="model-Q4_K_M.gguf" />
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span class="font-medium">GGUF URL</span>
							<input class="border rounded px-3 py-2 bg-background" bind:value={form.gguf_url} placeholder="https://..." />
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
										class="px-3 py-2 rounded border text-sm flex flex-col items-start {selectedGPUs.includes(gpu.index) ? 'bg-primary text-primary-foreground border-primary' : 'hover:bg-muted'}"
										onclick={() => toggleGPU(gpu.index)}
									>
										<span class="font-medium">GPU {gpu.index}: {gpu.name}</span>
										<span class="text-xs opacity-75">{Math.round(gpu.memory_mb / 1024)} GB ({Math.round(gpu.memory_free_mb / 1024)} GB free)</span>
									</button>
								{/each}
							</div>
							<span class="text-xs text-muted-foreground">
								{selectedGPUs.length === 0 ? 'No GPU selected = use all GPUs' : `Selected: GPU ${selectedGPUs.join(', ')}`}
							</span>
						{:else}
							<input 
								class="border rounded px-3 py-2 bg-background" 
								bind:value={form.gpu_device} 
								placeholder="0, 1, or 0,1 (empty = all GPUs)"
							/>
							<span class="text-xs text-muted-foreground">GPU list unavailable. Enter index manually.</span>
						{/if}
					</div>

					<!-- Capabilities -->
					<div class="flex flex-col gap-2 text-sm">
						<span class="font-medium">Capabilities</span>
						<div class="flex gap-2">
							{#each capabilities as c}
								<button type="button" class={`px-3 py-1 rounded border text-sm ${(form.capabilities || []).includes(c) ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'}`} onclick={() => toggleCapability(c)}>
									{c}
								</button>
							{/each}
						</div>
					</div>

					<!-- Memory Recommendations -->
					{#if memoryRecommendation}
						<div class="p-3 rounded-lg border {memoryRecommendation.willFit ? 'bg-green-500/10 border-green-500/30' : 'bg-yellow-500/10 border-yellow-500/30'}">
							<div class="flex items-start justify-between gap-4">
								<div class="flex-1">
									<div class="text-sm font-medium mb-1">
										{memoryRecommendation.willFit ? '✅' : '⚠️'} Memory Recommendation
									</div>
									<div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs text-muted-foreground">
										<div>
											<span class="text-foreground font-medium">{memoryRecommendation.modelParams}</span>
											<span class="block">{memoryRecommendation.quantization}</span>
										</div>
										<div>
											<span class="text-foreground font-medium">~{memoryRecommendation.modelSizeGB} GB</span>
											<span class="block">Model Size</span>
										</div>
										<div>
											<span class="text-foreground font-medium">{memoryRecommendation.totalGPUMemoryGB} GB</span>
											<span class="block">GPU Available</span>
										</div>
										<div>
											<span class="text-foreground font-medium">{memoryRecommendation.recommendedMemFraction}</span>
											<span class="block">Rec. Mem Frac</span>
										</div>
									</div>
									{#if memoryRecommendation.warning}
										<p class="text-xs text-yellow-600 dark:text-yellow-400 mt-2">{memoryRecommendation.warning}</p>
									{/if}
								</div>
								<button
									type="button"
									class="text-xs px-2 py-1 rounded bg-primary/10 hover:bg-primary/20 text-primary"
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
						<div class="grid gap-3 sm:grid-cols-3 pt-2 border-t">
							<label class="flex flex-col gap-1 text-sm">
								<span>Tensor Parallel</span>
								<input type="number" min="0" class="border rounded px-3 py-2 bg-background" bind:value={form.vllm_tensor_parallel} />
							</label>
							<label class="flex flex-col gap-1 text-sm">
								<span>Max Model Len</span>
								<input type="number" min="0" class="border rounded px-3 py-2 bg-background" bind:value={form.vllm_max_model_len} />
							</label>
							<label class="flex flex-col gap-1 text-sm">
								<span>GPU Utilization</span>
								<input type="number" step="0.01" min="0" max="1" class="border rounded px-3 py-2 bg-background" bind:value={form.vllm_gpu_utilization} />
							</label>
						</div>
					{:else if form.provider === 'llama.cpp'}
						<div class="grid gap-3 sm:grid-cols-3 pt-2 border-t">
							<label class="flex flex-col gap-1 text-sm">
								<span>n_gpu_layers</span>
								<input type="number" min="0" class="border rounded px-3 py-2 bg-background" bind:value={form.llama_n_gpu_layers} />
							</label>
							<label class="flex flex-col gap-1 text-sm">
								<span>main_gpu</span>
								<input type="number" min="0" class="border rounded px-3 py-2 bg-background" bind:value={form.llama_main_gpu} />
							</label>
							<label class="flex flex-col gap-1 text-sm">
								<span>tensor_split</span>
								<input class="border rounded px-3 py-2 bg-background" bind:value={form.llama_tensor_split} placeholder="0.5,0.5" />
							</label>
						</div>
					{:else if form.provider === 'sglang'}
						<div class="grid gap-3 sm:grid-cols-2 pt-2 border-t">
							<label class="flex flex-col gap-1 text-sm">
								<span>Tensor Parallel</span>
								<input type="number" min="0" class="border rounded px-3 py-2 bg-background" bind:value={form.sglang_tensor_parallel} />
							</label>
							<label class="flex flex-col gap-1 text-sm">
								<span>Mem Fraction</span>
								<input type="number" step="0.01" min="0" max="1" class="border rounded px-3 py-2 bg-background" bind:value={form.sglang_mem_fraction} />
							</label>
						</div>
					{:else if form.provider === 'tgi'}
						<div class="grid gap-3 sm:grid-cols-1 pt-2 border-t">
							<label class="flex flex-col gap-1 text-sm">
								<span>Num Shards</span>
								<input type="number" min="1" class="border rounded px-3 py-2 bg-background" bind:value={form.tgi_num_shard} />
							</label>
						</div>
					{/if}

					<div class="flex gap-2 pt-2">
						<button class="px-4 py-2 rounded bg-primary text-primary-foreground disabled:opacity-50" onclick={() => submit(true)} disabled={busy}>
							Load & Start
						</button>
						<button class="px-4 py-2 rounded border hover:bg-muted disabled:opacity-50" onclick={() => submit(false)} disabled={busy}>
							Download Local
						</button>
					</div>
				</div>

				<!-- Saved Models (persisted configs) -->
				{#if savedModels.length > 0}
					<div class="border rounded-lg overflow-hidden bg-card">
						<div class="px-4 py-3 border-b bg-muted/50">
							<h2 class="font-semibold">Saved Models <span class="text-xs text-muted-foreground font-normal">(click to load)</span></h2>
						</div>
						<div class="divide-y">
							{#each savedModels as saved}
								{@const isRunning = models.some(m => m.alias === saved.alias)}
								<div class="px-4 py-3 hover:bg-muted/30 flex items-center gap-4">
									<div class="flex-1 min-w-0">
										<div class="flex items-center gap-2">
											<span class="font-medium">{saved.alias}</span>
											{#if saved.auto_start}
												<span class="text-xs px-1.5 py-0.5 rounded bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300">auto-start</span>
											{/if}
											{#if isRunning}
												<span class="text-xs px-1.5 py-0.5 rounded bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300">running</span>
											{/if}
										</div>
										<div class="text-xs text-muted-foreground mt-1">
											{saved.provider} · {saved.format} · {saved.hf_repo || saved.gguf_url || 'local'}
										</div>
									</div>
									<div class="flex gap-1 flex-shrink-0">
										{#if !isRunning}
											<button class="px-2 py-1 text-xs rounded border bg-primary/10 text-primary hover:bg-primary/20" onclick={() => loadSavedModelConfig(saved)}>Load</button>
										{/if}
										<button 
											class="px-2 py-1 text-xs rounded border hover:bg-muted" 
											onclick={() => toggleAutoStart(saved)}
											title={saved.auto_start ? 'Disable auto-start' : 'Enable auto-start'}
										>
											{saved.auto_start ? '⏸' : '▶'}
										</button>
										<button class="px-2 py-1 text-xs rounded border text-red-500 hover:bg-red-500/10" onclick={() => deleteSavedModel(saved.alias)}>✕</button>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Running Models List -->
				<div class="border rounded-lg overflow-hidden bg-card">
					<div class="px-4 py-3 border-b bg-muted/50 flex items-center justify-between">
						<h2 class="font-semibold">Running Models</h2>
					</div>
					<div class="divide-y">
						{#if (models || []).length === 0}
							<div class="px-4 py-8 text-center text-muted-foreground">No models loaded</div>
						{:else}
							{#each (models || []) as m}
								{@const isSaved = savedModels.some(s => s.alias === m.alias)}
								<div class="px-4 py-3 hover:bg-muted/30 cursor-pointer flex items-start gap-4" onclick={() => selectModel(m)}>
									<div class="flex-1 min-w-0">
										<div class="flex items-center gap-2">
											<span class="font-semibold">{m.alias}</span>
											{#if m.pinned}
												<span class="text-xs px-1.5 py-0.5 rounded bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300">pinned</span>
											{/if}
											{#if isSaved}
												<span class="text-xs px-1.5 py-0.5 rounded bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300">saved</span>
											{/if}
											<span class={`text-sm font-medium ${statusColor(m.status)}`}>{m.status}</span>
										</div>
										<div class="text-xs text-muted-foreground mt-1">
											{m.provider} · {m.format} · {(m.capabilities || []).join(', ') || 'chat'}
										</div>
										{#if m.endpoint}
											<div class="text-xs text-muted-foreground truncate">{m.endpoint}</div>
										{/if}
										{#if m.last_error}
											<div class="text-xs text-red-500 mt-1">{m.last_error}</div>
										{/if}
									</div>
									<div class="flex gap-1 flex-shrink-0">
										{#if m.status === 'running' || m.status === 'starting'}
											<button class="px-2 py-1 text-xs rounded border hover:bg-muted" onclick={(e) => { e.stopPropagation(); stopModel(m.alias); }}>Stop</button>
										{:else}
											<button class="px-2 py-1 text-xs rounded border bg-green-500/10 text-green-600 hover:bg-green-500/20" onclick={(e) => { e.stopPropagation(); startModel(m); }}>Start</button>
										{/if}
										{#if !isSaved}
											<button class="px-2 py-1 text-xs rounded border bg-blue-500/10 text-blue-600 hover:bg-blue-500/20" onclick={(e) => { e.stopPropagation(); saveModelConfig(m.alias); }} title="Save config for restart">Save</button>
										{/if}
										<button class="px-2 py-1 text-xs rounded border hover:bg-muted" onclick={(e) => { e.stopPropagation(); evictModel(m.alias); }}>Evict</button>
										<button class="px-2 py-1 text-xs rounded border hover:bg-muted" onclick={(e) => { e.stopPropagation(); togglePin(m); }}>
											{m.pinned ? 'Unpin' : 'Pin'}
										</button>
									</div>
								</div>
							{/each}
						{/if}
					</div>
				</div>
			</div>

			<!-- Right: Details Panel -->
			<div class="border rounded-lg bg-card h-fit sticky top-4">
				{#if selectedModel}
					<div class="px-4 py-3 border-b bg-muted/50 flex items-center justify-between">
						<h2 class="font-semibold">{selectedModel.alias}</h2>
						<button class="text-muted-foreground hover:text-foreground" onclick={closeDetails}>✕</button>
					</div>
					<div class="p-4 space-y-4">
						<!-- Health -->
						<div>
							<h3 class="text-sm font-medium mb-2">Health</h3>
							{#if selectedHealth}
								<div class="flex items-center gap-2">
									<span class={`w-2 h-2 rounded-full ${selectedHealth.status === 'healthy' ? 'bg-green-500' : selectedHealth.status === 'unhealthy' ? 'bg-red-500' : 'bg-yellow-500'}`}></span>
									<span class="text-sm">{selectedHealth.status}</span>
									{#if selectedHealth.response_time_ms}
										<span class="text-xs text-muted-foreground">({selectedHealth.response_time_ms}ms)</span>
									{/if}
								</div>
								{#if selectedHealth.error}
									<div class="text-xs text-red-500 mt-1">{selectedHealth.error}</div>
								{/if}
							{:else}
								<div class="text-sm text-muted-foreground">Loading...</div>
							{/if}
						</div>

						<!-- Info -->
						<div>
							<h3 class="text-sm font-medium mb-2">Details</h3>
							<dl class="text-sm space-y-1">
								<div class="flex justify-between"><dt class="text-muted-foreground">Provider</dt><dd>{selectedModel.provider}</dd></div>
								<div class="flex justify-between"><dt class="text-muted-foreground">Format</dt><dd>{selectedModel.format}</dd></div>
								<div class="flex justify-between"><dt class="text-muted-foreground">Status</dt><dd class={statusColor(selectedModel.status)}>{selectedModel.status}</dd></div>
								{#if selectedModel.endpoint}
									<div class="flex justify-between"><dt class="text-muted-foreground">Endpoint</dt><dd class="truncate max-w-[200px]">{selectedModel.endpoint}</dd></div>
								{/if}
								{#if selectedModel.last_used}
									<div class="flex justify-between"><dt class="text-muted-foreground">Last Used</dt><dd>{formatDate(selectedModel.last_used)}</dd></div>
								{/if}
							</dl>
						</div>

						<!-- Logs -->
						<div>
							<h3 class="text-sm font-medium mb-2">Logs (last 200 lines)</h3>
							<pre class="text-xs bg-muted/50 p-2 rounded overflow-auto max-h-48 whitespace-pre-wrap">{selectedLogs || 'No logs'}</pre>
						</div>

						<!-- Metrics -->
						{#if selectedMetrics}
							<div>
								<h3 class="text-sm font-medium mb-2">Metrics</h3>
								<pre class="text-xs bg-muted/50 p-2 rounded overflow-auto max-h-32 whitespace-pre-wrap">{selectedMetrics}</pre>
							</div>
						{/if}

						<!-- Actions -->
						<div class="flex gap-2 pt-2 border-t">
							<button class="px-3 py-1.5 text-sm rounded border hover:bg-muted" onclick={() => deleteModelArtifacts(selectedModel!.alias)}>
								Delete Artifacts
							</button>
							<button class="px-3 py-1.5 text-sm rounded border hover:bg-muted" onclick={() => fetchLogs(selectedModel!.alias)}>
								Refresh Logs
							</button>
						</div>
					</div>
				{:else}
					<div class="p-8 text-center text-muted-foreground">
						Select a model to view details
					</div>
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
							class="px-3 py-1.5 text-sm rounded-md border transition-colors flex items-center gap-1.5 {hfProviderFilter === pf.id ? 'bg-primary text-primary-foreground border-primary' : 'hover:bg-muted'}"
							onclick={() => { hfProviderFilter = pf.id; hfSearchQuery = ''; hfSearchResults = []; hfSearchPerformed = false; hfSearchError = ''; }}
							title={pf.description}
						>
							<span>{pf.icon}</span>
							{pf.label}
						</button>
					{/each}
				</div>
				
				<!-- Size filter selector -->
				<div class="flex items-center gap-2 flex-wrap">
					<span class="text-xs text-muted-foreground">Size:</span>
					{#each hfSizeFilters as sf}
						<button 
							class="px-2 py-1 text-xs rounded border transition-colors {hfSizeFilter === sf.id ? 'bg-primary text-primary-foreground border-primary' : 'hover:bg-muted'}"
							onclick={() => { hfSizeFilter = sf.id; }}
							title={sf.range}
						>
							{sf.label}
						</button>
					{/each}
				</div>
				
				<!-- Search -->
				<div class="flex gap-2">
					<div class="relative flex-1">
						<Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
						<input 
							type="text" 
							class="w-full border rounded-md pl-10 pr-4 py-2 bg-background"
							placeholder="Search {hfProviderFilters.find(c => c.id === hfProviderFilter)?.label || 'models'}..."
							bind:value={hfSearchQuery}
							onkeydown={(e) => e.key === 'Enter' && searchHF()}
						/>
					</div>
					<button 
						class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 flex items-center gap-2 disabled:opacity-50"
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
				<div class="text-xs text-muted-foreground bg-muted/50 rounded px-3 py-2">
					{#if hfProviderFilter === 'llama.cpp'}
						🦙 GGUF quantized models for <strong>llama.cpp</strong> — efficient CPU/GPU inference
					{:else if hfProviderFilter === 'vllm'}
						⚡ Models for <strong>vLLM</strong> — high-throughput production serving
					{:else if hfProviderFilter === 'sglang'}
						🚀 Models for <strong>SGLang</strong> — fast inference with RadixAttention
					{:else if hfProviderFilter === 'tgi'}
						🤗 Models for <strong>TGI</strong> — HuggingFace Text Generation Inference
					{:else if hfProviderFilter === 'embedding'}
						📊 Embedding models for <strong>vLLM/SGLang</strong> — feature extraction & RAG
					{:else}
						💡 Filter by provider to find compatible models
					{/if}
				</div>
				
				<!-- Results -->
				<div class="border rounded-lg bg-card">
					<div class="px-4 py-3 border-b bg-muted/50">
						<h2 class="font-semibold">
							{#if hfSearchResults.length > 0}
								{@const filtered = getFilteredHFModels(hfSearchResults)}
								Search Results ({filtered.length}{hfSizeFilter !== 'any' ? ` of ${hfSearchResults.length}` : ''})
							{:else}
								{@const filtered = getFilteredHFModels(hfPopularModels)}
								Popular {hfProviderFilters.find(c => c.id === hfProviderFilter)?.label || ''} Models ({filtered.length}{hfSizeFilter !== 'any' ? ` of ${hfPopularModels.length}` : ''})
							{/if}
						</h2>
					</div>
					<div class="divide-y max-h-[600px] overflow-y-auto">
						{#each getFilteredHFModels(hfSearchResults.length > 0 ? hfSearchResults : hfPopularModels) as m}
							<div 
								class="px-4 py-3 hover:bg-muted/30 cursor-pointer flex items-start gap-3 {hfSelectedModel?.id === m.id ? 'bg-primary/10' : ''}"
								onclick={() => selectHFModel(m)}
							>
								<div class="flex-1 min-w-0">
									<div class="font-medium truncate">{m.id}</div>
									<div class="text-xs text-muted-foreground flex flex-wrap gap-2 mt-1">
										<span>⬇️ {formatNumber(m.downloads || 0)}</span>
										<span>❤️ {formatNumber(m.likes || 0)}</span>
										{#if extractModelSizeB(m)}
											<span class="px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-600 dark:text-blue-400 text-xs font-medium">{extractModelSizeB(m)}B</span>
										{/if}
										{#if m.pipeline_tag}
											<span class="px-1.5 py-0.5 rounded bg-muted text-xs">{m.pipeline_tag}</span>
										{/if}
									</div>
									{#if m.tags?.length}
										<div class="flex flex-wrap gap-1 mt-1">
											{#each m.tags.slice(0, 5) as tag}
												<span class="px-1.5 py-0.5 rounded bg-muted/50 text-xs">{tag}</span>
											{/each}
										</div>
									{/if}
								</div>
								<a 
									href="https://huggingface.co/{m.id}" 
									target="_blank" 
									class="p-1 hover:bg-muted rounded"
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
										⚠️ {hfSearchError}
									</div>
									<div class="text-xs mt-2">Try a different search term or check model name spelling</div>
								{:else if hfSearchPerformed && hfSearchResults.length === 0}
									<div>No models found</div>
									<div class="text-xs mt-2">Try a different search term</div>
								{:else}
									Search for models or wait for popular models to load
								{/if}
							</div>
						{/each}
					</div>
				</div>
			</div>
			
			<!-- Right: Selected model details -->
			<div class="border rounded-lg bg-card">
				<div class="px-4 py-3 border-b bg-muted/50">
					<h2 class="font-semibold">Model Details</h2>
				</div>
				{#if hfSelectedModel}
					<div class="p-4 space-y-4">
						<div>
							<div class="text-sm text-muted-foreground">Model ID</div>
							<div class="font-mono text-sm break-all">{hfSelectedModel.id}</div>
						</div>
						<div>
							<div class="text-sm text-muted-foreground">Author</div>
							<div>{hfSelectedModel.author || hfSelectedModel.id.split('/')[0]}</div>
						</div>
						<div class="flex gap-4">
							<div>
								<div class="text-sm text-muted-foreground">Downloads</div>
								<div>{formatNumber(hfSelectedModel.downloads || 0)}</div>
							</div>
							<div>
								<div class="text-sm text-muted-foreground">Likes</div>
								<div>{formatNumber(hfSelectedModel.likes || 0)}</div>
							</div>
						</div>
						
						<!-- Files (GGUF/safetensors) -->
						{#if hfModelFiles.length > 0}
							<div>
								<div class="text-sm text-muted-foreground mb-2">Available Files</div>
								<div class="space-y-1 max-h-64 overflow-y-auto">
									{#each hfModelFiles as f}
										<div class="flex items-center justify-between gap-2 p-2 rounded bg-muted/30 text-sm">
											<span class="truncate flex-1" title={f.rfilename}>{f.rfilename}</span>
											<div class="flex items-center gap-1">
												{#if f.size}
													<span class="text-xs text-muted-foreground">{formatSize(f.size)}</span>
												{/if}
												<button 
													class="px-2 py-1 text-xs rounded bg-primary text-primary-foreground hover:bg-primary/90"
													onclick={() => useHFModel(hfSelectedModel!, f)}
												>
													Use
												</button>
											</div>
										</div>
									{/each}
								</div>
							</div>
						{/if}
						
						<!-- Recommended provider -->
						<div class="text-xs text-muted-foreground bg-muted/30 rounded px-3 py-2">
							{#if hfProviderFilter === 'llama.cpp'}
								Recommended: <strong>llama.cpp</strong>
							{:else if hfProviderFilter === 'tgi' || hfSelectedModel.tags?.includes('text-generation-inference')}
								Recommended: <strong>TGI</strong>
							{:else if hfProviderFilter === 'sglang'}
								Recommended: <strong>SGLang</strong>
							{:else if hfProviderFilter === 'embedding'}
								Recommended: <strong>SGLang</strong> or <strong>vLLM</strong>
							{:else}
								Recommended: <strong>vLLM</strong> or <strong>SGLang</strong>
							{/if}
						</div>
						
						<div class="flex gap-2 pt-2">
							<button 
								class="flex-1 px-4 py-2 rounded bg-primary text-primary-foreground hover:bg-primary/90"
								onclick={() => useHFModel(hfSelectedModel!)}
							>
								Use This Model
							</button>
							<a 
								href="https://huggingface.co/{hfSelectedModel.id}" 
								target="_blank"
								class="px-4 py-2 rounded border hover:bg-muted flex items-center gap-2"
							>
								<ExternalLink class="h-4 w-4" />
								View on HF
							</a>
						</div>
					</div>
				{:else}
					<div class="p-8 text-center text-muted-foreground">
						Select a model to view details
					</div>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Cache Tab -->
	{#if activeTab === 'cache'}
		<div class="border rounded-lg bg-card">
			<div class="px-4 py-3 border-b bg-muted/50 flex items-center justify-between flex-wrap gap-2">
				<h2 class="font-semibold">Cache Artifacts ({formatSize(totalCacheSize)})</h2>
				<div class="flex items-center gap-2">
					<input id="evictLimitMB" type="number" class="border rounded px-3 py-1.5 w-24 text-sm bg-background" placeholder="MB" />
					<button class="px-3 py-1.5 text-sm rounded border hover:bg-muted" onclick={evictCacheToLimit}>
						Evict to Limit
					</button>
					<button 
						class="px-3 py-1.5 text-sm rounded border border-red-500 text-red-500 hover:bg-red-500/10 disabled:opacity-50"
						onclick={clearAllCache}
						disabled={busy || (artifacts || []).length === 0}
					>
						🗑️ Clear All
					</button>
					<button class="px-3 py-1.5 text-sm rounded border hover:bg-muted" onclick={loadCache}>
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
							<tr><td class="px-4 py-8 text-center text-muted-foreground" colspan="5">No cache artifacts</td></tr>
						{:else}
							{#each (artifacts || []) as a}
								<tr class="hover:bg-muted/30">
									<td class="px-4 py-2 break-all max-w-md">{a.path}</td>
									<td class="px-4 py-2">{formatSize(a.size)}</td>
									<td class="px-4 py-2">{a.format}</td>
									<td class="px-4 py-2 text-xs text-muted-foreground">{a.root}</td>
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
			<div class="border rounded-lg p-4 space-y-4 bg-card">
				<h2 class="text-lg font-semibold">Convert to TensorRT</h2>
				<div class="grid gap-3">
					<label class="flex flex-col gap-1 text-sm">
						<span class="font-medium">HF Model ID</span>
						<input class="border rounded px-3 py-2 bg-background" bind:value={trtForm.hf_model} placeholder="meta-llama/Llama-3.1-8B-Instruct" />
					</label>
					<div class="grid grid-cols-2 gap-3">
						<label class="flex flex-col gap-1 text-sm">
							<span>Max Batch Size</span>
							<input type="number" min="1" class="border rounded px-3 py-2 bg-background" bind:value={trtForm.max_batch_size} />
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span>Max Input Len</span>
							<input type="number" min="1" class="border rounded px-3 py-2 bg-background" bind:value={trtForm.max_input_len} />
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span>Max Output Len</span>
							<input type="number" min="1" class="border rounded px-3 py-2 bg-background" bind:value={trtForm.max_output_len} />
						</label>
						<label class="flex flex-col gap-1 text-sm">
							<span>Dtype</span>
							<select class="border rounded px-3 py-2 bg-background" bind:value={trtForm.dtype}>
								<option value="float16">float16</option>
								<option value="bfloat16">bfloat16</option>
								<option value="float32">float32</option>
							</select>
						</label>
					</div>
				</div>
				<button class="px-4 py-2 rounded bg-primary text-primary-foreground disabled:opacity-50" onclick={convertToTRT} disabled={busy}>
					Start Conversion
				</button>
			</div>

			<!-- Engines List -->
			<div class="border rounded-lg bg-card">
				<div class="px-4 py-3 border-b bg-muted/50">
					<h2 class="font-semibold">Cached TRT Engines</h2>
				</div>
				<div class="divide-y">
					{#if (trtEngines || []).length === 0}
						<div class="px-4 py-8 text-center text-muted-foreground">No TRT engines</div>
					{:else}
						{#each (trtEngines || []) as e}
							<div class="px-4 py-3">
								<div class="flex items-start justify-between">
									<div>
										<div class="font-semibold">{e.model_id}</div>
										<div class="text-xs text-muted-foreground mt-1">
											{formatSize(e.size_bytes)} · CUDA {e.cuda_version} · TRT {e.trt_version} · SM {e.gpu_sm}
										</div>
										<div class="text-xs text-muted-foreground">Created: {formatDate(e.created_at)}</div>
									</div>
									<div class="flex items-center gap-2">
										<span class={`text-xs px-1.5 py-0.5 rounded ${e.compatible ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300' : 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300'}`}>
											{e.compatible ? 'compatible' : 'needs reconvert'}
										</span>
										<button class="px-2 py-1 text-xs rounded border hover:bg-muted" onclick={() => deleteTRTEngine(e.model_id)}>
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
		<div class="border rounded-lg bg-card">
			<div class="px-4 py-3 border-b bg-muted/50 flex items-center justify-between">
				<h2 class="font-semibold">Repository Downloads</h2>
				<button class="px-3 py-1 text-sm rounded border hover:bg-muted" onclick={loadRepoDownloads}>
					Refresh
				</button>
			</div>
			<div class="divide-y">
				{#if repoDownloads.length === 0}
					<div class="px-4 py-8 text-center text-muted-foreground">
						<p>No active downloads</p>
						<p class="text-sm mt-2">Use "Download Local" button in the Models tab to start downloading a model.</p>
					</div>
				{:else}
					{#each repoDownloads as dl}
						<div class="px-4 py-4">
							<div class="flex items-start justify-between mb-2">
								<div>
									<div class="font-semibold">{dl.model_id}</div>
									<div class="text-xs text-muted-foreground mt-1">
										{dl.completed_files}/{dl.total_files} files · {formatSize(dl.downloaded_size)}/{formatSize(dl.total_size)}
									</div>
									{#if dl.local_path}
										<div class="text-xs text-muted-foreground mt-1">📁 {dl.local_path}</div>
									{/if}
								</div>
								<span class={`text-xs px-2 py-1 rounded font-medium ${
									dl.status === 'completed' ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300' :
									dl.status === 'downloading' ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300' :
									dl.status === 'failed' ? 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300' :
									'bg-gray-100 dark:bg-gray-900/30 text-gray-700 dark:text-gray-300'
								}`}>
									{dl.status === 'downloading' ? '⏳ ' : dl.status === 'completed' ? '✅ ' : dl.status === 'failed' ? '❌ ' : ''}
									{dl.status}
								</span>
							</div>
							<!-- Progress bar -->
							{#if dl.status === 'downloading'}
								<div class="w-full bg-muted rounded-full h-2 mt-2">
									<div class="bg-primary h-2 rounded-full transition-all" style="width: {dl.progress}%"></div>
								</div>
								<div class="text-xs text-muted-foreground mt-1 text-right">{dl.progress.toFixed(1)}%</div>
							{/if}
							{#if dl.error}
								<div class="text-xs text-red-500 mt-2">{dl.error}</div>
							{/if}
						</div>
					{/each}
				{/if}
			</div>
		</div>
	{/if}
</div>
