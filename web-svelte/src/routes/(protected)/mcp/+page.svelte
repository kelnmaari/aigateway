<script lang="ts">
	import { onMount } from 'svelte';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import {
		faServer,
		faPlus,
		faPlay,
		faSquare,
		faArrowsRotate,
		faSpinner,
		faTrash,
		faGear,
		faWrench,
		faDatabase,
		faXmark,
		faMagnifyingGlass,
		faArrowUpRightFromSquare,
		faCode,
		faRocket,
		faCloud,
		faBrain,
		faBoxOpen,
		faChevronLeft,
		faChevronRight,
		faCopy,
		faCheck,
		faTag,
		faGlobe
	} from '@fortawesome/free-solid-svg-icons';
	import { faGithub } from '@fortawesome/free-brands-svg-icons';
	import { mcpApi, type MCPServer } from '$lib/api/mcp';
	import { cn, formatRelativeTime, debounce, copyToClipboard } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import { authStore } from '$lib';
	import * as m from '$lib/paraglide/messages';

	// Admin check
	const isAdmin = $derived(authStore.isAdmin);

	let servers = $state<MCPServer[]>([]);
	let categories = $state<string[]>([]);
	let totalServers = $state(0);
	let isLoading = $state(true);

	// Filters
	let searchQuery = $state('');
	let categoryFilter = $state('');
	let sortBy = $state<'created_at' | 'name' | 'category'>('created_at');
	let sortOrder = $state<'asc' | 'desc'>('desc');

	// Pagination
	let currentPage = $state(0);
	let pageSize = 12;

	// Modals
	let showCreateModal = $state(false);
	let showDetailModal = $state(false);
	let selectedServer = $state<MCPServer | null>(null);
	let copiedGuide = $state(false);

	// Create form
	let serverName = $state('');
	let serverDescription = $state('');
	let serverCategory = $state('');
	let serverType = $state<'stdio' | 'sse'>('stdio');
	let serverCommand = $state('');
	let serverArgs = $state('');
	let serverUrl = $state('');
	let serverTags = $state('');
	let serverInstallGuide = $state('');
	let serverGithubUrl = $state('');
	let serverWebsiteUrl = $state('');
	let isCreating = $state(false);
	let createError = $state('');

	onMount(async () => {
		await Promise.all([loadCategories(), loadServers()]);
	});

	const debouncedSearch = debounce(() => {
		currentPage = 0;
		loadServers();
	}, 300);

	async function loadCategories() {
		try {
			const response = await mcpApi.getCategories();
			categories = response.categories || [];
		} catch (error) {
			console.error('Failed to load categories:', error);
		}
	}

	async function loadServers() {
		isLoading = true;
		try {
			const response = await mcpApi.getServers({
				limit: pageSize,
				offset: currentPage * pageSize,
				search: searchQuery || undefined,
				category: categoryFilter || undefined,
				sort_by: sortBy,
				sort_order: sortOrder,
				active_only: true
			});
			servers = response.servers || [];
			totalServers = response.total || servers.length;
		} catch (error) {
			console.error('Failed to load MCP servers:', error);
			servers = [];
			totalServers = 0;
		} finally {
			isLoading = false;
		}
	}

	function clearFilters() {
		searchQuery = '';
		categoryFilter = '';
		sortBy = 'created_at';
		sortOrder = 'desc';
		currentPage = 0;
		loadServers();
	}

	function openCreateModal() {
		serverName = '';
		serverDescription = '';
		serverCategory = '';
		serverType = 'stdio';
		serverCommand = '';
		serverArgs = '';
		serverUrl = '';
		serverTags = '';
		serverInstallGuide = '';
		serverGithubUrl = '';
		serverWebsiteUrl = '';
		createError = '';
		showCreateModal = true;
	}

	async function handleCreateServer() {
		if (!serverName.trim()) {
			createError = 'Name is required';
			return;
		}

		if (serverType === 'stdio' && !serverCommand.trim()) {
			createError = 'Command is required for stdio servers';
			return;
		}

		if (serverType === 'sse' && !serverUrl.trim()) {
			createError = 'URL is required for SSE servers';
			return;
		}

		isCreating = true;
		createError = '';

		try {
			const newServer = await mcpApi.createServer({
				name: serverName.trim(),
				description: serverDescription.trim() || undefined,
				category: serverCategory || undefined,
				type: serverType,
				command: serverType === 'stdio' ? serverCommand.trim() : undefined,
				args: serverType === 'stdio' && serverArgs.trim()
					? serverArgs.split(' ').filter(Boolean)
					: undefined,
				url: serverType === 'sse' ? serverUrl.trim() : undefined,
				enabled: true,
				tags: serverTags.trim() ? serverTags.split(',').map(t => t.trim()).filter(Boolean) : undefined,
				installation_guide: serverInstallGuide.trim() || undefined,
				github_url: serverGithubUrl.trim() || undefined,
				website_url: serverWebsiteUrl.trim() || undefined
			});

			servers = [newServer, ...servers];
			totalServers++;
			showCreateModal = false;
		} catch (error) {
			createError = error instanceof Error ? error.message : 'Failed to create server';
		} finally {
			isCreating = false;
		}
	}

	async function handleStartServer(server: MCPServer) {
		try {
			await mcpApi.startServer(server.id);
			servers = servers.map((s) => (s.id === server.id ? { ...s, status: 'running' } : s));
		} catch (error) {
			console.error('Failed to start server:', error);
			alert(m.alert_failed_start_server());
		}
	}

	async function handleStopServer(server: MCPServer) {
		try {
			await mcpApi.stopServer(server.id);
			servers = servers.map((s) => (s.id === server.id ? { ...s, status: 'stopped' } : s));
		} catch (error) {
			console.error('Failed to stop server:', error);
			alert(m.alert_failed_stop_server());
		}
	}

	async function handleDeleteServer(server: MCPServer) {
		if (!confirm(m.confirm_delete_mcp({ name: server.name }))) return;

		try {
			await mcpApi.deleteServer(server.id);
			servers = servers.filter((s) => s.id !== server.id);
			totalServers--;
		} catch (error) {
			console.error('Failed to delete server:', error);
			alert(m.alert_failed_delete_server());
		}
	}

	async function viewDetails(server: MCPServer) {
		try {
			// Fetch full details
			const fullServer = await mcpApi.getServer(server.id);
			selectedServer = fullServer;
			showDetailModal = true;
		} catch {
			selectedServer = server;
			showDetailModal = true;
		}
	}

	async function copyInstallGuide() {
		if (selectedServer?.installation_guide) {
			const success = await copyToClipboard(selectedServer.installation_guide);
			if (success) {
				copiedGuide = true;
				setTimeout(() => { copiedGuide = false; }, 2000);
			}
		}
	}

	function getCategoryIcon(category: string | undefined) {
		switch (category) {
			case 'development': return faCode;
			case 'productivity': return faRocket;
			case 'database': return faDatabase;
			case 'cloud': return faCloud;
			case 'ai': return faBrain;
			default: return faBoxOpen;
		}
	}

	function formatCategory(category: string | undefined): string {
		if (!category) return 'Other';
		return category.charAt(0).toUpperCase() + category.slice(1);
	}

	function getStatusClass(status: string) {
		switch (status) {
			case 'running':
				return 'bg-green-500/10 text-green-500';
			case 'error':
				return 'bg-red-500/10 text-red-500';
			default:
				return 'bg-muted text-muted-foreground';
		}
	}

	function nextPage() {
		if ((currentPage + 1) * pageSize < totalServers) {
			currentPage++;
			loadServers();
		}
	}

	function prevPage() {
		if (currentPage > 0) {
			currentPage--;
			loadServers();
		}
	}

	const totalPages = $derived(Math.ceil(totalServers / pageSize));
	const showingStart = $derived(currentPage * pageSize + 1);
	const showingEnd = $derived(Math.min((currentPage + 1) * pageSize, totalServers));
</script>

<svelte:head>
	<title>{m.admin_mcp()} Servers | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-[1600px] px-4 py-8 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="mb-6 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-foreground">{m.mcp_title()}</h1>
			<p class="mt-1 text-muted-foreground">{m.mcp_subtitle()}</p>
		</div>
		<div class="flex gap-2">
			<Button variant="outline" onclick={() => loadServers()} disabled={isLoading}>
				<FontAwesomeIcon icon={faArrowsRotate} class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
				{m.common_refresh()}
			</Button>
			{#if isAdmin}
				<Button onclick={openCreateModal}>
					<FontAwesomeIcon icon={faPlus} class="mr-2 h-4 w-4" />
					{m.mcp_add_server()}
				</Button>
			{/if}
		</div>
	</div>

	<!-- Filters -->
	<div class="mb-6 rounded-lg border border-border bg-card p-4">
		<div class="flex flex-wrap items-center gap-4">
			<!-- Search -->
			<div class="relative flex-1 min-w-[200px]">
				<FontAwesomeIcon icon={faMagnifyingGlass} class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
				<input
					type="text"
					placeholder={m.mcp_search()}
					class="w-full rounded-lg border border-input bg-background py-2 pl-9 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					bind:value={searchQuery}
					oninput={debouncedSearch}
				/>
			</div>

			<!-- Category Filter -->
			<div class="flex items-center gap-2">
				<label for="category-filter" class="text-sm text-muted-foreground">{m.mcp_category()}:</label>
				<select
					id="category-filter"
					bind:value={categoryFilter}
					onchange={() => { currentPage = 0; loadServers(); }}
					class="rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
				>
					<option value="">{m.mcp_all_categories()}</option>
					{#each categories as cat}
						<option value={cat}>{formatCategory(cat)}</option>
					{/each}
				</select>
			</div>

			<!-- Sort By -->
			<div class="flex items-center gap-2">
				<label for="sort-filter" class="text-sm text-muted-foreground">Sort by:</label>
				<select
					id="sort-filter"
					bind:value={sortBy}
					onchange={() => { currentPage = 0; loadServers(); }}
					class="rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
				>
					<option value="created_at">Newest First</option>
					<option value="name">Name (A-Z)</option>
					<option value="category">Category</option>
				</select>
			</div>

			<!-- Clear Filters -->
			<Button variant="outline" size="sm" onclick={clearFilters}>
				<FontAwesomeIcon icon={faXmark} class="mr-1 h-4 w-4" />
				Clear
			</Button>
		</div>

		<!-- Stats -->
		<div class="mt-3 text-sm text-muted-foreground">
			{#if totalServers === 0}
				No servers found
			{:else}
				Showing {showingStart}-{showingEnd} of {totalServers} servers
			{/if}
		</div>
	</div>

	<!-- Servers Grid -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<FontAwesomeIcon icon={faSpinner} class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if servers.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<FontAwesomeIcon icon={faServer} class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-lg font-medium">No MCP servers found</p>
			<p class="mt-1 text-muted-foreground">
				{isAdmin ? 'Try adjusting your filters or add a new server' : 'No servers available in the catalog'}
			</p>
			{#if isAdmin}
				<Button class="mt-6" onclick={openCreateModal}>
					<FontAwesomeIcon icon={faPlus} class="mr-2 h-4 w-4" />
					Add Server
				</Button>
			{/if}
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each servers as server (server.id)}
				{@const CategoryIcon = getCategoryIcon(server.category)}
				<div class="flex flex-col rounded-xl border border-border bg-card p-5 transition-shadow hover:shadow-md">
					<!-- Header -->
					<div class="mb-3 flex items-start justify-between">
						<div class={cn(
							'flex h-10 w-10 items-center justify-center rounded-lg',
							server.status === 'running' ? 'bg-green-500/10 text-green-500' : 'bg-primary/10 text-primary'
						)}>
							<FontAwesomeIcon icon={CategoryIcon} class="h-5 w-5" />
						</div>
						<span class="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
							{formatCategory(server.category)}
						</span>
					</div>

					<!-- Content -->
					<h3 class="font-semibold text-foreground">{server.name}</h3>
					{#if server.description}
						<p class="mt-1 line-clamp-2 text-sm text-muted-foreground">{server.description}</p>
					{/if}

					<!-- Tags -->
					{#if server.tags && server.tags.length > 0}
						<div class="mt-3 flex flex-wrap gap-1">
							{#each server.tags.slice(0, 3) as tag}
								<span class="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">{tag}</span>
							{/each}
							{#if server.tags.length > 3}
								<span class="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">+{server.tags.length - 3}</span>
							{/if}
						</div>
					{/if}

					<!-- Status -->
					{#if server.status}
						<div class="mt-3">
							<span class={cn('rounded-full px-2 py-0.5 text-xs font-medium', getStatusClass(server.status))}>
								{server.status}
							</span>
						</div>
					{/if}

					<!-- Footer -->
					<div class="mt-auto flex items-center gap-2 pt-4">
						<Button variant="outline" size="sm" class="flex-1" onclick={() => viewDetails(server)}>
							Details
						</Button>
						{#if server.github_url}
							<a href={server.github_url} target="_blank" rel="noopener noreferrer" class="rounded-lg border border-input p-2 text-muted-foreground hover:bg-accent hover:text-foreground">
								<FontAwesomeIcon icon={faGithub} class="h-4 w-4" />
							</a>
						{/if}
						{#if server.website_url}
							<a href={server.website_url} target="_blank" rel="noopener noreferrer" class="rounded-lg border border-input p-2 text-muted-foreground hover:bg-accent hover:text-foreground">
								<FontAwesomeIcon icon={faArrowUpRightFromSquare} class="h-4 w-4" />
							</a>
						{/if}
					</div>
				</div>
			{/each}
		</div>

		<!-- Pagination -->
		{#if totalPages > 1}
			<div class="mt-6 flex items-center justify-center gap-4">
				<Button variant="outline" size="sm" onclick={prevPage} disabled={currentPage === 0}>
					<FontAwesomeIcon icon={faChevronLeft} class="mr-1 h-4 w-4" />
					Previous
				</Button>
				<span class="text-sm text-muted-foreground">
					Page {currentPage + 1} of {totalPages}
				</span>
				<Button variant="outline" size="sm" onclick={nextPage} disabled={currentPage >= totalPages - 1}>
					Next
					<FontAwesomeIcon icon={faChevronRight} class="ml-1 h-4 w-4" />
				</Button>
			</div>
		{/if}
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
				<h2 class="text-lg font-semibold">Add MCP Server</h2>
				<button onclick={() => (showCreateModal = false)} class="rounded p-1 text-muted-foreground hover:bg-accent">
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			<form onsubmit={(e) => { e.preventDefault(); handleCreateServer(); }} class="max-h-[70vh] overflow-y-auto p-6">
				{#if createError}
					<div class="mb-4 rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
						{createError}
					</div>
				{/if}

				<!-- Basic Info -->
				<div class="space-y-4">
					<div>
						<label for="server-name" class="mb-1.5 block text-sm font-medium">{m.common_name()} *</label>
						<input
							id="server-name"
							type="text"
							bind:value={serverName}
							placeholder={m.placeholder_server_name()}
							required
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						/>
					</div>

					<div>
						<label for="server-desc" class="mb-1.5 block text-sm font-medium">Description</label>
						<textarea
							id="server-desc"
							bind:value={serverDescription}
							rows="2"
							placeholder={m.placeholder_server_desc()}
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						></textarea>
					</div>

					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="server-category" class="mb-1.5 block text-sm font-medium">Category</label>
							<select
								id="server-category"
								bind:value={serverCategory}
								class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
							>
								<option value="">Select Category</option>
								<option value="development">Development</option>
								<option value="productivity">Productivity</option>
								<option value="database">Database</option>
								<option value="cloud">Cloud</option>
								<option value="ai">AI</option>
								<option value="other">Other</option>
							</select>
						</div>

						<div>
							<label for="server-type" class="mb-1.5 block text-sm font-medium">Type *</label>
							<div class="flex gap-2">
								<button
									type="button"
									onclick={() => (serverType = 'stdio')}
									class={cn(
										'flex-1 rounded-lg border px-3 py-2 text-sm transition-colors',
										serverType === 'stdio' ? 'border-primary bg-primary/10 text-primary' : 'border-input'
									)}
								>
									STDIO
								</button>
								<button
									type="button"
									onclick={() => (serverType = 'sse')}
									class={cn(
										'flex-1 rounded-lg border px-3 py-2 text-sm transition-colors',
										serverType === 'sse' ? 'border-primary bg-primary/10 text-primary' : 'border-input'
									)}
								>
									SSE
								</button>
							</div>
						</div>
					</div>

					{#if serverType === 'stdio'}
						<div>
							<label for="server-command" class="mb-1.5 block text-sm font-medium">Command *</label>
							<input
								id="server-command"
								type="text"
								bind:value={serverCommand}
								placeholder="npx"
								class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
							/>
						</div>

						<div>
							<label for="server-args" class="mb-1.5 block text-sm font-medium">Arguments</label>
							<input
								id="server-args"
								type="text"
								bind:value={serverArgs}
								placeholder="-y @modelcontextprotocol/server-filesystem"
								class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
							/>
						</div>
					{:else}
						<div>
							<label for="server-url" class="mb-1.5 block text-sm font-medium">URL *</label>
							<input
								id="server-url"
								type="url"
								bind:value={serverUrl}
								placeholder="http://localhost:3000/sse"
								class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
							/>
						</div>
					{/if}

					<div>
						<label for="server-tags" class="mb-1.5 block text-sm font-medium">Tags</label>
						<input
							id="server-tags"
							type="text"
							bind:value={serverTags}
							placeholder="filesystem, storage, tools (comma separated)"
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						/>
					</div>

					<div>
						<label for="server-guide" class="mb-1.5 block text-sm font-medium">Installation Guide</label>
						<textarea
							id="server-guide"
							bind:value={serverInstallGuide}
							rows="3"
							placeholder="npm install -g @modelcontextprotocol/server-filesystem"
							class="w-full rounded-lg border border-input bg-background px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						></textarea>
					</div>

					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="server-github" class="mb-1.5 block text-sm font-medium">GitHub URL</label>
							<input
								id="server-github"
								type="url"
								bind:value={serverGithubUrl}
								placeholder="https://github.com/..."
								class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
							/>
						</div>
						<div>
							<label for="server-website" class="mb-1.5 block text-sm font-medium">Website URL</label>
							<input
								id="server-website"
								type="url"
								bind:value={serverWebsiteUrl}
								placeholder="https://..."
								class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
							/>
						</div>
					</div>
				</div>
			</form>

			<div class="flex justify-end gap-3 border-t border-border px-6 py-4">
				<Button variant="outline" type="button" onclick={() => (showCreateModal = false)}>
					{m.common_cancel()}
				</Button>
				<Button onclick={handleCreateServer} disabled={isCreating}>
					{#if isCreating}
						<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					{m.common_create()}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Detail Modal -->
{#if showDetailModal && selectedServer}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showDetailModal = false)}
		role="dialog"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && (showDetailModal = false)}
	>
		<div class="my-8 w-full max-w-2xl rounded-xl border border-border bg-card shadow-xl">
			<div class="flex items-center justify-between border-b border-border px-6 py-4">
				<div>
					<h2 class="text-lg font-semibold">{selectedServer.name}</h2>
					{#if selectedServer.category}
						<span class="mt-1 inline-block rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
							{formatCategory(selectedServer.category)}
						</span>
					{/if}
				</div>
				<button onclick={() => (showDetailModal = false)} class="rounded p-1 text-muted-foreground hover:bg-accent">
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			<div class="max-h-[60vh] overflow-y-auto p-6">
				<!-- Description -->
				{#if selectedServer.description}
					<div class="mb-6">
						<h3 class="mb-2 text-sm font-medium text-muted-foreground">Description</h3>
						<p class="text-foreground">{selectedServer.description}</p>
					</div>
				{/if}

				<!-- Installation Guide -->
				{#if selectedServer.installation_guide}
					<div class="mb-6">
						<div class="mb-2 flex items-center justify-between">
							<h3 class="text-sm font-medium text-muted-foreground">Installation Guide</h3>
							<Button variant="ghost" size="sm" onclick={copyInstallGuide}>
								{#if copiedGuide}
									<FontAwesomeIcon icon={faCheck} class="mr-1 h-3 w-3" />
									Copied
								{:else}
									<FontAwesomeIcon icon={faCopy} class="mr-1 h-3 w-3" />
									Copy
								{/if}
							</Button>
						</div>
						<pre class="overflow-x-auto rounded-lg bg-muted p-4 text-sm">{selectedServer.installation_guide}</pre>
					</div>
				{/if}

				<!-- Tags -->
				{#if selectedServer.tags && selectedServer.tags.length > 0}
					<div class="mb-6">
						<h3 class="mb-2 flex items-center gap-2 text-sm font-medium text-muted-foreground">
							<FontAwesomeIcon icon={faTag} class="h-4 w-4" />
							Tags
						</h3>
						<div class="flex flex-wrap gap-2">
							{#each selectedServer.tags as tag}
								<span class="rounded-full bg-muted px-3 py-1 text-sm">{tag}</span>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Links -->
				{#if selectedServer.github_url || selectedServer.website_url}
					<div class="mb-6">
						<h3 class="mb-2 text-sm font-medium text-muted-foreground">Links</h3>
						<div class="flex gap-2">
							{#if selectedServer.github_url}
								<a href={selectedServer.github_url} target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-2 rounded-lg border border-input px-3 py-2 text-sm hover:bg-accent">
									<FontAwesomeIcon icon={faGithub} class="h-4 w-4" />
									GitHub
								</a>
							{/if}
							{#if selectedServer.website_url}
								<a href={selectedServer.website_url} target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-2 rounded-lg border border-input px-3 py-2 text-sm hover:bg-accent">
									<FontAwesomeIcon icon={faGlobe} class="h-4 w-4" />
									Website
								</a>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Tools -->
				{#if selectedServer.tools && selectedServer.tools.length > 0}
					<div class="mb-6">
						<h3 class="mb-3 flex items-center gap-2 text-sm font-medium text-muted-foreground">
							<FontAwesomeIcon icon={faWrench} class="h-4 w-4" />
							Tools ({selectedServer.tools.length})
						</h3>
						<div class="space-y-2">
							{#each selectedServer.tools as tool}
								<div class="rounded-lg border border-border p-3">
									<p class="font-medium">{tool.name}</p>
									{#if tool.description}
										<p class="mt-1 text-sm text-muted-foreground">{tool.description}</p>
									{/if}
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Resources -->
				{#if selectedServer.resources && selectedServer.resources.length > 0}
					<div class="mb-6">
						<h3 class="mb-3 flex items-center gap-2 text-sm font-medium text-muted-foreground">
							<FontAwesomeIcon icon={faDatabase} class="h-4 w-4" />
							Resources ({selectedServer.resources.length})
						</h3>
						<div class="space-y-2">
							{#each selectedServer.resources as resource}
								<div class="rounded-lg border border-border p-3">
									<p class="font-medium">{resource.name}</p>
									<p class="mt-1 text-xs text-muted-foreground">{resource.uri}</p>
									{#if resource.description}
										<p class="mt-1 text-sm text-muted-foreground">{resource.description}</p>
									{/if}
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Server Info -->
				<div class="border-t border-border pt-4">
					<div class="flex items-center justify-between text-sm text-muted-foreground">
						<span>Type: {selectedServer.type.toUpperCase()}</span>
						{#if selectedServer.created_at}
							<span>Added: {formatRelativeTime(selectedServer.created_at)}</span>
						{/if}
					</div>
				</div>
			</div>

			<!-- Actions -->
			<div class="flex justify-between border-t border-border px-6 py-4">
				{#if isAdmin}
					<div class="flex gap-2">
						{#if selectedServer.status === 'running'}
							<Button variant="outline" size="sm" onclick={() => handleStopServer(selectedServer!)}>
								<FontAwesomeIcon icon={faSquare} class="mr-1 h-3 w-3" />
								Stop
							</Button>
						{:else}
							<Button variant="outline" size="sm" onclick={() => handleStartServer(selectedServer!)}>
								<FontAwesomeIcon icon={faPlay} class="mr-1 h-3 w-3" />
								Start
							</Button>
						{/if}
						<Button
							variant="outline"
							size="sm"
							onclick={() => handleDeleteServer(selectedServer!)}
							class="text-destructive hover:bg-destructive/10"
						>
							<FontAwesomeIcon icon={faTrash} class="mr-1 h-3 w-3" />
							Delete
						</Button>
					</div>
				{:else}
					<div></div>
				{/if}
				<Button variant="outline" onclick={() => (showDetailModal = false)}>
					Close
				</Button>
			</div>
		</div>
	</div>
{/if}
