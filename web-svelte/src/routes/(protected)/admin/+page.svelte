<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		Users,
		Key,
		Activity,
		TrendingUp,
		Clock,
		Cpu,
		HardDrive,
		Loader2,
		Building2,
		Server,
		Database,
		BrainCircuit,
		CheckCircle,
		XCircle,
		AlertCircle,
		Zap,
		Thermometer,
		Gauge
	} from 'lucide-svelte';
	import { adminApi, type AdminStats, type SystemMetrics, type RAGStats, type GPUMetricsResponse, type GPUDevice, type YzmaHealthResponse, type YzmaGPUResponse } from '$lib/api/admin';
	import { cn } from '$lib/utils';

	let stats = $state<AdminStats | null>(null);
	let metrics = $state<SystemMetrics | null>(null);
	let ragStats = $state<RAGStats | null>(null);
	let gpuMetrics = $state<GPUMetricsResponse | null>(null);
	let yzmaHealth = $state<YzmaHealthResponse | null>(null);
	let yzmaGPU = $state<YzmaGPUResponse | null>(null);
	let isLoading = $state(true);
	let gpuUpdateInterval: ReturnType<typeof setInterval> | null = null;

	onMount(async () => {
		await loadData();
		// Auto-refresh GPU metrics every 10 seconds
		gpuUpdateInterval = setInterval(updateGPUMetrics, 10000);
	});

	onDestroy(() => {
		if (gpuUpdateInterval) {
			clearInterval(gpuUpdateInterval);
		}
	});

	async function loadData() {
		isLoading = true;
		try {
			const [statsRes, metricsRes, ragRes, gpuRes, yzmaHealthRes, yzmaGPURes] = await Promise.allSettled([
				adminApi.getStats(),
				adminApi.getMetrics(),
				adminApi.getRAGStats(),
				adminApi.getGPUMetrics(),
				adminApi.getYzmaHealth(),
				adminApi.getYzmaGPUInfo()
			]);

			if (statsRes.status === 'fulfilled') {
				stats = statsRes.value;
			}
			if (metricsRes.status === 'fulfilled') {
				metrics = metricsRes.value;
			}
			if (ragRes.status === 'fulfilled') {
				ragStats = ragRes.value;
			}
			if (gpuRes.status === 'fulfilled') {
				gpuMetrics = gpuRes.value;
			}
			if (yzmaHealthRes.status === 'fulfilled') {
				yzmaHealth = yzmaHealthRes.value;
			}
			if (yzmaGPURes.status === 'fulfilled') {
				yzmaGPU = yzmaGPURes.value;
			}
		} catch (error) {
			console.error('Failed to load admin data:', error);
		} finally {
			isLoading = false;
		}
	}

	async function updateGPUMetrics() {
		try {
			const [gpuRes, healthRes] = await Promise.allSettled([
				adminApi.getGPUMetrics(),
				adminApi.getYzmaHealth()
			]);
			if (gpuRes.status === 'fulfilled') {
				gpuMetrics = gpuRes.value;
			}
			if (healthRes.status === 'fulfilled') {
				yzmaHealth = healthRes.value;
			}
		} catch (error) {
			// Silently fail for GPU/health updates
		}
	}

	function getTempColor(temp: number): string {
		if (temp > 80) return 'text-red-500';
		if (temp > 70) return 'text-amber-500';
		return 'text-green-500';
	}

	function getUtilColor(util: number): string {
		if (util > 90) return 'bg-red-500';
		if (util > 70) return 'bg-amber-500';
		return 'bg-green-500';
	}

	function formatBytes(bytes: number | undefined): string {
		if (bytes == null || bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
	}

	function formatUptime(seconds: number | undefined): string {
		if (seconds == null || seconds === 0) return '—';
		const days = Math.floor(seconds / 86400);
		const hours = Math.floor((seconds % 86400) / 3600);
		const mins = Math.floor((seconds % 3600) / 60);
		if (days > 0) return `${days}d ${hours}h`;
		if (hours > 0) return `${hours}h ${mins}m`;
		return `${mins}m`;
	}

	function formatMB(mb: number | undefined): string {
		if (mb == null) return '—';
		if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`;
		return `${mb.toFixed(0)} MB`;
	}

	// Derived values for progress bars
	const cpuPercent = $derived(metrics?.system?.cpu_percent ?? 0);
	const memPercent = $derived(metrics?.system?.memory_percent ?? 0);
	const diskPercent = $derived(metrics?.system?.disk_percent ?? 0);
	const appMemPercent = $derived(
		metrics?.app?.heap_sys_mb
			? ((metrics.app.heap_alloc_mb ?? 0) / metrics.app.heap_sys_mb) * 100
			: 0
	);
</script>

{#if isLoading}
	<div class="flex items-center justify-center py-20">
		<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
	</div>
{:else}
	<div class="space-y-6">
		<!-- Stats Cards -->
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10 text-blue-500">
						<Users class="h-5 w-5" />
					</div>
					<div>
						<p class="text-sm text-muted-foreground">Total Users</p>
						<p class="text-2xl font-bold">{stats?.total_users ?? 0}</p>
					</div>
				</div>
			</div>

			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/10 text-green-500">
						<Key class="h-5 w-5" />
					</div>
					<div>
						<p class="text-sm text-muted-foreground">API Keys</p>
						<p class="text-2xl font-bold">{stats?.total_api_keys ?? 0}</p>
					</div>
				</div>
			</div>

			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/10 text-purple-500">
						<Building2 class="h-5 w-5" />
					</div>
					<div>
						<p class="text-sm text-muted-foreground">Tenants</p>
						<p class="text-2xl font-bold">{stats?.total_tenants ?? 0}</p>
					</div>
				</div>
			</div>

			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-500/10 text-amber-500">
						<Activity class="h-5 w-5" />
					</div>
					<div>
						<p class="text-sm text-muted-foreground">Total Requests</p>
						<p class="text-2xl font-bold">{stats?.total_requests ?? 0}</p>
					</div>
				</div>
			</div>
		</div>

		<!-- System Metrics -->
		{#if metrics?.system}
			<div class="rounded-xl border border-border bg-card p-6">
				<h2 class="mb-4 text-lg font-semibold">System Resources</h2>
				<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
					<!-- CPU -->
					<div>
						<div class="mb-2 flex items-center justify-between">
							<span class="flex items-center gap-2 text-sm text-muted-foreground">
								<Cpu class="h-4 w-4" />
								CPU ({metrics.system.cpu_cores} cores)
							</span>
							<span class="text-sm font-medium">{cpuPercent.toFixed(1)}%</span>
						</div>
						<div class="h-2 overflow-hidden rounded-full bg-muted">
							<div
								class={cn(
									'h-full transition-all',
									cpuPercent > 80 ? 'bg-red-500' : cpuPercent > 50 ? 'bg-amber-500' : 'bg-green-500'
								)}
								style="width: {Math.min(cpuPercent, 100)}%"
							></div>
						</div>
					</div>

					<!-- Memory -->
					<div>
						<div class="mb-2 flex items-center justify-between">
							<span class="flex items-center gap-2 text-sm text-muted-foreground">
								<HardDrive class="h-4 w-4" />
								Memory
							</span>
							<span class="text-sm font-medium">
								{formatBytes(metrics.system.memory_used)} / {formatBytes(metrics.system.memory_total)}
							</span>
						</div>
						<div class="h-2 overflow-hidden rounded-full bg-muted">
							<div
								class={cn(
									'h-full transition-all',
									memPercent > 80 ? 'bg-red-500' : memPercent > 50 ? 'bg-amber-500' : 'bg-green-500'
								)}
								style="width: {Math.min(memPercent, 100)}%"
							></div>
						</div>
					</div>

					<!-- Disk -->
					<div>
						<div class="mb-2 flex items-center justify-between">
							<span class="flex items-center gap-2 text-sm text-muted-foreground">
								<Database class="h-4 w-4" />
								Disk
							</span>
							<span class="text-sm font-medium">
								{formatBytes(metrics.system.disk_used)} / {formatBytes(metrics.system.disk_total)}
							</span>
						</div>
						<div class="h-2 overflow-hidden rounded-full bg-muted">
							<div
								class={cn(
									'h-full transition-all',
									diskPercent > 80 ? 'bg-red-500' : diskPercent > 50 ? 'bg-amber-500' : 'bg-green-500'
								)}
								style="width: {Math.min(diskPercent, 100)}%"
							></div>
						</div>
					</div>

					<!-- Uptime -->
					<div>
						<div class="mb-2 flex items-center gap-2 text-sm text-muted-foreground">
							<Clock class="h-4 w-4" />
							Uptime
						</div>
						<div class="space-y-1">
							<p class="text-lg font-semibold">{formatUptime(metrics.system.uptime)}</p>
							<p class="text-xs text-muted-foreground">App: {formatUptime(metrics.system.app_uptime)}</p>
						</div>
					</div>
				</div>
			</div>
		{/if}

		<!-- Yzma/llama.cpp Status -->
		{#if yzmaHealth || yzmaGPU}
			<div class="rounded-xl border border-border bg-card p-6">
				<div class="mb-4 flex items-center justify-between">
					<h2 class="text-lg font-semibold flex items-center gap-2">
						<BrainCircuit class="h-5 w-5 text-purple-500" />
						LLM Backend Status
					</h2>
					{#if yzmaHealth}
						{@const statusColor = yzmaHealth.status === 'ready' ? 'text-green-500' : 
							yzmaHealth.status === 'waiting_for_models' ? 'text-amber-500' : 'text-blue-500'}
						<span class={cn("text-sm font-medium flex items-center gap-1.5", statusColor)}>
							{#if yzmaHealth.status === 'ready'}
								<CheckCircle class="h-4 w-4" />
								Ready
							{:else if yzmaHealth.status === 'waiting_for_models'}
								<Loader2 class="h-4 w-4 animate-spin" />
								Loading Models...
							{:else}
								<Loader2 class="h-4 w-4 animate-spin" />
								Initializing...
							{/if}
						</span>
					{/if}
				</div>
				
				<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
					<!-- Backend Status -->
					<div class="rounded-lg border border-border/50 p-3">
						<div class="mb-1 text-xs font-medium text-muted-foreground">Backend</div>
						<div class="flex items-center gap-2">
							{#if yzmaGPU?.backend_ready}
								<CheckCircle class="h-4 w-4 text-green-500" />
								<span class="font-medium text-green-500">Initialized</span>
							{:else}
								<Loader2 class="h-4 w-4 animate-spin text-blue-500" />
								<span class="font-medium text-blue-500">Starting...</span>
							{/if}
						</div>
					</div>

					<!-- GPU Support -->
					<div class="rounded-lg border border-border/50 p-3">
						<div class="mb-1 text-xs font-medium text-muted-foreground">GPU Support</div>
						<div class="flex items-center gap-2">
							{#if yzmaGPU?.gpu?.supports_gpu}
								<CheckCircle class="h-4 w-4 text-green-500" />
								<span class="font-medium text-green-500">
									{yzmaGPU.gpu.max_devices} Device{yzmaGPU.gpu.max_devices !== 1 ? 's' : ''}
								</span>
							{:else}
								<XCircle class="h-4 w-4 text-red-500" />
								<span class="font-medium text-red-500">CPU Only</span>
							{/if}
						</div>
					</div>

					<!-- Loaded Models -->
					<div class="rounded-lg border border-border/50 p-3">
						<div class="mb-1 text-xs font-medium text-muted-foreground">Loaded Models</div>
						<div class="flex items-center gap-2">
							{#if yzmaHealth?.loaded_model_count && yzmaHealth.loaded_model_count > 0}
								<CheckCircle class="h-4 w-4 text-green-500" />
								<span class="font-medium">{yzmaHealth.loaded_model_count} Model{yzmaHealth.loaded_model_count !== 1 ? 's' : ''}</span>
							{:else}
								<AlertCircle class="h-4 w-4 text-amber-500" />
								<span class="font-medium text-amber-500">None</span>
							{/if}
						</div>
					</div>

					<!-- GPU Layers -->
					<div class="rounded-lg border border-border/50 p-3">
						<div class="mb-1 text-xs font-medium text-muted-foreground">GPU Layers</div>
						<span class="font-medium">
							{#if yzmaGPU?.gpu?.n_gpu_layers === -1}
								All (Auto)
							{:else if yzmaGPU?.gpu?.n_gpu_layers === 0}
								CPU Only
							{:else}
								{yzmaGPU?.gpu?.n_gpu_layers ?? '—'}
							{/if}
						</span>
					</div>
				</div>

				<!-- Configuration Details -->
				{#if yzmaGPU?.gpu}
					<div class="mt-4 pt-4 border-t border-border/50">
						<div class="grid gap-2 text-xs text-muted-foreground sm:grid-cols-2 lg:grid-cols-4">
							<div>
								<span class="font-medium">Context Size:</span>
								{yzmaGPU.gpu.context_size.toLocaleString()}
							</div>
							<div>
								<span class="font-medium">Batch Size:</span>
								{yzmaGPU.gpu.batch_size.toLocaleString()}
							</div>
							<div>
								<span class="font-medium">Flash Attention:</span>
								{yzmaGPU.gpu.flash_attention ? 'Enabled' : 'Disabled'}
							</div>
							{#if yzmaGPU.gpu.tensor_split && yzmaGPU.gpu.tensor_split.length > 0}
								<div>
									<span class="font-medium">Tensor Split:</span>
									{yzmaGPU.gpu.tensor_split.map(v => (v * 100).toFixed(0) + '%').join(' / ')}
								</div>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Model List -->
				{#if yzmaHealth?.loaded_models && yzmaHealth.loaded_models.length > 0}
					<div class="mt-4 pt-4 border-t border-border/50">
						<div class="text-xs font-medium text-muted-foreground mb-2">Loaded Models:</div>
						<div class="flex flex-wrap gap-2">
							{#each yzmaHealth.loaded_models as model}
								<span class="px-2 py-1 rounded bg-muted text-xs font-mono">{model}</span>
							{/each}
						</div>
					</div>
				{/if}
			</div>
		{/if}

		<!-- GPU Metrics -->
		{#if gpuMetrics?.enabled && gpuMetrics.data?.devices && gpuMetrics.data.devices.length > 0}
			<div class="rounded-xl border border-border bg-card p-6">
				<div class="mb-4 flex items-center justify-between">
					<h2 class="text-lg font-semibold flex items-center gap-2">
						<Zap class="h-5 w-5 text-green-500" />
						GPU ({gpuMetrics.data.device_count} {gpuMetrics.data.device_count === 1 ? 'device' : 'devices'})
					</h2>
					<span class="text-xs text-muted-foreground">Auto-refresh: 10s</span>
				</div>
				<div class="space-y-4">
					{#each gpuMetrics.data.devices as gpu, index}
						{@const memUsedGB = gpu.memory_used_mb / 1024}
						{@const memTotalGB = gpu.memory_total_mb / 1024}
						{@const memPercent = gpu.memory_usage_percent || (gpu.memory_used_mb / gpu.memory_total_mb * 100)}
						{@const powerPercent = gpu.power_limit_w > 0 ? (gpu.power_usage_w / gpu.power_limit_w * 100) : 0}
						
						<div class={cn(
							"rounded-lg border border-border/50 p-4",
							index > 0 && "mt-3"
						)}>
							<!-- GPU Header -->
							<div class="mb-3 flex items-center justify-between">
								<div>
									<span class="text-xs font-medium text-muted-foreground">GPU {index}</span>
									<h3 class="font-semibold">{gpu.name.replace('NVIDIA GeForce ', '').replace('NVIDIA ', '')}</h3>
								</div>
								<div class="flex items-center gap-4 text-sm">
									<!-- Temperature -->
									<div class="flex items-center gap-1.5">
										<Thermometer class={cn("h-4 w-4", getTempColor(gpu.temperature_c))} />
										<span class={cn("font-bold", getTempColor(gpu.temperature_c))}>{gpu.temperature_c}°C</span>
									</div>
									<!-- Power -->
									<div class="flex items-center gap-1.5">
										<Zap class="h-4 w-4 text-amber-500" />
										<span class="font-medium">{gpu.power_usage_w.toFixed(0)}W</span>
										<span class="text-xs text-muted-foreground">/ {gpu.power_limit_w.toFixed(0)}W</span>
									</div>
									<!-- Clock -->
									<div class="flex items-center gap-1.5">
										<Gauge class="h-4 w-4 text-blue-500" />
										<span class="font-medium">{gpu.clock_graphics_mhz} MHz</span>
									</div>
									{#if gpu.fan_speed_percent > 0}
										<div class="text-muted-foreground">
											Fan: {gpu.fan_speed_percent}%
										</div>
									{/if}
								</div>
							</div>
							
							<!-- Progress Bars -->
							<div class="grid gap-4 sm:grid-cols-2">
								<!-- GPU Utilization -->
								<div>
									<div class="mb-1.5 flex items-center justify-between">
										<span class="text-xs font-medium text-muted-foreground">GPU Load</span>
										<span class={cn("text-sm font-bold", 
											gpu.utilization_gpu_percent > 90 ? 'text-red-500' : 
											gpu.utilization_gpu_percent > 70 ? 'text-amber-500' : 'text-green-500'
										)}>{gpu.utilization_gpu_percent}%</span>
									</div>
									<div class="h-2.5 overflow-hidden rounded-full bg-muted">
										<div
											class={cn("h-full transition-all duration-500", getUtilColor(gpu.utilization_gpu_percent))}
											style="width: {Math.min(gpu.utilization_gpu_percent, 100)}%"
										></div>
									</div>
								</div>
								
								<!-- VRAM -->
								<div>
									<div class="mb-1.5 flex items-center justify-between">
										<span class="text-xs font-medium text-muted-foreground">VRAM</span>
										<span class="text-sm font-bold text-blue-500">
											{memUsedGB.toFixed(1)} / {memTotalGB.toFixed(1)} GB
										</span>
									</div>
									<div class="h-2.5 overflow-hidden rounded-full bg-muted">
										<div
											class="h-full bg-blue-500 transition-all duration-500"
											style="width: {Math.min(memPercent, 100)}%"
										></div>
									</div>
								</div>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- App Metrics (Go Runtime) -->
		{#if metrics?.app}
			<div class="rounded-xl border border-border bg-card p-6">
				<h2 class="mb-4 text-lg font-semibold">Application Runtime</h2>
				<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
					<!-- Go Heap -->
					<div>
						<div class="mb-2 flex items-center justify-between">
							<span class="flex items-center gap-2 text-sm text-muted-foreground">
								<Server class="h-4 w-4" />
								Go Heap
							</span>
							<span class="text-sm font-medium">
								{formatMB(metrics.app.heap_alloc_mb)} / {formatMB(metrics.app.heap_sys_mb)}
							</span>
						</div>
						<div class="h-2 overflow-hidden rounded-full bg-muted">
							<div
								class={cn(
									'h-full transition-all',
									appMemPercent > 80 ? 'bg-red-500' : appMemPercent > 50 ? 'bg-amber-500' : 'bg-cyan-500'
								)}
								style="width: {Math.min(appMemPercent, 100)}%"
							></div>
						</div>
					</div>

					<!-- Goroutines -->
					<div>
						<div class="mb-2 flex items-center gap-2 text-sm text-muted-foreground">
							<Activity class="h-4 w-4" />
							Goroutines
						</div>
						<p class="text-xl font-semibold">{metrics.app.num_goroutines}</p>
					</div>

					<!-- GC Runs -->
					<div>
						<div class="mb-2 flex items-center gap-2 text-sm text-muted-foreground">
							<TrendingUp class="h-4 w-4" />
							GC Runs
						</div>
						<p class="text-xl font-semibold">{metrics.app.num_gc}</p>
					</div>

					<!-- Alloc Rate -->
					<div>
						<div class="mb-2 flex items-center gap-2 text-sm text-muted-foreground">
							<TrendingUp class="h-4 w-4" />
							Alloc Rate
						</div>
						<p class="text-xl font-semibold">{metrics.app.alloc_rate_mb_s?.toFixed(2) ?? '0'} MB/s</p>
					</div>
				</div>
			</div>
		{/if}

		<!-- RAG / Vector Store Stats -->
		{#if ragStats}
			<div class="rounded-xl border border-border bg-card p-6">
				<h2 class="mb-4 text-lg font-semibold flex items-center gap-2">
					<BrainCircuit class="h-5 w-5" />
					RAG / Vector Store
				</h2>
				<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
					<!-- Status -->
					<div>
						<div class="mb-2 flex items-center gap-2 text-sm text-muted-foreground">
							{#if ragStats.health_status === 'healthy'}
								<CheckCircle class="h-4 w-4 text-green-500" />
							{:else if ragStats.health_status === 'unhealthy'}
								<XCircle class="h-4 w-4 text-red-500" />
							{:else}
								<AlertCircle class="h-4 w-4 text-amber-500" />
							{/if}
							Status
						</div>
						<p class="text-xl font-semibold capitalize">{ragStats.health_status}</p>
						<p class="text-xs text-muted-foreground mt-1">
							{ragStats.enabled ? 'Enabled' : 'Disabled'}
						</p>
					</div>

					<!-- Provider -->
					<div>
						<div class="mb-2 flex items-center gap-2 text-sm text-muted-foreground">
							<Database class="h-4 w-4" />
							Provider
						</div>
						<p class="text-xl font-semibold capitalize">{ragStats.provider || '—'}</p>
					</div>

					<!-- Total Vectors -->
					<div>
						<div class="mb-2 flex items-center gap-2 text-sm text-muted-foreground">
							<Activity class="h-4 w-4" />
							Total Vectors
						</div>
						<p class="text-xl font-semibold">
							{ragStats.vector_stats?.total_vectors?.toLocaleString() ?? '0'}
						</p>
					</div>

					<!-- Dimensions -->
					<div>
						<div class="mb-2 flex items-center gap-2 text-sm text-muted-foreground">
							<TrendingUp class="h-4 w-4" />
							Dimensions
						</div>
						<p class="text-xl font-semibold">
							{ragStats.vector_stats?.dimensions ?? '—'}
						</p>
					</div>
				</div>
			</div>
		{/if}

		<!-- Quick Actions -->
		<div class="rounded-xl border border-border bg-card p-6">
			<h2 class="mb-4 text-lg font-semibold">Quick Actions</h2>
			<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
				<a
					href="/admin/users"
					class="flex items-center gap-3 rounded-lg border border-border p-4 transition-colors hover:bg-accent"
				>
					<Users class="h-5 w-5 text-muted-foreground" />
					<span class="font-medium">Manage Users</span>
				</a>
				<a
					href="/admin/api-keys"
					class="flex items-center gap-3 rounded-lg border border-border p-4 transition-colors hover:bg-accent"
				>
					<Key class="h-5 w-5 text-muted-foreground" />
					<span class="font-medium">View API Keys</span>
				</a>
				<a
					href="/admin/settings"
					class="flex items-center gap-3 rounded-lg border border-border p-4 transition-colors hover:bg-accent"
				>
					<Server class="h-5 w-5 text-muted-foreground" />
					<span class="font-medium">System Settings</span>
				</a>
				<a
					href="/admin/logs"
					class="flex items-center gap-3 rounded-lg border border-border p-4 transition-colors hover:bg-accent"
				>
					<Activity class="h-5 w-5 text-muted-foreground" />
					<span class="font-medium">View Logs</span>
				</a>
			</div>
		</div>
	</div>
{/if}
