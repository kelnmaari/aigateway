<script lang="ts">
	import { page } from '$app/stores';
	import { Activity, BarChart3, MessageSquare, Settings, GitBranch, History } from 'lucide-svelte';
	import * as m from '$lib/paraglide/messages';

	const navLinks = $derived([
		{ href: '/admin/gitlab', label: m.admin_gitlab_integrations(), icon: GitBranch, exact: true },
		{ href: '/admin/gitlab/queue', label: m.admin_gitlab_queue(), icon: Activity },
		{ href: '/admin/gitlab/analytics', label: m.admin_gitlab_analytics(), icon: BarChart3 },
		{ href: '/admin/gitlab/feedback', label: m.admin_gitlab_feedback(), icon: MessageSquare },
		{ href: '/admin/gitlab/history', label: m.admin_gitlab_scan_history?.() || 'Scan History', icon: History },
		{ href: '/admin/gitlab/settings', label: m.settings_title(), icon: Settings }
	]);

	function isActive(href: string, exact: boolean = false): boolean {
		if (exact) {
			return $page.url.pathname === href;
		}
		return $page.url.pathname === href || $page.url.pathname.startsWith(href + '/');
	}
</script>

<nav class="flex items-center gap-1 border-b pb-4 mb-6">
	{#each navLinks as link}
		{@const Icon = link.icon}
		<a
			href={link.href}
			class="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors
				{isActive(link.href, link.exact) 
					? 'bg-primary/10 text-primary' 
					: 'text-muted-foreground hover:text-foreground hover:bg-muted'}"
		>
			<Icon class="h-4 w-4" />
			{link.label}
		</a>
	{/each}
</nav>

