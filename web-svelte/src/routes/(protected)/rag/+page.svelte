<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Database,
		Plus,
		RefreshCw,
		Loader2,
		Trash2,
		Play,
		FileText,
		Globe,
		Server,
		X,
		Search
	} from 'lucide-svelte';
	import { ragApi, type RAGSource } from '$lib/api/rag';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	let sources = $state<RAGSource[]>([]);
	let isLoading = $state(true);

	// Modals
	let showCreateModal = $state(false);
	let showDetailModal = $state(false);
	let selectedSource = $state<RAGSource | null>(null);

	// Create form
	let sourceName = $state('');
	let sourceType = $state<'file' | 'url'>('file');
	let sourceConfig = $state('');
	let isCreating = $state(false);
	let createError = $state('');

	onMount(async () => {
		await loadSources();
	});

	async function loadSources() {
		isLoading = true;
		try {
			const response = await ragApi.getSources();
			sources = response.sources || [];
		} catch (error) {
			console.error('Failed to load RAG sources:', error);
		} finally {
			isLoading = false;
		}
	}

	function openCreateModal() {
		sourceName = '';
		sourceType = 'file';
		sourceConfig = '';
		createError = '';
		showCreateModal = true;
	}

	async function handleCreateSource() {
		if (!sourceName.trim()) {
			createError = 'Name is required';
			return;
		}

		isCreating = true;
		createError = '';

		try {
			let config: Record<string, unknown> = {};
			if (sourceConfig.trim()) {
				config = JSON.parse(sourceConfig);
			}

			const newSource = await ragApi.createSource({
				name: sourceName.trim(),
				type: sourceType,
				config
			});

			sources = [newSource, ...sources];
			showCreateModal = false;
		} catch (error) {
			createError = error instanceof Error ? error.message : 'Failed to create source';
		} finally {
			isCreating = false;
		}
	}

	async function handleReindex(source: RAGSource) {
		try {
			await ragApi.reindexSource(source.id);
			sources = sources.map((s) => (s.id === source.id ? { ...s, status: 'indexing' } : s));
		} catch (error) {
			console.error('Failed to reindex:', error);
			alert('Failed to start reindexing');
		}
	}

	async function handleDelete(source: RAGSource) {
		if (!confirm(`Delete RAG source "${source.name}"?`)) return;

		try {
			await ragApi.deleteSource(source.id);
			sources = sources.filter((s) => s.id !== source.id);
		} catch (error) {
			console.error('Failed to delete:', error);
			alert('Failed to delete source');
		}
	}

	function getTypeIcon(type: string) {
		switch (type) {
			case 'file':
				return FileText;
			case 'url':
				return Globe;
			default:
				return Server;
		}
	}

	function getStatusClass(status: string) {
		switch (status) {
			case 'active':
				return 'bg-green-500/10 text-green-500';
			case 'indexing':
				return 'bg-amber-500/10 text-amber-500';
			case 'error':
				return 'bg-red-500/10 text-red-500';
			default:
				return 'bg-muted text-muted-foreground';
		}
	}
</script>

<svelte:head>
	<title>{m.nav_rag()} | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-foreground">{m.nav_rag()} Sources</h1>
			<p class="mt-1 text-muted-foreground">Manage your knowledge bases for retrieval-augmented generation</p>
		</div>
		<div class="flex gap-2">
			<Button variant="outline" onclick={loadSources} disabled={isLoading}>
				<RefreshCw class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
				{m.common_refresh()}
			</Button>
			<Button onclick={openCreateModal}>
				<Plus class="mr-2 h-4 w-4" />
				Add Source
			</Button>
		</div>
	</div>

	<!-- Sources Grid -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if sources.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Database class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-lg font-medium">No RAG sources yet</p>
			<p class="mt-1 text-muted-foreground">Add your first knowledge source to enable RAG</p>
			<Button class="mt-6" onclick={openCreateModal}>
				<Plus class="mr-2 h-4 w-4" />
				Add Source
			</Button>
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each sources as source (source.id)}
				{@const TypeIcon = getTypeIcon(source.type)}
				<div class="rounded-xl border border-border bg-card p-5">
					<div class="mb-4 flex items-start justify-between">
						<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
							<TypeIcon class="h-5 w-5" />
						</div>
						<span class={cn('rounded-full px-2 py-0.5 text-xs font-medium', getStatusClass(source.status))}>
							{#if source.status === 'indexing'}
								<Loader2 class="mr-1 inline h-3 w-3 animate-spin" />
							{/if}
							{source.status}
						</span>
					</div>

					<h3 class="font-semibold text-foreground">{source.name}</h3>
					<p class="mt-1 text-sm text-muted-foreground capitalize">{source.type} source</p>

					<div class="mt-4 flex items-center gap-4 text-xs text-muted-foreground">
						<span>{source.document_count} docs</span>
						<span>{source.chunk_count} chunks</span>
					</div>

					{#if source.last_indexed_at}
						<p class="mt-2 text-xs text-muted-foreground">
							Indexed {formatRelativeTime(source.last_indexed_at)}
						</p>
					{/if}

					{#if source.error_message}
						<p class="mt-2 text-xs text-red-500">{source.error_message}</p>
					{/if}

					<div class="mt-4 flex gap-2">
						<Button
							variant="outline"
							size="sm"
							onclick={() => handleReindex(source)}
							disabled={source.status === 'indexing'}
						>
							<RefreshCw class="mr-1 h-3 w-3" />
							Reindex
						</Button>
						<Button
							variant="outline"
							size="sm"
							onclick={() => handleDelete(source)}
							class="text-destructive hover:bg-destructive/10"
						>
							<Trash2 class="h-3 w-3" />
						</Button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Create Modal -->
{#if showCreateModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showCreateModal = false)}
		role="dialog"
		tabindex="-1"
	>
		<div class="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl">
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Add RAG Source</h2>
				<button onclick={() => (showCreateModal = false)} class="rounded p-1 text-muted-foreground hover:bg-accent">
					<X class="h-5 w-5" />
				</button>
			</div>

			<form onsubmit={(e) => { e.preventDefault(); handleCreateSource(); }} class="space-y-4">
				{#if createError}
					<div class="rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
						{createError}
					</div>
				{/if}

				<div class="space-y-2">
					<label for="source-name" class="text-sm font-medium">{m.common_name()} *</label>
					<input
						id="source-name"
						type="text"
						bind:value={sourceName}
						placeholder="My Knowledge Base"
						required
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>

				<div class="space-y-2">
					<label class="text-sm font-medium">Type</label>
					<div class="flex gap-2">
						<button
							type="button"
							onclick={() => (sourceType = 'file')}
							class={cn(
								'flex flex-1 items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors',
								sourceType === 'file' ? 'border-primary bg-primary/10 text-primary' : 'border-input'
							)}
						>
							<FileText class="h-4 w-4" />
							Files
						</button>
						<button
							type="button"
							onclick={() => (sourceType = 'url')}
							class={cn(
								'flex flex-1 items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors',
								sourceType === 'url' ? 'border-primary bg-primary/10 text-primary' : 'border-input'
							)}
						>
							<Globe class="h-4 w-4" />
							URL
						</button>
					</div>
				</div>

				<div class="space-y-2">
					<label for="source-config" class="text-sm font-medium">Configuration (JSON)</label>
					<textarea
						id="source-config"
						bind:value={sourceConfig}
						placeholder={`{"path": "/data/docs"}`}
						rows="3"
						class="w-full rounded-lg border border-input bg-background px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					></textarea>
				</div>

				<div class="flex justify-end gap-3 pt-2">
					<Button variant="outline" type="button" onclick={() => (showCreateModal = false)}>
						{m.common_cancel()}
					</Button>
					<Button type="submit" disabled={isCreating}>
						{#if isCreating}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						{m.common_create()}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}

