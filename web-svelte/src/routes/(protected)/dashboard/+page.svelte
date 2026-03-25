<script lang="ts">
	import { onMount } from 'svelte';
	import { authStore } from '$lib';
	import { api } from '$lib/api';
	import { cn, formatNumber, formatRelativeTime } from '$lib/utils';
	import * as m from '$lib/paraglide/messages';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import {
		faComments,
		faKey,
		faMicrochip,
		faChartLine,
		faPlus,
		faArrowRight,
		faSpinner,
		faBuilding,
		faDownload,
		faXmark,
		faWandMagicSparkles
	} from '@fortawesome/free-solid-svg-icons';
	import * as Card from '$lib/components/ui/card';

	// Dashboard data
	let stats = $state({
		conversations: 0,
		apiKeys: 0,
		models: 0,
		requests: 0
	});
	let recentConversations = $state<Array<{ id: string; title: string; updated_at: string }>>([]);
	let availableModels = $state<Array<{ id: string; name?: string }>>([]);
	let tenants = $state<Array<{ id: string; name: string; role: string }>>([]);
	let loading = $state(true);
	
	// Version check state
	let currentVersion = $state('');
	let latestVersion = $state('');
	let hasUpdate = $state(false);
	let updateDismissed = $state(false);
	let versionCheckError = $state('');

	onMount(async () => {
		await loadDashboardData();
		await checkForUpdates();
	});
	
	// Compare semantic versions: returns 1 if a > b, -1 if a < b, 0 if equal
	function compareVersions(a: string, b: string): number {
		const normalize = (v: string) => v.replace(/^v/, '').split(/[-+]/)[0]; // Remove 'v' prefix and pre-release suffix
		const partsA = normalize(a).split('.').map(Number);
		const partsB = normalize(b).split('.').map(Number);
		
		for (let i = 0; i < Math.max(partsA.length, partsB.length); i++) {
			const numA = partsA[i] || 0;
			const numB = partsB[i] || 0;
			if (numA > numB) return 1;
			if (numA < numB) return -1;
		}
		return 0;
	}
	
	async function checkForUpdates() {
		try {
			// Get current version from system info
			const sysInfo = await api.get<{ version: string; git_commit: string }>('/api/system/info');
			currentVersion = sysInfo.version || 'unknown';
			
			// Fetch latest version from GitLab Package Registry (external source)
			// This works even without updating the server - checks remote packages
			try {
				const gitlabRes = await fetch('https://gitlab.alexue4.dev/api/v4/projects/146/packages?order_by=version&sort=desc&per_page=10', {
					headers: { 'Accept': 'application/json' }
				});
				if (gitlabRes.ok) {
					const packages = await gitlabRes.json();
					if (packages && packages.length > 0) {
						// Find the latest stable version (e.g., "4.0.0", not "4.0.0-feature.xxx")
						// Stable versions don't have a hyphen after the semver part
						const stableVersions = packages
							.map((p: { version: string }) => p.version)
							.filter((v: string) => /^\d+\.\d+\.\d+$/.test(v))
							.sort((a: string, b: string) => compareVersions(b, a));
						
						if (stableVersions.length > 0) {
							latestVersion = stableVersions[0];
						} else {
							// Fallback to any latest version if no stable found
							latestVersion = packages[0].version?.replace(/^v/, '') || '';
						}
						
						// Check if update is available
						if (currentVersion !== 'dev' && currentVersion !== 'unknown' && latestVersion) {
							hasUpdate = compareVersions(latestVersion, currentVersion) > 0;
							console.log('[Version Check via Package Registry]', { current: currentVersion, latest: latestVersion, hasUpdate });
						}
						return; // Success - don't need fallback
					}
				}
			} catch (gitlabErr) {
				console.warn('GitLab Package Registry unreachable, using local changelogs as fallback');
			}
			
			// Fallback to local changelogs if GitLab is unreachable
			const changelogsRes = await api.get<{ changelogs: Array<{ version: string }> }>('/api/system/changelogs');
			const changelogs = changelogsRes.changelogs || [];
			if (changelogs.length > 0) {
				const versions = changelogs.map(c => c.version).sort((a, b) => compareVersions(b, a));
				latestVersion = versions[0];
				if (currentVersion !== 'dev' && currentVersion !== 'unknown') {
					hasUpdate = compareVersions(latestVersion, currentVersion) > 0;
					console.log('[Version Check via Changelogs]', { current: currentVersion, latest: latestVersion, hasUpdate });
				}
			}
		} catch (err) {
			console.error('Version check failed:', err);
			versionCheckError = 'Failed to check for updates';
		}
	}

	async function loadDashboardData() {
		loading = true;
		try {
			// Load dashboard stats - use batch API endpoint
			const [statsRes, conversationsRes, modelsRes, tenantsRes] = await Promise.allSettled([
				api.get<{ conversations: number; api_keys: number; requests: number }>('/api/dashboard/stats'),
				api.get<{ conversations: Array<{ id: string; title: string; updated_at: string }> }>(
					'/api/conversations?limit=5'
				),
				api.get<{ object: string; data: Array<{ id: string; owned_by?: string }> }>('/v1/models'),
				api.get<{ tenants: Array<{ id: string; name: string; role: string }> }>('/api/users/me/tenants')
			]);

			if (statsRes.status === 'fulfilled') {
				stats.conversations = statsRes.value.conversations || 0;
				stats.apiKeys = statsRes.value.api_keys || 0;
				stats.requests = statsRes.value.requests || 0;
			}

			if (conversationsRes.status === 'fulfilled') {
				recentConversations = conversationsRes.value.conversations || [];
			}

			if (modelsRes.status === 'fulfilled') {
				// OpenAI format: { object: "list", data: [...] }
				const modelData = modelsRes.value.data || [];
				availableModels = modelData.slice(0, 6).map((m) => ({ id: m.id, name: m.id }));
				stats.models = modelData.length;
			}

			if (tenantsRes.status === 'fulfilled') {
				tenants = tenantsRes.value.tenants || [];
			}
		} catch (err) {
			console.error('Failed to load dashboard data:', err);
		} finally {
			loading = false;
		}
	}

	const statCards = $derived([
		{
			label: 'Conversations',
			value: stats.conversations,
			icon: faComments,
			href: '/chat',
			color: 'text-blue-500'
		},
		{
			label: 'API Keys',
			value: stats.apiKeys,
			icon: faKey,
			href: '/api-keys',
			color: 'text-emerald-500'
		},
		{
			label: 'Models',
			value: stats.models,
			icon: faMicrochip,
			href: '/chat',
			color: 'text-purple-500'
		},
		{
			label: 'Requests',
			value: stats.requests,
			icon: faChartLine,
			href: '/api-keys',
			color: 'text-orange-500'
		}
	]);
</script>

<svelte:head>
	<title>Dashboard - AIGateway</title>
</svelte:head>

<div class="mx-auto max-w-[1600px] px-4 py-8 sm:px-6 lg:px-8">
	<!-- Update Available Banner -->
	{#if hasUpdate && !updateDismissed}
		<div class="mb-6 relative overflow-hidden rounded-lg border border-emerald-500/30 bg-gradient-to-r from-emerald-500/10 via-teal-500/10 to-cyan-500/10 p-4">
			<div class="absolute inset-0 bg-grid-pattern opacity-5"></div>
			<div class="relative flex items-center justify-between">
				<div class="flex items-center gap-4">
					<div class="flex h-12 w-12 items-center justify-center rounded-full bg-emerald-500/20 text-emerald-500">
						<FontAwesomeIcon icon={faWandMagicSparkles} class="h-6 w-6" />
					</div>
					<div>
						<h3 class="font-semibold text-foreground flex items-center gap-2">
							{m.dashboard_update_available()}
							<span class="inline-flex items-center rounded-full bg-emerald-500/20 px-2 py-0.5 text-xs font-medium text-emerald-600 dark:text-emerald-400">
								v{latestVersion}
							</span>
						</h3>
						<p class="text-sm text-muted-foreground">
							{m.dashboard_update_current({ version: currentVersion })}
						</p>
					</div>
				</div>
				<div class="flex items-center gap-2">
					<a 
						href="/admin/about" 
						class="inline-flex items-center gap-2 rounded-lg bg-emerald-500 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-600 transition-colors"
					>
						<FontAwesomeIcon icon={faDownload} class="h-4 w-4" />
						{m.dashboard_update_view_changelog()}
					</a>
					<button 
						onclick={() => updateDismissed = true}
						class="p-2 text-muted-foreground hover:text-foreground rounded-lg hover:bg-muted transition-colors"
						title={m.common_close()}
					>
						<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Header -->
	<div class="mb-8">
		<h1 class="text-2xl font-bold text-foreground">
			{m.dashboard_welcome({ name: authStore.user?.full_name || authStore.user?.username || '' })}
		</h1>
		<p class="mt-1 text-muted-foreground">{m.dashboard_welcome_desc()}</p>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-20">
			<FontAwesomeIcon icon={faSpinner} class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else}
		<!-- Stats Grid -->
		<div class="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			{#each statCards as stat}
				<a href={stat.href} class="group">
					<Card.Root class="transition-shadow hover:shadow-md">
						<Card.Content class="flex items-center gap-4 p-6">
							<div class={cn('rounded-lg bg-background p-3', stat.color)}>
								<FontAwesomeIcon icon={stat.icon} class="h-6 w-6" />
							</div>
							<div>
								<p class="text-sm text-muted-foreground">{stat.label}</p>
								<p class="text-2xl font-bold text-foreground">{formatNumber(stat.value)}</p>
							</div>
						</Card.Content>
					</Card.Root>
				</a>
			{/each}
		</div>

		<div class="grid gap-8 lg:grid-cols-2">
			<!-- Quick Actions -->
			<Card.Root>
				<Card.Header>
					<Card.Title>{m.dashboard_quick_actions_title()}</Card.Title>
					<Card.Description>{m.dashboard_quick_actions_desc()}</Card.Description>
				</Card.Header>
				<Card.Content class="grid gap-3">
					<a
						href="/chat"
						class="flex items-center justify-between rounded-lg border border-border p-4 transition-colors hover:bg-accent"
					>
						<div class="flex items-center gap-3">
							<div class="rounded-lg bg-primary/10 p-2 text-primary">
								<FontAwesomeIcon icon={faPlus} class="h-5 w-5" />
							</div>
							<div>
								<p class="font-medium text-foreground">New Chat</p>
								<p class="text-sm text-muted-foreground">Start a conversation with AI</p>
							</div>
						</div>
						<FontAwesomeIcon icon={faArrowRight} class="h-5 w-5 text-muted-foreground" />
					</a>

					<a
						href="/api-keys"
						class="flex items-center justify-between rounded-lg border border-border p-4 transition-colors hover:bg-accent"
					>
						<div class="flex items-center gap-3">
							<div class="rounded-lg bg-emerald-500/10 p-2 text-emerald-500">
								<FontAwesomeIcon icon={faKey} class="h-5 w-5" />
							</div>
							<div>
								<p class="font-medium text-foreground">Create API Key</p>
								<p class="text-sm text-muted-foreground">Generate a new API key</p>
							</div>
						</div>
						<FontAwesomeIcon icon={faArrowRight} class="h-5 w-5 text-muted-foreground" />
					</a>

					{#if authStore.isAdmin}
						<a
							href="/admin"
							class="flex items-center justify-between rounded-lg border border-border p-4 transition-colors hover:bg-accent"
						>
							<div class="flex items-center gap-3">
								<div class="rounded-lg bg-purple-500/10 p-2 text-purple-500">
									<FontAwesomeIcon icon={faChartLine} class="h-5 w-5" />
								</div>
								<div>
									<p class="font-medium text-foreground">Admin Panel</p>
									<p class="text-sm text-muted-foreground">Manage users and settings</p>
								</div>
							</div>
							<FontAwesomeIcon icon={faArrowRight} class="h-5 w-5 text-muted-foreground" />
						</a>
					{/if}
				</Card.Content>
			</Card.Root>

			<!-- Recent Conversations -->
			<Card.Root>
				<Card.Header>
					<Card.Title>Recent Conversations</Card.Title>
					<Card.Description>Continue where you left off</Card.Description>
				</Card.Header>
				<Card.Content>
					{#if recentConversations.length === 0}
						<div class="py-8 text-center">
							<FontAwesomeIcon icon={faComments} class="mx-auto h-12 w-12 text-muted-foreground/50" />
							<p class="mt-2 text-sm text-muted-foreground">No conversations yet</p>
							<a
								href="/chat"
								class="mt-4 inline-flex items-center gap-2 text-sm font-medium text-primary hover:underline"
							>
								<FontAwesomeIcon icon={faPlus} class="h-4 w-4" />
								Start your first chat
							</a>
						</div>
					{:else}
						<div class="space-y-2">
							{#each recentConversations as conversation}
								<a
									href="/chat/{conversation.id}"
									class="flex items-center justify-between rounded-lg border border-border p-3 transition-colors hover:bg-accent"
								>
									<div class="flex items-center gap-3">
										<FontAwesomeIcon icon={faComments} class="h-5 w-5 text-muted-foreground" />
										<div>
											<p class="font-medium text-foreground">
												{conversation.title || 'Untitled'}
											</p>
											<p class="text-xs text-muted-foreground">
												{formatRelativeTime(conversation.updated_at)}
											</p>
										</div>
									</div>
									<FontAwesomeIcon icon={faArrowRight} class="h-4 w-4 text-muted-foreground" />
								</a>
							{/each}
						</div>
						<a
							href="/chat"
							class="mt-4 inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline"
						>
							View all conversations
							<FontAwesomeIcon icon={faArrowRight} class="h-4 w-4" />
						</a>
					{/if}
				</Card.Content>
			</Card.Root>

			<!-- Available Models -->
			<Card.Root>
				<Card.Header>
					<Card.Title>Available Models</Card.Title>
					<Card.Description>Models ready to use</Card.Description>
				</Card.Header>
				<Card.Content>
					{#if availableModels.length === 0}
						<div class="py-8 text-center">
							<FontAwesomeIcon icon={faMicrochip} class="mx-auto h-12 w-12 text-muted-foreground/50" />
							<p class="mt-2 text-sm text-muted-foreground">No models available</p>
						</div>
					{:else}
						<div class="grid gap-2 sm:grid-cols-2">
							{#each availableModels as model}
								<div class="flex items-center gap-2 rounded-lg border border-border p-3">
									<FontAwesomeIcon icon={faMicrochip} class="h-4 w-4 text-muted-foreground" />
									<span class="truncate text-sm text-foreground">{model.name || model.id}</span>
								</div>
							{/each}
						</div>
					{/if}
				</Card.Content>
			</Card.Root>

			<!-- Organizations -->
			<Card.Root>
				<Card.Header>
					<Card.Title>Your Organizations</Card.Title>
					<Card.Description>Teams and workspaces you belong to</Card.Description>
				</Card.Header>
				<Card.Content>
					{#if tenants.length === 0}
						<div class="py-8 text-center">
							<FontAwesomeIcon icon={faBuilding} class="mx-auto h-12 w-12 text-muted-foreground/50" />
							<p class="mt-2 text-sm text-muted-foreground">No organizations yet</p>
							<a
								href="/tenants"
								class="mt-4 inline-flex items-center gap-2 text-sm font-medium text-primary hover:underline"
							>
								<FontAwesomeIcon icon={faPlus} class="h-4 w-4" />
								Create organization
							</a>
						</div>
					{:else}
						<div class="space-y-2">
							{#each tenants as tenant}
								<a
									href="/tenants"
									class="flex items-center justify-between rounded-lg border border-border p-3 transition-colors hover:bg-accent"
								>
									<div class="flex items-center gap-3">
										<FontAwesomeIcon icon={faBuilding} class="h-5 w-5 text-muted-foreground" />
										<div>
											<p class="font-medium text-foreground">{tenant.name}</p>
											<p class="text-xs text-muted-foreground capitalize">{tenant.role}</p>
										</div>
									</div>
									<FontAwesomeIcon icon={faArrowRight} class="h-4 w-4 text-muted-foreground" />
								</a>
							{/each}
						</div>
					{/if}
				</Card.Content>
			</Card.Root>
		</div>
	{/if}
</div>

