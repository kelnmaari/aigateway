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
	import { adminApi, type AdminStats, type SystemMetrics, type RAGStats, type GPUMetricsResponse, type GPUDevice, type BackendStatusResponse, type DockerImagesResponse, type DockerImageStatus } from '$lib/api/admin';
	import { cn } from '$lib/utils';

	let stats = $state<AdminStats | null>(null);
	let metrics = $state<SystemMetrics | null>(null);
	let ragStats = $state<RAGStats | null>(null);
	let gpuMetrics = $state<GPUMetricsResponse | null>(null);
	let backendStatus = $state<BackendStatusResponse | null>(null);
	let dockerImages = $state<DockerImagesResponse | null>(null);
	let pullingImages = $state<Set<string>>(new Set());
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
			const [statsRes, metricsRes, ragRes, gpuRes, backendRes, dockerImagesRes] = await Promise.allSettled([
				adminApi.getStats(),
				adminApi.getMetrics(),
				adminApi.getRAGStats(),
				adminApi.getGPUMetrics(),
				adminApi.getBackendStatus(),
				adminApi.getDockerImages()
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
			if (backendRes.status === 'fulfilled') {
				backendStatus = backendRes.value;
			}
			if (dockerImagesRes.status === 'fulfilled') {
				dockerImages = dockerImagesRes.value;
			}
		} catch (error) {
			console.error('Failed to load admin data:', error);
		} finally {
			isLoading = false;
		}
	}

	async function updateGPUMetrics() {
		try {
			const [gpuRes, backendRes, dockerRes] = await Promise.allSettled([
				adminApi.getGPUMetrics(),
				adminApi.getBackendStatus(),
				adminApi.getDockerImages()
			]);
			if (gpuRes.status === 'fulfilled') {
				gpuMetrics = gpuRes.value;
			}
			if (backendRes.status === 'fulfilled') {
				backendStatus = backendRes.value;
			}
			if (dockerRes.status === 'fulfilled') {
				dockerImages = dockerRes.value;
			}
		} catch (error) {
			// Silently fail for GPU/health updates
		}
	}

	async function pullDockerImage(image: string) {
		pullingImages = new Set([...pullingImages, image]);
		try {
			await adminApi.pullDockerImage(image);
			// Poll for completion every 5 seconds
			const pollInterval = setInterval(async () => {
				try {
					const res = await adminApi.getDockerImages();
					dockerImages = res;
					const img = res.images.find(i => i.image === image);
					if (img?.exists) {
						clearInterval(pollInterval);
						pullingImages = new Set([...pullingImages].filter(i => i !== image));
					}
				} catch {
					// ignore
				}
			}, 5000);
			// Timeout after 30 minutes
			setTimeout(() => {
				clearInterval(pollInterval);
				pullingImages = new Set([...pullingImages].filter(i => i !== image));
			}, 30 * 60 * 1000);
		} catch (error) {
			console.error('Failed to start image pull:', error);
			pullingImages = new Set([...pullingImages].filter(i => i !== image));
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

		<!-- Docker Inference Backend Status -->
		{#if backendStatus}
			<div class="rounded-xl border border-border bg-card p-6">
				<div class="mb-4 flex items-center justify-between">
					<h2 class="text-lg font-semibold flex items-center gap-2">
						<BrainCircuit class="h-5 w-5 text-purple-500" />
						LLM Inference Backend
					</h2>
					<span class={cn("text-sm font-medium flex items-center gap-1.5", backendStatus.ready ? 'text-green-500' : 'text-amber-500')}>
						{#if backendStatus.ready}
							<CheckCircle class="h-4 w-4" />
							Ready
						{:else}
							<AlertCircle class="h-4 w-4" />
							Not Configured
						{/if}
					</span>
				</div>
				
				<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
					<!-- Backend Type -->
					<div class="rounded-lg border border-border/50 p-3">
						<div class="mb-1 text-xs font-medium text-muted-foreground">Backend Type</div>
						<div class="flex items-center gap-2">
							<Server class="h-4 w-4 text-blue-500" />
							<span class="font-medium">Docker Containers</span>
						</div>
					</div>

					<!-- Running Models -->
					<div class="rounded-lg border border-border/50 p-3">
						<div class="mb-1 text-xs font-medium text-muted-foreground">Running Models</div>
						<div class="flex items-center gap-2">
							{#if (backendStatus.running_models ?? 0) > 0}
								<CheckCircle class="h-4 w-4 text-green-500" />
								<span class="font-medium text-green-500">{backendStatus.running_models}</span>
							{:else}
								<AlertCircle class="h-4 w-4 text-amber-500" />
								<span class="font-medium text-amber-500">None</span>
							{/if}
						</div>
					</div>

					<!-- Loaded Models -->
					<div class="rounded-lg border border-border/50 p-3">
						<div class="mb-1 text-xs font-medium text-muted-foreground">Loaded Models</div>
						<div class="flex items-center gap-2">
							<span class="font-medium">{backendStatus.loaded_models ?? 0}</span>
						</div>
					</div>

					<!-- Max Concurrent -->
					<div class="rounded-lg border border-border/50 p-3">
						<div class="mb-1 text-xs font-medium text-muted-foreground">Max Concurrent</div>
						<span class="font-medium">{backendStatus.max_running_models ?? 'Unlimited'}</span>
					</div>
				</div>

				<!-- Docker Images Status -->
				{#if dockerImages?.images}
					<div class="mt-4 pt-4 border-t border-border/50">
						<div class="text-xs font-medium text-muted-foreground mb-3">Inference Provider Images:</div>
						<div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
							{#each dockerImages.images as img}
								{@const isPulling = pullingImages.has(img.image)}
								<div class="flex items-center justify-between rounded-lg border border-border/50 p-2.5">
									<div class="flex items-center gap-2 min-w-0 flex-1">
										{#if img.exists}
											<CheckCircle class="h-4 w-4 text-green-500 flex-shrink-0" />
										{:else if isPulling}
											<Loader2 class="h-4 w-4 text-blue-500 animate-spin flex-shrink-0" />
										{:else}
											<XCircle class="h-4 w-4 text-red-500 flex-shrink-0" />
										{/if}
										<div class="min-w-0">
											<div class="text-sm font-medium capitalize">{img.provider}</div>
											<div class="text-xs text-muted-foreground truncate" title={img.image}>{img.image.split('/').pop()}</div>
										</div>
									</div>
									<div class="flex items-center gap-2 flex-shrink-0 ml-2">
										{#if img.exists}
											<span class="text-xs text-muted-foreground">{img.size}</span>
										{:else if isPulling}
											<span class="text-xs text-blue-500">Pulling...</span>
										{:else}
											<button
												onclick={() => pullDockerImage(img.image)}
												class="px-2 py-1 text-xs bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors"
											>
												Pull
											</button>
										{/if}
									</div>
								</div>
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
