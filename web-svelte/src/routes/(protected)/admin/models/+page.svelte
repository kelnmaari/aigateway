<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { inferenceApi, type ModelInfo, type ArtifactInfo, type TRTEngine, type Provider, type Format, type Capability, type LoadRequest } from '$lib/api/inference';

	const providers: Provider[] = ['vllm', 'sglang', 'tgi', 'tensorrt-llm', 'llama.cpp'];
	const formats: Format[] = ['hf', 'gguf', 'trt', 'other'];
	const capabilities: Capability[] = ['chat', 'embeddings', 'vision'];

	let models: ModelInfo[] = $state([]);
	let artifacts: ArtifactInfo[] = $state([]);
	let trtEngines: TRTEngine[] = $state([]);
	let busy = $state(false);
	let msg = $state('');
	let msgType = $state<'info' | 'error' | 'success'>('info');
	let activeTab = $state<'models' | 'cache' | 'trt' | 'logs'>('models');
	
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
		vllm_tensor_parallel: 0,
		vllm_max_model_len: 0,
		vllm_gpu_utilization: 0.92,
		llama_main_gpu: 0,
		llama_tensor_split: '',
		llama_n_gpu_layers: 0,
		sglang_tensor_parallel: 0,
		sglang_mem_fraction: 0.9,
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
	});

	onDestroy(() => {
		if (logsInterval) clearInterval(logsInterval);
	});

	async function refreshAll() {
		await Promise.all([loadModels(), loadCache(), loadTRTEngines()]);
	}

	async function loadModels() {
		try {
			models = await inferenceApi.listModels();
		} catch (e: any) {
			showMsg(e?.message || 'Не удалось получить список моделей', 'error');
		}
	}

	async function loadCache() {
		try {
			artifacts = await inferenceApi.listCache();
		} catch (e: any) {
			console.error(e);
		}
	}

	async function loadTRTEngines() {
		try {
			trtEngines = await inferenceApi.listTRTEngines();
		} catch (e: any) {
			console.error(e);
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
				await inferenceApi.prepare(req);
				showMsg('Артефакты подготавливаются...', 'success');
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
			showMsg('Кеш очищен', 'success');
			await loadCache();
		} catch (e: any) {
			showMsg(e?.message || 'Ошибка очистки кеша', 'error');
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

	const totalCacheSize = $derived(artifacts.reduce((sum, a) => sum + (a.size || 0), 0));
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
			Models ({models.length})
		</button>
		<button class={`px-4 py-2 -mb-px ${activeTab === 'cache' ? 'border-b-2 border-primary font-semibold' : 'text-muted-foreground'}`} onclick={() => activeTab = 'cache'}>
			Cache ({formatSize(totalCacheSize)})
		</button>
		<button class={`px-4 py-2 -mb-px ${activeTab === 'trt' ? 'border-b-2 border-primary font-semibold' : 'text-muted-foreground'}`} onclick={() => activeTab = 'trt'}>
			TRT Engines ({trtEngines.length})
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
							<select class="border rounded px-3 py-2 bg-background" bind:value={form.provider}>
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
							<input class="border rounded px-3 py-2 bg-background" bind:value={form.hf_repo} placeholder="meta-llama/Llama-3.1-8B-Instruct" />
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
							Prepare Only
						</button>
					</div>
				</div>

				<!-- Models List -->
				<div class="border rounded-lg overflow-hidden bg-card">
					<div class="px-4 py-3 border-b bg-muted/50 flex items-center justify-between">
						<h2 class="font-semibold">Running Models</h2>
					</div>
					<div class="divide-y">
						{#if models.length === 0}
							<div class="px-4 py-8 text-center text-muted-foreground">No models loaded</div>
						{:else}
							{#each models as m}
								<div class="px-4 py-3 hover:bg-muted/30 cursor-pointer flex items-start gap-4" onclick={() => selectModel(m)}>
									<div class="flex-1 min-w-0">
										<div class="flex items-center gap-2">
											<span class="font-semibold">{m.alias}</span>
											{#if m.pinned}
												<span class="text-xs px-1.5 py-0.5 rounded bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300">pinned</span>
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
										<button class="px-2 py-1 text-xs rounded border hover:bg-muted" onclick={(e) => { e.stopPropagation(); stopModel(m.alias); }}>Stop</button>
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
						{#if artifacts.length === 0}
							<tr><td class="px-4 py-8 text-center text-muted-foreground" colspan="5">No cache artifacts</td></tr>
						{:else}
							{#each artifacts as a}
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
					{#if trtEngines.length === 0}
						<div class="px-4 py-8 text-center text-muted-foreground">No TRT engines</div>
					{:else}
						{#each trtEngines as e}
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
</div>
