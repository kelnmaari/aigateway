<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { authStore, themeStore } from '$lib';
	import { cn } from '$lib/utils';
	import { locales, getLocale, setLocale } from '$lib/paraglide/runtime';
	import * as m from '$lib/paraglide/messages';
	import { api } from '$lib/api/client';
	import {
		Home,
		MessageSquare,
		Key,
		Building2,
		User,
		Settings,
		Shield,
		LogOut,
		Sun,
		Moon,
		Globe,
		Menu,
		X,
		ChevronDown,
		FolderOpen,
		Database,
		Server,
		GitBranch,
		Info,
		FileText
	} from 'lucide-svelte';
	import { Tooltip } from '$lib/components/ui/tooltip';
	import { ActiveJobsIndicator } from '$lib/components/gitlab';

	const STORAGE_KEY = 'PARAGLIDE_LOCALE';

	let mobileMenuOpen = $state(false);
	let userMenuOpen = $state(false);
	let appVersion = $state('');

	const navItems = [
		{ href: '/dashboard', label: () => m.nav_dashboard(), icon: Home },
		{ href: '/chat', label: () => m.nav_chat(), icon: MessageSquare },
		{ href: '/api-keys', label: () => m.nav_apiKeys(), icon: Key },
		{ href: '/tenants', label: () => m.nav_tenants(), icon: Building2 },
		{ href: '/files', label: () => m.nav_files(), icon: FolderOpen },
		{ href: '/rag', label: () => m.nav_rag(), icon: Database },
		{ href: '/mcp', label: () => m.nav_mcp(), icon: Server },
		{ href: '/gitlab', label: () => 'GitLab', icon: GitBranch }
	];

	// Admin-only menu items
	const adminItems = [{ href: '/admin', label: () => m.nav_admin(), icon: Shield }];

	function isActive(href: string): boolean {
		return $page.url.pathname === href || $page.url.pathname.startsWith(href + '/');
	}

	async function handleLogout() {
		try {
			authStore.logout();
		} catch {
			// Ignore logout errors
		}
		goto('/login');
	}

	function closeMenus() {
		mobileMenuOpen = false;
		userMenuOpen = false;
	}

	function toggleLocale() {
		const current = getLocale();
		const currentIndex = locales.indexOf(current);
		const nextIndex = (currentIndex + 1) % locales.length;
		const newLocale = locales[nextIndex];

		setLocale(newLocale);
		localStorage.setItem(STORAGE_KEY, newLocale);
		window.location.reload();
	}

	function getLocaleLabel(): string {
		const current = getLocale();
		return current === 'en' ? 'English' : 'Русский';
	}

	onMount(async () => {
		try {
			const info = await api.get<{ version: string }>('/api/system/info');
			appVersion = info.version || '';
		} catch {
			appVersion = '';
		}
	});
</script>

<svelte:window
	onclick={(e) => {
		if (!(e.target as HTMLElement).closest('[data-user-menu]')) {
			userMenuOpen = false;
		}
	}}
/>

<nav
	class="border-border bg-background/95 supports-[backdrop-filter]:bg-background/60 sticky top-0 z-50 border-b backdrop-blur"
>
	<div class="mx-auto max-w-[1600px] px-4 sm:px-6 lg:px-8">
		<div class="flex h-16 items-center justify-between">
			<!-- Logo -->
			<div class="flex items-center gap-8">
				<a href="/dashboard" class="flex items-center gap-2">
					<div
						class="bg-primary text-primary-foreground flex h-8 w-8 items-center justify-center rounded-lg text-lg"
					>
						🤖
					</div>
					<span class="text-foreground hidden font-semibold sm:block">AIGateway</span>
				</a>

				<!-- Desktop Navigation -->
				<div class="hidden md:flex md:items-center md:gap-0.5">
					{#each navItems as item}
						<a
							href={item.href}
							class={cn(
								'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors',
								isActive(item.href)
									? 'bg-accent text-accent-foreground'
									: 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
							)}
						>
							<item.icon class="h-4 w-4" />
							{item.label()}
						</a>
					{/each}

					{#if authStore.isAdmin}
						{#each adminItems as item}
							<a
								href={item.href}
								class={cn(
									'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors',
									isActive(item.href)
										? 'bg-accent text-accent-foreground'
										: 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
								)}
							>
								<item.icon class="h-4 w-4" />
								{item.label()}
							</a>
						{/each}
					{/if}
				</div>
			</div>

			<!-- Right side -->
			<div class="flex items-center gap-2">
				<!-- Theme Toggle -->
				<Tooltip text={themeStore.isDark ? m.theme_light() : m.theme_dark()} position="bottom">
					<button
						onclick={() => themeStore.toggle()}
						class="text-muted-foreground hover:bg-accent hover:text-accent-foreground hidden h-9 w-9 items-center justify-center rounded-md sm:flex"
					>
						{#if themeStore.isDark}
							<Sun class="h-4 w-4" />
						{:else}
							<Moon class="h-4 w-4" />
						{/if}
					</button>
				</Tooltip>

				<!-- Locale Toggle -->
				<Tooltip text={m.settings_language()} position="bottom">
					<button
						onclick={toggleLocale}
						class="text-muted-foreground hover:bg-accent hover:text-accent-foreground hidden h-9 items-center gap-1 rounded-md px-2 text-sm sm:flex"
					>
						<Globe class="h-4 w-4" />
						<span class="uppercase">{getLocale()}</span>
					</button>
				</Tooltip>

				<!-- Active Jobs Indicator -->
				<div class="hidden sm:block">
					<ActiveJobsIndicator />
				</div>

				<!-- User Menu -->
				<div class="relative" data-user-menu>
					<button
						onclick={() => (userMenuOpen = !userMenuOpen)}
						class="hover:bg-accent flex items-center gap-2 rounded-md px-2 py-1.5 text-sm"
					>
						<div
							class="bg-primary/10 text-primary flex h-7 w-7 items-center justify-center rounded-full"
						>
							<User class="h-4 w-4" />
						</div>
						<span class="text-foreground hidden max-w-[100px] truncate sm:block">
							{authStore.user?.username || 'User'}
						</span>
						<ChevronDown class="text-muted-foreground hidden h-4 w-4 sm:block" />
					</button>

					{#if userMenuOpen}
						<div
							class="border-border bg-card absolute right-0 mt-2 w-56 origin-top-right rounded-md border py-1 shadow-lg"
						>
							<div class="border-border border-b px-4 py-2">
								<p class="text-foreground text-sm font-medium">
									{authStore.user?.full_name || authStore.user?.username}
								</p>
								<p class="text-muted-foreground text-xs">{authStore.user?.email}</p>
								{#if authStore.isAdmin}
									<span
										class="bg-primary/10 text-primary mt-1 inline-block rounded px-1.5 py-0.5 text-xs font-medium"
									>
										Admin
									</span>
								{/if}
							</div>

							<a
								href="/profile"
								onclick={closeMenus}
								class="text-muted-foreground hover:bg-accent hover:text-accent-foreground flex items-center gap-2 px-4 py-2 text-sm"
							>
								<User class="h-4 w-4" />
								{m.nav_profile()}
							</a>

							<a
								href="/settings"
								onclick={closeMenus}
								class="text-muted-foreground hover:bg-accent hover:text-accent-foreground flex items-center gap-2 px-4 py-2 text-sm"
							>
								<Settings class="h-4 w-4" />
								{m.nav_settings()}
							</a>

							<a
								href="/about"
								onclick={closeMenus}
								class="text-muted-foreground hover:bg-accent hover:text-accent-foreground flex items-center gap-2 px-4 py-2 text-sm"
							>
								<Info class="h-4 w-4" />
								{m.nav_about()}
							</a>
							<a
								href="/docs"
								onclick={closeMenus}
								class="text-muted-foreground hover:bg-accent hover:text-accent-foreground flex items-center gap-2 px-4 py-2 text-sm"
							>
								<FileText class="h-4 w-4" />
								{m.nav_docs?.() || 'Documentation'}
							</a>

							<div class="border-border border-t">
								{#if appVersion}
									<div class="text-muted-foreground px-4 py-2 text-xs">
										v{appVersion}
									</div>
								{/if}
								<button
									onclick={handleLogout}
									class="text-destructive hover:bg-destructive/10 flex w-full items-center gap-2 px-4 py-2 text-sm"
								>
									<LogOut class="h-4 w-4" />
									{m.auth_logout()}
								</button>
							</div>
						</div>
					{/if}
				</div>

				<!-- Mobile menu button -->
				<button
					onclick={() => (mobileMenuOpen = !mobileMenuOpen)}
					class="text-muted-foreground hover:bg-accent hover:text-accent-foreground flex h-9 w-9 items-center justify-center rounded-md md:hidden"
				>
					{#if mobileMenuOpen}
						<X class="h-5 w-5" />
					{:else}
						<Menu class="h-5 w-5" />
					{/if}
				</button>
			</div>
		</div>
	</div>

	<!-- Mobile menu -->
	{#if mobileMenuOpen}
		<div class="border-border border-t md:hidden">
			<div class="space-y-1 px-4 py-3">
				{#each navItems as item}
					<a
						href={item.href}
						onclick={closeMenus}
						class={cn(
							'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium',
							isActive(item.href)
								? 'bg-accent text-accent-foreground'
								: 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
						)}
					>
						<item.icon class="h-5 w-5" />
						{item.label()}
					</a>
				{/each}

				{#if authStore.isAdmin}
					<div class="border-border my-2 border-t"></div>
					{#each adminItems as item}
						<a
							href={item.href}
							onclick={closeMenus}
							class={cn(
								'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium',
								isActive(item.href)
									? 'bg-accent text-accent-foreground'
									: 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
							)}
						>
							<item.icon class="h-5 w-5" />
							{item.label()}
						</a>
					{/each}
				{/if}

				<div class="border-border my-2 border-t"></div>

				<!-- Mobile theme & locale toggles -->
				<div class="flex items-center gap-2 px-3 py-2">
					<button
						onclick={() => themeStore.toggle()}
						class="text-muted-foreground hover:bg-accent flex h-9 w-9 items-center justify-center rounded-md"
					>
						{#if themeStore.isDark}
							<Sun class="h-5 w-5" />
						{:else}
							<Moon class="h-5 w-5" />
						{/if}
					</button>
					<button
						onclick={toggleLocale}
						class="text-muted-foreground hover:bg-accent flex h-9 items-center gap-1.5 rounded-md px-3 text-sm"
					>
						<Globe class="h-4 w-4" />
						{getLocaleLabel()}
					</button>
				</div>
			</div>
		</div>
	{/if}
</nav>
