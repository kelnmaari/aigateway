<script lang="ts">
	import { page } from '$app/stores';
	import { Activity, BarChart3, MessageSquare, Settings, GitBranch } from 'lucide-svelte';

	const navLinks = [
		{ href: '/admin/gitlab', label: 'Integrations', icon: GitBranch, exact: true },
		{ href: '/admin/gitlab/queue', label: 'Queue', icon: Activity },
		{ href: '/admin/gitlab/analytics', label: 'Analytics', icon: BarChart3 },
		{ href: '/admin/gitlab/feedback', label: 'Feedback', icon: MessageSquare },
		{ href: '/admin/gitlab/settings', label: 'Settings', icon: Settings }
	];

	function isActive(href: string, exact: boolean = false): boolean {
		if (exact) {
			return $page.url.pathname === href;
		}
		return $page.url.pathname === href || $page.url.pathname.startsWith(href + '/');
	}
</script>

<nav class="flex items-center gap-1 border-b pb-4 mb-6">
	{#each navLinks as link}
		<a
			href={link.href}
			class="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors
				{isActive(link.href, link.exact) 
					? 'bg-primary/10 text-primary' 
					: 'text-muted-foreground hover:text-foreground hover:bg-muted'}"
		>
			<svelte:component this={link.icon} class="h-4 w-4" />
			{link.label}
		</a>
	{/each}
</nav>

