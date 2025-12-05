<script lang="ts">
	import { onMount } from 'svelte';
	import {
		GitBranch,
		Plus,
		RefreshCw,
		Loader2,
		Trash2,
		Settings,
		X,
		Search,
		ExternalLink,
		CheckCircle,
		XCircle,
		AlertCircle,
		Play,
		Eye,
		Folder,
		Activity,
		Server,
		BarChart3,
		MessageSquare,
		Cog
	} from 'lucide-svelte';
	import { gitlabApi, type GitLabIntegration, type GitLabQueueStats } from '$lib/api/gitlab';
	import { cn, formatRelativeTime, debounce } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';

	// Sub-navigation links
	const subNavLinks = [
		{ href: '/admin/gitlab/queue', label: 'Queue', icon: Activity },
		{ href: '/admin/gitlab/analytics', label: 'Analytics', icon: BarChart3 },
		{ href: '/admin/gitlab/feedback', label: 'Feedback', icon: MessageSquare },
		{ href: '/admin/gitlab/settings', label: 'Settings', icon: Cog }
	];

	let integrations = $state<GitLabIntegration[]>([]);
	let queueStats = $state<GitLabQueueStats | null>(null);
	let totalIntegrations = $state(0);
	let isLoading = $state(true);

	// Filters
	let searchQuery = $state('');
	let statusFilter = $state('');

	// Pagination
	let currentPage = $state(0);
	let pageSize = 10;

	// Modals
	let showCreateModal = $state(false);
	let showDetailModal = $state(false);
	let selectedIntegration = $state<GitLabIntegration | null>(null);
	let testResult = $state<{ success: boolean; error?: string; user?: any } | null>(null);
	let isTesting = $state(false);

	// Create form
	let formName = $state('');
	let formBaseUrl = $state('https://gitlab.com');
	let formAccessToken = $state('');
	let formWebhookSecret = $state('');
	let isCreating = $state(false);
	let createError = $state('');

	onMount(async () => {
		// Load data separately to prevent one failure from blocking everything
		await loadIntegrations();
		await loadQueueStats();
	});

	const debouncedSearch = debounce(() => {
		currentPage = 0;
		loadIntegrations();
	}, 300);

	async function loadIntegrations() {
		isLoading = true;
		try {
			const response = await gitlabApi.listIntegrations({
				limit: pageSize,
				offset: currentPage * pageSize,
				search: searchQuery || undefined,
				status: statusFilter || undefined
			});
			integrations = response.data || [];
			totalIntegrations = response.total || integrations.length;
		} catch (error) {
			console.error('Failed to load integrations:', error);
			integrations = [];
			totalIntegrations = 0;
		} finally {
			isLoading = false;
		}
	}

	async function loadQueueStats() {
		try {
			queueStats = await gitlabApi.getQueueStatus();
		} catch (error) {
			console.error('Failed to load queue stats:', error);
		}
	}

	function clearFilters() {
		searchQuery = '';
		statusFilter = '';
		currentPage = 0;
		loadIntegrations();
	}

	function openCreateModal() {
		formName = '';
		formBaseUrl = 'https://gitlab.com';
		formAccessToken = '';
		formWebhookSecret = '';
		createError = '';
		showCreateModal = true;
	}

	async function handleCreate() {
		if (!formName.trim()) {
			createError = 'Name is required';
			return;
		}
		if (!formBaseUrl.trim()) {
			createError = 'GitLab URL is required';
			return;
		}
		if (!formAccessToken.trim()) {
			createError = 'Access Token is required';
			return;
		}

		isCreating = true;
		createError = '';

		try {
			const newIntegration = await gitlabApi.createIntegration({
				name: formName.trim(),
				base_url: formBaseUrl.trim(),
				access_token: formAccessToken.trim(),
				webhook_secret: formWebhookSecret.trim() || undefined
			});

			integrations = [newIntegration, ...integrations];
			totalIntegrations++;
			showCreateModal = false;
		} catch (error) {
			createError = error instanceof Error ? error.message : 'Failed to create integration';
		} finally {
			isCreating = false;
		}
	}

	async function handleTest(integration: GitLabIntegration) {
		isTesting = true;
		testResult = null;
		selectedIntegration = integration;
		showDetailModal = true;

		try {
			testResult = await gitlabApi.testIntegration(integration.id);
			// Refresh to update status
			await loadIntegrations();
		} catch (error) {
			testResult = {
				success: false,
				error: error instanceof Error ? error.message : 'Test failed'
			};
		} finally {
			isTesting = false;
		}
	}

	async function handleDelete(integration: GitLabIntegration) {
		if (!confirm(`Delete integration "${integration.name}"? This will also delete all projects and reviews.`)) {
			return;
		}

		try {
			await gitlabApi.deleteIntegration(integration.id);
			integrations = integrations.filter((i) => i.id !== integration.id);
			totalIntegrations--;
		} catch (error) {
			alert('Failed to delete integration');
		}
	}

	function getStatusIcon(status: string) {
		switch (status) {
			case 'active':
				return CheckCircle;
			case 'error':
				return XCircle;
			default:
				return AlertCircle;
		}
	}

	function getStatusColor(status: string) {
		switch (status) {
			case 'active':
				return 'text-green-500';
			case 'error':
				return 'text-red-500';
			default:
				return 'text-yellow-500';
		}
	}

	const totalPages = $derived(Math.ceil(totalIntegrations / pageSize));
</script>

<div class="space-y-6">
	<!-- Sub Navigation -->
	<div class="flex items-center gap-2 border-b pb-4">
		{#each subNavLinks as link}
			<a
				href={link.href}
				class="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
			>
				<svelte:component this={link.icon} class="h-4 w-4" />
				{link.label}
			</a>
		{/each}
	</div>

	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-2xl font-bold text-foreground">GitLab Integrations</h2>
			<p class="text-muted-foreground">Manage GitLab connections for AI-powered MR reviews</p>
		</div>
		<Button onclick={openCreateModal}>
			<Plus class="mr-2 h-4 w-4" />
			Add Integration
		</Button>
	</div>

	<!-- Queue Stats -->
	{#if queueStats}
		<div class="grid grid-cols-2 gap-4 sm:grid-cols-4 lg:grid-cols-6">
			<div class="rounded-lg border bg-card p-4">
				<div class="flex items-center gap-2">
					<Activity class="h-4 w-4 text-yellow-500" />
					<span class="text-sm text-muted-foreground">Pending</span>
				</div>
				<p class="mt-1 text-2xl font-bold">{queueStats.pending}</p>
			</div>
			<div class="rounded-lg border bg-card p-4">
				<div class="flex items-center gap-2">
					<Loader2 class="h-4 w-4 animate-spin text-blue-500" />
					<span class="text-sm text-muted-foreground">Processing</span>
				</div>
				<p class="mt-1 text-2xl font-bold">{queueStats.processing}</p>
			</div>
			<div class="rounded-lg border bg-card p-4">
				<div class="flex items-center gap-2">
					<CheckCircle class="h-4 w-4 text-green-500" />
					<span class="text-sm text-muted-foreground">Completed Today</span>
				</div>
				<p class="mt-1 text-2xl font-bold">{queueStats.completed_today}</p>
			</div>
			<div class="rounded-lg border bg-card p-4">
				<div class="flex items-center gap-2">
					<XCircle class="h-4 w-4 text-red-500" />
					<span class="text-sm text-muted-foreground">Failed</span>
				</div>
				<p class="mt-1 text-2xl font-bold">{queueStats.failed}</p>
			</div>
			<div class="rounded-lg border bg-card p-4">
				<div class="flex items-center gap-2">
					<Server class="h-4 w-4 text-purple-500" />
					<span class="text-sm text-muted-foreground">Workers</span>
				</div>
				<p class="mt-1 text-2xl font-bold">{queueStats.active_workers}/{queueStats.total_workers}</p>
			</div>
			<div class="rounded-lg border bg-card p-4">
				<div class="flex items-center gap-2">
					<Activity class="h-4 w-4 text-cyan-500" />
					<span class="text-sm text-muted-foreground">Last 24h</span>
				</div>
				<p class="mt-1 text-2xl font-bold">{queueStats.jobs_last_24_hours}</p>
			</div>
		</div>
	{/if}

	<!-- Filters -->
	<div class="flex flex-wrap items-center gap-4">
		<div class="relative flex-1 min-w-[200px]">
			<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
			<input
				type="text"
				placeholder="Search integrations..."
				bind:value={searchQuery}
				oninput={debouncedSearch}
				class="h-10 w-full rounded-md border bg-background pl-10 pr-4 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
			/>
		</div>
		<select
			bind:value={statusFilter}
			onchange={() => { currentPage = 0; loadIntegrations(); }}
			class="h-10 rounded-md border bg-background px-3 text-sm"
		>
			<option value="">All Statuses</option>
			<option value="active">Active</option>
			<option value="disabled">Disabled</option>
			<option value="error">Error</option>
		</select>
		<Button variant="outline" onclick={clearFilters}>
			<RefreshCw class="mr-2 h-4 w-4" />
			Reset
		</Button>
	</div>

	<!-- Integrations Table -->
	<div class="rounded-lg border bg-card">
		{#if isLoading}
			<div class="flex items-center justify-center py-12">
				<Loader2 class="h-8 w-8 animate-spin text-primary" />
			</div>
		{:else if integrations.length === 0}
			<div class="flex flex-col items-center justify-center py-12 text-center">
				<GitBranch class="h-12 w-12 text-muted-foreground/50" />
				<p class="mt-4 text-lg font-medium text-muted-foreground">No integrations found</p>
				<p class="text-sm text-muted-foreground">Add a GitLab integration to start reviewing MRs</p>
				<Button class="mt-4" onclick={openCreateModal}>
					<Plus class="mr-2 h-4 w-4" />
					Add Integration
				</Button>
			</div>
		{:else}
			<table class="w-full">
				<thead>
					<tr class="border-b text-left text-sm text-muted-foreground">
						<th class="px-4 py-3 font-medium">Name</th>
						<th class="px-4 py-3 font-medium">URL</th>
						<th class="px-4 py-3 font-medium">Status</th>
						<th class="px-4 py-3 font-medium">Projects</th>
						<th class="px-4 py-3 font-medium">Last Sync</th>
						<th class="px-4 py-3 font-medium">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each integrations as integration}
						<tr class="border-b last:border-0 hover:bg-muted/50">
							<td class="px-4 py-3">
								<div class="font-medium">{integration.name}</div>
							</td>
							<td class="px-4 py-3">
								<a
									href={integration.base_url}
									target="_blank"
									rel="noopener"
									class="flex items-center gap-1 text-sm text-primary hover:underline"
								>
									{integration.base_url}
									<ExternalLink class="h-3 w-3" />
								</a>
							</td>
							<td class="px-4 py-3">
								<div class="flex items-center gap-2">
									<svelte:component
										this={getStatusIcon(integration.status)}
										class={cn('h-4 w-4', getStatusColor(integration.status))}
									/>
									<span class="text-sm capitalize">{integration.status}</span>
								</div>
								{#if integration.last_error}
									<p class="mt-1 text-xs text-red-500 truncate max-w-[200px]" title={integration.last_error}>
										{integration.last_error}
									</p>
								{/if}
							</td>
							<td class="px-4 py-3">
								<span class="text-sm">{integration.project_count || 0}</span>
							</td>
							<td class="px-4 py-3">
								<span class="text-sm text-muted-foreground">
									{integration.last_sync_at ? formatRelativeTime(integration.last_sync_at) : 'Never'}
								</span>
							</td>
							<td class="px-4 py-3">
								<div class="flex items-center gap-2">
									<button
										onclick={() => handleTest(integration)}
										class="rounded p-1.5 hover:bg-muted"
										title="Test Connection"
									>
										<Play class="h-4 w-4" />
									</button>
									<a
										href={`/admin/gitlab/${integration.id}`}
										class="rounded p-1.5 hover:bg-muted"
										title="View Projects"
									>
										<Folder class="h-4 w-4" />
									</a>
									<button
										onclick={() => handleDelete(integration)}
										class="rounded p-1.5 text-red-500 hover:bg-red-500/10"
										title="Delete"
									>
										<Trash2 class="h-4 w-4" />
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>

			<!-- Pagination -->
			{#if totalPages > 1}
				<div class="flex items-center justify-between border-t px-4 py-3">
					<p class="text-sm text-muted-foreground">
						Showing {currentPage * pageSize + 1} to {Math.min((currentPage + 1) * pageSize, totalIntegrations)} of {totalIntegrations}
					</p>
					<div class="flex items-center gap-2">
						<Button
							variant="outline"
							size="sm"
							disabled={currentPage === 0}
							onclick={() => { currentPage--; loadIntegrations(); }}
						>
							Previous
						</Button>
						<Button
							variant="outline"
							size="sm"
							disabled={currentPage >= totalPages - 1}
							onclick={() => { currentPage++; loadIntegrations(); }}
						>
							Next
						</Button>
					</div>
				</div>
			{/if}
		{/if}
	</div>
</div>

<!-- Create Modal -->
{#if showCreateModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showCreateModal = false)}
		role="dialog"
		aria-modal="true"
	>
		<div class="w-full max-w-lg rounded-lg bg-card p-6 shadow-xl">
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-semibold">Add GitLab Integration</h3>
				<button onclick={() => (showCreateModal = false)} class="rounded p-1 hover:bg-muted">
					<X class="h-5 w-5" />
				</button>
			</div>

			{#if createError}
				<div class="mb-4 rounded-md bg-red-500/10 p-3 text-sm text-red-500">
					{createError}
				</div>
			{/if}

			<div class="space-y-4">
				<div>
					<label class="mb-1.5 block text-sm font-medium">Name *</label>
					<input
						type="text"
						bind:value={formName}
						placeholder="My GitLab"
						class="h-10 w-full rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
					/>
				</div>

				<div>
					<label class="mb-1.5 block text-sm font-medium">GitLab URL *</label>
					<input
						type="url"
						bind:value={formBaseUrl}
						placeholder="https://gitlab.com"
						class="h-10 w-full rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
					/>
					<p class="mt-1 text-xs text-muted-foreground">Use https://gitlab.com for GitLab.com or your self-hosted URL</p>
				</div>

				<div>
					<label class="mb-1.5 block text-sm font-medium">Personal Access Token *</label>
					<input
						type="password"
						bind:value={formAccessToken}
						placeholder="glpat-xxxxxxxxxxxx"
						class="h-10 w-full rounded-md border bg-background px-3 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary"
					/>
					<p class="mt-1 text-xs text-muted-foreground">
						Requires <code class="bg-muted px-1 rounded">api</code> and <code class="bg-muted px-1 rounded">read_repository</code> scopes
					</p>
				</div>

				<div>
					<label class="mb-1.5 block text-sm font-medium">Webhook Secret (optional)</label>
					<input
						type="text"
						bind:value={formWebhookSecret}
						placeholder="Leave empty to auto-generate"
						class="h-10 w-full rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
					/>
				</div>
			</div>

			<div class="mt-6 flex justify-end gap-3">
				<Button variant="outline" onclick={() => (showCreateModal = false)}>
					Cancel
				</Button>
				<Button onclick={handleCreate} disabled={isCreating}>
					{#if isCreating}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					Create Integration
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Detail/Test Modal -->
{#if showDetailModal && selectedIntegration}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showDetailModal = false)}
		role="dialog"
		aria-modal="true"
	>
		<div class="w-full max-w-lg rounded-lg bg-card p-6 shadow-xl">
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-semibold">{selectedIntegration.name}</h3>
				<button onclick={() => (showDetailModal = false)} class="rounded p-1 hover:bg-muted">
					<X class="h-5 w-5" />
				</button>
			</div>

			<div class="space-y-4">
				<div>
					<span class="text-sm text-muted-foreground">URL:</span>
					<a href={selectedIntegration.base_url} target="_blank" class="ml-2 text-primary hover:underline">
						{selectedIntegration.base_url}
					</a>
				</div>

				<div>
					<span class="text-sm text-muted-foreground">Status:</span>
					<span class={cn('ml-2 capitalize', getStatusColor(selectedIntegration.status))}>
						{selectedIntegration.status}
					</span>
				</div>

				<!-- Test Result -->
				{#if isTesting}
					<div class="flex items-center gap-2 rounded-md bg-muted p-4">
						<Loader2 class="h-5 w-5 animate-spin" />
						<span>Testing connection...</span>
					</div>
				{:else if testResult}
					<div class={cn(
						'rounded-md p-4',
						testResult.success ? 'bg-green-500/10' : 'bg-red-500/10'
					)}>
						{#if testResult.success}
							<div class="flex items-center gap-2 text-green-500">
								<CheckCircle class="h-5 w-5" />
								<span class="font-medium">Connection successful!</span>
							</div>
							{#if testResult.user}
								<div class="mt-2 text-sm">
									<p>Connected as: <strong>{testResult.user.name}</strong> (@{testResult.user.username})</p>
									{#if testResult.user.is_admin}
										<p class="text-green-600">✓ Admin access</p>
									{/if}
								</div>
							{/if}
						{:else}
							<div class="flex items-center gap-2 text-red-500">
								<XCircle class="h-5 w-5" />
								<span class="font-medium">Connection failed</span>
							</div>
							{#if testResult.error}
								<p class="mt-2 text-sm text-red-500">{testResult.error}</p>
							{/if}
						{/if}
					</div>
				{/if}
			</div>

			<div class="mt-6 flex justify-end gap-3">
				<Button variant="outline" onclick={() => (showDetailModal = false)}>
					Close
				</Button>
				<a href={`/admin/gitlab/${selectedIntegration.id}`}>
					<Button>
						<Folder class="mr-2 h-4 w-4" />
						View Projects
					</Button>
				</a>
			</div>
		</div>
	</div>
{/if}

