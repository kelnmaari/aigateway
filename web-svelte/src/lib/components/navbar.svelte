<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { authStore, themeStore } from '$lib';
	import { cn } from '$lib/utils';
	import { locales, getLocale, setLocale } from '$lib/paraglide/runtime';
	import * as m from '$lib/paraglide/messages';
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
		ChevronDown
	} from 'lucide-svelte';

	const STORAGE_KEY = 'PARAGLIDE_LOCALE';

	let mobileMenuOpen = $state(false);
	let userMenuOpen = $state(false);

	const navItems = [
		{ href: '/dashboard', label: () => m.nav_dashboard(), icon: Home },
		{ href: '/chat', label: () => m.nav_chat(), icon: MessageSquare },
		{ href: '/api-keys', label: () => m.nav_apiKeys(), icon: Key },
		{ href: '/tenants', label: () => m.nav_tenants(), icon: Building2 }
	];

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
</script>

<svelte:window
	onclick={(e) => {
		if (!(e.target as HTMLElement).closest('[data-user-menu]')) {
			userMenuOpen = false;
		}
	}}
/>

<nav class="sticky top-0 z-50 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
	<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
		<div class="flex h-16 items-center justify-between">
			<!-- Logo -->
			<div class="flex items-center gap-8">
				<a href="/dashboard" class="flex items-center gap-2">
					<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-lg text-primary-foreground">
						🤖
					</div>
					<span class="hidden font-semibold text-foreground sm:block">AIGateway</span>
				</a>

				<!-- Desktop Navigation -->
				<div class="hidden md:flex md:items-center md:gap-1">
					{#each navItems as item}
						<a
							href={item.href}
							class={cn(
								'flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors',
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
									'flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors',
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
				<button
					onclick={() => themeStore.toggle()}
					class="hidden h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-accent-foreground sm:flex"
					title={themeStore.isDark ? m.theme_light() : m.theme_dark()}
				>
					{#if themeStore.isDark}
						<Sun class="h-4 w-4" />
					{:else}
						<Moon class="h-4 w-4" />
					{/if}
				</button>

				<!-- Locale Toggle -->
				<button
					onclick={toggleLocale}
					class="hidden h-9 items-center gap-1 rounded-md px-2 text-sm text-muted-foreground hover:bg-accent hover:text-accent-foreground sm:flex"
					title="Change language"
				>
					<Globe class="h-4 w-4" />
					<span class="uppercase">{getLocale()}</span>
				</button>

				<!-- User Menu -->
				<div class="relative" data-user-menu>
					<button
						onclick={() => (userMenuOpen = !userMenuOpen)}
						class="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-accent"
					>
						<div class="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-primary">
							<User class="h-4 w-4" />
						</div>
						<span class="hidden max-w-[100px] truncate text-foreground sm:block">
							{authStore.user?.username || 'User'}
						</span>
						<ChevronDown class="hidden h-4 w-4 text-muted-foreground sm:block" />
					</button>

					{#if userMenuOpen}
						<div
							class="absolute right-0 mt-2 w-56 origin-top-right rounded-md border border-border bg-card py-1 shadow-lg"
						>
							<div class="border-b border-border px-4 py-2">
								<p class="text-sm font-medium text-foreground">
									{authStore.user?.full_name || authStore.user?.username}
								</p>
								<p class="text-xs text-muted-foreground">{authStore.user?.email}</p>
								{#if authStore.isAdmin}
									<span class="mt-1 inline-block rounded bg-primary/10 px-1.5 py-0.5 text-xs font-medium text-primary">
										Admin
									</span>
								{/if}
							</div>

							<a
								href="/profile"
								onclick={closeMenus}
								class="flex items-center gap-2 px-4 py-2 text-sm text-muted-foreground hover:bg-accent hover:text-accent-foreground"
							>
								<User class="h-4 w-4" />
								{m.nav_profile()}
							</a>

							<a
								href="/settings"
								onclick={closeMenus}
								class="flex items-center gap-2 px-4 py-2 text-sm text-muted-foreground hover:bg-accent hover:text-accent-foreground"
							>
								<Settings class="h-4 w-4" />
								{m.nav_settings()}
							</a>

							<div class="border-t border-border">
								<button
									onclick={handleLogout}
									class="flex w-full items-center gap-2 px-4 py-2 text-sm text-destructive hover:bg-destructive/10"
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
					class="flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-accent-foreground md:hidden"
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
		<div class="border-t border-border md:hidden">
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
					<div class="my-2 border-t border-border"></div>
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

				<div class="my-2 border-t border-border"></div>

				<!-- Mobile theme & locale toggles -->
				<div class="flex items-center gap-2 px-3 py-2">
					<button
						onclick={() => themeStore.toggle()}
						class="flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-accent"
					>
						{#if themeStore.isDark}
							<Sun class="h-5 w-5" />
						{:else}
							<Moon class="h-5 w-5" />
						{/if}
					</button>
					<button
						onclick={toggleLocale}
						class="flex h-9 items-center gap-1.5 rounded-md px-3 text-sm text-muted-foreground hover:bg-accent"
					>
						<Globe class="h-4 w-4" />
						{getLocaleLabel()}
					</button>
				</div>
			</div>
		</div>
	{/if}
</nav>
