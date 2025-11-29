<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		Download,
		Plus,
		X,
		RefreshCw,
		Loader2,
		Trash2,
		Play,
		Square,
		CheckCircle,
		AlertCircle,
		Search
	} from 'lucide-svelte';
	import { downloadsApi, type Download as DownloadItem } from '$lib/api/downloads';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	let downloads = $state<DownloadItem[]>([]);
	let isLoading = $state(true);
	let pollInterval: ReturnType<typeof setInterval>;

	// Modal
	let showAddModal = $state(false);
	let modelName = $state('');
	let downloadSource = $state<'ollama' | 'huggingface'>('ollama');
	let isStarting = $state(false);
	let addError = $state('');

	onMount(async () => {
		await loadDownloads();
		// Poll for updates
		pollInterval = setInterval(loadDownloads, 3000);
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
	});

	async function loadDownloads() {
		try {
			const response = await downloadsApi.getDownloads();
			downloads = response.downloads || [];
		} catch (error) {
			console.error('Failed to load downloads:', error);
		} finally {
			isLoading = false;
		}
	}

	function openAddModal() {
		modelName = '';
		addError = '';
		showAddModal = true;
	}

	async function handleStartDownload() {
		if (!modelName.trim()) {
			addError = 'Model name is required';
			return;
		}

		isStarting = true;
		addError = '';

		try {
			if (downloadSource === 'ollama') {
				// Pull via Ollama/Yzma
				await downloadsApi.pullOllamaModel(modelName.trim());
			} else {
				// Download from HuggingFace
				await downloadsApi.startDownload(modelName.trim());
			}
			await loadDownloads();
			showAddModal = false;
			modelName = '';
		} catch (error) {
			addError = error instanceof Error ? error.message : 'Failed to start download';
		} finally {
			isStarting = false;
		}
	}

	async function handleCancel(download: DownloadItem) {
		try {
			await downloadsApi.cancelDownload(download.id);
			downloads = downloads.map((d) => (d.id === download.id ? { ...d, status: 'cancelled' } : d));
		} catch (error) {
			console.error('Failed to cancel:', error);
		}
	}

	async function handlePause(download: DownloadItem) {
		try {
			await downloadsApi.pauseDownload(download.id);
			downloads = downloads.map((d) => (d.id === download.id ? { ...d, status: 'pending' } : d));
		} catch (error) {
			console.error('Failed to pause:', error);
		}
	}

	function formatSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const units = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(1024));
		return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
	}

	function formatSpeed(bytesPerSec: number): string {
		return `${formatSize(bytesPerSec)}/s`;
	}

	function formatEta(seconds: number): string {
		if (seconds < 60) return `${seconds}s`;
		if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
		return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
	}

	function getStatusIcon(status: string) {
		switch (status) {
			case 'completed':
				return CheckCircle;
			case 'failed':
			case 'cancelled':
				return AlertCircle;
			case 'downloading':
				return Loader2;
			default:
				return Download;
		}
	}

	function getStatusClass(status: string) {
		switch (status) {
			case 'completed':
				return 'text-green-500';
			case 'failed':
			case 'cancelled':
				return 'text-red-500';
			case 'downloading':
				return 'text-blue-500';
			default:
				return 'text-muted-foreground';
		}
	}
</script>

<svelte:head>
	<title>{m.admin_downloads()} | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-foreground">{m.admin_downloads()}</h1>
			<p class="mt-1 text-muted-foreground">Download and manage AI models</p>
		</div>
		<Button onclick={openAddModal}>
			<Plus class="mr-2 h-4 w-4" />
			Download Model
		</Button>
	</div>

	<!-- Downloads List -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if downloads.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Download class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-lg font-medium">No downloads</p>
			<p class="mt-1 text-muted-foreground">Start downloading a model to see it here</p>
			<Button class="mt-6" onclick={openAddModal}>
				<Plus class="mr-2 h-4 w-4" />
				Download Model
			</Button>
		</div>
	{:else}
		<div class="space-y-4">
			{#each downloads as download (download.id)}
				{@const StatusIcon = getStatusIcon(download.status)}
				<div class="rounded-xl border border-border bg-card p-5">
					<div class="flex items-start justify-between">
						<div class="flex items-start gap-4">
							<div class={cn('mt-1', getStatusClass(download.status))}>
								<StatusIcon class={cn('h-6 w-6', download.status === 'downloading' && 'animate-spin')} />
							</div>
							<div class="flex-1">
								<h3 class="font-semibold text-foreground">{download.model_name}</h3>
								<p class="text-sm text-muted-foreground capitalize">{download.source} • {download.status}</p>

								{#if download.status === 'downloading'}
									<div class="mt-3">
										<div class="mb-1 flex items-center justify-between text-sm">
											<span>
												{formatSize(download.downloaded_size)} / {formatSize(download.total_size)}
											</span>
											<span>
												{download.progress.toFixed(1)}%
												{#if download.speed}
													• {formatSpeed(download.speed)}
												{/if}
												{#if download.eta}
													• ETA: {formatEta(download.eta)}
												{/if}
											</span>
										</div>
										<div class="h-2 overflow-hidden rounded-full bg-muted">
											<div
												class="h-full bg-primary transition-all"
												style="width: {download.progress}%"
											></div>
										</div>
									</div>
								{/if}

								{#if download.error_message}
									<p class="mt-2 text-sm text-red-500">{download.error_message}</p>
								{/if}

								{#if download.completed_at}
									<p class="mt-2 text-xs text-muted-foreground">
										Completed {formatRelativeTime(download.completed_at)}
									</p>
								{/if}
							</div>
						</div>

						<div class="flex gap-2">
							{#if download.status === 'downloading'}
								<Button variant="outline" size="sm" onclick={() => handleCancel(download)}>
									<Square class="mr-1 h-3 w-3" />
									Cancel
								</Button>
							{:else if download.status === 'failed' || download.status === 'cancelled'}
								<Button variant="outline" size="sm" onclick={() => handleRetry(download)}>
									<RefreshCw class="mr-1 h-3 w-3" />
									Retry
								</Button>
							{/if}
							{#if download.status !== 'downloading'}
								<Button
									variant="outline"
									size="sm"
									onclick={() => handleDelete(download)}
									class="text-destructive hover:bg-destructive/10"
								>
									<Trash2 class="h-4 w-4" />
								</Button>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Add Modal -->
{#if showAddModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showAddModal = false)}
		role="dialog"
		tabindex="-1"
	>
		<div class="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl">
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Download Model</h2>
				<button onclick={() => (showAddModal = false)} class="rounded p-1 text-muted-foreground hover:bg-accent">
					<X class="h-5 w-5" />
				</button>
			</div>

			<form onsubmit={(e) => { e.preventDefault(); handleStartDownload(); }} class="space-y-4">
				{#if addError}
					<div class="rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
						{addError}
					</div>
				{/if}

				<div class="space-y-2">
					<label class="text-sm font-medium">Source</label>
					<div class="flex gap-2">
						<button
							type="button"
							onclick={() => (downloadSource = 'ollama')}
							class={cn(
								'flex-1 rounded-lg border px-3 py-2 text-sm transition-colors',
								downloadSource === 'ollama' ? 'border-primary bg-primary/10 text-primary' : 'border-input'
							)}
						>
							Ollama
						</button>
						<button
							type="button"
							onclick={() => (downloadSource = 'huggingface')}
							class={cn(
								'flex-1 rounded-lg border px-3 py-2 text-sm transition-colors',
								downloadSource === 'huggingface' ? 'border-primary bg-primary/10 text-primary' : 'border-input'
							)}
						>
							Hugging Face
						</button>
					</div>
				</div>

				<div class="space-y-2">
					<label for="model-name" class="text-sm font-medium">Model {downloadSource === 'ollama' ? 'Name' : 'ID'} *</label>
					<input
						id="model-name"
						type="text"
						bind:value={modelName}
						placeholder={downloadSource === 'ollama' ? 'llama3.2:latest' : 'TheBloke/Llama-2-7B-GGUF'}
						required
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					/>
					<p class="text-xs text-muted-foreground">
						{downloadSource === 'ollama'
							? 'Enter an Ollama model name (e.g., llama3.2, mistral, codellama)'
							: 'Enter a Hugging Face model ID (e.g., TheBloke/Llama-2-7B-GGUF)'}
					</p>
				</div>

				<div class="flex justify-end gap-3 pt-2">
					<Button variant="outline" type="button" onclick={() => (showAddModal = false)}>
						{m.common_cancel()}
					</Button>
					<Button type="submit" disabled={isStarting}>
						{#if isStarting}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						{downloadSource === 'ollama' ? 'Pull Model' : 'Start Download'}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}

