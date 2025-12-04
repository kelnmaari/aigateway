<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Server,
		RefreshCw,
		Loader2,
		Download,
		Trash2,
		Play,
		Search,
		Cpu,
		HardDrive,
		Zap,
		ExternalLink,
		FileText,
		Scale,
		Clock,
		Heart,
		ArrowDownToLine,
		Tag,
		BookOpen,
		GitBranch,
		Globe,
		ChevronDown,
		ChevronUp,
		X
	} from 'lucide-svelte';
	import { api } from '$lib/api/client';
	import { cn, formatRelativeTime, debounce } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	// Types
	interface YzmaModel {
		name: string;
		path?: string;
		modified_at?: string;
		size?: number;
		digest?: string;
		loaded?: boolean;
		details?: {
			family?: string;
			parameter_size?: string;
			quantization_level?: string;
		};
	}

	interface RunningModel {
		path: string;    // Full relative path
		name: string;    // Basename
		alias?: string;
		size?: number;
		vram?: number;
	}

	interface YzmaStats {
		models_count: number;
		loaded_count: number;
		total_size: number;
		memory_used: number;
		available_memory: number;
	}

	interface YzmaModelMetadata {
		description?: string;
		architecture?: string;
		context_length?: number;
		embedding_size?: number;
		num_layers?: number;
		num_heads?: number;
		num_kv_heads?: number;
		vocab_size?: number;
		model_size?: number;
		file_size_bytes?: number;
		quantization?: string;
		file_type?: string;
		general_name?: string;
		general_author?: string;
		general_base_model?: string;
		license?: string;
		raw_metadata?: Record<string, string>;
	}

	interface HFModelConfig {
		architectures?: string[];
		model_type?: string;
		vocab_size?: number;
		hidden_size?: number;
		num_attention_heads?: number;
		num_hidden_layers?: number;
		max_position_embeddings?: number;
	}

	interface HFModelCardData {
		language?: string[];
		license?: string;
		tags?: string[];
		datasets?: string[];
		metrics?: string[];
		base_model?: string[];
	}

	interface HFModel {
		id: string;
		author: string;
		modelId: string;
		downloads: number;
		likes: number;
		tags: string[];
		pipeline_tag?: string;
		library_name?: string;
		createdAt?: string;
		lastModified?: string;
		config?: HFModelConfig;
		cardData?: HFModelCardData;
		total_size?: number;
		gguf_files?: HFFile[];
		has_gguf?: boolean;
		parameter_size?: string;
	}

	interface HFFile {
		rfilename: string;
		size: number;
		download_url?: string;
		lfs?: {
			size: number;
			oid: string;
		};
	}

	// Detailed model info (loaded separately)
	interface HFModelDetails extends HFModel {
		siblings?: HFFile[];
		description?: string;
	}

	// State
	let activeTab = $state<'local' | 'huggingface'>('local');

	// Local models state
	let yzmaModels = $state<YzmaModel[]>([]);
	let runningModels = $state<RunningModel[]>([]);
	let yzmaStats = $state<YzmaStats | null>(null);
	let isLoadingLocal = $state(true);
	
	// Selected local model for details
	let selectedYzmaModel = $state<YzmaModel | null>(null);
	let yzmaMetadata = $state<YzmaModelMetadata | null>(null);
	let isLoadingMetadata = $state(false);

	// HuggingFace state
	let hfModels = $state<HFModel[]>([]);
	let hfSearch = $state('');
	let hfAuthor = $state('');
	let hfSort = $state<'downloads' | 'trending' | 'updated'>('downloads');
	let hfLimit = $state(30);
	let isLoadingHF = $state(false);
	let hfSearched = $state(false);

	// Selected model for file download
	let selectedHFModel = $state<HFModelDetails | null>(null);
	let hfFiles = $state<HFFile[]>([]);
	let isLoadingFiles = $state(false);
	let downloadingFiles = $state<Set<string>>(new Set());
	let showFilesSection = $state(true);

	onMount(async () => {
		await loadLocalModels();
	});

	// === Local Models ===
	async function loadLocalModels() {
		isLoadingLocal = true;
		try {
			const [modelsRes, runningRes, statsRes] = await Promise.allSettled([
				api.get<{ models: YzmaModel[] }>('/api/ui/yzma/models'),
				api.get<{ models: RunningModel[] }>('/api/ui/yzma/loaded'),
				api.get<YzmaStats>('/api/ui/yzma/stats')
			]);

			if (modelsRes.status === 'fulfilled') {
				yzmaModels = modelsRes.value.models || [];
			}
			if (runningRes.status === 'fulfilled') {
				runningModels = runningRes.value.models || [];
			}
			if (statsRes.status === 'fulfilled') {
				yzmaStats = statsRes.value;
			}
		} catch (error) {
			console.error('Failed to load models:', error);
		} finally {
			isLoadingLocal = false;
		}
	}

	function formatSize(bytes: number | undefined | null): string {
		if (!bytes || bytes === 0 || isNaN(bytes)) return '0 B';
		const gb = bytes / (1024 * 1024 * 1024);
		if (gb >= 1) return `${gb.toFixed(1)} GB`;
		const mb = bytes / (1024 * 1024);
		if (mb >= 1) return `${mb.toFixed(0)} MB`;
		const kb = bytes / 1024;
		if (kb >= 1) return `${kb.toFixed(0)} KB`;
		return `${bytes} B`;
	}

	function isModelRunning(path: string): boolean {
		return runningModels.some((m) => m.path === path || m.name === path);
	}

	async function loadModel(path: string) {
		try {
			await api.post('/api/ui/yzma/load', { model_path: path });
			await loadLocalModels();
		} catch (error) {
			console.error('Failed to load model:', error);
			alert('Failed to load model');
		}
	}

	async function unloadModel(path: string) {
		try {
			await api.post('/api/ui/yzma/unload', { model_path: path });
			await loadLocalModels();
		} catch (error) {
			console.error('Failed to unload model:', error);
			alert('Failed to unload model');
		}
	}

	async function deleteModel(path: string, name: string) {
		if (!confirm(`Delete model "${name}"? This cannot be undone.`)) return;
		try {
			await api.post('/api/ui/yzma/delete', { model_path: path });
			await loadLocalModels();
		} catch (error) {
			console.error('Failed to delete model:', error);
			alert('Failed to delete model');
		}
	}

	async function viewLocalModelDetails(model: YzmaModel) {
		selectedYzmaModel = model;
		isLoadingMetadata = true;
		yzmaMetadata = null;
		
		try {
			const response = await api.get<{
				loaded: boolean;
				metadata?: YzmaModelMetadata;
				message?: string;
			}>(`/api/ui/yzma/metadata/${model.path || model.name}`);
			
			if (response.metadata) {
				yzmaMetadata = response.metadata;
			}
		} catch (error) {
			console.error('Failed to load model metadata:', error);
		} finally {
			isLoadingMetadata = false;
		}
	}

	// === HuggingFace ===
	const debouncedHFSearch = debounce(searchHuggingFace, 500);

	async function searchHuggingFace() {
		if (!hfSearch && !hfAuthor) return;

		isLoadingHF = true;
		hfSearched = true;
		try {
			const params = new URLSearchParams();
			if (hfSearch) params.append('search', hfSearch);
			if (hfAuthor) params.append('author', hfAuthor);
			params.append('sort', hfSort);
			params.append('limit', String(hfLimit));

			const response = await api.get<{ models: HFModel[] }>(`/api/ui/huggingface/search?${params}`);
			hfModels = response.models || [];
		} catch (error) {
			console.error('Failed to search HuggingFace:', error);
			hfModels = [];
		} finally {
			isLoadingHF = false;
		}
	}

	function quickSearch(query: string) {
		hfSearch = query;
		hfAuthor = '';
		searchHuggingFace();
	}

	async function selectModel(model: HFModel) {
		selectedHFModel = model as HFModelDetails;
		isLoadingFiles = true;
		hfFiles = [];
		showFilesSection = true;

		try {
			// Load detailed model info including files
			const response = await api.get<{ model_id: string; files: HFFile[] }>(
				`/api/ui/huggingface/gguf-files/${model.id}`
			);
			hfFiles = response.files || [];

			// Try to load full model details (with config and cardData)
			try {
				const detailsResponse = await api.get<HFModelDetails>(
					`/api/ui/huggingface/model/${model.id}`
				);
				if (detailsResponse) {
					selectedHFModel = { ...model, ...detailsResponse };
				}
			} catch {
				// Full details not available, use basic info
				console.log('Full model details not available, using search result');
			}
		} catch (error) {
			console.error('Failed to load model files:', error);
		} finally {
			isLoadingFiles = false;
		}
	}

	function extractQuantFromFilename(filename: string): string | null {
		const match = filename.match(/[_-](Q\d+[_-]?[KMSXL0-9]*)/i);
		return match ? match[1].toUpperCase() : null;
	}

	function extractSizeFromFilename(filename: string): string | null {
		const match = filename.match(/(\d+\.?\d*)[_-]?[bB]/);
		return match ? `${match[1]}B` : null;
	}

	function formatDate(dateStr?: string): string {
		if (!dateStr) return '—';
		try {
			return new Date(dateStr).toLocaleDateString('ru-RU', {
				year: 'numeric',
				month: 'short',
				day: 'numeric'
			});
		} catch {
			return '—';
		}
	}

	async function downloadFile(model: HFModel, file: HFFile) {
		const key = `${model.id}/${file.rfilename}`;
		downloadingFiles.add(key);
		downloadingFiles = new Set(downloadingFiles);

		try {
			await api.post('/api/ui/huggingface/download', {
				model_id: model.id,
				filename: file.rfilename,
				total_size: file.lfs?.size || file.size
			});
			alert('Download started! Check the Downloads tab for progress.');
		} catch (error) {
			console.error('Failed to start download:', error);
			alert('Failed to start download');
		} finally {
			downloadingFiles.delete(key);
			downloadingFiles = new Set(downloadingFiles);
		}
	}

	function formatNumber(num: number): string {
		if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
		if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
		return String(num);
	}
</script>

<div class="space-y-6">
	<!-- Sub-tabs -->
	<div class="flex items-center justify-between">
		<div class="flex gap-2 border-b border-border">
			<button
				onclick={() => (activeTab = 'local')}
				class={cn(
					'border-b-2 px-4 py-2 text-sm font-medium transition-colors',
					activeTab === 'local'
						? 'border-primary text-primary'
						: 'border-transparent text-muted-foreground hover:text-foreground'
				)}
			>
				<Cpu class="mr-2 inline h-4 w-4" />
				Local Models (yzma)
			</button>
			<button
				onclick={() => (activeTab = 'huggingface')}
				class={cn(
					'border-b-2 px-4 py-2 text-sm font-medium transition-colors',
					activeTab === 'huggingface'
						? 'border-primary text-primary'
						: 'border-transparent text-muted-foreground hover:text-foreground'
				)}
			>
				🤗 Hugging Face Browser
			</button>
		</div>

		{#if activeTab === 'local'}
			<Button variant="outline" onclick={loadLocalModels} disabled={isLoadingLocal}>
				<RefreshCw class={cn('mr-2 h-4 w-4', isLoadingLocal && 'animate-spin')} />
				{m.common_refresh()}
			</Button>
		{/if}
	</div>

	<!-- Local Models Tab -->
	{#if activeTab === 'local'}
		{#if isLoadingLocal}
			<div class="flex items-center justify-center py-20">
				<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
			</div>
		{:else}
			<!-- Stats -->
			{#if yzmaStats}
				<div class="grid gap-4 sm:grid-cols-4">
					<div class="rounded-lg border border-border bg-card p-4">
						<div class="flex items-center gap-3">
							<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
								<Server class="h-5 w-5" />
							</div>
							<div>
								<p class="text-2xl font-bold">{yzmaStats.models_count}</p>
								<p class="text-xs text-muted-foreground">Models</p>
							</div>
						</div>
					</div>
					<div class="rounded-lg border border-border bg-card p-4">
						<div class="flex items-center gap-3">
							<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/10 text-green-500">
								<Zap class="h-5 w-5" />
							</div>
							<div>
								<p class="text-2xl font-bold">{yzmaStats.loaded_count}</p>
								<p class="text-xs text-muted-foreground">Loaded</p>
							</div>
						</div>
					</div>
					<div class="rounded-lg border border-border bg-card p-4">
						<div class="flex items-center gap-3">
							<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10 text-blue-500">
								<HardDrive class="h-5 w-5" />
							</div>
							<div>
								<p class="text-2xl font-bold">{formatSize(yzmaStats.total_size)}</p>
								<p class="text-xs text-muted-foreground">Total Size</p>
							</div>
						</div>
					</div>
					<div class="rounded-lg border border-border bg-card p-4">
						<div class="flex items-center gap-3">
							<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/10 text-purple-500">
								<Cpu class="h-5 w-5" />
							</div>
							<div>
								<p class="text-2xl font-bold">{yzmaStats.memory_used ? formatSize(yzmaStats.memory_used) : '0 B'}</p>
								<p class="text-xs text-muted-foreground">Memory Used</p>
							</div>
						</div>
					</div>
				</div>
			{/if}

			<!-- Running Models -->
			{#if runningModels.length > 0}
				<section>
					<h3 class="mb-3 flex items-center gap-2 font-medium text-green-500">
						<Play class="h-4 w-4" />
						Running Models ({runningModels.length})
					</h3>
					<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
						{#each runningModels as model}
							<div class="rounded-lg border border-green-500/20 bg-green-500/5 p-4">
								<div class="flex items-start justify-between">
									<div>
										<p class="font-medium">{model.name}</p>
										<p class="text-sm text-muted-foreground">
											{formatSize(model.size)}
											{#if model.vram}
												• VRAM: {formatSize(model.vram)}
											{/if}
										</p>
									</div>
									<Button variant="ghost" size="sm" onclick={() => unloadModel(model.path)}>
										Unload
									</Button>
								</div>
							</div>
						{/each}
					</div>
				</section>
			{/if}

			<!-- All Models -->
			{#if yzmaModels.length === 0}
				<div class="rounded-lg border border-dashed border-border py-16 text-center">
					<Server class="mx-auto h-12 w-12 text-muted-foreground/40" />
					<p class="mt-4 text-lg font-medium">No GGUF models installed</p>
					<p class="mt-1 text-muted-foreground">Download models from the Hugging Face Browser tab</p>
				</div>
			{:else}
				<div class="overflow-hidden rounded-lg border border-border">
					<table class="w-full">
						<thead class="border-b border-border bg-muted/50">
							<tr>
								<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Model</th>
								<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Size</th>
								<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Details</th>
								<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Modified</th>
								<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Status</th>
								<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">Actions</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border">
							{#each yzmaModels as model (model.path || model.name)}
								{@const running = isModelRunning(model.path || model.name)}
								<tr class="hover:bg-muted/30">
									<td class="px-4 py-3">
										<div class="flex items-center gap-2">
											<Server class="h-4 w-4 text-muted-foreground" />
											<span class="font-medium">{model.name}</span>
										</div>
									</td>
									<td class="px-4 py-3 text-sm">{formatSize(model.size)}</td>
									<td class="px-4 py-3 text-sm text-muted-foreground">
										{#if model.details}
											{model.details.parameter_size || ''}
											{model.details.quantization_level || ''}
										{/if}
									</td>
									<td class="px-4 py-3 text-sm text-muted-foreground">
										{model.modified_at ? formatRelativeTime(model.modified_at) : '-'}
									</td>
									<td class="px-4 py-3">
										{#if running}
											<span class="inline-flex items-center gap-1 rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-500">
												<span class="h-1.5 w-1.5 rounded-full bg-green-500"></span>
												Running
											</span>
										{:else}
											<span class="text-sm text-muted-foreground">Idle</span>
										{/if}
									</td>
									<td class="px-4 py-3 text-right">
										<div class="flex items-center justify-end gap-1">
											<button
												onclick={() => viewLocalModelDetails(model)}
												class="rounded p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
												title="View details"
											>
												<FileText class="h-4 w-4" />
											</button>
										{#if !running}
											<button
												onclick={() => loadModel(model.path || model.name)}
												class="rounded p-1.5 text-muted-foreground hover:bg-primary/10 hover:text-primary"
												title="Load model"
											>
												<Play class="h-4 w-4" />
											</button>
										{/if}
										<button
											onclick={() => deleteModel(model.path || model.name, model.name)}
											disabled={running}
												class={cn(
													'rounded p-1.5',
													running
														? 'cursor-not-allowed text-muted-foreground/50'
														: 'text-muted-foreground hover:bg-destructive/10 hover:text-destructive'
												)}
												title={running ? 'Cannot delete running model' : 'Delete model'}
											>
												<Trash2 class="h-4 w-4" />
											</button>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		{/if}
	{/if}

	<!-- HuggingFace Browser Tab -->
	{#if activeTab === 'huggingface'}
		<div class="space-y-6">
			<!-- Search Form -->
			<div class="rounded-lg border border-border bg-card p-4">
				<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
					<div class="lg:col-span-2">
						<label for="hf-search" class="mb-1.5 block text-sm font-medium">Search models</label>
						<div class="relative">
							<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
							<input
								id="hf-search"
								type="text"
								bind:value={hfSearch}
								placeholder="e.g., llama, mistral, phi"
								class="w-full rounded-lg border border-input bg-background py-2 pl-9 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
								onkeydown={(e) => e.key === 'Enter' && searchHuggingFace()}
							/>
						</div>
					</div>
					<div>
						<label for="hf-author" class="mb-1.5 block text-sm font-medium">Author</label>
						<input
							id="hf-author"
							type="text"
							bind:value={hfAuthor}
							placeholder="e.g., TheBloke"
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						/>
					</div>
					<div>
						<label for="hf-sort" class="mb-1.5 block text-sm font-medium">Sort by</label>
						<select
							id="hf-sort"
							bind:value={hfSort}
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						>
							<option value="downloads">Most Downloads</option>
							<option value="trending">Trending</option>
							<option value="updated">Recently Updated</option>
						</select>
					</div>
					<div class="flex items-end">
						<Button onclick={searchHuggingFace} disabled={isLoadingHF} class="w-full">
							{#if isLoadingHF}
								<Loader2 class="mr-2 h-4 w-4 animate-spin" />
							{:else}
								<Search class="mr-2 h-4 w-4" />
							{/if}
							Search
						</Button>
					</div>
				</div>

				<!-- Quick Filters -->
				<div class="mt-4 flex flex-wrap gap-2">
					<span class="text-sm font-medium text-muted-foreground">Quick:</span>
					<button onclick={() => quickSearch('llama')} class="rounded-md bg-muted px-2.5 py-1 text-xs font-medium hover:bg-muted/80">
						🦙 Llama
					</button>
					<button onclick={() => quickSearch('mistral')} class="rounded-md bg-muted px-2.5 py-1 text-xs font-medium hover:bg-muted/80">
						🌪️ Mistral
					</button>
					<button onclick={() => quickSearch('phi')} class="rounded-md bg-muted px-2.5 py-1 text-xs font-medium hover:bg-muted/80">
						φ Phi
					</button>
					<button onclick={() => quickSearch('gemma')} class="rounded-md bg-muted px-2.5 py-1 text-xs font-medium hover:bg-muted/80">
						💎 Gemma
					</button>
					<button onclick={() => quickSearch('qwen')} class="rounded-md bg-muted px-2.5 py-1 text-xs font-medium hover:bg-muted/80">
						🌟 Qwen
					</button>
				</div>
			</div>

			<!-- Results -->
			{#if !hfSearched}
				<div class="rounded-lg border border-dashed border-border py-16 text-center">
					<Search class="mx-auto h-12 w-12 text-muted-foreground/40" />
					<p class="mt-4 text-lg font-medium">Search for GGUF models</p>
					<p class="mt-1 text-muted-foreground">Use the search form above or click a quick filter</p>
				</div>
			{:else if isLoadingHF}
				<div class="flex items-center justify-center py-20">
					<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
				</div>
			{:else if hfModels.length === 0}
				<div class="rounded-lg border border-dashed border-border py-16 text-center">
					<Search class="mx-auto h-12 w-12 text-muted-foreground/40" />
					<p class="mt-4 text-muted-foreground">No models found matching your search</p>
				</div>
			{:else}
				<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
					{#each hfModels as model (model.id)}
						<div class="flex flex-col rounded-lg border border-border bg-card p-4 transition-shadow hover:shadow-md">
							<div class="mb-2 flex items-start justify-between">
								<div class="flex-1">
									<p class="text-xs text-muted-foreground">{model.author}</p>
									<h3 class="font-medium text-foreground">{model.modelId || model.id.split('/')[1] || model.id}</h3>
								</div>
								<a
									href={`https://huggingface.co/${model.id}`}
									target="_blank"
									rel="noopener noreferrer"
									class="shrink-0 rounded p-1 text-muted-foreground hover:bg-accent"
								>
									<ExternalLink class="h-4 w-4" />
								</a>
							</div>

							<div class="mb-3 flex gap-3 text-xs text-muted-foreground">
								<span>↓ {formatNumber(model.downloads)}</span>
								<span>❤️ {formatNumber(model.likes)}</span>
							</div>

							{#if model.tags && model.tags.length > 0}
								<div class="mb-3 flex flex-wrap gap-1">
									{#each model.tags.slice(0, 4) as tag}
										<span class="rounded bg-muted px-1.5 py-0.5 text-xs">{tag}</span>
									{/each}
								</div>
							{/if}

							<div class="mt-auto">
								<Button variant="outline" size="sm" class="w-full" onclick={() => selectModel(model)}>
									<Download class="mr-2 h-4 w-4" />
									View Files
								</Button>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- Model Details Modal -->
{#if selectedHFModel}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
		onclick={(e) => e.target === e.currentTarget && (selectedHFModel = null)}
		role="dialog"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && (selectedHFModel = null)}
	>
		<div class="flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden rounded-xl border border-border bg-card shadow-2xl">
			<!-- Header -->
			<div class="flex items-start justify-between border-b border-border bg-muted/30 px-6 py-4">
				<div class="flex-1">
					<div class="flex items-center gap-3">
						<h2 class="text-xl font-bold">{selectedHFModel.modelId || selectedHFModel.id.split('/')[1] || selectedHFModel.id}</h2>
						<a
							href={`https://huggingface.co/${selectedHFModel.id}`}
							target="_blank"
							rel="noopener noreferrer"
							class="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
							title="Open on HuggingFace"
						>
							<ExternalLink class="h-4 w-4" />
						</a>
					</div>
					<p class="mt-1 text-sm text-muted-foreground">by {selectedHFModel.author}</p>
					
					<!-- Quick Stats -->
					<div class="mt-3 flex flex-wrap gap-4 text-sm">
						<span class="flex items-center gap-1.5 text-muted-foreground">
							<ArrowDownToLine class="h-4 w-4" />
							{formatNumber(selectedHFModel.downloads)} downloads
						</span>
						<span class="flex items-center gap-1.5 text-muted-foreground">
							<Heart class="h-4 w-4" />
							{formatNumber(selectedHFModel.likes)} likes
						</span>
						{#if selectedHFModel.parameter_size}
							<span class="flex items-center gap-1.5 text-muted-foreground">
								<Scale class="h-4 w-4" />
								{selectedHFModel.parameter_size}
							</span>
						{/if}
						{#if selectedHFModel.lastModified}
							<span class="flex items-center gap-1.5 text-muted-foreground">
								<Clock class="h-4 w-4" />
								Updated {formatDate(selectedHFModel.lastModified)}
							</span>
						{/if}
					</div>
				</div>
				<button 
					onclick={() => (selectedHFModel = null)} 
					class="rounded-lg p-2 text-muted-foreground hover:bg-accent"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			<!-- Content -->
			<div class="flex-1 overflow-y-auto">
				<!-- GGUF Files Section -->
				<div class="border-b border-border">
					<button
						onclick={() => (showFilesSection = !showFilesSection)}
						class="flex w-full items-center justify-between px-6 py-3 text-left hover:bg-muted/30"
					>
						<div class="flex items-center gap-2 font-semibold">
							<FileText class="h-5 w-5 text-primary" />
							GGUF Files ({hfFiles.length})
						</div>
						{#if showFilesSection}
							<ChevronUp class="h-5 w-5 text-muted-foreground" />
						{:else}
							<ChevronDown class="h-5 w-5 text-muted-foreground" />
						{/if}
					</button>
					
					{#if showFilesSection}
						<div class="max-h-64 overflow-y-auto px-6 pb-4">
							{#if isLoadingFiles}
								<div class="flex items-center justify-center py-8">
									<Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
								</div>
							{:else if hfFiles.length === 0}
								<div class="py-6 text-center text-muted-foreground">
									No GGUF files found in this repository
								</div>
							{:else}
								<div class="space-y-2">
									{#each hfFiles as file}
										{@const key = `${selectedHFModel.id}/${file.rfilename}`}
										{@const isDownloading = downloadingFiles.has(key)}
										{@const fileSize = file.lfs?.size || file.size}
										{@const quant = extractQuantFromFilename(file.rfilename)}
										<div class="flex items-center justify-between rounded-lg border border-border bg-background p-3 transition-colors hover:border-primary/50">
											<div class="min-w-0 flex-1 pr-4">
												<p class="truncate font-mono text-sm font-medium" title={file.rfilename}>
													{file.rfilename}
												</p>
												<div class="mt-1 flex items-center gap-3 text-xs text-muted-foreground">
													<span class="flex items-center gap-1">
														<HardDrive class="h-3 w-3" />
														{formatSize(fileSize)}
													</span>
													{#if quant}
														<span class="rounded bg-primary/10 px-1.5 py-0.5 font-medium text-primary">
															{quant}
														</span>
													{/if}
												</div>
											</div>
											<Button
												size="sm"
												onclick={() => downloadFile(selectedHFModel!, file)}
												disabled={isDownloading}
											>
												{#if isDownloading}
													<Loader2 class="mr-2 h-4 w-4 animate-spin" />
												{:else}
													<Download class="mr-2 h-4 w-4" />
												{/if}
												Download
											</Button>
										</div>
									{/each}
								</div>
							{/if}
						</div>
					{/if}
				</div>

				<!-- Model Information Section -->
				<div class="space-y-6 px-6 py-4">
					<!-- Description -->
					{#if selectedHFModel.description}
						<div>
							<h3 class="mb-2 flex items-center gap-2 text-sm font-semibold text-muted-foreground">
								<FileText class="h-4 w-4" />
								Description
							</h3>
							<div class="max-h-48 overflow-y-auto rounded-lg border border-border bg-muted/30 p-4">
								<p class="whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
									{selectedHFModel.description}
								</p>
							</div>
						</div>
					{/if}

					<!-- Tags -->
					{#if selectedHFModel.tags && selectedHFModel.tags.length > 0}
						<div>
							<h3 class="mb-2 flex items-center gap-2 text-sm font-semibold text-muted-foreground">
								<Tag class="h-4 w-4" />
								Tags
							</h3>
							<div class="flex flex-wrap gap-1.5">
								{#each selectedHFModel.tags as tag}
									<span class="rounded-md bg-muted px-2 py-1 text-xs">{tag}</span>
								{/each}
							</div>
						</div>
					{/if}

					<!-- Model Config -->
					{#if selectedHFModel.config}
						<div>
							<h3 class="mb-2 flex items-center gap-2 text-sm font-semibold text-muted-foreground">
								<Cpu class="h-4 w-4" />
								Model Architecture
							</h3>
							<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
								{#if selectedHFModel.config.architectures?.length}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Architecture</p>
										<p class="font-medium">{selectedHFModel.config.architectures.join(', ')}</p>
									</div>
								{/if}
								{#if selectedHFModel.config.model_type}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Model Type</p>
										<p class="font-medium">{selectedHFModel.config.model_type}</p>
									</div>
								{/if}
								{#if selectedHFModel.config.vocab_size}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Vocab Size</p>
										<p class="font-medium">{selectedHFModel.config.vocab_size.toLocaleString()}</p>
									</div>
								{/if}
								{#if selectedHFModel.config.hidden_size}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Hidden Size</p>
										<p class="font-medium">{selectedHFModel.config.hidden_size.toLocaleString()}</p>
									</div>
								{/if}
								{#if selectedHFModel.config.num_attention_heads}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Attention Heads</p>
										<p class="font-medium">{selectedHFModel.config.num_attention_heads}</p>
									</div>
								{/if}
								{#if selectedHFModel.config.num_hidden_layers}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Hidden Layers</p>
										<p class="font-medium">{selectedHFModel.config.num_hidden_layers}</p>
									</div>
								{/if}
								{#if selectedHFModel.config.max_position_embeddings}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Max Context Length</p>
										<p class="font-medium">{selectedHFModel.config.max_position_embeddings.toLocaleString()}</p>
									</div>
								{/if}
							</div>
						</div>
					{/if}

					<!-- Card Data (License, Languages, Base Model) -->
					{#if selectedHFModel.cardData}
						<div class="grid gap-4 sm:grid-cols-2">
							{#if selectedHFModel.cardData.license}
								<div>
									<h3 class="mb-2 flex items-center gap-2 text-sm font-semibold text-muted-foreground">
										<Scale class="h-4 w-4" />
										License
									</h3>
									<span class="inline-flex items-center rounded-md bg-blue-500/10 px-2.5 py-1 text-sm font-medium text-blue-500">
										{selectedHFModel.cardData.license}
									</span>
								</div>
							{/if}

							{#if selectedHFModel.cardData.language?.length}
								<div>
									<h3 class="mb-2 flex items-center gap-2 text-sm font-semibold text-muted-foreground">
										<Globe class="h-4 w-4" />
										Languages
									</h3>
									<div class="flex flex-wrap gap-1.5">
										{#each selectedHFModel.cardData.language as lang}
											<span class="rounded-md bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-500">
												{lang}
											</span>
										{/each}
									</div>
								</div>
							{/if}

							{#if selectedHFModel.cardData.base_model?.length}
								<div class="sm:col-span-2">
									<h3 class="mb-2 flex items-center gap-2 text-sm font-semibold text-muted-foreground">
										<GitBranch class="h-4 w-4" />
										Base Model
									</h3>
									<div class="flex flex-wrap gap-2">
										{#each selectedHFModel.cardData.base_model as baseModel}
											<a
												href={`https://huggingface.co/${baseModel}`}
												target="_blank"
												rel="noopener noreferrer"
												class="inline-flex items-center gap-1.5 rounded-md bg-purple-500/10 px-2.5 py-1 text-sm font-medium text-purple-500 hover:bg-purple-500/20"
											>
												{baseModel}
												<ExternalLink class="h-3 w-3" />
											</a>
										{/each}
									</div>
								</div>
							{/if}

							{#if selectedHFModel.cardData.datasets?.length}
								<div class="sm:col-span-2">
									<h3 class="mb-2 flex items-center gap-2 text-sm font-semibold text-muted-foreground">
										<BookOpen class="h-4 w-4" />
										Training Datasets
									</h3>
									<div class="flex flex-wrap gap-1.5">
										{#each selectedHFModel.cardData.datasets.slice(0, 10) as dataset}
											<span class="rounded-md bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-500">
												{dataset}
											</span>
										{/each}
										{#if selectedHFModel.cardData.datasets.length > 10}
											<span class="text-xs text-muted-foreground">
												+{selectedHFModel.cardData.datasets.length - 10} more
											</span>
										{/if}
									</div>
								</div>
							{/if}
						</div>
					{/if}

					<!-- Pipeline/Library -->
					{#if selectedHFModel.pipeline_tag || selectedHFModel.library_name}
						<div class="flex flex-wrap gap-4">
							{#if selectedHFModel.pipeline_tag}
								<div>
									<h3 class="mb-1 text-xs font-semibold text-muted-foreground">Pipeline</h3>
									<span class="rounded-md bg-cyan-500/10 px-2.5 py-1 text-sm font-medium text-cyan-500">
										{selectedHFModel.pipeline_tag}
									</span>
								</div>
							{/if}
							{#if selectedHFModel.library_name}
								<div>
									<h3 class="mb-1 text-xs font-semibold text-muted-foreground">Library</h3>
									<span class="rounded-md bg-pink-500/10 px-2.5 py-1 text-sm font-medium text-pink-500">
										{selectedHFModel.library_name}
									</span>
								</div>
							{/if}
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}

<!-- Local Model Details Modal -->
{#if selectedYzmaModel}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
		onclick={(e) => e.target === e.currentTarget && (selectedYzmaModel = null)}
		role="dialog"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && (selectedYzmaModel = null)}
	>
		<div class="flex max-h-[85vh] w-full max-w-3xl flex-col overflow-hidden rounded-xl border border-border bg-card shadow-2xl">
			<!-- Header -->
			<div class="flex items-start justify-between border-b border-border bg-muted/30 px-6 py-4">
				<div class="flex-1">
					<div class="flex items-center gap-3">
						<Server class="h-6 w-6 text-primary" />
						<h2 class="text-xl font-bold">{selectedYzmaModel.name}</h2>
						{#if isModelRunning(selectedYzmaModel.path || selectedYzmaModel.name)}
							<span class="rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-500">
								● Running
							</span>
						{/if}
					</div>
					{#if selectedYzmaModel.path && selectedYzmaModel.path !== selectedYzmaModel.name}
						<p class="mt-1 font-mono text-sm text-muted-foreground">{selectedYzmaModel.path}</p>
					{/if}
					
					<!-- Quick Stats -->
					<div class="mt-3 flex flex-wrap gap-4 text-sm">
						<span class="flex items-center gap-1.5 text-muted-foreground">
							<HardDrive class="h-4 w-4" />
							{formatSize(selectedYzmaModel.size)}
						</span>
						{#if yzmaMetadata?.quantization}
							<span class="flex items-center gap-1.5">
								<Tag class="h-4 w-4 text-muted-foreground" />
								<span class="rounded bg-primary/10 px-1.5 py-0.5 font-medium text-primary">
									{yzmaMetadata.quantization}
								</span>
							</span>
						{/if}
					</div>
				</div>
				<button 
					onclick={() => (selectedYzmaModel = null)} 
					class="rounded-lg p-2 text-muted-foreground hover:bg-accent"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			<!-- Content -->
			<div class="flex-1 overflow-y-auto p-6">
				{#if isLoadingMetadata}
					<div class="flex items-center justify-center py-16">
						<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
					</div>
				{:else if !isModelRunning(selectedYzmaModel.path || selectedYzmaModel.name)}
					<!-- Model not loaded - show message -->
					<div class="rounded-lg border border-amber-500/30 bg-amber-500/10 p-6 text-center">
						<Zap class="mx-auto h-12 w-12 text-amber-500" />
						<h3 class="mt-4 text-lg font-semibold">Model Not Loaded</h3>
						<p class="mt-2 text-muted-foreground">
							Load the model to see detailed architecture information (layers, attention heads, context size, etc.)
						</p>
						<Button 
							class="mt-4" 
							onclick={async () => {
								await loadModel(selectedYzmaModel!.path || selectedYzmaModel!.name);
								await viewLocalModelDetails(selectedYzmaModel!);
							}}
						>
							<Play class="mr-2 h-4 w-4" />
							Load Model
						</Button>
					</div>
				{:else if yzmaMetadata}
					<div class="space-y-6">
						<!-- Description -->
						{#if yzmaMetadata.description}
							<div>
								<h3 class="mb-2 text-sm font-semibold text-muted-foreground">Description</h3>
								<p class="rounded-lg border border-border bg-background p-3 font-mono text-sm">
									{yzmaMetadata.description}
								</p>
							</div>
						{/if}

						<!-- Model Architecture -->
						<div>
							<h3 class="mb-3 flex items-center gap-2 text-sm font-semibold text-muted-foreground">
								<Cpu class="h-4 w-4" />
								Model Architecture
							</h3>
							<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
								{#if yzmaMetadata.architecture}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Architecture</p>
										<p class="font-medium">{yzmaMetadata.architecture}</p>
									</div>
								{/if}
								{#if yzmaMetadata.context_length}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Context Length</p>
										<p class="font-medium">{yzmaMetadata.context_length.toLocaleString()}</p>
									</div>
								{/if}
								{#if yzmaMetadata.num_layers}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Layers</p>
										<p class="font-medium">{yzmaMetadata.num_layers}</p>
									</div>
								{/if}
								{#if yzmaMetadata.num_heads}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Attention Heads</p>
										<p class="font-medium">{yzmaMetadata.num_heads}</p>
									</div>
								{/if}
								{#if yzmaMetadata.num_kv_heads}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">KV Heads</p>
										<p class="font-medium">{yzmaMetadata.num_kv_heads}</p>
									</div>
								{/if}
								{#if yzmaMetadata.embedding_size}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Embedding Size</p>
										<p class="font-medium">{yzmaMetadata.embedding_size.toLocaleString()}</p>
									</div>
								{/if}
								{#if yzmaMetadata.vocab_size}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Vocab Size</p>
										<p class="font-medium">{yzmaMetadata.vocab_size.toLocaleString()}</p>
									</div>
								{/if}
								{#if yzmaMetadata.model_size}
									<div class="rounded-lg border border-border p-3">
										<p class="text-xs text-muted-foreground">Model Size (memory)</p>
										<p class="font-medium">{formatSize(yzmaMetadata.model_size)}</p>
									</div>
								{/if}
							</div>
						</div>

						<!-- Model Info -->
						{#if yzmaMetadata.general_name || yzmaMetadata.general_author || yzmaMetadata.license}
							<div class="grid gap-4 sm:grid-cols-2">
								{#if yzmaMetadata.general_name}
									<div>
										<h3 class="mb-1 text-xs font-semibold text-muted-foreground">Name</h3>
										<p class="font-medium">{yzmaMetadata.general_name}</p>
									</div>
								{/if}
								{#if yzmaMetadata.general_author}
									<div>
										<h3 class="mb-1 text-xs font-semibold text-muted-foreground">Author</h3>
										<p class="font-medium">{yzmaMetadata.general_author}</p>
									</div>
								{/if}
								{#if yzmaMetadata.general_base_model}
									<div>
										<h3 class="mb-1 text-xs font-semibold text-muted-foreground">Base Model</h3>
										<p class="font-medium">{yzmaMetadata.general_base_model}</p>
									</div>
								{/if}
								{#if yzmaMetadata.license}
									<div>
										<h3 class="mb-1 text-xs font-semibold text-muted-foreground">License</h3>
										<span class="inline-flex items-center rounded-md bg-blue-500/10 px-2.5 py-1 text-sm font-medium text-blue-500">
											{yzmaMetadata.license}
										</span>
									</div>
								{/if}
							</div>
						{/if}

						<!-- Raw Metadata (collapsible) -->
						{#if yzmaMetadata.raw_metadata && Object.keys(yzmaMetadata.raw_metadata).length > 0}
							<details class="rounded-lg border border-border">
								<summary class="cursor-pointer px-4 py-3 font-medium hover:bg-muted/30">
									Raw GGUF Metadata ({Object.keys(yzmaMetadata.raw_metadata).length} keys)
								</summary>
								<div class="max-h-64 overflow-y-auto border-t border-border p-4">
									<div class="space-y-1 font-mono text-xs">
										{#each Object.entries(yzmaMetadata.raw_metadata).sort((a, b) => a[0].localeCompare(b[0])) as [key, value]}
											<div class="flex">
												<span class="w-64 shrink-0 truncate text-muted-foreground" title={key}>{key}</span>
												<span class="truncate" title={value}>{value}</span>
											</div>
										{/each}
									</div>
								</div>
							</details>
						{/if}
					</div>
				{:else}
					<div class="py-10 text-center text-muted-foreground">
						No metadata available
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="flex items-center justify-end gap-3 border-t border-border px-6 py-4">
				{#if isModelRunning(selectedYzmaModel.path || selectedYzmaModel.name)}
					<Button variant="outline" onclick={() => unloadModel(selectedYzmaModel!.path || selectedYzmaModel!.name)}>
						Unload
					</Button>
				{:else}
					<Button onclick={() => loadModel(selectedYzmaModel!.path || selectedYzmaModel!.name)}>
						<Play class="mr-2 h-4 w-4" />
						Load
					</Button>
				{/if}
				<Button variant="outline" onclick={() => (selectedYzmaModel = null)}>
					Close
				</Button>
			</div>
		</div>
	</div>
{/if}
