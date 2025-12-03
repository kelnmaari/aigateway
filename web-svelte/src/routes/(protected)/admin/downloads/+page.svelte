<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		Download,
		RefreshCw,
		Loader2,
		Trash2,
		CheckCircle,
		XCircle,
		Clock,
		HardDrive,
		Pause,
		Play,
		AlertTriangle
	} from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	interface DownloadItem {
		id: string;
		model_name: string;
		file_name: string;
		status: 'pending' | 'downloading' | 'completed' | 'failed' | 'paused';
		progress: number;
		downloaded_bytes: number;
		total_bytes: number;
		speed: number;
		eta: number;
		error?: string;
		started_at: string;
		completed_at?: string;
	}

	let downloads = $state<DownloadItem[]>([]);
	let isLoading = $state(true);
	let refreshInterval: number | null = null;

	onMount(async () => {
		await loadDownloads();
		// Auto-refresh every 2 seconds
		refreshInterval = setInterval(loadDownloads, 2000);
	});

	onDestroy(() => {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}
	});

	async function loadDownloads() {
		try {
			const response = await fetch('/api/ui/huggingface/downloads', {
				headers: {
					Authorization: `Bearer ${localStorage.getItem('access_token')}`
				}
			});
			if (response.ok) {
				const data = await response.json();
				downloads = data.downloads || [];
			}
		} catch (error) {
			console.error('Failed to load downloads:', error);
		} finally {
			isLoading = false;
		}
	}

	async function cancelDownload(id: string) {
		try {
			await fetch(`/api/ui/huggingface/downloads/${id}/cancel`, {
				method: 'POST',
				headers: {
					Authorization: `Bearer ${localStorage.getItem('access_token')}`
				}
			});
			await loadDownloads();
		} catch (error) {
			console.error('Failed to cancel download:', error);
		}
	}

	async function clearCompleted() {
		// Filter out completed and failed downloads locally since API might not have clear endpoint
		downloads = downloads.filter((d) => d.status === 'downloading' || d.status === 'pending');
	}

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	function formatSpeed(bytesPerSecond: number): string {
		return formatBytes(bytesPerSecond) + '/s';
	}

	function formatETA(seconds: number): string {
		if (seconds <= 0) return '--';
		if (seconds < 60) return `${Math.round(seconds)}s`;
		if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
		return `${Math.round(seconds / 3600)}h ${Math.round((seconds % 3600) / 60)}m`;
	}

	function getStatusIcon(status: string) {
		switch (status) {
			case 'completed':
				return CheckCircle;
			case 'failed':
				return XCircle;
			case 'downloading':
				return Download;
			case 'paused':
				return Pause;
			default:
				return Clock;
		}
	}

	function getStatusClass(status: string): string {
		switch (status) {
			case 'completed':
				return 'text-green-500 bg-green-500/10';
			case 'failed':
				return 'text-red-500 bg-red-500/10';
			case 'downloading':
				return 'text-blue-500 bg-blue-500/10';
			case 'paused':
				return 'text-amber-500 bg-amber-500/10';
			default:
				return 'text-muted-foreground bg-muted';
		}
	}

	const activeDownloads = $derived(downloads.filter((d) => d.status === 'downloading'));
	const completedDownloads = $derived(downloads.filter((d) => d.status === 'completed'));
	const failedDownloads = $derived(downloads.filter((d) => d.status === 'failed'));
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-lg font-semibold">Model Downloads</h2>
			<p class="text-sm text-muted-foreground">HuggingFace model download queue</p>
		</div>
		<div class="flex gap-2">
			{#if completedDownloads.length > 0 || failedDownloads.length > 0}
				<Button variant="outline" size="sm" onclick={clearCompleted}>
					<Trash2 class="mr-2 h-4 w-4" />
					Clear Completed
				</Button>
			{/if}
			<Button variant="outline" size="sm" onclick={loadDownloads} disabled={isLoading}>
				<RefreshCw class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
				{m.common_refresh()}
			</Button>
		</div>
	</div>

	<!-- Stats -->
	<div class="grid gap-4 sm:grid-cols-4">
		<div class="rounded-lg border border-border bg-card p-4">
			<div class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10 text-blue-500">
					<Download class="h-5 w-5" />
				</div>
				<div>
					<p class="text-2xl font-bold">{activeDownloads.length}</p>
					<p class="text-xs text-muted-foreground">Active</p>
				</div>
			</div>
		</div>

		<div class="rounded-lg border border-border bg-card p-4">
			<div class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-500/10 text-amber-500">
					<Clock class="h-5 w-5" />
				</div>
				<div>
					<p class="text-2xl font-bold">{downloads.filter((d) => d.status === 'pending').length}</p>
					<p class="text-xs text-muted-foreground">Pending</p>
				</div>
			</div>
		</div>

		<div class="rounded-lg border border-border bg-card p-4">
			<div class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/10 text-green-500">
					<CheckCircle class="h-5 w-5" />
				</div>
				<div>
					<p class="text-2xl font-bold">{completedDownloads.length}</p>
					<p class="text-xs text-muted-foreground">Completed</p>
				</div>
			</div>
		</div>

		<div class="rounded-lg border border-border bg-card p-4">
			<div class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-red-500/10 text-red-500">
					<XCircle class="h-5 w-5" />
				</div>
				<div>
					<p class="text-2xl font-bold">{failedDownloads.length}</p>
					<p class="text-xs text-muted-foreground">Failed</p>
				</div>
			</div>
		</div>
	</div>

	<!-- Downloads List -->
	{#if isLoading && downloads.length === 0}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if downloads.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Download class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-lg font-medium">No active downloads</p>
			<p class="mt-1 text-muted-foreground">
				Start downloading models from the Models tab → HuggingFace Browser
			</p>
		</div>
	{:else}
		<div class="space-y-3">
			{#each downloads as download (download.id)}
				{@const StatusIcon = getStatusIcon(download.status)}
				<div class="rounded-lg border border-border bg-card p-4">
					<div class="flex items-start justify-between gap-4">
						<div class="flex items-start gap-3">
							<div class={cn('flex h-10 w-10 shrink-0 items-center justify-center rounded-lg', getStatusClass(download.status))}>
								{#if download.status === 'downloading'}
									<Loader2 class="h-5 w-5 animate-spin" />
								{:else}
									<StatusIcon class="h-5 w-5" />
								{/if}
							</div>
							<div>
								<h3 class="font-medium text-foreground">{download.model_name}</h3>
								<p class="text-sm text-muted-foreground">{download.file_name}</p>
								{#if download.error}
									<p class="mt-1 flex items-center gap-1 text-sm text-red-500">
										<AlertTriangle class="h-3.5 w-3.5" />
										{download.error}
									</p>
								{/if}
							</div>
						</div>

						<div class="flex items-center gap-2">
							{#if download.status === 'downloading' || download.status === 'pending'}
								<Button variant="ghost" size="sm" onclick={() => cancelDownload(download.id)}>
									<XCircle class="h-4 w-4" />
								</Button>
							{/if}
						</div>
					</div>

					{#if download.status === 'downloading'}
						<div class="mt-4">
							<div class="mb-2 flex items-center justify-between text-sm">
								<span class="text-muted-foreground">
									{formatBytes(download.downloaded_bytes)} / {formatBytes(download.total_bytes)}
								</span>
								<span class="font-medium text-foreground">{download.progress.toFixed(1)}%</span>
							</div>
							<div class="h-2 overflow-hidden rounded-full bg-muted">
								<div
									class="h-full bg-primary transition-all"
									style="width: {download.progress}%"
								></div>
							</div>
							<div class="mt-2 flex items-center justify-between text-xs text-muted-foreground">
								<span>{formatSpeed(download.speed)}</span>
								<span>ETA: {formatETA(download.eta)}</span>
							</div>
						</div>
					{:else if download.status === 'completed'}
						<div class="mt-3 text-sm text-muted-foreground">
							<span class="flex items-center gap-1">
								<HardDrive class="h-3.5 w-3.5" />
								{formatBytes(download.total_bytes)}
							</span>
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

