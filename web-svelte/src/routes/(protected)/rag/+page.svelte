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
		Search,
		Settings,
		CheckCircle,
		AlertCircle,
		Clock,
		Zap
	} from 'lucide-svelte';
	import { ragApi, type RAGSource } from '$lib/api/rag';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	let sources = $state<RAGSource[]>([]);
	let isLoading = $state(true);

	// Modals
	let showCreateModal = $state(false);
	let showDeleteModal = $state(false);
	let sourceToDelete = $state<RAGSource | null>(null);

	// Create form
	let sourceName = $state('');
	let sourceDescription = $state('');
	let sourceType = $state<'api' | 'database' | 'file' | 'web'>('file');
	let isShared = $state(false);
	let isCreating = $state(false);
	let createError = $state('');
	let isTesting = $state(false);
	let testResult = $state<{ success: boolean; message: string } | null>(null);

	// API Config
	let apiUrl = $state('');
	let apiMethod = $state<'GET' | 'POST'>('GET');
	let apiAuthType = $state<'none' | 'bearer' | 'api_key' | 'basic'>('none');
	let apiToken = $state('');
	let apiUsername = $state('');
	let apiPassword = $state('');

	// Database Config
	let dbHost = $state('localhost');
	let dbPort = $state(5432);
	let dbName = $state('');
	let dbUsername = $state('');
	let dbPassword = $state('');
	let dbQuery = $state('');

	// File Config
	let selectedFiles = $state<File[]>([]);

	// Web Config
	let webUrl = $state('');
	let webDepth = $state(1);

	// Indexing Config
	let chunkSize = $state(512);
	let chunkOverlap = $state(50);

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

	function resetForm() {
		sourceName = '';
		sourceDescription = '';
		sourceType = 'file';
		isShared = false;
		createError = '';
		testResult = null;
		// API
		apiUrl = '';
		apiMethod = 'GET';
		apiAuthType = 'none';
		apiToken = '';
		apiUsername = '';
		apiPassword = '';
		// Database
		dbHost = 'localhost';
		dbPort = 5432;
		dbName = '';
		dbUsername = '';
		dbPassword = '';
		dbQuery = '';
		// File
		selectedFiles = [];
		// Web
		webUrl = '';
		webDepth = 1;
		// Indexing
		chunkSize = 512;
		chunkOverlap = 50;
	}

	function openCreateModal() {
		resetForm();
		showCreateModal = true;
	}

	function buildConfig(): Record<string, unknown> {
		const config: Record<string, unknown> = {
			indexing: {
				chunk_size: chunkSize,
				chunk_overlap: chunkOverlap
			}
		};

		switch (sourceType) {
			case 'api':
				config.api = {
					url: apiUrl,
					method: apiMethod,
					auth_type: apiAuthType,
					...(apiAuthType === 'bearer' || apiAuthType === 'api_key' ? { token: apiToken } : {}),
					...(apiAuthType === 'basic' ? { username: apiUsername, password: apiPassword } : {})
				};
				break;
			case 'database':
				config.database = {
					host: dbHost,
					port: dbPort,
					name: dbName,
					username: dbUsername,
					password: dbPassword,
					query: dbQuery
				};
				break;
			case 'file':
				config.file = {
					files: selectedFiles.map(f => f.name)
				};
				break;
			case 'web':
				config.web = {
					url: webUrl,
					depth: webDepth
				};
				break;
		}

		return config;
	}

	async function handleTestConnection() {
		isTesting = true;
		testResult = null;
		try {
			const config = buildConfig();
			const result = await ragApi.testConnection({
				type: sourceType,
				config
			});
			testResult = { success: true, message: result.message || 'Connection successful!' };
		} catch (error) {
			testResult = { success: false, message: error instanceof Error ? error.message : 'Connection failed' };
		} finally {
			isTesting = false;
		}
	}

	async function handleCreateSource() {
		if (!sourceName.trim()) {
			createError = 'Name is required';
			return;
		}

		isCreating = true;
		createError = '';

		try {
			const config = buildConfig();

			const newSource = await ragApi.createSource({
				name: sourceName.trim(),
				description: sourceDescription.trim(),
				type: sourceType,
				config,
				shared: isShared
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

	function openDeleteModal(source: RAGSource) {
		sourceToDelete = source;
		showDeleteModal = true;
	}

	async function confirmDelete() {
		if (!sourceToDelete) return;

		try {
			await ragApi.deleteSource(sourceToDelete.id);
			sources = sources.filter((s) => s.id !== sourceToDelete!.id);
			showDeleteModal = false;
			sourceToDelete = null;
		} catch (error) {
			console.error('Failed to delete:', error);
			alert('Failed to delete source');
		}
	}

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files) {
			selectedFiles = Array.from(input.files);
		}
	}

	function getTypeIcon(type: string) {
		switch (type) {
			case 'file':
				return FileText;
			case 'web':
				return Globe;
			case 'api':
				return Zap;
			case 'database':
				return Database;
			default:
				return Server;
		}
	}

	function getStatusInfo(status: string) {
		switch (status) {
			case 'active':
				return { icon: CheckCircle, class: 'text-green-500', bg: 'bg-green-500/10', label: 'Active' };
			case 'indexing':
				return { icon: Loader2, class: 'text-amber-500', bg: 'bg-amber-500/10', label: 'Indexing' };
			case 'error':
				return { icon: AlertCircle, class: 'text-red-500', bg: 'bg-red-500/10', label: 'Error' };
			case 'pending':
				return { icon: Clock, class: 'text-blue-500', bg: 'bg-blue-500/10', label: 'Pending' };
			default:
				return { icon: Clock, class: 'text-muted-foreground', bg: 'bg-muted', label: status };
		}
	}

	function formatNumber(num: number | undefined): string {
		if (!num) return '0';
		if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
		if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
		return num.toString();
	}
</script>

<svelte:head>
	<title>{m.nav_rag()} | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-foreground">RAG Data Sources</h1>
			<p class="mt-1 text-muted-foreground">Manage knowledge sources for Retrieval-Augmented Generation</p>
		</div>
		<div class="flex gap-2">
			<Button variant="outline" onclick={loadSources} disabled={isLoading}>
				<RefreshCw class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
				{m.common_refresh()}
			</Button>
			<Button onclick={openCreateModal}>
				<Plus class="mr-2 h-4 w-4" />
				Add Data Source
			</Button>
		</div>
	</div>

	<!-- Sources Table -->
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
				Add Data Source
			</Button>
		</div>
	{:else}
		<div class="rounded-lg border border-border">
			<table class="w-full">
				<thead class="border-b border-border bg-muted/50">
					<tr>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">{m.common_name()}</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Type</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">{m.common_status()}</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Last Sync</th>
						<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">Chunks</th>
						<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">Tokens</th>
						<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">{m.common_actions()}</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border">
					{#each sources as source (source.id)}
						{@const TypeIcon = getTypeIcon(source.type)}
						{@const statusInfo = getStatusInfo(source.status)}
						<tr class="hover:bg-muted/30">
							<td class="px-4 py-3">
								<div class="flex items-center gap-3">
									<div class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
										<TypeIcon class="h-4 w-4" />
									</div>
									<div>
										<p class="font-medium">{source.name}</p>
										{#if source.description}
											<p class="text-xs text-muted-foreground">{source.description}</p>
										{/if}
									</div>
								</div>
							</td>
							<td class="px-4 py-3 text-sm capitalize">{source.type}</td>
							<td class="px-4 py-3">
								<span class={cn('inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium', statusInfo.bg, statusInfo.class)}>
									<svelte:component this={statusInfo.icon} class={cn('h-3 w-3', source.status === 'indexing' && 'animate-spin')} />
									{statusInfo.label}
								</span>
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{source.last_indexed_at ? formatRelativeTime(source.last_indexed_at) : 'Never'}
							</td>
							<td class="px-4 py-3 text-right text-sm">{formatNumber(source.chunk_count)}</td>
							<td class="px-4 py-3 text-right text-sm">{formatNumber(source.token_count)}</td>
							<td class="px-4 py-3">
								<div class="flex justify-end gap-1">
									<Button
										variant="ghost"
										size="sm"
										onclick={() => handleReindex(source)}
										disabled={source.status === 'indexing'}
										title="Reindex"
									>
										<RefreshCw class="h-4 w-4" />
									</Button>
									<Button
										variant="ghost"
										size="sm"
										onclick={() => openDeleteModal(source)}
										class="text-destructive hover:bg-destructive/10"
										title="Delete"
									>
										<Trash2 class="h-4 w-4" />
									</Button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<!-- Create Modal -->
{#if showCreateModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showCreateModal = false)}
		role="dialog"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && (showCreateModal = false)}
	>
		<div class="my-8 w-full max-w-2xl rounded-xl border border-border bg-card shadow-xl">
			<div class="flex items-center justify-between border-b border-border px-6 py-4">
				<h2 class="text-lg font-semibold">Add Data Source</h2>
				<button onclick={() => (showCreateModal = false)} class="rounded p-1 text-muted-foreground hover:bg-accent">
					<X class="h-5 w-5" />
				</button>
			</div>

			<form onsubmit={(e) => { e.preventDefault(); handleCreateSource(); }} class="max-h-[70vh] overflow-y-auto p-6">
				{#if createError}
					<div class="mb-4 rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
						{createError}
					</div>
				{/if}

				<!-- Basic Info -->
				<div class="space-y-4">
					<div>
						<label for="source-name" class="mb-1.5 block text-sm font-medium">{m.common_name()} *</label>
						<input
							id="source-name"
							type="text"
							bind:value={sourceName}
							required
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						/>
					</div>

					<div>
						<label for="source-desc" class="mb-1.5 block text-sm font-medium">Description</label>
						<textarea
							id="source-desc"
							bind:value={sourceDescription}
							rows="2"
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						></textarea>
					</div>

					<div>
						<label class="mb-1.5 block text-sm font-medium">Source Type *</label>
						<select
							bind:value={sourceType}
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						>
							<option value="file">File Upload</option>
							<option value="api">REST API</option>
							<option value="database">PostgreSQL Database</option>
							<option value="web">Web Scraping</option>
						</select>
					</div>
				</div>

				<!-- API Configuration -->
				{#if sourceType === 'api'}
					<div class="mt-6 rounded-lg border border-border p-4">
						<h3 class="mb-4 flex items-center gap-2 font-medium">
							<Zap class="h-4 w-4" />
							API Configuration
						</h3>
						<div class="space-y-4">
							<div>
								<label for="api-url" class="mb-1.5 block text-sm font-medium">API URL *</label>
								<input
									id="api-url"
									type="url"
									bind:value={apiUrl}
									placeholder="https://api.example.com/data"
									class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
								/>
							</div>
							<div class="grid grid-cols-2 gap-4">
								<div>
									<label for="api-method" class="mb-1.5 block text-sm font-medium">HTTP Method</label>
									<select id="api-method" bind:value={apiMethod} class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm">
										<option value="GET">GET</option>
										<option value="POST">POST</option>
									</select>
								</div>
								<div>
									<label for="api-auth" class="mb-1.5 block text-sm font-medium">Authentication</label>
									<select id="api-auth" bind:value={apiAuthType} class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm">
										<option value="none">None</option>
										<option value="bearer">Bearer Token</option>
										<option value="api_key">API Key</option>
										<option value="basic">Basic Auth</option>
									</select>
								</div>
							</div>
							{#if apiAuthType === 'bearer' || apiAuthType === 'api_key'}
								<div>
									<label for="api-token" class="mb-1.5 block text-sm font-medium">Token / API Key</label>
									<input
										id="api-token"
										type="password"
										bind:value={apiToken}
										class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
									/>
								</div>
							{/if}
							{#if apiAuthType === 'basic'}
								<div class="grid grid-cols-2 gap-4">
									<div>
										<label for="api-username" class="mb-1.5 block text-sm font-medium">Username</label>
										<input id="api-username" type="text" bind:value={apiUsername} class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm" />
									</div>
									<div>
										<label for="api-password" class="mb-1.5 block text-sm font-medium">Password</label>
										<input id="api-password" type="password" bind:value={apiPassword} class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm" />
									</div>
								</div>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Database Configuration -->
				{#if sourceType === 'database'}
					<div class="mt-6 rounded-lg border border-border p-4">
						<h3 class="mb-4 flex items-center gap-2 font-medium">
							<Database class="h-4 w-4" />
							Database Configuration
						</h3>
						<div class="space-y-4">
							<div class="grid grid-cols-3 gap-4">
								<div class="col-span-2">
									<label for="db-host" class="mb-1.5 block text-sm font-medium">Host *</label>
									<input id="db-host" type="text" bind:value={dbHost} placeholder="localhost" class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm" />
								</div>
								<div>
									<label for="db-port" class="mb-1.5 block text-sm font-medium">Port *</label>
									<input id="db-port" type="number" bind:value={dbPort} class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm" />
								</div>
							</div>
							<div>
								<label for="db-name" class="mb-1.5 block text-sm font-medium">Database Name *</label>
								<input id="db-name" type="text" bind:value={dbName} class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm" />
							</div>
							<div class="grid grid-cols-2 gap-4">
								<div>
									<label for="db-user" class="mb-1.5 block text-sm font-medium">Username *</label>
									<input id="db-user" type="text" bind:value={dbUsername} class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm" />
								</div>
								<div>
									<label for="db-pass" class="mb-1.5 block text-sm font-medium">Password *</label>
									<input id="db-pass" type="password" bind:value={dbPassword} class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm" />
								</div>
							</div>
							<div>
								<label for="db-query" class="mb-1.5 block text-sm font-medium">SQL Query *</label>
								<textarea
									id="db-query"
									bind:value={dbQuery}
									rows="3"
									placeholder="SELECT id, title, content FROM documents"
									class="w-full rounded-lg border border-input bg-background px-3 py-2 font-mono text-sm"
								></textarea>
							</div>
						</div>
					</div>
				{/if}

				<!-- File Configuration -->
				{#if sourceType === 'file'}
					<div class="mt-6 rounded-lg border border-border p-4">
						<h3 class="mb-4 flex items-center gap-2 font-medium">
							<FileText class="h-4 w-4" />
							File Configuration
						</h3>
						<div>
							<label class="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-border py-8 transition-colors hover:border-primary hover:bg-primary/5">
								<FileText class="mb-2 h-8 w-8 text-muted-foreground" />
								<span class="text-sm text-muted-foreground">Click to select files</span>
								<span class="mt-1 text-xs text-muted-foreground">Supported: TXT, PDF, DOC, DOCX, MD</span>
								<input
									type="file"
									multiple
									accept=".txt,.pdf,.doc,.docx,.md"
									class="hidden"
									onchange={handleFileSelect}
								/>
							</label>
							{#if selectedFiles.length > 0}
								<div class="mt-3 space-y-2">
									{#each selectedFiles as file}
										<div class="flex items-center gap-2 rounded bg-muted px-3 py-2 text-sm">
											<FileText class="h-4 w-4" />
											<span>{file.name}</span>
											<span class="ml-auto text-muted-foreground">{(file.size / 1024).toFixed(1)} KB</span>
										</div>
									{/each}
								</div>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Web Configuration -->
				{#if sourceType === 'web'}
					<div class="mt-6 rounded-lg border border-border p-4">
						<h3 class="mb-4 flex items-center gap-2 font-medium">
							<Globe class="h-4 w-4" />
							Web Scraping Configuration
						</h3>
						<div class="space-y-4">
							<div>
								<label for="web-url" class="mb-1.5 block text-sm font-medium">Website URL *</label>
								<input
									id="web-url"
									type="url"
									bind:value={webUrl}
									placeholder="https://example.com"
									class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
								/>
							</div>
							<div>
								<label for="web-depth" class="mb-1.5 block text-sm font-medium">Crawl Depth</label>
								<input
									id="web-depth"
									type="number"
									bind:value={webDepth}
									min="1"
									max="5"
									class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
								/>
								<p class="mt-1 text-xs text-muted-foreground">Max depth: 5 levels</p>
							</div>
						</div>
					</div>
				{/if}

				<!-- Indexing Settings -->
				<div class="mt-6 rounded-lg border border-border p-4">
					<h3 class="mb-4 flex items-center gap-2 font-medium">
						<Settings class="h-4 w-4" />
						Indexing Settings
					</h3>
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="chunk-size" class="mb-1.5 block text-sm font-medium">Chunk Size</label>
							<input
								id="chunk-size"
								type="number"
								bind:value={chunkSize}
								min="100"
								max="2000"
								class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
							/>
						</div>
						<div>
							<label for="chunk-overlap" class="mb-1.5 block text-sm font-medium">Chunk Overlap</label>
							<input
								id="chunk-overlap"
								type="number"
								bind:value={chunkOverlap}
								min="0"
								max="500"
								class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
							/>
						</div>
					</div>
				</div>

				<!-- Shared Checkbox -->
				<div class="mt-4">
					<label class="flex items-center gap-2 text-sm">
						<input type="checkbox" bind:checked={isShared} class="rounded border-input" />
						Share with organization
					</label>
				</div>

				<!-- Test Result -->
				{#if testResult}
					<div class={cn('mt-4 rounded-lg p-3 text-sm', testResult.success ? 'bg-green-500/10 text-green-600' : 'bg-red-500/10 text-red-600')}>
						{testResult.message}
					</div>
				{/if}
			</form>

			<div class="flex justify-end gap-3 border-t border-border px-6 py-4">
				<Button variant="outline" type="button" onclick={() => (showCreateModal = false)}>
					{m.common_cancel()}
				</Button>
				{#if sourceType !== 'file'}
					<Button variant="outline" type="button" onclick={handleTestConnection} disabled={isTesting}>
						{#if isTesting}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						Test Connection
					</Button>
				{/if}
				<Button onclick={handleCreateSource} disabled={isCreating}>
					{#if isCreating}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					Save
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Delete Confirmation Modal -->
{#if showDeleteModal && sourceToDelete}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showDeleteModal = false)}
		role="dialog"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && (showDeleteModal = false)}
	>
		<div class="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl">
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Confirm Delete</h2>
				<button onclick={() => (showDeleteModal = false)} class="rounded p-1 text-muted-foreground hover:bg-accent">
					<X class="h-5 w-5" />
				</button>
			</div>
			<p class="text-muted-foreground">
				Are you sure you want to delete "<strong>{sourceToDelete.name}</strong>"? This action cannot be undone.
			</p>
			<div class="mt-6 flex justify-end gap-3">
				<Button variant="outline" onclick={() => (showDeleteModal = false)}>
					{m.common_cancel()}
				</Button>
				<Button variant="destructive" onclick={confirmDelete}>
					{m.common_delete()}
				</Button>
			</div>
		</div>
	</div>
{/if}
