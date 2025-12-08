<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { authStore } from '$lib/stores/auth.svelte';
	import {
		LayoutDashboard,
		Users,
		Key,
		Settings,
		FileText,
		Shield,
		Server,
		Mail,
		Download,
		Cpu,
		GitBranch
	} from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import * as m from '$lib/paraglide/messages';

	let { children } = $props();

	const tabs = [
		{ id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard, href: '/admin' },
		{ id: 'users', label: m.admin_users, icon: Users, href: '/admin/users' },
		{ id: 'invitations', label: 'Invitations', icon: Mail, href: '/admin/invitations' },
		{ id: 'api-keys', label: 'API Keys', icon: Key, href: '/admin/api-keys' },
		{ id: 'models', label: m.admin_models, icon: Cpu, href: '/admin/models' },
		{ id: 'downloads', label: 'Downloads', icon: Download, href: '/admin/downloads' },
		{ id: 'mcp', label: 'MCP', icon: Server, href: '/admin/mcp' },
		{ id: 'gitlab', label: 'GitLab', icon: GitBranch, href: '/admin/gitlab' },
		{ id: 'settings', label: m.admin_settings, icon: Settings, href: '/admin/settings' },
		{ id: 'logs', label: m.admin_logs, icon: FileText, href: '/admin/logs' }
	];

	function getCurrentTab(): string {
		const path = $page.url.pathname;
		if (path === '/admin') return 'dashboard';
		const segment = path.split('/')[2];
		return segment || 'dashboard';
	}

	onMount(() => {
		// Redirect non-admins
		if (!authStore.isAdmin) {
			goto('/dashboard');
		}
	});
</script>

<svelte:head>
	<title>{m.admin_title()} | AI Gateway</title>
</svelte:head>

<div class="min-h-[calc(100vh-4rem)]">
	<!-- Admin Header -->
	<div class="border-b border-border bg-card/50">
		<div class="mx-auto max-w-7xl px-4 py-4 sm:px-6 lg:px-8">
			<div class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-primary-foreground">
					<Shield class="h-5 w-5" />
				</div>
				<div>
					<h1 class="text-xl font-bold text-foreground">{m.admin_title()}</h1>
					<p class="text-sm text-muted-foreground">System administration and configuration</p>
				</div>
			</div>
		</div>
	</div>

	<!-- Tabs Navigation -->
	<div class="border-b border-border bg-background">
		<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
			<nav class="-mb-px flex gap-1 overflow-x-auto" aria-label="Admin tabs">
				{#each tabs as tab}
					<a
						href={tab.href}
						class={cn(
							'flex items-center gap-2 whitespace-nowrap border-b-2 px-4 py-3 text-sm font-medium transition-colors',
							getCurrentTab() === tab.id
								? 'border-primary text-primary'
								: 'border-transparent text-muted-foreground hover:border-border hover:text-foreground'
						)}
					>
						<tab.icon class="h-4 w-4" />
						{typeof tab.label === 'function' ? tab.label() : tab.label}
					</a>
				{/each}
			</nav>
		</div>
	</div>

	<!-- Content -->
	<div class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
		{@render children()}
	</div>
</div>

