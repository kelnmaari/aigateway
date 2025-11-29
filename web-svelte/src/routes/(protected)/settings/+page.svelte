<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Settings,
		Bell,
		Globe,
		Moon,
		Sun,
		Monitor,
		Palette,
		Save,
		Loader2,
		Check
	} from 'lucide-svelte';
	import { themeStore } from '$lib';
	import { locales, getLocale, setLocale } from '$lib/paraglide/runtime';
	import { cn } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	const STORAGE_KEY = 'PARAGLIDE_LOCALE';

	// Settings state
	let theme = $state<'light' | 'dark' | 'system'>('system');
	let locale = $state(getLocale());
	let notifications = $state({
		email: true,
		browser: false,
		sound: false
	});
	let chatSettings = $state({
		enterToSend: true,
		showTimestamps: true,
		codeHighlight: true
	});

	let isSaving = $state(false);
	let saved = $state(false);

	onMount(() => {
		// Load settings from localStorage
		const savedSettings = localStorage.getItem('user_settings');
		if (savedSettings) {
			try {
				const parsed = JSON.parse(savedSettings);
				if (parsed.notifications) notifications = parsed.notifications;
				if (parsed.chatSettings) chatSettings = parsed.chatSettings;
			} catch {
				// Ignore parse errors
			}
		}

		// Set theme from store
		theme = themeStore.isDark ? 'dark' : 'light';
	});

	function handleThemeChange(newTheme: 'light' | 'dark' | 'system') {
		theme = newTheme;
		if (newTheme === 'light') {
			themeStore.setLight();
		} else if (newTheme === 'dark') {
			themeStore.setDark();
		} else {
			// System preference
			const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
			prefersDark ? themeStore.setDark() : themeStore.setLight();
		}
	}

	function handleLocaleChange(newLocale: string) {
		locale = newLocale;
		setLocale(newLocale);
		localStorage.setItem(STORAGE_KEY, newLocale);
	}

	async function saveSettings() {
		isSaving = true;

		// Save to localStorage
		const settings = {
			notifications,
			chatSettings
		};
		localStorage.setItem('user_settings', JSON.stringify(settings));

		// Simulate API call
		await new Promise((resolve) => setTimeout(resolve, 500));

		isSaving = false;
		saved = true;
		setTimeout(() => (saved = false), 2000);
	}

	function getLocaleLabel(loc: string): string {
		return loc === 'en' ? 'English' : 'Русский';
	}
</script>

<svelte:head>
	<title>{m.nav_settings()} | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-3xl px-4 py-8 sm:px-6 lg:px-8">
	<div class="mb-8">
		<h1 class="text-2xl font-bold text-foreground">{m.nav_settings()}</h1>
		<p class="mt-1 text-muted-foreground">Customize your experience</p>
	</div>

	<div class="space-y-6">
		<!-- Appearance -->
		<div class="rounded-xl border border-border bg-card p-6">
			<div class="mb-4 flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
					<Palette class="h-5 w-5" />
				</div>
				<div>
					<h2 class="font-semibold text-foreground">Appearance</h2>
					<p class="text-sm text-muted-foreground">Customize the look and feel</p>
				</div>
			</div>

			<div class="space-y-4">
				<!-- Theme -->
				<div>
					<label class="mb-2 block text-sm font-medium">Theme</label>
					<div class="flex gap-2">
						<button
							onclick={() => handleThemeChange('light')}
							class={cn(
								'flex flex-1 items-center justify-center gap-2 rounded-lg border px-4 py-3 transition-colors',
								theme === 'light' ? 'border-primary bg-primary/10 text-primary' : 'border-input hover:bg-accent'
							)}
						>
							<Sun class="h-4 w-4" />
							{m.theme_light()}
						</button>
						<button
							onclick={() => handleThemeChange('dark')}
							class={cn(
								'flex flex-1 items-center justify-center gap-2 rounded-lg border px-4 py-3 transition-colors',
								theme === 'dark' ? 'border-primary bg-primary/10 text-primary' : 'border-input hover:bg-accent'
							)}
						>
							<Moon class="h-4 w-4" />
							{m.theme_dark()}
						</button>
						<button
							onclick={() => handleThemeChange('system')}
							class={cn(
								'flex flex-1 items-center justify-center gap-2 rounded-lg border px-4 py-3 transition-colors',
								theme === 'system' ? 'border-primary bg-primary/10 text-primary' : 'border-input hover:bg-accent'
							)}
						>
							<Monitor class="h-4 w-4" />
							{m.theme_system()}
						</button>
					</div>
				</div>

				<!-- Language -->
				<div>
					<label class="mb-2 block text-sm font-medium">Language</label>
					<div class="flex gap-2">
						{#each locales as loc}
							<button
								onclick={() => handleLocaleChange(loc)}
								class={cn(
									'flex flex-1 items-center justify-center gap-2 rounded-lg border px-4 py-3 transition-colors',
									locale === loc ? 'border-primary bg-primary/10 text-primary' : 'border-input hover:bg-accent'
								)}
							>
								<Globe class="h-4 w-4" />
								{getLocaleLabel(loc)}
							</button>
						{/each}
					</div>
					<p class="mt-2 text-xs text-muted-foreground">
						Page will reload after changing language
					</p>
				</div>
			</div>
		</div>

		<!-- Notifications -->
		<div class="rounded-xl border border-border bg-card p-6">
			<div class="mb-4 flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
					<Bell class="h-5 w-5" />
				</div>
				<div>
					<h2 class="font-semibold text-foreground">Notifications</h2>
					<p class="text-sm text-muted-foreground">Manage your notification preferences</p>
				</div>
			</div>

			<div class="space-y-4">
				<label class="flex items-center justify-between">
					<div>
						<p class="font-medium">Email notifications</p>
						<p class="text-sm text-muted-foreground">Receive updates via email</p>
					</div>
					<input
						type="checkbox"
						bind:checked={notifications.email}
						class="h-5 w-5 rounded border-input text-primary focus:ring-primary"
					/>
				</label>

				<label class="flex items-center justify-between">
					<div>
						<p class="font-medium">Browser notifications</p>
						<p class="text-sm text-muted-foreground">Show desktop notifications</p>
					</div>
					<input
						type="checkbox"
						bind:checked={notifications.browser}
						class="h-5 w-5 rounded border-input text-primary focus:ring-primary"
					/>
				</label>

				<label class="flex items-center justify-between">
					<div>
						<p class="font-medium">Sound notifications</p>
						<p class="text-sm text-muted-foreground">Play sound for new messages</p>
					</div>
					<input
						type="checkbox"
						bind:checked={notifications.sound}
						class="h-5 w-5 rounded border-input text-primary focus:ring-primary"
					/>
				</label>
			</div>
		</div>

		<!-- Chat Settings -->
		<div class="rounded-xl border border-border bg-card p-6">
			<div class="mb-4 flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
					<Settings class="h-5 w-5" />
				</div>
				<div>
					<h2 class="font-semibold text-foreground">Chat Settings</h2>
					<p class="text-sm text-muted-foreground">Customize chat behavior</p>
				</div>
			</div>

			<div class="space-y-4">
				<label class="flex items-center justify-between">
					<div>
						<p class="font-medium">Press Enter to send</p>
						<p class="text-sm text-muted-foreground">Use Shift+Enter for new line</p>
					</div>
					<input
						type="checkbox"
						bind:checked={chatSettings.enterToSend}
						class="h-5 w-5 rounded border-input text-primary focus:ring-primary"
					/>
				</label>

				<label class="flex items-center justify-between">
					<div>
						<p class="font-medium">Show timestamps</p>
						<p class="text-sm text-muted-foreground">Display time for each message</p>
					</div>
					<input
						type="checkbox"
						bind:checked={chatSettings.showTimestamps}
						class="h-5 w-5 rounded border-input text-primary focus:ring-primary"
					/>
				</label>

				<label class="flex items-center justify-between">
					<div>
						<p class="font-medium">Code highlighting</p>
						<p class="text-sm text-muted-foreground">Syntax highlighting in code blocks</p>
					</div>
					<input
						type="checkbox"
						bind:checked={chatSettings.codeHighlight}
						class="h-5 w-5 rounded border-input text-primary focus:ring-primary"
					/>
				</label>
			</div>
		</div>

		<!-- Save Button -->
		<div class="flex justify-end">
			<Button onclick={saveSettings} disabled={isSaving}>
				{#if isSaving}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
					Saving...
				{:else if saved}
					<Check class="mr-2 h-4 w-4" />
					Saved!
				{:else}
					<Save class="mr-2 h-4 w-4" />
					{m.common_save()}
				{/if}
			</Button>
		</div>
	</div>
</div>

