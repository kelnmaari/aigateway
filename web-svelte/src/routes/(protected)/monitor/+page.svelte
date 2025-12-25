<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		Activity,
		Cpu,
		HardDrive,
		Loader2,
		RefreshCw,
		Server,
		Zap,
		MemoryStick,
		Gauge
	} from 'lucide-svelte';
	import { api } from '$lib/api/client';
	import { cn, formatNumber, formatBytes } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	interface SystemMetrics {
		cpu_percent: number;
		memory_used: number;
		memory_total: number;
		memory_percent: number;
		goroutines: number;
		uptime_seconds: number;
		requests_per_second: number;
		active_connections: number;
		gpu?: {
			name: string;
			memory_used: number;
			memory_total: number;
			utilization: number;
			temperature: number;
		};
	}

	let metrics = $state<SystemMetrics | null>(null);
	let isLoading = $state(true);
	let pollInterval: ReturnType<typeof setInterval>;

	onMount(async () => {
		await loadMetrics();
		// Poll every 3 seconds
		pollInterval = setInterval(loadMetrics, 3000);
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
	});

	async function loadMetrics() {
		try {
			const response = await api.get<SystemMetrics>('/api/admin/performance/metrics');
			metrics = response;
		} catch (error) {
			console.error('Failed to load metrics:', error);
		} finally {
			isLoading = false;
		}
	}

	function formatUptime(seconds: number): string {
		const days = Math.floor(seconds / 86400);
		const hours = Math.floor((seconds % 86400) / 3600);
		const mins = Math.floor((seconds % 3600) / 60);
		
		if (days > 0) return `${days}d ${hours}h`;
		if (hours > 0) return `${hours}h ${mins}m`;
		return `${mins}m`;
	}

	function getStatusColor(percent: number): string {
		if (percent < 50) return 'text-green-500';
		if (percent < 80) return 'text-amber-500';
		return 'text-red-500';
	}

	function getProgressColor(percent: number): string {
		if (percent < 50) return 'bg-green-500';
		if (percent < 80) return 'bg-amber-500';
		return 'bg-red-500';
	}
</script>

<svelte:head>
	<title>System Monitor | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-[1600px] px-4 py-8 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-foreground">{m.monitor_title()}</h1>
			<p class="mt-1 text-muted-foreground">{m.monitor_subtitle()}</p>
		</div>
		<Button variant="outline" onclick={loadMetrics} disabled={isLoading}>
			<RefreshCw class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
			{m.common_refresh()}
		</Button>
	</div>

	{#if isLoading && !metrics}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if metrics}
		<!-- Main Stats Grid -->
		<div class="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<!-- CPU -->
			<div class="rounded-xl border border-border bg-card p-5">
				<div class="mb-3 flex items-center justify-between">
					<div class="flex items-center gap-2">
						<Cpu class="h-5 w-5 text-muted-foreground" />
						<span class="font-medium">CPU</span>
					</div>
					<span class={cn('text-2xl font-bold', getStatusColor(metrics.cpu_percent))}>
						{metrics.cpu_percent.toFixed(1)}%
					</span>
				</div>
				<div class="h-2 overflow-hidden rounded-full bg-muted">
					<div
						class={cn('h-full transition-all', getProgressColor(metrics.cpu_percent))}
						style="width: {metrics.cpu_percent}%"
					></div>
				</div>
			</div>

			<!-- Memory -->
			<div class="rounded-xl border border-border bg-card p-5">
				<div class="mb-3 flex items-center justify-between">
					<div class="flex items-center gap-2">
						<MemoryStick class="h-5 w-5 text-muted-foreground" />
						<span class="font-medium">Memory</span>
					</div>
					<span class={cn('text-2xl font-bold', getStatusColor(metrics.memory_percent))}>
						{metrics.memory_percent.toFixed(1)}%
					</span>
				</div>
				<div class="h-2 overflow-hidden rounded-full bg-muted">
					<div
						class={cn('h-full transition-all', getProgressColor(metrics.memory_percent))}
						style="width: {metrics.memory_percent}%"
					></div>
				</div>
				<p class="mt-2 text-xs text-muted-foreground">
					{formatBytes(metrics.memory_used)} / {formatBytes(metrics.memory_total)}
				</p>
			</div>

			<!-- Requests/sec -->
			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10 text-blue-500">
						<Zap class="h-5 w-5" />
					</div>
					<div>
						<p class="text-sm text-muted-foreground">Requests/sec</p>
						<p class="text-2xl font-bold">{metrics.requests_per_second.toFixed(1)}</p>
					</div>
				</div>
			</div>

			<!-- Uptime -->
			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/10 text-green-500">
						<Activity class="h-5 w-5" />
					</div>
					<div>
						<p class="text-sm text-muted-foreground">Uptime</p>
						<p class="text-2xl font-bold">{formatUptime(metrics.uptime_seconds)}</p>
					</div>
				</div>
			</div>
		</div>

		<!-- Secondary Stats -->
		<div class="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<Server class="h-5 w-5 text-muted-foreground" />
					<div>
						<p class="text-sm text-muted-foreground">Goroutines</p>
						<p class="text-xl font-bold">{formatNumber(metrics.goroutines)}</p>
					</div>
				</div>
			</div>

			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<Activity class="h-5 w-5 text-muted-foreground" />
					<div>
						<p class="text-sm text-muted-foreground">Active Connections</p>
						<p class="text-xl font-bold">{formatNumber(metrics.active_connections)}</p>
					</div>
				</div>
			</div>

			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<Gauge class="h-5 w-5 text-muted-foreground" />
					<div>
						<p class="text-sm text-muted-foreground">Health Status</p>
						<p class="text-xl font-bold text-green-500">Healthy</p>
					</div>
				</div>
			</div>
		</div>

		<!-- GPU (if available) -->
		{#if metrics.gpu}
			<div class="rounded-xl border border-border bg-card p-6">
				<h2 class="mb-4 flex items-center gap-2 font-semibold">
					<HardDrive class="h-5 w-5" />
					GPU: {metrics.gpu.name}
				</h2>

				<div class="grid gap-4 sm:grid-cols-3">
					<div>
						<p class="mb-1 text-sm text-muted-foreground">Utilization</p>
						<div class="flex items-center gap-2">
							<div class="h-2 flex-1 overflow-hidden rounded-full bg-muted">
								<div
									class={cn('h-full transition-all', getProgressColor(metrics.gpu.utilization))}
									style="width: {metrics.gpu.utilization}%"
								></div>
							</div>
							<span class="text-sm font-medium">{metrics.gpu.utilization}%</span>
						</div>
					</div>

					<div>
						<p class="mb-1 text-sm text-muted-foreground">Memory</p>
						<div class="flex items-center gap-2">
							<div class="h-2 flex-1 overflow-hidden rounded-full bg-muted">
								<div
									class={cn('h-full transition-all', getProgressColor((metrics.gpu.memory_used / metrics.gpu.memory_total) * 100))}
									style="width: {(metrics.gpu.memory_used / metrics.gpu.memory_total) * 100}%"
								></div>
							</div>
							<span class="text-sm font-medium">
								{formatBytes(metrics.gpu.memory_used)} / {formatBytes(metrics.gpu.memory_total)}
							</span>
						</div>
					</div>

					<div>
						<p class="mb-1 text-sm text-muted-foreground">Temperature</p>
						<p class={cn('text-xl font-bold', metrics.gpu.temperature > 80 ? 'text-red-500' : 'text-foreground')}>
							{metrics.gpu.temperature}°C
						</p>
					</div>
				</div>
			</div>
		{/if}
	{:else}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Server class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-lg font-medium">Unable to load metrics</p>
			<p class="mt-1 text-muted-foreground">Check server connection</p>
		</div>
	{/if}
</div>

