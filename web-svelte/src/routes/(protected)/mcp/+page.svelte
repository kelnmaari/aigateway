<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Server,
		Plus,
		Play,
		Square,
		RefreshCw,
		Loader2,
		Trash2,
		Settings,
		Wrench,
		Database,
		X
	} from 'lucide-svelte';
	import { mcpApi, type MCPServer } from '$lib/api/mcp';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	let servers = $state<MCPServer[]>([]);
	let isLoading = $state(true);

	// Modals
	let showCreateModal = $state(false);
	let showDetailModal = $state(false);
	let selectedServer = $state<MCPServer | null>(null);

	// Create form
	let serverName = $state('');
	let serverType = $state<'stdio' | 'sse'>('stdio');
	let serverCommand = $state('');
	let serverArgs = $state('');
	let serverUrl = $state('');
	let isCreating = $state(false);
	let createError = $state('');

	onMount(async () => {
		await loadServers();
	});

	async function loadServers() {
		isLoading = true;
		try {
			const response = await mcpApi.getServers();
			servers = response.servers || [];
		} catch (error) {
			console.error('Failed to load MCP servers:', error);
		} finally {
			isLoading = false;
		}
	}

	function openCreateModal() {
		serverName = '';
		serverType = 'stdio';
		serverCommand = '';
		serverArgs = '';
		serverUrl = '';
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
				type: serverType,
				command: serverType === 'stdio' ? serverCommand.trim() : undefined,
				args: serverType === 'stdio' && serverArgs.trim()
					? serverArgs.split(' ').filter(Boolean)
					: undefined,
				url: serverType === 'sse' ? serverUrl.trim() : undefined,
				enabled: true
			});

			servers = [newServer, ...servers];
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
			alert('Failed to start server');
		}
	}

	async function handleStopServer(server: MCPServer) {
		try {
			await mcpApi.stopServer(server.id);
			servers = servers.map((s) => (s.id === server.id ? { ...s, status: 'stopped' } : s));
		} catch (error) {
			console.error('Failed to stop server:', error);
			alert('Failed to stop server');
		}
	}

	async function handleDeleteServer(server: MCPServer) {
		if (!confirm(`Delete MCP server "${server.name}"?`)) return;

		try {
			await mcpApi.deleteServer(server.id);
			servers = servers.filter((s) => s.id !== server.id);
		} catch (error) {
			console.error('Failed to delete server:', error);
			alert('Failed to delete server');
		}
	}

	function viewDetails(server: MCPServer) {
		selectedServer = server;
		showDetailModal = true;
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
</script>

<svelte:head>
	<title>{m.admin_mcp()} | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-foreground">{m.admin_mcp()} Servers</h1>
			<p class="mt-1 text-muted-foreground">Manage Model Context Protocol servers for tool integration</p>
		</div>
		<div class="flex gap-2">
			<Button variant="outline" onclick={loadServers} disabled={isLoading}>
				<RefreshCw class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
				{m.common_refresh()}
			</Button>
			<Button onclick={openCreateModal}>
				<Plus class="mr-2 h-4 w-4" />
				Add Server
			</Button>
		</div>
	</div>

	<!-- Servers List -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if servers.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Server class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-lg font-medium">No MCP servers configured</p>
			<p class="mt-1 text-muted-foreground">Add an MCP server to enable tool integrations</p>
			<Button class="mt-6" onclick={openCreateModal}>
				<Plus class="mr-2 h-4 w-4" />
				Add Server
			</Button>
		</div>
	{:else}
		<div class="space-y-4">
			{#each servers as server (server.id)}
				<div class="rounded-xl border border-border bg-card p-5">
					<div class="flex items-start justify-between">
						<div class="flex items-start gap-4">
							<div class={cn(
								'flex h-12 w-12 items-center justify-center rounded-lg',
								server.status === 'running' ? 'bg-green-500/10 text-green-500' : 'bg-muted text-muted-foreground'
							)}>
								<Server class="h-6 w-6" />
							</div>
							<div>
								<div class="flex items-center gap-2">
									<h3 class="font-semibold text-foreground">{server.name}</h3>
									<span class={cn('rounded-full px-2 py-0.5 text-xs font-medium', getStatusClass(server.status))}>
										{server.status}
									</span>
								</div>
								<p class="mt-1 text-sm text-muted-foreground">
									{server.type.toUpperCase()}
									{#if server.command}
										• {server.command}
									{:else if server.url}
										• {server.url}
									{/if}
								</p>
								{#if server.tools && server.tools.length > 0}
									<div class="mt-2 flex items-center gap-2">
										<Wrench class="h-4 w-4 text-muted-foreground" />
										<span class="text-sm text-muted-foreground">{server.tools.length} tools</span>
									</div>
								{/if}
								{#if server.resources && server.resources.length > 0}
									<div class="mt-1 flex items-center gap-2">
										<Database class="h-4 w-4 text-muted-foreground" />
										<span class="text-sm text-muted-foreground">{server.resources.length} resources</span>
									</div>
								{/if}
								{#if server.error_message}
									<p class="mt-2 text-sm text-red-500">{server.error_message}</p>
								{/if}
							</div>
						</div>

						<div class="flex gap-2">
							{#if server.status === 'running'}
								<Button variant="outline" size="sm" onclick={() => handleStopServer(server)}>
									<Square class="mr-1 h-3 w-3" />
									Stop
								</Button>
							{:else}
								<Button variant="outline" size="sm" onclick={() => handleStartServer(server)}>
									<Play class="mr-1 h-3 w-3" />
									Start
								</Button>
							{/if}
							<Button variant="outline" size="sm" onclick={() => viewDetails(server)}>
								<Settings class="h-4 w-4" />
							</Button>
							<Button
								variant="outline"
								size="sm"
								onclick={() => handleDeleteServer(server)}
								class="text-destructive hover:bg-destructive/10"
							>
								<Trash2 class="h-4 w-4" />
							</Button>
						</div>
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
				<h2 class="text-lg font-semibold">Add MCP Server</h2>
				<button onclick={() => (showCreateModal = false)} class="rounded p-1 text-muted-foreground hover:bg-accent">
					<X class="h-5 w-5" />
				</button>
			</div>

			<form onsubmit={(e) => { e.preventDefault(); handleCreateServer(); }} class="space-y-4">
				{#if createError}
					<div class="rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
						{createError}
					</div>
				{/if}

				<div class="space-y-2">
					<label for="server-name" class="text-sm font-medium">{m.common_name()} *</label>
					<input
						id="server-name"
						type="text"
						bind:value={serverName}
						placeholder="My MCP Server"
						required
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>

				<div class="space-y-2">
					<label class="text-sm font-medium">Type</label>
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

				{#if serverType === 'stdio'}
					<div class="space-y-2">
						<label for="server-command" class="text-sm font-medium">Command *</label>
						<input
							id="server-command"
							type="text"
							bind:value={serverCommand}
							placeholder="npx"
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						/>
					</div>

					<div class="space-y-2">
						<label for="server-args" class="text-sm font-medium">Arguments</label>
						<input
							id="server-args"
							type="text"
							bind:value={serverArgs}
							placeholder="-y @modelcontextprotocol/server-filesystem"
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						/>
					</div>
				{:else}
					<div class="space-y-2">
						<label for="server-url" class="text-sm font-medium">URL *</label>
						<input
							id="server-url"
							type="url"
							bind:value={serverUrl}
							placeholder="http://localhost:3000/sse"
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						/>
					</div>
				{/if}

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

<!-- Detail Modal -->
{#if showDetailModal && selectedServer}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showDetailModal = false)}
		role="dialog"
		tabindex="-1"
	>
		<div class="w-full max-w-lg rounded-xl border border-border bg-card shadow-xl">
			<div class="flex items-center justify-between border-b border-border p-4">
				<h2 class="text-lg font-semibold">{selectedServer.name}</h2>
				<button onclick={() => (showDetailModal = false)} class="rounded p-1 text-muted-foreground hover:bg-accent">
					<X class="h-5 w-5" />
				</button>
			</div>

			<div class="max-h-[60vh] overflow-y-auto p-4">
				<!-- Tools -->
				{#if selectedServer.tools && selectedServer.tools.length > 0}
					<div class="mb-6">
						<h3 class="mb-3 flex items-center gap-2 font-medium">
							<Wrench class="h-4 w-4" />
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
					<div>
						<h3 class="mb-3 flex items-center gap-2 font-medium">
							<Database class="h-4 w-4" />
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

				{#if (!selectedServer.tools || selectedServer.tools.length === 0) && (!selectedServer.resources || selectedServer.resources.length === 0)}
					<p class="py-8 text-center text-muted-foreground">No tools or resources available</p>
				{/if}
			</div>
		</div>
	</div>
{/if}

