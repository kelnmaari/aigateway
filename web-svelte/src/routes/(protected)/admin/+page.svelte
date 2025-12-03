<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Users,
		Key,
		Server,
		Activity,
		TrendingUp,
		Clock,
		Cpu,
		HardDrive,
		Loader2
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

	function formatUptime(seconds: number | undefined): string {
		if (seconds == null) return '—';
		const days = Math.floor(seconds / 86400);
		const hours = Math.floor((seconds % 86400) / 3600);
		const mins = Math.floor((seconds % 3600) / 60);
		if (days > 0) return `${days}d ${hours}h ${mins}m`;
		if (hours > 0) return `${hours}h ${mins}m`;
		return `${mins}m`;
	}

	function formatBytes(bytes: number | undefined): string {
		if (bytes == null) return '—';
		const gb = bytes / (1024 * 1024 * 1024);
		if (gb >= 1) return `${gb.toFixed(1)} GB`;
		const mb = bytes / (1024 * 1024);
		return `${mb.toFixed(0)} MB`;
	}

	function formatPercent(value: number | undefined): string {
		if (value == null) return '0';
		return value.toFixed(1);
	}

	function getPercentValue(value: number | undefined): number {
		return value ?? 0;
	}

	// Derived memory percentage
	const memPercent = $derived(
		metrics?.memory_total ? ((metrics.memory_used ?? 0) / metrics.memory_total) * 100 : 0
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
						<p class="text-2xl font-bold">{stats?.users ?? 0}</p>
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
						<p class="text-2xl font-bold">{stats?.api_keys ?? 0}</p>
					</div>
				</div>
			</div>

			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/10 text-purple-500">
						<Server class="h-5 w-5" />
					</div>
					<div>
						<p class="text-sm text-muted-foreground">Models</p>
						<p class="text-2xl font-bold">{stats?.models ?? 0}</p>
					</div>
				</div>
			</div>

			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-500/10 text-amber-500">
						<Activity class="h-5 w-5" />
					</div>
					<div>
						<p class="text-sm text-muted-foreground">Requests Today</p>
						<p class="text-2xl font-bold">{stats?.requests_today ?? 0}</p>
					</div>
				</div>
			</div>
		</div>

		<!-- System Metrics -->
		{#if metrics}
			<div class="rounded-xl border border-border bg-card p-6">
				<h2 class="mb-4 text-lg font-semibold">System Health</h2>
				<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
					<div>
						<div class="mb-2 flex items-center justify-between">
							<span class="flex items-center gap-2 text-sm text-muted-foreground">
								<Cpu class="h-4 w-4" />
								CPU
							</span>
							<span class="text-sm font-medium">{formatPercent(metrics.cpu_percent)}%</span>
						</div>
						<div class="h-2 overflow-hidden rounded-full bg-muted">
							<div
								class={cn(
									'h-full transition-all',
									getPercentValue(metrics.cpu_percent) > 80 ? 'bg-red-500' : getPercentValue(metrics.cpu_percent) > 50 ? 'bg-amber-500' : 'bg-green-500'
								)}
								style="width: {Math.min(getPercentValue(metrics.cpu_percent), 100)}%"
							></div>
						</div>
					</div>

					<div>
						<div class="mb-2 flex items-center justify-between">
							<span class="flex items-center gap-2 text-sm text-muted-foreground">
								<HardDrive class="h-4 w-4" />
								Memory
							</span>
							<span class="text-sm font-medium">
								{formatBytes(metrics.memory_used)} / {formatBytes(metrics.memory_total)}
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

					<div>
						<div class="mb-2 flex items-center gap-2 text-sm text-muted-foreground">
							<Clock class="h-4 w-4" />
							Uptime
						</div>
						<p class="text-xl font-semibold">{formatUptime(metrics.uptime)}</p>
					</div>

					<div>
						<div class="mb-2 flex items-center gap-2 text-sm text-muted-foreground">
							<TrendingUp class="h-4 w-4" />
							Requests/sec
						</div>
						<p class="text-xl font-semibold">{formatPercent(metrics.requests_per_second)}</p>
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

