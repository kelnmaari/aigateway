<script lang="ts">
	import { onMount } from 'svelte';
	import { authStore } from '$lib';
	import { api } from '$lib/api';
	import { cn, formatNumber, formatRelativeTime } from '$lib/utils';
	import {
		MessageSquare,
		Key,
		Cpu,
		Activity,
		Plus,
		ArrowRight,
		Loader2,
		Building2
	} from 'lucide-svelte';
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

	onMount(async () => {
		await loadDashboardData();
	});

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
			icon: MessageSquare,
			href: '/chat',
			color: 'text-blue-500'
		},
		{
			label: 'API Keys',
			value: stats.apiKeys,
			icon: Key,
			href: '/api-keys',
			color: 'text-emerald-500'
		},
		{
			label: 'Models',
			value: stats.models,
			icon: Cpu,
			href: '/chat',
			color: 'text-purple-500'
		},
		{
			label: 'Requests',
			value: stats.requests,
			icon: Activity,
			href: '/api-keys',
			color: 'text-orange-500'
		}
	]);
</script>

<svelte:head>
	<title>Dashboard - AIGateway</title>
</svelte:head>

<div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="mb-8">
		<h1 class="text-2xl font-bold text-foreground">
			Welcome back, {authStore.user?.full_name || authStore.user?.username}
		</h1>
		<p class="mt-1 text-muted-foreground">Here's what's happening with your AI gateway</p>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else}
		<!-- Stats Grid -->
		<div class="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			{#each statCards as stat}
				<a href={stat.href} class="group">
					<Card.Root class="transition-shadow hover:shadow-md">
						<Card.Content class="flex items-center gap-4 p-6">
							<div class={cn('rounded-lg bg-background p-3', stat.color)}>
								<stat.icon class="h-6 w-6" />
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
					<Card.Title>Quick Actions</Card.Title>
					<Card.Description>Get started with common tasks</Card.Description>
				</Card.Header>
				<Card.Content class="grid gap-3">
					<a
						href="/chat"
						class="flex items-center justify-between rounded-lg border border-border p-4 transition-colors hover:bg-accent"
					>
						<div class="flex items-center gap-3">
							<div class="rounded-lg bg-primary/10 p-2 text-primary">
								<Plus class="h-5 w-5" />
							</div>
							<div>
								<p class="font-medium text-foreground">New Chat</p>
								<p class="text-sm text-muted-foreground">Start a conversation with AI</p>
							</div>
						</div>
						<ArrowRight class="h-5 w-5 text-muted-foreground" />
					</a>

					<a
						href="/api-keys"
						class="flex items-center justify-between rounded-lg border border-border p-4 transition-colors hover:bg-accent"
					>
						<div class="flex items-center gap-3">
							<div class="rounded-lg bg-emerald-500/10 p-2 text-emerald-500">
								<Key class="h-5 w-5" />
							</div>
							<div>
								<p class="font-medium text-foreground">Create API Key</p>
								<p class="text-sm text-muted-foreground">Generate a new API key</p>
							</div>
						</div>
						<ArrowRight class="h-5 w-5 text-muted-foreground" />
					</a>

					{#if authStore.isAdmin}
						<a
							href="/admin"
							class="flex items-center justify-between rounded-lg border border-border p-4 transition-colors hover:bg-accent"
						>
							<div class="flex items-center gap-3">
								<div class="rounded-lg bg-purple-500/10 p-2 text-purple-500">
									<Activity class="h-5 w-5" />
								</div>
								<div>
									<p class="font-medium text-foreground">Admin Panel</p>
									<p class="text-sm text-muted-foreground">Manage users and settings</p>
								</div>
							</div>
							<ArrowRight class="h-5 w-5 text-muted-foreground" />
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
							<MessageSquare class="mx-auto h-12 w-12 text-muted-foreground/50" />
							<p class="mt-2 text-sm text-muted-foreground">No conversations yet</p>
							<a
								href="/chat"
								class="mt-4 inline-flex items-center gap-2 text-sm font-medium text-primary hover:underline"
							>
								<Plus class="h-4 w-4" />
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
										<MessageSquare class="h-5 w-5 text-muted-foreground" />
										<div>
											<p class="font-medium text-foreground">
												{conversation.title || 'Untitled'}
											</p>
											<p class="text-xs text-muted-foreground">
												{formatRelativeTime(conversation.updated_at)}
											</p>
										</div>
									</div>
									<ArrowRight class="h-4 w-4 text-muted-foreground" />
								</a>
							{/each}
						</div>
						<a
							href="/chat"
							class="mt-4 inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline"
						>
							View all conversations
							<ArrowRight class="h-4 w-4" />
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
							<Cpu class="mx-auto h-12 w-12 text-muted-foreground/50" />
							<p class="mt-2 text-sm text-muted-foreground">No models available</p>
						</div>
					{:else}
						<div class="grid gap-2 sm:grid-cols-2">
							{#each availableModels as model}
								<div class="flex items-center gap-2 rounded-lg border border-border p-3">
									<Cpu class="h-4 w-4 text-muted-foreground" />
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
							<Building2 class="mx-auto h-12 w-12 text-muted-foreground/50" />
							<p class="mt-2 text-sm text-muted-foreground">No organizations yet</p>
							<a
								href="/tenants"
								class="mt-4 inline-flex items-center gap-2 text-sm font-medium text-primary hover:underline"
							>
								<Plus class="h-4 w-4" />
								Create organization
							</a>
						</div>
					{:else}
						<div class="space-y-2">
							{#each tenants as tenant}
								<a
									href="/tenants/{tenant.id}"
									class="flex items-center justify-between rounded-lg border border-border p-3 transition-colors hover:bg-accent"
								>
									<div class="flex items-center gap-3">
										<Building2 class="h-5 w-5 text-muted-foreground" />
										<div>
											<p class="font-medium text-foreground">{tenant.name}</p>
											<p class="text-xs text-muted-foreground capitalize">{tenant.role}</p>
										</div>
									</div>
									<ArrowRight class="h-4 w-4 text-muted-foreground" />
								</a>
							{/each}
						</div>
					{/if}
				</Card.Content>
			</Card.Root>
		</div>
	{/if}
</div>

