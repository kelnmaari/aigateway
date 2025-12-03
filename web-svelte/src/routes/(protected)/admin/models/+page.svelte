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
		ExternalLink
	} from 'lucide-svelte';
	import { api } from '$lib/api/client';
	import { cn, formatRelativeTime, debounce } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	// Types
	interface YzmaModel {
		name: string;
		modified_at: string;
		size: number;
		digest: string;
		details?: {
			family?: string;
			parameter_size?: string;
			quantization_level?: string;
		};
	}

	interface RunningModel {
		name: string;
		model: string;
		size: number;
		vram?: number;
		expires_at?: string;
	}

	interface YzmaStats {
		models_count: number;
		loaded_count: number;
		total_size: number;
		memory_used: number;
		available_memory: number;
	}

	interface HFModel {
		id: string;
		author: string;
		modelId: string; // Backend uses camelCase
		downloads: number;
		likes: number;
		tags: string[];
		created_at: string;
		updated_at: string;
	}

	interface HFFile {
		filename: string;
		size: number;
		download_url: string;
	}

	// State
	let activeTab = $state<'local' | 'huggingface'>('local');

	// Local models state
	let yzmaModels = $state<YzmaModel[]>([]);
	let runningModels = $state<RunningModel[]>([]);
	let yzmaStats = $state<YzmaStats | null>(null);
	let isLoadingLocal = $state(true);

	// HuggingFace state
	let hfModels = $state<HFModel[]>([]);
	let hfSearch = $state('');
	let hfAuthor = $state('');
	let hfSort = $state<'downloads' | 'trending' | 'updated'>('downloads');
	let hfLimit = $state(30);
	let isLoadingHF = $state(false);
	let hfSearched = $state(false);

	// Selected model for file download
	let selectedHFModel = $state<HFModel | null>(null);
	let hfFiles = $state<HFFile[]>([]);
	let isLoadingFiles = $state(false);
	let downloadingFiles = $state<Set<string>>(new Set());

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

	function formatSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const gb = bytes / (1024 * 1024 * 1024);
		if (gb >= 1) return `${gb.toFixed(1)} GB`;
		const mb = bytes / (1024 * 1024);
		return `${mb.toFixed(0)} MB`;
	}

	function isModelRunning(name: string): boolean {
		return runningModels.some((m) => m.name === name || m.model === name);
	}

	async function loadModel(name: string) {
		try {
			await api.post('/api/ui/yzma/load', { model: name });
			await loadLocalModels();
		} catch (error) {
			console.error('Failed to load model:', error);
			alert('Failed to load model');
		}
	}

	async function unloadModel(name: string) {
		try {
			await api.post('/api/ui/yzma/unload', { model: name });
			await loadLocalModels();
		} catch (error) {
			console.error('Failed to unload model:', error);
			alert('Failed to unload model');
		}
	}

	async function deleteModel(name: string) {
		if (!confirm(`Delete model "${name}"? This cannot be undone.`)) return;
		try {
			await api.post('/api/ui/yzma/delete', { model: name });
			await loadLocalModels();
		} catch (error) {
			console.error('Failed to delete model:', error);
			alert('Failed to delete model');
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
		selectedHFModel = model;
		isLoadingFiles = true;
		hfFiles = [];

		try {
			// Use the gguf-files endpoint with model_id in path
			const response = await api.get<{ files: HFFile[] }>(
				`/api/ui/huggingface/gguf-files/${model.id}`
			);
			hfFiles = response.files || [];
		} catch (error) {
			console.error('Failed to load model files:', error);
		} finally {
			isLoadingFiles = false;
		}
	}

	async function downloadFile(model: HFModel, file: HFFile) {
		const key = `${model.id}/${file.filename}`;
		downloadingFiles.add(key);
		downloadingFiles = new Set(downloadingFiles);

		try {
			await api.post('/api/ui/huggingface/download', {
				model_id: model.id,
				filename: file.filename
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
								<p class="text-2xl font-bold">{formatSize(yzmaStats.memory_used)}</p>
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
									<Button variant="ghost" size="sm" onclick={() => unloadModel(model.name)}>
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
							{#each yzmaModels as model (model.digest)}
								{@const running = isModelRunning(model.name)}
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
										{formatRelativeTime(model.modified_at)}
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
											{#if !running}
												<button
													onclick={() => loadModel(model.name)}
													class="rounded p-1.5 text-muted-foreground hover:bg-primary/10 hover:text-primary"
													title="Load model"
												>
													<Play class="h-4 w-4" />
												</button>
											{/if}
											<button
												onclick={() => deleteModel(model.name)}
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

<!-- Files Modal -->
{#if selectedHFModel}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (selectedHFModel = null)}
		role="dialog"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && (selectedHFModel = null)}
	>
		<div class="w-full max-w-2xl rounded-xl border border-border bg-card shadow-xl">
			<div class="flex items-center justify-between border-b border-border px-6 py-4">
				<div>
					<h2 class="text-lg font-semibold">{selectedHFModel.modelId || selectedHFModel.id.split('/')[1] || selectedHFModel.id}</h2>
					<p class="text-sm text-muted-foreground">by {selectedHFModel.author}</p>
				</div>
				<button onclick={() => (selectedHFModel = null)} class="rounded p-1 text-muted-foreground hover:bg-accent">
					✕
				</button>
			</div>

			<div class="max-h-96 overflow-y-auto p-6">
				{#if isLoadingFiles}
					<div class="flex items-center justify-center py-10">
						<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
					</div>
				{:else if hfFiles.length === 0}
					<div class="py-10 text-center text-muted-foreground">
						No GGUF files found in this repository
					</div>
				{:else}
					<div class="space-y-2">
						{#each hfFiles as file}
							{@const key = `${selectedHFModel.id}/${file.filename}`}
							{@const isDownloading = downloadingFiles.has(key)}
							<div class="flex items-center justify-between rounded-lg border border-border p-3">
								<div>
									<p class="font-medium">{file.filename}</p>
									<p class="text-sm text-muted-foreground">{formatSize(file.size)}</p>
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
		</div>
	</div>
{/if}
