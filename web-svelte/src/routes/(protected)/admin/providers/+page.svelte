<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Server,
		Plus,
		Pencil,
		Trash2,
		Loader2,
		RefreshCw,
		Activity,
		Search,
		X,
		Globe,
		Zap,
		Eye,
		EyeOff,
		ChevronDown,
		ChevronUp
	} from 'lucide-svelte';
	import {
		registryApi,
		type ModelProvider,
		type ModelRegistry,
		type ProviderType,
		type CreateProviderRequest,
		type UpdateProviderRequest
	} from '$lib/api/registry';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import { toast } from 'svelte-sonner';

	// ==================== State ====================

	let providers = $state<ModelProvider[]>([]);
	let models = $state<ModelRegistry[]>([]);
	let isLoading = $state(true);
	let isLoadingModels = $state(false);
	let registryDisabled = $state(false);

	// Dialog state
	let showFormDialog = $state(false);
	let isEditing = $state(false);
	let editingProviderId = $state<string | null>(null);
	let isSaving = $state(false);
	let formError = $state('');

	// Form fields
	let formName = $state('');
	let formType = $state<ProviderType>('openai');
	let formBaseUrl = $state('');
	let formApiKey = $state('');
	let formEnabled = $state(true);
	let formPriority = $state(10);

	// Action loading states (per provider ID)
	let healthCheckLoading = $state<Record<string, boolean>>({});
	let discoverLoading = $state<Record<string, boolean>>({});
	let deleteLoading = $state<Record<string, boolean>>({});

	// Models section toggle
	let showModels = $state(false);

	// API key visibility per provider
	let showApiKey = $state(false);

	// ==================== Provider Type Metadata ====================

	const providerTypes: ProviderType[] = ['openai', 'anthropic', 'gemini', 'deepseek', 'vllm', 'custom'];

	const providerTypeConfig: Record<ProviderType, { label: string; color: string; defaultUrl: string }> = {
		openai: { label: 'OpenAI', color: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400', defaultUrl: 'https://api.openai.com/v1' },
		anthropic: { label: 'Anthropic', color: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400', defaultUrl: 'https://api.anthropic.com' },
		gemini: { label: 'Gemini', color: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400', defaultUrl: 'https://generativelanguage.googleapis.com' },
		deepseek: { label: 'DeepSeek', color: 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/30 dark:text-indigo-400', defaultUrl: 'https://api.deepseek.com' },
		vllm: { label: 'vLLM', color: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400', defaultUrl: 'http://localhost:8000/v1' },
		custom: { label: 'Custom', color: 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400', defaultUrl: '' }
	};

	const healthColors: Record<string, string> = {
		healthy: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400',
		unhealthy: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
		unknown: 'bg-gray-100 text-gray-600 dark:bg-gray-800/50 dark:text-gray-400',
		checking: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
	};

	// ==================== Lifecycle ====================

	onMount(async () => {
		await loadProviders();
	});

	// ==================== Data Loading ====================

	async function loadProviders() {
		isLoading = true;
		try {
			const response = await registryApi.listProviders();
			providers = response.providers || [];
		} catch (error) {
			const msg = error instanceof Error ? error.message : String(error);
			if (msg.includes('404') || msg.includes('endpoint not found')) {
				registryDisabled = true;
			} else {
				console.error('Failed to load providers:', error);
				toast.error('Failed to load providers');
			}
		} finally {
			isLoading = false;
		}
	}

	async function loadModels() {
		if (registryDisabled) return;
		isLoadingModels = true;
		try {
			const response = await registryApi.listModels();
			models = response.models || [];
		} catch (error) {
			const msg = error instanceof Error ? error.message : String(error);
			if (!msg.includes('404')) {
				console.error('Failed to load registry models:', error);
				toast.error('Failed to load registry models');
			}
		} finally {
			isLoadingModels = false;
		}
	}

	// ==================== Form Helpers ====================

	function openCreateDialog() {
		isEditing = false;
		editingProviderId = null;
		formName = '';
		formType = 'openai';
		formBaseUrl = providerTypeConfig['openai'].defaultUrl;
		formApiKey = '';
		formEnabled = true;
		formPriority = 10;
		formError = '';
		showApiKey = false;
		showFormDialog = true;
	}

	function openEditDialog(provider: ModelProvider) {
		isEditing = true;
		editingProviderId = provider.id;
		formName = provider.name;
		formType = provider.provider_type;
		formBaseUrl = provider.base_url;
		formApiKey = '';
		formEnabled = provider.enabled;
		formPriority = provider.priority;
		formError = '';
		showApiKey = false;
		showFormDialog = true;
	}

	function closeDialog() {
		showFormDialog = false;
		editingProviderId = null;
		formError = '';
	}

	function handleTypeChange() {
		if (!isEditing) {
			formBaseUrl = providerTypeConfig[formType].defaultUrl;
		}
	}

	// ==================== CRUD Operations ====================

	async function handleSave() {
		if (!formName.trim()) {
			formError = 'Provider name is required';
			return;
		}
		if (!formBaseUrl.trim()) {
			formError = 'Base URL is required';
			return;
		}

		isSaving = true;
		formError = '';

		try {
			if (isEditing && editingProviderId) {
				const data: UpdateProviderRequest = {
					name: formName.trim(),
					provider_type: formType,
					base_url: formBaseUrl.trim(),
					enabled: formEnabled,
					priority: formPriority
				};
				if (formApiKey.trim()) {
					data.api_key = formApiKey.trim();
				}
				const updated = await registryApi.updateProvider(editingProviderId, data);
				providers = providers.map((p) => (p.id === editingProviderId ? updated : p));
				toast.success(`Provider "${formName}" updated`);
			} else {
				const data: CreateProviderRequest = {
					name: formName.trim(),
					provider_type: formType,
					base_url: formBaseUrl.trim(),
					enabled: formEnabled,
					priority: formPriority
				};
				if (formApiKey.trim()) {
					data.api_key = formApiKey.trim();
				}
				const created = await registryApi.createProvider(data);
				providers = [...providers, created];
				toast.success(`Provider "${formName}" created`);
			}
			closeDialog();
		} catch (error) {
			const msg = error instanceof Error ? error.message : 'Operation failed';
			formError = msg;
			console.error('Save provider failed:', error);
		} finally {
			isSaving = false;
		}
	}

	async function handleDelete(provider: ModelProvider) {
		if (!confirm(`Delete provider "${provider.name}"? This cannot be undone.`)) return;

		deleteLoading = { ...deleteLoading, [provider.id]: true };
		try {
			await registryApi.deleteProvider(provider.id);
			providers = providers.filter((p) => p.id !== provider.id);
			toast.success(`Provider "${provider.name}" deleted`);
		} catch (error) {
			console.error('Delete provider failed:', error);
			toast.error(`Failed to delete provider "${provider.name}"`);
		} finally {
			deleteLoading = { ...deleteLoading, [provider.id]: false };
		}
	}

	// ==================== Provider Actions ====================

	async function handleHealthCheck(provider: ModelProvider) {
		healthCheckLoading = { ...healthCheckLoading, [provider.id]: true };
		try {
			const result = await registryApi.healthCheckProvider(provider.id);
			// Update the provider health status locally
			providers = providers.map((p) => {
				if (p.id === provider.id) {
					return {
						...p,
						health_status: result.status === 'healthy' ? 'healthy' : 'unhealthy',
						last_health_check: new Date().toISOString(),
						error_message: result.status !== 'healthy' ? result.message : undefined
					};
				}
				return p;
			});
			if (result.status === 'healthy') {
				toast.success(`${provider.name}: healthy (${result.response_time_ms ?? 0}ms)`);
			} else {
				toast.error(`${provider.name}: ${result.message}`);
			}
		} catch (error) {
			const msg = error instanceof Error ? error.message : 'Health check failed';
			providers = providers.map((p) => {
				if (p.id === provider.id) {
					return { ...p, health_status: 'unhealthy', error_message: msg };
				}
				return p;
			});
			toast.error(`${provider.name}: ${msg}`);
		} finally {
			healthCheckLoading = { ...healthCheckLoading, [provider.id]: false };
		}
	}

	async function handleDiscover(provider: ModelProvider) {
		discoverLoading = { ...discoverLoading, [provider.id]: true };
		try {
			const result = await registryApi.discoverModels(provider.id);
			toast.success(`${provider.name}: discovered ${result.models_found} model(s)`);
			// Refresh models if the section is visible
			if (showModels) {
				await loadModels();
			}
		} catch (error) {
			const msg = error instanceof Error ? error.message : 'Model discovery failed';
			toast.error(`${provider.name}: ${msg}`);
		} finally {
			discoverLoading = { ...discoverLoading, [provider.id]: false };
		}
	}

	function toggleModelsSection() {
		showModels = !showModels;
		if (showModels && models.length === 0) {
			loadModels();
		}
	}

	// ==================== Helpers ====================

	function getProviderName(providerId: string): string {
		const p = providers.find((pr) => pr.id === providerId);
		return p?.name ?? providerId.substring(0, 8);
	}

	function getHealthColor(status: string): string {
		return healthColors[status] || healthColors['unknown'];
	}

	let sortedProviders = $derived(
		[...providers].sort((a, b) => a.priority - b.priority)
	);
</script>

<div class="space-y-6">
	<!-- Header -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<h2 class="text-lg font-semibold text-foreground">Model Providers</h2>
			<p class="text-sm text-muted-foreground">
				Manage external AI model providers and their configurations
			</p>
		</div>
		<div class="flex items-center gap-2">
			<Button variant="outline" size="sm" onclick={loadProviders} disabled={isLoading}>
				<RefreshCw class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
				Refresh
			</Button>
			<Button size="sm" onclick={openCreateDialog}>
				<Plus class="mr-2 h-4 w-4" />
				Add Provider
			</Button>
		</div>
	</div>

	<!-- Providers Table -->
	{#if registryDisabled}
		<div class="rounded-lg border border-amber-300 bg-amber-50 dark:border-amber-700 dark:bg-amber-950/30 py-16 text-center">
			<Server class="mx-auto h-12 w-12 text-amber-500/60" />
			<p class="mt-4 text-lg font-medium text-amber-700 dark:text-amber-400">Model Registry is disabled</p>
			<p class="mt-1 text-sm text-amber-600/70 dark:text-amber-500/70">
				Enable it in server configuration: <code class="bg-amber-100 dark:bg-amber-900/50 px-1 py-0.5 rounded text-xs">model_registry.enabled: true</code>
			</p>
		</div>
	{:else if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if providers.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Server class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-lg font-medium text-muted-foreground">No providers configured</p>
			<p class="mt-1 text-sm text-muted-foreground/70">
				Add an AI provider to start routing model requests
			</p>
			<Button class="mt-4" size="sm" onclick={openCreateDialog}>
				<Plus class="mr-2 h-4 w-4" />
				Add Provider
			</Button>
		</div>
	{:else}
		<div class="overflow-hidden rounded-lg border border-border">
			<table class="w-full">
				<thead class="border-b border-border bg-muted/50">
					<tr>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							Name
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							Type
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground hidden md:table-cell">
							Base URL
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							Health
						</th>
						<th class="px-4 py-3 text-center text-xs font-medium uppercase text-muted-foreground hidden sm:table-cell">
							Enabled
						</th>
						<th class="px-4 py-3 text-center text-xs font-medium uppercase text-muted-foreground hidden lg:table-cell">
							Priority
						</th>
						<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">
							Actions
						</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border">
					{#each sortedProviders as provider (provider.id)}
						<tr class="hover:bg-muted/30 transition-colors">
							<!-- Name -->
							<td class="px-4 py-3">
								<div class="flex items-center gap-2">
									<Globe class="h-4 w-4 text-muted-foreground shrink-0" />
									<div>
										<span class="font-medium text-foreground">{provider.name}</span>
										{#if provider.error_message}
											<p class="text-xs text-destructive mt-0.5 max-w-[200px] truncate" title={provider.error_message}>
												{provider.error_message}
											</p>
										{/if}
									</div>
								</div>
							</td>

							<!-- Type Badge -->
							<td class="px-4 py-3">
								<span class={cn(
									'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
									providerTypeConfig[provider.provider_type]?.color ?? providerTypeConfig['custom'].color
								)}>
									{providerTypeConfig[provider.provider_type]?.label ?? provider.provider_type}
								</span>
							</td>

							<!-- Base URL -->
							<td class="px-4 py-3 hidden md:table-cell">
								<span class="text-sm text-muted-foreground font-mono max-w-[250px] truncate block" title={provider.base_url}>
									{provider.base_url}
								</span>
							</td>

							<!-- Health Badge -->
							<td class="px-4 py-3">
								<div class="flex items-center gap-1.5">
									<span class={cn(
										'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
										getHealthColor(provider.health_status)
									)}>
										{provider.health_status}
									</span>
									{#if provider.last_health_check}
										<span class="text-xs text-muted-foreground hidden xl:inline" title={provider.last_health_check}>
											{formatRelativeTime(provider.last_health_check)}
										</span>
									{/if}
								</div>
							</td>

							<!-- Enabled -->
							<td class="px-4 py-3 text-center hidden sm:table-cell">
								{#if provider.enabled}
									<span class="inline-flex h-5 w-5 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-900/30">
										<span class="h-2 w-2 rounded-full bg-emerald-500"></span>
									</span>
								{:else}
									<span class="inline-flex h-5 w-5 items-center justify-center rounded-full bg-gray-100 dark:bg-gray-800/50">
										<span class="h-2 w-2 rounded-full bg-gray-400"></span>
									</span>
								{/if}
							</td>

							<!-- Priority -->
							<td class="px-4 py-3 text-center hidden lg:table-cell">
								<span class="text-sm text-muted-foreground">{provider.priority}</span>
							</td>

							<!-- Actions -->
							<td class="px-4 py-3">
								<div class="flex items-center justify-end gap-1">
									<button
										onclick={() => handleHealthCheck(provider)}
										disabled={healthCheckLoading[provider.id]}
										class="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors disabled:opacity-50"
										title="Health Check"
									>
										{#if healthCheckLoading[provider.id]}
											<Loader2 class="h-4 w-4 animate-spin" />
										{:else}
											<Activity class="h-4 w-4" />
										{/if}
									</button>
									<button
										onclick={() => handleDiscover(provider)}
										disabled={discoverLoading[provider.id]}
										class="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors disabled:opacity-50"
										title="Discover Models"
									>
										{#if discoverLoading[provider.id]}
											<Loader2 class="h-4 w-4 animate-spin" />
										{:else}
											<Search class="h-4 w-4" />
										{/if}
									</button>
									<button
										onclick={() => openEditDialog(provider)}
										class="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
										title="Edit"
									>
										<Pencil class="h-4 w-4" />
									</button>
									<button
										onclick={() => handleDelete(provider)}
										disabled={deleteLoading[provider.id]}
										class="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors disabled:opacity-50"
										title="Delete"
									>
										{#if deleteLoading[provider.id]}
											<Loader2 class="h-4 w-4 animate-spin" />
										{:else}
											<Trash2 class="h-4 w-4" />
										{/if}
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<!-- Models Section Toggle -->
	<div class="border-t border-border pt-4">
		<button
			onclick={toggleModelsSection}
			class="flex w-full items-center justify-between rounded-lg border border-border bg-card/50 px-4 py-3 text-left hover:bg-muted/30 transition-colors"
		>
			<div class="flex items-center gap-2">
				<Zap class="h-4 w-4 text-muted-foreground" />
				<span class="text-sm font-medium text-foreground">Registry Models</span>
				{#if models.length > 0}
					<span class="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
						{models.length}
					</span>
				{/if}
			</div>
			{#if showModels}
				<ChevronUp class="h-4 w-4 text-muted-foreground" />
			{:else}
				<ChevronDown class="h-4 w-4 text-muted-foreground" />
			{/if}
		</button>

		{#if showModels}
			<div class="mt-4">
				{#if isLoadingModels}
					<div class="flex items-center justify-center py-12">
						<Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
					</div>
				{:else if models.length === 0}
					<div class="rounded-lg border border-dashed border-border py-10 text-center">
						<Zap class="mx-auto h-10 w-10 text-muted-foreground/40" />
						<p class="mt-3 text-sm text-muted-foreground">
							No models in registry. Use "Discover Models" on a provider to populate.
						</p>
					</div>
				{:else}
					<div class="overflow-hidden rounded-lg border border-border">
						<table class="w-full">
							<thead class="border-b border-border bg-muted/50">
								<tr>
									<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
										Model ID
									</th>
									<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
										Name
									</th>
									<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground hidden md:table-cell">
										Provider
									</th>
									<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground hidden lg:table-cell">
										Capabilities
									</th>
									<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground hidden sm:table-cell">
										Context
									</th>
									<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
										Status
									</th>
									<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
										Health
									</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-border">
								{#each models as model (model.id)}
									<tr class="hover:bg-muted/30 transition-colors">
										<td class="px-4 py-3">
											<span class="text-sm font-mono text-foreground">{model.model_id}</span>
										</td>
										<td class="px-4 py-3">
											<span class="text-sm text-foreground">{model.model_name}</span>
										</td>
										<td class="px-4 py-3 hidden md:table-cell">
											<span class="text-sm text-muted-foreground">
												{getProviderName(model.provider_id)}
											</span>
										</td>
										<td class="px-4 py-3 hidden lg:table-cell">
											<div class="flex flex-wrap gap-1">
												{#each (model.capabilities || []).slice(0, 3) as cap}
													<span class="inline-flex rounded-md bg-accent px-1.5 py-0.5 text-xs text-accent-foreground">
														{cap}
													</span>
												{/each}
												{#if (model.capabilities || []).length > 3}
													<span class="inline-flex rounded-md bg-accent px-1.5 py-0.5 text-xs text-accent-foreground">
														+{model.capabilities.length - 3}
													</span>
												{/if}
											</div>
										</td>
										<td class="px-4 py-3 hidden sm:table-cell">
											<span class="text-sm text-muted-foreground">
												{model.context_length ? `${(model.context_length / 1024).toFixed(0)}K` : '-'}
											</span>
										</td>
										<td class="px-4 py-3">
											<span class={cn(
												'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
												model.status === 'active'
													? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400'
													: 'bg-gray-100 text-gray-600 dark:bg-gray-800/50 dark:text-gray-400'
											)}>
												{model.status}
											</span>
										</td>
										<td class="px-4 py-3">
											<span class={cn(
												'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
												getHealthColor(model.health_status)
											)}>
												{model.health_status}
											</span>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>

<!-- Create/Edit Provider Dialog -->
{#if showFormDialog}
	<!-- Backdrop -->
	<div
		class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm"
		onclick={closeDialog}
		onkeydown={(e) => e.key === 'Escape' && closeDialog()}
		role="button"
		tabindex="-1"
	></div>

	<!-- Dialog -->
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<div
			class="relative w-full max-w-lg rounded-xl border border-border bg-background p-6 shadow-lg"
			onclick={(e) => e.stopPropagation()}
			role="dialog"
			aria-modal="true"
			aria-label={isEditing ? 'Edit Provider' : 'Add Provider'}
		>
			<!-- Header -->
			<div class="mb-6 flex items-center justify-between">
				<h3 class="text-lg font-semibold text-foreground">
					{isEditing ? 'Edit Provider' : 'Add Provider'}
				</h3>
				<button
					onclick={closeDialog}
					class="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<!-- Form -->
			<form
				onsubmit={(e) => { e.preventDefault(); handleSave(); }}
				class="space-y-4"
			>
				<!-- Name -->
				<div>
					<label for="provider-name" class="mb-1.5 block text-sm font-medium text-foreground">
						Name
					</label>
					<input
						id="provider-name"
						type="text"
						bind:value={formName}
						placeholder="e.g. OpenAI Production"
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>

				<!-- Type -->
				<div>
					<label for="provider-type" class="mb-1.5 block text-sm font-medium text-foreground">
						Provider Type
					</label>
					<select
						id="provider-type"
						bind:value={formType}
						onchange={handleTypeChange}
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
					>
						{#each providerTypes as pt}
							<option value={pt}>{providerTypeConfig[pt].label}</option>
						{/each}
					</select>
				</div>

				<!-- Base URL -->
				<div>
					<label for="provider-url" class="mb-1.5 block text-sm font-medium text-foreground">
						Base URL
					</label>
					<input
						id="provider-url"
						type="url"
						bind:value={formBaseUrl}
						placeholder="https://api.example.com/v1"
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm font-mono text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>

				<!-- API Key -->
				<div>
					<label for="provider-apikey" class="mb-1.5 block text-sm font-medium text-foreground">
						API Key
						{#if isEditing}
							<span class="font-normal text-muted-foreground">(leave blank to keep current)</span>
						{/if}
					</label>
					<div class="relative">
						<input
							id="provider-apikey"
							type={showApiKey ? 'text' : 'password'}
							bind:value={formApiKey}
							placeholder={isEditing ? '********' : 'sk-...'}
							class="w-full rounded-lg border border-input bg-background px-3 py-2 pr-10 text-sm font-mono text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
						/>
						<button
							type="button"
							onclick={() => showApiKey = !showApiKey}
							class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground"
						>
							{#if showApiKey}
								<EyeOff class="h-4 w-4" />
							{:else}
								<Eye class="h-4 w-4" />
							{/if}
						</button>
					</div>
				</div>

				<!-- Enabled + Priority row -->
				<div class="flex gap-4">
					<div class="flex-1">
						<label for="provider-priority" class="mb-1.5 block text-sm font-medium text-foreground">
							Priority
						</label>
						<input
							id="provider-priority"
							type="number"
							bind:value={formPriority}
							min="1"
							max="100"
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
						/>
						<p class="mt-1 text-xs text-muted-foreground">Lower = higher priority</p>
					</div>
					<div class="flex items-center gap-2 pt-6">
						<input
							id="provider-enabled"
							type="checkbox"
							bind:checked={formEnabled}
							class="h-4 w-4 rounded border-input text-primary focus:ring-ring"
						/>
						<label for="provider-enabled" class="text-sm font-medium text-foreground">
							Enabled
						</label>
					</div>
				</div>

				<!-- Error -->
				{#if formError}
					<div class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
						{formError}
					</div>
				{/if}

				<!-- Actions -->
				<div class="flex justify-end gap-2 pt-2">
					<Button variant="outline" type="button" onclick={closeDialog}>
						Cancel
					</Button>
					<Button type="submit" disabled={isSaving}>
						{#if isSaving}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						{isEditing ? 'Save Changes' : 'Create Provider'}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
