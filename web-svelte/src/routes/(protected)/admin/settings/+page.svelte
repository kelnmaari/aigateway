<script lang="ts">
	import { onMount } from 'svelte';
	import { Settings, Save, RefreshCw, Loader2, AlertTriangle, Check } from 'lucide-svelte';
	import { adminApi, type Setting } from '$lib/api/admin';
	import { cn } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	let settings = $state<Setting[]>([]);
	let isLoading = $state(true);
	let editedValues = $state<Record<string, string>>({});
	let savingKeys = $state<Set<string>>(new Set());
	let savedKeys = $state<Set<string>>(new Set());

	// Group settings by category
	let groupedSettings = $derived(() => {
		const groups: Record<string, Setting[]> = {};
		for (const setting of settings) {
			const category = setting.category || 'general';
			if (!groups[category]) groups[category] = [];
			groups[category].push(setting);
		}
		return groups;
	});

	onMount(async () => {
		await loadSettings();
	});

	async function loadSettings() {
		isLoading = true;
		try {
			const response = await adminApi.getSettings();
			settings = response.settings || [];
			// Initialize edited values
			editedValues = {};
			for (const s of settings) {
				editedValues[s.key] = s.value;
			}
		} catch (error) {
			console.error('Failed to load settings:', error);
		} finally {
			isLoading = false;
		}
	}

	function hasChanges(key: string): boolean {
		const original = settings.find((s) => s.key === key);
		return original ? editedValues[key] !== original.value : false;
	}

	async function saveSetting(key: string) {
		savingKeys.add(key);
		savingKeys = new Set(savingKeys);

		try {
			await adminApi.updateSetting(key, editedValues[key]);
			// Update original value
			settings = settings.map((s) => (s.key === key ? { ...s, value: editedValues[key] } : s));
			// Show saved indicator
			savedKeys.add(key);
			savedKeys = new Set(savedKeys);
			setTimeout(() => {
				savedKeys.delete(key);
				savedKeys = new Set(savedKeys);
			}, 2000);
		} catch (error) {
			console.error('Failed to save setting:', error);
			alert('Failed to save setting');
		} finally {
			savingKeys.delete(key);
			savingKeys = new Set(savingKeys);
		}
	}

	function renderInput(setting: Setting) {
		const value = editedValues[setting.key] ?? setting.value;

		if (setting.type === 'boolean') {
			return {
				type: 'checkbox',
				checked: value === 'true'
			};
		}

		if (setting.type === 'number') {
			return {
				type: 'number',
				value
			};
		}

		if (setting.type === 'json' || value.length > 100) {
			return {
				type: 'textarea',
				value
			};
		}

		return {
			type: 'text',
			value
		};
	}

	function getCategoryLabel(category: string): string {
		const labels: Record<string, string> = {
			general: 'General',
			server: 'Server',
			auth: 'Authentication',
			ollama: 'Ollama',
			storage: 'Storage',
			logging: 'Logging',
			security: 'Security'
		};
		return labels[category] || category.charAt(0).toUpperCase() + category.slice(1);
	}
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h2 class="text-lg font-semibold">{m.admin_settings()}</h2>
		<Button variant="outline" onclick={loadSettings} disabled={isLoading}>
			<RefreshCw class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
			{m.common_refresh()}
		</Button>
	</div>

	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if settings.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Settings class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-muted-foreground">No settings available</p>
		</div>
	{:else}
		<div class="space-y-8">
			{#each Object.entries(groupedSettings()) as [category, categorySettings]}
				<section class="rounded-xl border border-border bg-card">
					<div class="border-b border-border px-6 py-4">
						<h3 class="font-semibold">{getCategoryLabel(category)}</h3>
					</div>
					<div class="divide-y divide-border">
						{#each categorySettings as setting (setting.key)}
							{@const inputConfig = renderInput(setting)}
							<div class="flex items-start justify-between gap-4 px-6 py-4">
								<div class="flex-1">
									<div class="flex items-center gap-2">
										<label for={setting.key} class="font-medium text-foreground">
											{setting.key}
										</label>
										{#if setting.requires_restart}
											<span class="flex items-center gap-1 rounded bg-amber-500/10 px-1.5 py-0.5 text-xs text-amber-500">
												<AlertTriangle class="h-3 w-3" />
												Restart required
											</span>
										{/if}
										{#if setting.storage === 'yaml'}
											<span class="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
												YAML
											</span>
										{/if}
									</div>
									{#if setting.description}
										<p class="mt-1 text-sm text-muted-foreground">{setting.description}</p>
									{/if}
								</div>

								<div class="flex items-center gap-2">
									{#if inputConfig.type === 'checkbox'}
										<label class="relative inline-flex cursor-pointer items-center">
											<input
												type="checkbox"
												checked={editedValues[setting.key] === 'true'}
												onchange={(e) => (editedValues[setting.key] = e.currentTarget.checked ? 'true' : 'false')}
												class="peer sr-only"
											/>
											<div class="peer h-6 w-11 rounded-full bg-muted after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary peer-checked:after:translate-x-full"></div>
										</label>
									{:else if inputConfig.type === 'textarea'}
										<textarea
											id={setting.key}
											value={editedValues[setting.key]}
											oninput={(e) => (editedValues[setting.key] = e.currentTarget.value)}
											rows="3"
											class="w-64 rounded-lg border border-input bg-background px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2 focus:ring-ring"
										></textarea>
									{:else}
										<input
											id={setting.key}
											type={inputConfig.type}
											value={editedValues[setting.key]}
											oninput={(e) => (editedValues[setting.key] = e.currentTarget.value)}
											class="w-48 rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
										/>
									{/if}

									{#if hasChanges(setting.key)}
										<Button
											size="sm"
											onclick={() => saveSetting(setting.key)}
											disabled={savingKeys.has(setting.key)}
										>
											{#if savingKeys.has(setting.key)}
												<Loader2 class="h-4 w-4 animate-spin" />
											{:else}
												<Save class="h-4 w-4" />
											{/if}
										</Button>
									{:else if savedKeys.has(setting.key)}
										<span class="flex h-8 w-8 items-center justify-center text-green-500">
											<Check class="h-4 w-4" />
										</span>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				</section>
			{/each}
		</div>
	{/if}
</div>

