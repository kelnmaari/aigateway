<script lang="ts">
	import { onMount } from 'svelte';
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
		Database
	} from 'lucide-svelte';
	import { adminApi, type AdminStats, type SystemMetrics } from '$lib/api/admin';
	import { cn } from '$lib/utils';

	let stats = $state<AdminStats | null>(null);
	let metrics = $state<SystemMetrics | null>(null);
	let isLoading = $state(true);

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		isLoading = true;
		try {
			const [statsRes, metricsRes] = await Promise.allSettled([
				adminApi.getStats(),
				adminApi.getMetrics()
			]);

			if (statsRes.status === 'fulfilled') {
				stats = statsRes.value;
			}
			if (metricsRes.status === 'fulfilled') {
				metrics = metricsRes.value;
			}
		} catch (error) {
			console.error('Failed to load admin data:', error);
		} finally {
			isLoading = false;
		}
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
