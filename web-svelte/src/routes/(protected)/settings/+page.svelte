<script lang="ts">
	import { onMount } from 'svelte';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import {
		faGear,
		faBell,
		faGlobe,
		faMoon,
		faSun,
		faDesktop,
		faPalette,
		faFloppyDisk,
		faSpinner,
		faCheck
	} from '@fortawesome/free-solid-svg-icons';
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
		themeStore.set(newTheme);
	}

	function handleLocaleChange(newLocale: 'en' | 'ru') {
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

<div class="mx-auto max-w-[1600px] px-4 py-8 sm:px-6 lg:px-8">
	<div class="mb-8">
		<h1 class="text-2xl font-bold text-foreground">{m.nav_settings()}</h1>
		<p class="mt-1 text-muted-foreground">{m.settings_subtitle()}</p>
	</div>

	<div class="grid gap-6 lg:grid-cols-2">
		<!-- Left Column -->
		<div class="space-y-6">
			<!-- Appearance -->
			<div class="rounded-xl border border-border bg-card p-6">
				<div class="mb-4 flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
						<FontAwesomeIcon icon={faPalette} class="h-5 w-5" />
					</div>
					<div>
						<h2 class="font-semibold text-foreground">{m.settings_appearance()}</h2>
						<p class="text-sm text-muted-foreground">{m.settings_appearance_desc()}</p>
					</div>
				</div>

				<div class="space-y-4">
					<!-- Theme -->
					<div>
						<span class="mb-2 block text-sm font-medium">{m.settings_theme()}</span>
						<div class="flex gap-2" role="radiogroup" aria-label={m.settings_theme()}>
							<button
								onclick={() => handleThemeChange('light')}
								class={cn(
									'flex flex-1 items-center justify-center gap-2 rounded-lg border px-4 py-3 transition-colors',
									theme === 'light' ? 'border-primary bg-primary/10 text-primary' : 'border-input hover:bg-accent'
								)}
							>
								<FontAwesomeIcon icon={faSun} class="h-4 w-4" />
								{m.theme_light()}
							</button>
							<button
								onclick={() => handleThemeChange('dark')}
								class={cn(
									'flex flex-1 items-center justify-center gap-2 rounded-lg border px-4 py-3 transition-colors',
									theme === 'dark' ? 'border-primary bg-primary/10 text-primary' : 'border-input hover:bg-accent'
								)}
							>
								<FontAwesomeIcon icon={faMoon} class="h-4 w-4" />
								{m.theme_dark()}
							</button>
							<button
								onclick={() => handleThemeChange('system')}
								class={cn(
									'flex flex-1 items-center justify-center gap-2 rounded-lg border px-4 py-3 transition-colors',
									theme === 'system' ? 'border-primary bg-primary/10 text-primary' : 'border-input hover:bg-accent'
								)}
							>
								<FontAwesomeIcon icon={faDesktop} class="h-4 w-4" />
								{m.theme_system()}
							</button>
						</div>
					</div>

					<!-- Language -->
					<div>
						<span class="mb-2 block text-sm font-medium">{m.settings_language()}</span>
						<div class="flex gap-2" role="radiogroup" aria-label={m.settings_language()}>
							{#each locales as loc}
								<button
									onclick={() => handleLocaleChange(loc)}
									class={cn(
										'flex flex-1 items-center justify-center gap-2 rounded-lg border px-4 py-3 transition-colors',
										locale === loc ? 'border-primary bg-primary/10 text-primary' : 'border-input hover:bg-accent'
									)}
								>
									<FontAwesomeIcon icon={faGlobe} class="h-4 w-4" />
									{getLocaleLabel(loc)}
								</button>
							{/each}
						</div>
						<p class="mt-2 text-xs text-muted-foreground">
							{m.settings_language_reload()}
						</p>
					</div>
				</div>
			</div>

			<!-- Notifications -->
			<div class="rounded-xl border border-border bg-card p-6">
				<div class="mb-4 flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
						<FontAwesomeIcon icon={faBell} class="h-5 w-5" />
					</div>
					<div>
						<h2 class="font-semibold text-foreground">{m.settings_notifications()}</h2>
						<p class="text-sm text-muted-foreground">{m.settings_notifications_desc()}</p>
					</div>
				</div>

				<div class="space-y-4">
					<label class="flex items-center justify-between">
						<div>
							<p class="font-medium">{m.settings_notif_email()}</p>
							<p class="text-sm text-muted-foreground">{m.settings_notif_email_desc()}</p>
						</div>
						<input
							type="checkbox"
							bind:checked={notifications.email}
							class="h-5 w-5 rounded border-input text-primary focus:ring-primary"
						/>
					</label>

					<label class="flex items-center justify-between">
						<div>
							<p class="font-medium">{m.settings_notif_browser()}</p>
							<p class="text-sm text-muted-foreground">{m.settings_notif_browser_desc()}</p>
						</div>
						<input
							type="checkbox"
							bind:checked={notifications.browser}
							class="h-5 w-5 rounded border-input text-primary focus:ring-primary"
						/>
					</label>

					<label class="flex items-center justify-between">
						<div>
							<p class="font-medium">{m.settings_notif_sound()}</p>
							<p class="text-sm text-muted-foreground">{m.settings_notif_sound_desc()}</p>
						</div>
						<input
							type="checkbox"
							bind:checked={notifications.sound}
							class="h-5 w-5 rounded border-input text-primary focus:ring-primary"
						/>
					</label>
				</div>
			</div>
		</div>

		<!-- Right Column -->
		<div class="space-y-6">
			<!-- Chat Settings -->
			<div class="rounded-xl border border-border bg-card p-6">
				<div class="mb-4 flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
						<FontAwesomeIcon icon={faGear} class="h-5 w-5" />
					</div>
					<div>
						<h2 class="font-semibold text-foreground">{m.settings_chat()}</h2>
						<p class="text-sm text-muted-foreground">{m.settings_chat_desc()}</p>
					</div>
				</div>

				<div class="space-y-4">
					<label class="flex items-center justify-between">
						<div>
							<p class="font-medium">{m.settings_chat_enter()}</p>
							<p class="text-sm text-muted-foreground">{m.settings_chat_enter_desc()}</p>
						</div>
						<input
							type="checkbox"
							bind:checked={chatSettings.enterToSend}
							class="h-5 w-5 rounded border-input text-primary focus:ring-primary"
						/>
					</label>

					<label class="flex items-center justify-between">
						<div>
							<p class="font-medium">{m.settings_chat_timestamps()}</p>
							<p class="text-sm text-muted-foreground">{m.settings_chat_timestamps_desc()}</p>
						</div>
						<input
							type="checkbox"
							bind:checked={chatSettings.showTimestamps}
							class="h-5 w-5 rounded border-input text-primary focus:ring-primary"
						/>
					</label>

					<label class="flex items-center justify-between">
						<div>
							<p class="font-medium">{m.settings_chat_code()}</p>
							<p class="text-sm text-muted-foreground">{m.settings_chat_code_desc()}</p>
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
						<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
						{m.common_saving()}
					{:else if saved}
						<FontAwesomeIcon icon={faCheck} class="mr-2 h-4 w-4" />
						{m.success_saved()}
					{:else}
						<FontAwesomeIcon icon={faFloppyDisk} class="mr-2 h-4 w-4" />
						{m.common_save()}
					{/if}
				</Button>
			</div>
		</div>
	</div>
</div>

