<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { authStore } from '$lib/stores/auth.svelte';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import { faGauge, faUsers, faKey, faGear, faFileLines, faShield, faServer, faEnvelope, faDownload, faMicrochip, faCodeBranch, faCloud } from '@fortawesome/free-solid-svg-icons';
	import { cn } from '$lib/utils';
	import * as m from '$lib/paraglide/messages';

	let { children } = $props();

	// Scroll fade state for admin tabs
	let navEl: HTMLElement | undefined = $state();
	let showLeftFade = $state(false);
	let showRightFade = $state(true);

	function handleTabScroll() {
		if (!navEl) return;
		const { scrollLeft, scrollWidth, clientWidth } = navEl;
		showLeftFade = scrollLeft > 4;
		showRightFade = scrollLeft < scrollWidth - clientWidth - 4;
	}

	const tabs = [
		{ id: 'dashboard', label: 'Dashboard', icon: faGauge, href: '/admin' },
		{ id: 'users', label: m.admin_users, icon: faUsers, href: '/admin/users' },
		{ id: 'invitations', label: 'Invitations', icon: faEnvelope, href: '/admin/invitations' },
		{ id: 'api-keys', label: 'API Keys', icon: faKey, href: '/admin/api-keys' },
		{ id: 'models', label: m.admin_models, icon: faMicrochip, href: '/admin/models' },
		{ id: 'providers', label: 'Providers', icon: faCloud, href: '/admin/providers' },
		{ id: 'downloads', label: 'Downloads', icon: faDownload, href: '/admin/downloads' },
		{ id: 'mcp', label: 'MCP', icon: faServer, href: '/admin/mcp' },
		{ id: 'gitlab', label: 'GitLab', icon: faCodeBranch, href: '/admin/gitlab' },
		{ id: 'settings', label: m.admin_settings, icon: faGear, href: '/admin/settings' },
		{ id: 'logs', label: m.admin_logs, icon: faFileLines, href: '/admin/logs' }
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
		// Initial scroll check (tabs may already overflow)
		requestAnimationFrame(handleTabScroll);
	});
</script>

<svelte:head>
	<title>{m.admin_title()} | AI Gateway</title>
</svelte:head>

<div class="min-h-[calc(100vh-4rem)]">
	<!-- Admin Header -->
	<div class="border-b border-border bg-card/50">
		<div class="mx-auto max-w-[1600px] px-4 py-4 sm:px-6 lg:px-8">
			<div class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-primary-foreground">
					<FontAwesomeIcon icon={faShield} class="h-5 w-5" />
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
		<div class="mx-auto max-w-[1600px] px-4 sm:px-6 lg:px-8">
			<div class="relative">
				<!-- Left fade mask -->
				<div
					class={cn(
						'pointer-events-none absolute left-0 top-0 bottom-0 z-10 w-8 bg-gradient-to-r from-background to-transparent transition-opacity duration-200',
						showLeftFade ? 'opacity-100' : 'opacity-0'
					)}
				></div>
				<!-- Right fade mask -->
				<div
					class={cn(
						'pointer-events-none absolute right-0 top-0 bottom-0 z-10 w-8 bg-gradient-to-l from-background to-transparent transition-opacity duration-200',
						showRightFade ? 'opacity-100' : 'opacity-0'
					)}
				></div>

				<nav
					class="-mb-px flex gap-1 overflow-x-auto scrollbar-hide"
					aria-label="Admin tabs"
					bind:this={navEl}
					onscroll={handleTabScroll}
				>
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
							<FontAwesomeIcon icon={tab.icon} class="h-4 w-4" />
							{typeof tab.label === 'function' ? tab.label() : tab.label}
						</a>
					{/each}
				</nav>
			</div>
		</div>
	</div>

	<!-- Content -->
	<div class="mx-auto max-w-[1600px] px-4 py-6 sm:px-6 lg:px-8">
		{@render children()}
	</div>
</div>

