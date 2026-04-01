<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { cn } from '$lib/utils';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import {
		faNetworkWired,
		faPlus,
		faSpinner,
		faCircleCheck,
		faCircleXmark,
		faCirclePause,
		faClock,
		faServer,
		faMicrochip,
		faMemory,
		faTrash,
		faPlay,
		faStop,
		faUpload,
		faGear,
		faArrowsRotate,
		faXmark,
		faCopy,
		faTerminal,
		faTemperatureHalf,
		faBolt
	} from '@fortawesome/free-solid-svg-icons';
	import * as Card from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { toast } from 'svelte-sonner';

	// State
	let workers = $state<any[]>([]);
	let loading = $state(true);
	let refreshing = $state(false);

	// Add Worker modal
	let showAddModal = $state(false);
	let addForm = $state({
		name: '',
		address: '',
		node_type: 'gpu',
		max_running_models: 2
	});
	let addLoading = $state(false);
	let createdWorker = $state<{ api_key: string; worker: any } | null>(null);

	// Generate Config modal
	let showConfigModal = $state(false);
	let configForm = $state({
		name: '',
		node_type: 'gpu' as string,
		listen_addr: '0.0.0.0:9090',
		hf_cache_dir: '/data/models/hf',
		gguf_cache_dir: '/data/models/gguf',
		hf_token: ''
	});
	let configLoading = $state(false);
	let generatedConfig = $state<{ config_yaml: string; api_key: string; install_command: string; post_install: string } | null>(null);

	// Load Model modal
	let showLoadModal = $state(false);
	let loadTargetWorker = $state<any>(null);
	let loadForm = $state({
		alias: '',
		provider: 'vllm',
		hf_repo: '',
		format: 'hf',
		gpu_device: ''
	});
	let loadLoading = $state(false);

	// Worker models
	let workerModels = $state<Record<string, any[]>>({});
	let workerGPU = $state<Record<string, any>>({});

	onMount(async () => {
		await loadWorkers();
	});

	async function loadWorkers() {
		loading = true;
		try {
			const res = await api.get<{ workers: any[] }>('/api/admin/workers');
			workers = res.workers || [];
			// Load GPU info and models for each online worker
			for (const w of workers) {
				if (w.status === 'online') {
					loadWorkerDetails(w.id);
				}
			}
		} catch (err) {
			console.error('Failed to load workers:', err);
			toast.error('Failed to load workers');
		} finally {
			loading = false;
		}
	}

	async function loadWorkerDetails(workerId: string) {
		try {
			const [gpuRes, modelsRes] = await Promise.allSettled([
				api.get<any>(`/api/admin/workers/${workerId}/gpu`),
				api.get<{ models: any[] }>(`/api/admin/workers/${workerId}/models`)
			]);
			if (gpuRes.status === 'fulfilled') {
				workerGPU[workerId] = gpuRes.value;
			}
			if (modelsRes.status === 'fulfilled') {
				workerModels[workerId] = modelsRes.value.models || [];
			}
		} catch {
			// Silently fail — worker might be slow
		}
	}

	async function refreshWorkers() {
		refreshing = true;
		await loadWorkers();
		refreshing = false;
		toast.success('Workers refreshed');
	}

	async function addWorker() {
		addLoading = true;
		try {
			const res = await api.post<{ worker: any; api_key: string }>('/api/admin/workers', addForm);
			createdWorker = res;
			toast.success(`Worker "${addForm.name}" registered`);
			await loadWorkers();
		} catch (err: any) {
			toast.error(err.message || 'Failed to add worker');
		} finally {
			addLoading = false;
		}
	}

	async function deleteWorker(id: string, name: string) {
		if (!confirm(`Delete worker "${name}"? This will not stop running models.`)) return;
		try {
			await api.delete(`/api/admin/workers/${id}`);
			toast.success(`Worker "${name}" deleted`);
			await loadWorkers();
		} catch (err: any) {
			toast.error(err.message || 'Failed to delete worker');
		}
	}

	async function generateConfig() {
		configLoading = true;
		try {
			const res = await api.post<any>('/api/admin/workers/generate-config', configForm);
			generatedConfig = res;
		} catch (err: any) {
			toast.error(err.message || 'Failed to generate config');
		} finally {
			configLoading = false;
		}
	}

	async function loadModelOnWorker() {
		if (!loadTargetWorker) return;
		loadLoading = true;
		try {
			await api.post(`/api/admin/workers/${loadTargetWorker.id}/models/load`, loadForm);
			toast.success(`Model "${loadForm.alias}" loading on ${loadTargetWorker.name}`);
			showLoadModal = false;
			// Refresh worker models
			setTimeout(() => loadWorkerDetails(loadTargetWorker.id), 2000);
		} catch (err: any) {
			toast.error(err.message || 'Failed to load model');
		} finally {
			loadLoading = false;
		}
	}

	async function stopModelOnWorker(workerId: string, alias: string) {
		try {
			await api.post(`/api/admin/workers/${workerId}/models/stop`, { alias });
			toast.success(`Model "${alias}" stopped`);
			await loadWorkerDetails(workerId);
		} catch (err: any) {
			toast.error(err.message || 'Failed to stop model');
		}
	}

	function copyToClipboard(text: string) {
		navigator.clipboard.writeText(text);
		toast.success('Copied to clipboard');
	}

	function openLoadModal(worker: any) {
		loadTargetWorker = worker;
		loadForm = { alias: '', provider: 'vllm', hf_repo: '', format: 'hf', gpu_device: '' };
		showLoadModal = true;
	}

	function getStatusIcon(status: string) {
		switch (status) {
			case 'online': return faCircleCheck;
			case 'offline': return faCircleXmark;
			case 'draining': return faCirclePause;
			default: return faClock;
		}
	}

	function getStatusColor(status: string) {
		switch (status) {
			case 'online': return 'text-green-500';
			case 'offline': return 'text-red-500';
			case 'draining': return 'text-yellow-500';
			default: return 'text-muted-foreground';
		}
	}

	function getNodeTypeLabel(type: string) {
		switch (type) {
			case 'gpu': return 'GPU';
			case 'cpu': return 'CPU';
			case 'mixed': return 'Mixed';
			default: return type;
		}
	}
</script>

<svelte:head>
	<title>Workers | Administration | AI Gateway</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-2xl font-bold text-foreground">Remote Workers</h2>
			<p class="mt-1 text-sm text-muted-foreground">
				Manage remote GPU/CPU inference workers (Agent Mode)
			</p>
		</div>
		<div class="flex items-center gap-2">
			<Button variant="outline" size="sm" onclick={refreshWorkers} disabled={refreshing}>
				<FontAwesomeIcon icon={faArrowsRotate} class={cn('mr-2 h-4 w-4', refreshing && 'animate-spin')} />
				Refresh
			</Button>
			<Button variant="outline" size="sm" onclick={() => { configForm = { name: '', node_type: 'gpu', listen_addr: '0.0.0.0:9090', hf_cache_dir: '/data/models/hf', gguf_cache_dir: '/data/models/gguf', hf_token: '' }; generatedConfig = null; showConfigModal = true; }}>
				<FontAwesomeIcon icon={faGear} class="mr-2 h-4 w-4" />
				Generate Config
			</Button>
			<Button size="sm" onclick={() => { addForm = { name: '', address: '', node_type: 'gpu', max_running_models: 2 }; createdWorker = null; showAddModal = true; }}>
				<FontAwesomeIcon icon={faPlus} class="mr-2 h-4 w-4" />
				Add Worker
			</Button>
		</div>
	</div>

	{#if loading}
		<!-- Skeleton -->
		<div class="grid gap-4">
			{#each Array(2) as _}
				<div class="h-48 animate-pulse rounded-lg border border-border bg-card"></div>
			{/each}
		</div>
	{:else if workers.length === 0}
		<!-- Empty state -->
		<Card.Root>
			<Card.Content class="flex flex-col items-center justify-center py-16">
				<FontAwesomeIcon icon={faNetworkWired} class="h-12 w-12 text-muted-foreground/50" />
				<h3 class="mt-4 text-lg font-semibold text-foreground">No Workers Registered</h3>
				<p class="mt-2 max-w-md text-center text-sm text-muted-foreground">
					Add a remote GPU or CPU worker to distribute model inference across multiple machines.
					Install the agent binary on the worker, then register it here.
				</p>
				<div class="mt-6 flex gap-3">
					<Button variant="outline" onclick={() => { generatedConfig = null; showConfigModal = true; }}>
						<FontAwesomeIcon icon={faTerminal} class="mr-2 h-4 w-4" />
						Generate Agent Config
					</Button>
					<Button onclick={() => { createdWorker = null; showAddModal = true; }}>
						<FontAwesomeIcon icon={faPlus} class="mr-2 h-4 w-4" />
						Add Worker
					</Button>
				</div>
			</Card.Content>
		</Card.Root>
	{:else}
		<!-- Worker cards -->
		<div class="grid gap-4">
			{#each workers as worker}
				<Card.Root>
					<Card.Content class="p-6">
						<!-- Worker header -->
						<div class="flex items-start justify-between">
							<div class="flex items-center gap-4">
								<div class={cn('flex h-12 w-12 items-center justify-center rounded-lg', worker.status === 'online' ? 'bg-green-500/10' : 'bg-muted')}>
									<FontAwesomeIcon icon={faServer} class={cn('h-6 w-6', getStatusColor(worker.status))} />
								</div>
								<div>
									<h3 class="text-lg font-semibold text-foreground">{worker.name}</h3>
									<div class="flex items-center gap-3 text-sm text-muted-foreground">
										<span class="font-mono">{worker.address}</span>
										<span class="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-xs font-medium">
											{getNodeTypeLabel(worker.node_type)}
										</span>
										<span class={cn('inline-flex items-center gap-1', getStatusColor(worker.status))}>
											<FontAwesomeIcon icon={getStatusIcon(worker.status)} class="h-3.5 w-3.5" />
											{worker.status}
										</span>
									</div>
									{#if worker.last_error}
										<p class="mt-1 text-xs text-red-500">{worker.last_error}</p>
									{/if}
								</div>
							</div>
							<div class="flex items-center gap-2">
								{#if worker.status === 'online'}
									<Button variant="outline" size="sm" onclick={() => openLoadModal(worker)}>
										<FontAwesomeIcon icon={faPlay} class="mr-2 h-4 w-4" />
										Load Model
									</Button>
								{/if}
								<Button variant="ghost" size="icon" onclick={() => deleteWorker(worker.id, worker.name)}>
									<FontAwesomeIcon icon={faTrash} class="h-4 w-4 text-destructive" />
								</Button>
							</div>
						</div>

						<!-- GPU / System info -->
						{#if workerGPU[worker.id]}
							{@const sys = workerGPU[worker.id]}
							{#if sys.gpu_devices && sys.gpu_devices.length > 0}
								<div class="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
									{#each sys.gpu_devices as gpu}
										<div class="rounded-lg border border-border bg-muted/30 p-3">
											<div class="flex items-center justify-between">
												<span class="text-sm font-medium text-foreground">GPU {gpu.index}</span>
												<span class="text-xs text-muted-foreground">{gpu.name}</span>
											</div>
											<div class="mt-2 space-y-1">
												<div class="flex items-center justify-between text-xs">
													<span class="flex items-center gap-1 text-muted-foreground">
														<FontAwesomeIcon icon={faMemory} class="h-3 w-3" />
														VRAM
													</span>
													<span class="text-foreground">
														{Math.round(gpu.memory_used_mb)}MB / {Math.round(gpu.memory_total_mb)}MB
													</span>
												</div>
												<!-- VRAM bar -->
												<div class="h-1.5 w-full rounded-full bg-muted">
													<div
														class={cn('h-1.5 rounded-full', gpu.memory_used_mb / gpu.memory_total_mb > 0.9 ? 'bg-red-500' : 'bg-green-500')}
														style="width: {Math.round((gpu.memory_used_mb / gpu.memory_total_mb) * 100)}%"
													></div>
												</div>
												<div class="flex items-center justify-between text-xs text-muted-foreground">
													<span class="flex items-center gap-1">
														<FontAwesomeIcon icon={faMicrochip} class="h-3 w-3" />
														{gpu.utilization_gpu}%
													</span>
													<span class="flex items-center gap-1">
														<FontAwesomeIcon icon={faTemperatureHalf} class="h-3 w-3" />
														{gpu.temperature_c}°C
													</span>
													{#if gpu.power_draw_w}
														<span class="flex items-center gap-1">
															<FontAwesomeIcon icon={faBolt} class="h-3 w-3" />
															{Math.round(gpu.power_draw_w)}W
														</span>
													{/if}
												</div>
											</div>
										</div>
									{/each}
								</div>
							{/if}
						{/if}

						<!-- Running models -->
						{#if workerModels[worker.id] && workerModels[worker.id].length > 0}
							<div class="mt-4">
								<h4 class="text-sm font-medium text-muted-foreground mb-2">Running Models</h4>
								<div class="space-y-2">
									{#each workerModels[worker.id] as model}
										<div class="flex items-center justify-between rounded-lg border border-border p-3">
											<div class="flex items-center gap-3">
												<div class={cn(
													'h-2 w-2 rounded-full',
													model.status === 'running' ? 'bg-green-500' : model.status === 'starting' ? 'bg-yellow-500 animate-pulse' : 'bg-red-500'
												)}></div>
												<div>
													<span class="text-sm font-medium text-foreground">{model.alias}</span>
													<span class="ml-2 text-xs text-muted-foreground">{model.provider}</span>
												</div>
											</div>
											<Button variant="ghost" size="sm" onclick={() => stopModelOnWorker(worker.id, model.alias)}>
												<FontAwesomeIcon icon={faStop} class="mr-1 h-3.5 w-3.5" />
												Stop
											</Button>
										</div>
									{/each}
								</div>
							</div>
						{/if}
					</Card.Content>
				</Card.Root>
			{/each}
		</div>
	{/if}
</div>

<!-- ══════════════════════════════════════════════════════════════ -->
<!-- Add Worker Modal -->
<!-- ══════════════════════════════════════════════════════════════ -->

{#if showAddModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onclick={() => { if (!createdWorker) showAddModal = false; }}>
		<div class="w-full max-w-lg rounded-lg border border-border bg-card p-6 shadow-xl" onclick={(e) => e.stopPropagation()}>
			{#if createdWorker}
				<!-- Success: show API key -->
				<div class="text-center">
					<FontAwesomeIcon icon={faCircleCheck} class="h-12 w-12 text-green-500" />
					<h3 class="mt-3 text-lg font-semibold text-foreground">Worker Registered</h3>
					<p class="mt-1 text-sm text-muted-foreground">Save the API key — it won't be shown again.</p>
				</div>
				<div class="mt-4 rounded-lg bg-muted p-3">
					<label class="text-xs font-medium text-muted-foreground">API Key</label>
					<div class="mt-1 flex items-center gap-2">
						<code class="flex-1 break-all text-sm text-foreground">{createdWorker.api_key}</code>
						<Button variant="ghost" size="icon" onclick={() => copyToClipboard(createdWorker?.api_key || '')}>
							<FontAwesomeIcon icon={faCopy} class="h-4 w-4" />
						</Button>
					</div>
				</div>
				<div class="mt-4 flex justify-end">
					<Button onclick={() => showAddModal = false}>Done</Button>
				</div>
			{:else}
				<!-- Add form -->
				<div class="flex items-center justify-between mb-4">
					<h3 class="text-lg font-semibold text-foreground">Add Worker</h3>
					<Button variant="ghost" size="icon" onclick={() => showAddModal = false}>
						<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
					</Button>
				</div>
				<div class="space-y-4">
					<div>
						<label for="worker-name" class="text-sm font-medium text-foreground">Name</label>
						<input id="worker-name" type="text" bind:value={addForm.name} placeholder="gpu-worker-1"
							class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground" />
					</div>
					<div>
						<label for="worker-address" class="text-sm font-medium text-foreground">Address</label>
						<input id="worker-address" type="text" bind:value={addForm.address} placeholder="https://gpu-server:9090"
							class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground" />
					</div>
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="worker-type" class="text-sm font-medium text-foreground">Node Type</label>
							<select id="worker-type" bind:value={addForm.node_type}
								class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground">
								<option value="gpu">GPU</option>
								<option value="cpu">CPU</option>
								<option value="mixed">Mixed</option>
							</select>
						</div>
						<div>
							<label for="worker-max" class="text-sm font-medium text-foreground">Max Models</label>
							<input id="worker-max" type="number" bind:value={addForm.max_running_models} min="1" max="20"
								class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground" />
						</div>
					</div>
				</div>
				<div class="mt-6 flex justify-end gap-2">
					<Button variant="outline" onclick={() => showAddModal = false}>Cancel</Button>
					<Button onclick={addWorker} disabled={addLoading || !addForm.name || !addForm.address}>
						{#if addLoading}
							<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						Register Worker
					</Button>
				</div>
			{/if}
		</div>
	</div>
{/if}

<!-- ══════════════════════════════════════════════════════════════ -->
<!-- Generate Config Modal -->
<!-- ══════════════════════════════════════════════════════════════ -->

{#if showConfigModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onclick={() => showConfigModal = false}>
		<div class="w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-lg border border-border bg-card p-6 shadow-xl" onclick={(e) => e.stopPropagation()}>
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-semibold text-foreground">Generate Agent Config</h3>
				<Button variant="ghost" size="icon" onclick={() => showConfigModal = false}>
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</Button>
			</div>

			{#if generatedConfig}
				<!-- Generated config display -->
				<div class="space-y-4">
					<div>
						<div class="flex items-center justify-between">
							<label class="text-sm font-medium text-foreground">1. Install agent on the worker:</label>
							<Button variant="ghost" size="sm" onclick={() => copyToClipboard(generatedConfig?.install_command || '')}>
								<FontAwesomeIcon icon={faCopy} class="mr-1 h-3.5 w-3.5" />
								Copy
							</Button>
						</div>
						<pre class="mt-1 rounded-lg bg-muted p-3 text-xs text-foreground overflow-x-auto">{generatedConfig.install_command}</pre>
					</div>
					<div>
						<div class="flex items-center justify-between">
							<label class="text-sm font-medium text-foreground">2. Save this config as agent.yaml:</label>
							<Button variant="ghost" size="sm" onclick={() => copyToClipboard(generatedConfig?.config_yaml || '')}>
								<FontAwesomeIcon icon={faCopy} class="mr-1 h-3.5 w-3.5" />
								Copy
							</Button>
						</div>
						<pre class="mt-1 rounded-lg bg-muted p-3 text-xs text-foreground overflow-x-auto whitespace-pre-wrap">{generatedConfig.config_yaml}</pre>
					</div>
					<div>
						<label class="text-sm font-medium text-foreground">3. Deploy config and start:</label>
						<pre class="mt-1 rounded-lg bg-muted p-3 text-xs text-foreground overflow-x-auto">{generatedConfig.post_install}</pre>
					</div>
					<div class="rounded-lg border border-yellow-500/30 bg-yellow-500/10 p-3">
						<p class="text-xs text-yellow-600 dark:text-yellow-400">
							<strong>API Key</strong> (save it — won't be shown again):
						</p>
						<div class="mt-1 flex items-center gap-2">
							<code class="break-all text-sm font-mono text-foreground">{generatedConfig.api_key}</code>
							<Button variant="ghost" size="icon" onclick={() => copyToClipboard(generatedConfig?.api_key || '')}>
								<FontAwesomeIcon icon={faCopy} class="h-4 w-4" />
							</Button>
						</div>
					</div>
				</div>
				<div class="mt-6 flex justify-end">
					<Button onclick={() => showConfigModal = false}>Done</Button>
				</div>
			{:else}
				<!-- Config form -->
				<div class="space-y-4">
					<div>
						<label for="cfg-name" class="text-sm font-medium text-foreground">Node Name</label>
						<input id="cfg-name" type="text" bind:value={configForm.name} placeholder="gpu-worker-1"
							class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground" />
					</div>
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="cfg-type" class="text-sm font-medium text-foreground">Node Type</label>
							<select id="cfg-type" bind:value={configForm.node_type}
								class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground">
								<option value="gpu">GPU</option>
								<option value="cpu">CPU</option>
								<option value="mixed">Mixed</option>
							</select>
						</div>
						<div>
							<label for="cfg-addr" class="text-sm font-medium text-foreground">Listen Address</label>
							<input id="cfg-addr" type="text" bind:value={configForm.listen_addr}
								class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground" />
						</div>
					</div>
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="cfg-hf" class="text-sm font-medium text-foreground">HF Cache Dir</label>
							<input id="cfg-hf" type="text" bind:value={configForm.hf_cache_dir}
								class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground" />
						</div>
						<div>
							<label for="cfg-gguf" class="text-sm font-medium text-foreground">GGUF Cache Dir</label>
							<input id="cfg-gguf" type="text" bind:value={configForm.gguf_cache_dir}
								class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground" />
						</div>
					</div>
					<div>
						<label for="cfg-token" class="text-sm font-medium text-foreground">HuggingFace Token (optional)</label>
						<input id="cfg-token" type="password" bind:value={configForm.hf_token} placeholder="hf_..."
							class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground" />
					</div>
				</div>
				<div class="mt-6 flex justify-end gap-2">
					<Button variant="outline" onclick={() => showConfigModal = false}>Cancel</Button>
					<Button onclick={generateConfig} disabled={configLoading || !configForm.name}>
						{#if configLoading}
							<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						Generate
					</Button>
				</div>
			{/if}
		</div>
	</div>
{/if}

<!-- ══════════════════════════════════════════════════════════════ -->
<!-- Load Model Modal -->
<!-- ══════════════════════════════════════════════════════════════ -->

{#if showLoadModal && loadTargetWorker}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onclick={() => showLoadModal = false}>
		<div class="w-full max-w-lg rounded-lg border border-border bg-card p-6 shadow-xl" onclick={(e) => e.stopPropagation()}>
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-semibold text-foreground">Load Model on {loadTargetWorker.name}</h3>
				<Button variant="ghost" size="icon" onclick={() => showLoadModal = false}>
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</Button>
			</div>
			<div class="space-y-4">
				<div>
					<label for="load-alias" class="text-sm font-medium text-foreground">Model Alias</label>
					<input id="load-alias" type="text" bind:value={loadForm.alias} placeholder="my-model"
						class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground" />
				</div>
				<div>
					<label for="load-repo" class="text-sm font-medium text-foreground">HuggingFace Repository</label>
					<input id="load-repo" type="text" bind:value={loadForm.hf_repo} placeholder="meta-llama/Llama-3-8B"
						class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground" />
				</div>
				<div class="grid grid-cols-3 gap-4">
					<div>
						<label for="load-provider" class="text-sm font-medium text-foreground">Provider</label>
						<select id="load-provider" bind:value={loadForm.provider}
							class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground">
							<option value="vllm">vLLM</option>
							<option value="sglang">SGLang</option>
							<option value="tgi">TGI</option>
							<option value="tei">TEI</option>
							<option value="llama.cpp">llama.cpp</option>
						</select>
					</div>
					<div>
						<label for="load-format" class="text-sm font-medium text-foreground">Format</label>
						<select id="load-format" bind:value={loadForm.format}
							class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground">
							<option value="hf">HuggingFace</option>
							<option value="gguf">GGUF</option>
						</select>
					</div>
					<div>
						<label for="load-gpu" class="text-sm font-medium text-foreground">GPU Device</label>
						<input id="load-gpu" type="text" bind:value={loadForm.gpu_device} placeholder="0"
							class="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground" />
					</div>
				</div>
			</div>
			<div class="mt-6 flex justify-end gap-2">
				<Button variant="outline" onclick={() => showLoadModal = false}>Cancel</Button>
				<Button onclick={loadModelOnWorker} disabled={loadLoading || !loadForm.alias || !loadForm.hf_repo}>
					{#if loadLoading}
						<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					Load Model
				</Button>
			</div>
		</div>
	</div>
{/if}
