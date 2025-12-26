<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Key,
		Plus,
		Trash2,
		Copy,
		Check,
		Building2,
		User,
		Loader2,
		Eye,
		EyeOff,
		X
	} from 'lucide-svelte';
	import { apiKeysStore } from '$lib/stores/apikeys.svelte';
	import { apiKeysApi, type CreateAPIKeyRequest } from '$lib/api/apikeys';
	import { chatApi } from '$lib/api';
	import { cn, formatRelativeTime, copyToClipboard as copyText } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import { IconButton } from '$lib/components/ui/icon-button';
	import { FormLabel } from '$lib/components/ui/form-label';
	import { Tooltip } from '$lib/components/ui/tooltip';
	import * as m from '$lib/paraglide/messages';

	// Modals
	let showCreateModal = $state(false);
	let showKeyModal = $state(false);
	let copiedKey = $state(false);

	// Create form
	let keyName = $state('');
	let keyDescription = $state('');
	let keyRPM = $state(30);
	let keyRPH = $state(500);
	let allModels = $state(true);
	let selectedModels = $state<string[]>([]);
	let availableModels = $state<Array<{ id: string; name?: string }>>([]);
	let isCreating = $state(false);
	let createError = $state('');

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		apiKeysStore.setLoading(true);
		try {
			const [keysRes, tenantsRes, modelsRes] = await Promise.allSettled([
				apiKeysApi.getPersonalKeys(),
				apiKeysApi.getUserTenants(),
				chatApi.getModels()
			]);

			if (keysRes.status === 'fulfilled') {
				apiKeysStore.setPersonalKeys(keysRes.value.api_keys || []);
			}

			if (tenantsRes.status === 'fulfilled') {
				apiKeysStore.setTenants(tenantsRes.value.tenants || []);
			}

			if (modelsRes.status === 'fulfilled') {
				availableModels = modelsRes.value.data || [];
			}
		} catch (error) {
			console.error('Failed to load API keys:', error);
		} finally {
			apiKeysStore.setLoading(false);
		}
	}

	async function loadTenantKeys(tenantId: string) {
		const tenant = apiKeysStore.tenants.find((t) => t.id === tenantId);
		if (!tenant) return;

		apiKeysStore.setCurrentTenant(tenant);
		apiKeysStore.setLoading(true);

		try {
			const response = await apiKeysApi.getTenantKeys(tenantId);
			apiKeysStore.setTenantKeys(response.api_keys || []);
		} catch (error) {
			console.error('Failed to load tenant keys:', error);
		} finally {
			apiKeysStore.setLoading(false);
		}
	}

	function switchTab(tab: 'personal' | 'tenant') {
		apiKeysStore.setActiveTab(tab);
		if (tab === 'tenant' && apiKeysStore.tenants.length > 0 && !apiKeysStore.currentTenant) {
			loadTenantKeys(apiKeysStore.tenants[0].id);
		}
	}

	function openCreateModal() {
		keyName = '';
		keyDescription = '';
		keyRPM = 30;
		keyRPH = 500;
		allModels = true;
		selectedModels = [];
		createError = '';
		showCreateModal = true;
	}

	async function handleCreateKey() {
		if (!keyName.trim()) {
			createError = 'Name is required';
			return;
		}

		isCreating = true;
		createError = '';

		try {
			const data: CreateAPIKeyRequest = {
				name: keyName.trim(),
				description: keyDescription.trim() || undefined,
				rate_limit_rpm: keyRPM,
				rate_limit_rph: keyRPH,
				all_models: allModels,
				models: allModels ? undefined : selectedModels
			};

			let response;
			if (apiKeysStore.activeTab === 'personal') {
				response = await apiKeysApi.createPersonalKey(data);
				apiKeysStore.addPersonalKey(response.api_key);
			} else if (apiKeysStore.currentTenant) {
				response = await apiKeysApi.createTenantKey(apiKeysStore.currentTenant.id, data);
				apiKeysStore.addTenantKey(response.api_key);
			}

			if (response) {
				apiKeysStore.setCreatedKeyValue(response.key);
				showCreateModal = false;
				showKeyModal = true;
			}
		} catch (error) {
			createError = error instanceof Error ? error.message : 'Failed to create key';
		} finally {
			isCreating = false;
		}
	}

	async function handleDeleteKey(keyId: string) {
		if (!confirm(m.confirm_delete_apikey())) return;

		try {
			if (apiKeysStore.activeTab === 'personal') {
				await apiKeysApi.deletePersonalKey(keyId);
				apiKeysStore.removePersonalKey(keyId);
			} else if (apiKeysStore.currentTenant) {
				await apiKeysApi.deleteTenantKey(apiKeysStore.currentTenant.id, keyId);
				apiKeysStore.removeTenantKey(keyId);
			}
		} catch (error) {
			console.error('Failed to delete key:', error);
			alert(m.alert_failed_delete_apikey());
		}
	}

	async function copyToClipboard(text: string) {
		const success = await copyText(text);
		if (success) {
			copiedKey = true;
			setTimeout(() => (copiedKey = false), 2000);
		}
	}

	function maskKey(prefix: string | undefined | null): string {
		if (!prefix) return '••••••••••••••••';
		return prefix + '••••••••';
	}

	function toggleModel(modelId: string) {
		if (selectedModels.includes(modelId)) {
			selectedModels = selectedModels.filter((m) => m !== modelId);
		} else {
			selectedModels = [...selectedModels, modelId];
		}
	}
</script>

<svelte:head>
	<title>{m.nav_apiKeys()} | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-[1600px] px-4 py-8 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-foreground">{m.apikeys_title()}</h1>
			<p class="mt-1 text-muted-foreground">{m.apikeys_subtitle()}</p>
		</div>
		<Button onclick={openCreateModal}>
			<Plus class="mr-2 h-4 w-4" />
			{m.apikeys_create()}
		</Button>
	</div>

	<!-- Tabs -->
	<div class="mb-6 flex gap-2 border-b border-border">
		<button
			onclick={() => switchTab('personal')}
			class={cn(
				'flex items-center gap-2 border-b-2 px-4 py-2.5 text-sm font-medium transition-colors',
				apiKeysStore.activeTab === 'personal'
					? 'border-primary text-primary'
					: 'border-transparent text-muted-foreground hover:text-foreground'
			)}
		>
			<User class="h-4 w-4" />
			Personal Keys
		</button>
		{#if apiKeysStore.hasTenants}
			<button
				onclick={() => switchTab('tenant')}
				class={cn(
					'flex items-center gap-2 border-b-2 px-4 py-2.5 text-sm font-medium transition-colors',
					apiKeysStore.activeTab === 'tenant'
						? 'border-primary text-primary'
						: 'border-transparent text-muted-foreground hover:text-foreground'
				)}
			>
				<Building2 class="h-4 w-4" />
				Organization Keys
			</button>
		{/if}
	</div>

	<!-- Tenant Selector (for tenant tab) -->
	{#if apiKeysStore.activeTab === 'tenant' && apiKeysStore.hasTenants}
		<div class="mb-6">
			<label for="tenant-select" class="mb-2 block text-sm font-medium">Select Organization</label>
			<select
				id="tenant-select"
				value={apiKeysStore.currentTenant?.id || ''}
				onchange={(e) => loadTenantKeys(e.currentTarget.value)}
				class="w-full max-w-xs rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
			>
				{#each apiKeysStore.tenants as tenant (tenant.id)}
					<option value={tenant.id}>{tenant.name}</option>
				{/each}
			</select>
		</div>
	{/if}

	<!-- Keys Table -->
	{#if apiKeysStore.isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if apiKeysStore.currentKeys.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Key class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-muted-foreground">No API keys yet</p>
			<Button variant="outline" class="mt-4" onclick={openCreateModal}>
				<Plus class="mr-2 h-4 w-4" />
				Create your first key
			</Button>
		</div>
	{:else}
		<div class="overflow-hidden rounded-lg border border-border">
			<table class="w-full">
				<thead class="border-b border-border bg-muted/50">
					<tr>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.common_name()}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.table_key()}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.table_models()}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.common_created()}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.common_status()}
						</th>
						<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">
							{m.common_actions()}
						</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border">
					{#each apiKeysStore.currentKeys as key (key.id)}
						<tr class="hover:bg-muted/30">
							<td class="px-4 py-3">
								<div>
									<p class="font-medium text-foreground">{key.name}</p>
									{#if key.description}
										<p class="text-xs text-muted-foreground">{key.description}</p>
									{/if}
								</div>
							</td>
							<td class="px-4 py-3">
								<div class="flex items-center gap-2">
									<code class="rounded bg-muted px-2 py-1 font-mono text-xs">
										{maskKey(key.key_prefix)}
									</code>
								<IconButton
									tooltip={m.tooltip_copy()}
									size="sm"
									onclick={() => copyText(key.key_prefix || '')}
								>
									<Copy />
								</IconButton>
								</div>
							</td>
							<td class="px-4 py-3">
								{#if key.all_models}
									<span class="text-sm text-muted-foreground">{m.table_all_models()}</span>
								{:else}
									<span class="text-sm text-muted-foreground">{key.models?.length || 0} {m.table_models().toLowerCase()}</span>
								{/if}
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{formatRelativeTime(key.created_at)}
							</td>
							<td class="px-4 py-3">
								<span
									class={cn(
										'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
										key.status === 'active'
											? 'bg-green-500/10 text-green-500'
											: 'bg-red-500/10 text-red-500'
									)}
								>
									{key.status}
								</span>
							</td>
							<td class="px-4 py-3 text-right">
								<IconButton
									tooltip={m.tooltip_delete()}
									tooltipTitle={m.apikeys_revoke()}
									variant="destructive"
									onclick={() => handleDeleteKey(key.id)}
								>
									<Trash2 />
								</IconButton>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<!-- Create Key Modal -->
{#if showCreateModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showCreateModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showCreateModal = false)}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div class="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl">
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Create API Key</h2>
				<button
					onclick={() => (showCreateModal = false)}
					class="rounded p-1 text-muted-foreground hover:bg-accent"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			<form onsubmit={(e) => { e.preventDefault(); handleCreateKey(); }} class="space-y-4">
				{#if createError}
					<div class="rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
						{createError}
					</div>
				{/if}

				<div>
					<FormLabel label={m.form_apikey_name()} description={m.form_apikey_name_desc()} required for="key-name" />
					<input
						id="key-name"
						type="text"
						bind:value={keyName}
						placeholder={m.placeholder_apikey_name()}
						required
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>

				<div>
					<FormLabel label={m.form_apikey_description()} description={m.form_apikey_description_desc()} for="key-desc" />
					<input
						id="key-desc"
						type="text"
						bind:value={keyDescription}
						placeholder={m.placeholder_org_desc()}
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>

				<div class="grid grid-cols-2 gap-4">
					<div>
						<FormLabel label={m.form_apikey_rpm()} description={m.form_apikey_rpm_desc()} for="key-rpm" />
						<input
							id="key-rpm"
							type="number"
							bind:value={keyRPM}
							min="1"
							max="1000"
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						/>
					</div>
					<div>
						<FormLabel label={m.form_apikey_rph()} description={m.form_apikey_rph_desc()} for="key-rph" />
						<input
							id="key-rph"
							type="number"
							bind:value={keyRPH}
							min="1"
							max="10000"
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						/>
					</div>
				</div>

				<div class="space-y-3">
					<label class="flex items-center gap-2">
						<input
							type="checkbox"
							bind:checked={allModels}
							class="h-4 w-4 rounded border-input"
						/>
						<span class="text-sm font-medium">Access to all models</span>
					</label>

					{#if !allModels && availableModels.length > 0}
						<div class="max-h-40 space-y-2 overflow-y-auto rounded-lg border border-input p-3">
							{#each availableModels as model (model.id)}
								<label class="flex items-center gap-2">
									<input
										type="checkbox"
										checked={selectedModels.includes(model.id)}
										onchange={() => toggleModel(model.id)}
										class="h-4 w-4 rounded border-input"
									/>
									<span class="text-sm">{model.name || model.id}</span>
								</label>
							{/each}
						</div>
					{/if}
				</div>

				<div class="flex justify-end gap-3 pt-2">
					<Button variant="outline" type="button" onclick={() => (showCreateModal = false)}>
						{m.common_cancel()}
					</Button>
					<Button type="submit" disabled={isCreating}>
						{#if isCreating}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						{m.common_create()}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- Key Created Modal -->
{#if showKeyModal && apiKeysStore.createdKeyValue}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		role="dialog"
	>
		<div class="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl">
			<div class="mb-4 text-center">
				<div class="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-green-500/10 text-green-500">
					<Key class="h-6 w-6" />
				</div>
				<h2 class="text-lg font-semibold">API Key Created</h2>
				<p class="mt-1 text-sm text-muted-foreground">
					Copy your API key now. You won't be able to see it again!
				</p>
			</div>

			<div class="mb-4 rounded-lg border border-border bg-muted p-3">
				<code class="block break-all font-mono text-sm">{apiKeysStore.createdKeyValue}</code>
			</div>

			<div class="flex gap-3">
				<Button
					variant="outline"
					class="flex-1"
					onclick={() => copyToClipboard(apiKeysStore.createdKeyValue!)}
				>
					{#if copiedKey}
						<Check class="mr-2 h-4 w-4 text-green-500" />
						Copied!
					{:else}
						<Copy class="mr-2 h-4 w-4" />
						Copy Key
					{/if}
				</Button>
				<Button
					class="flex-1"
					onclick={() => {
						showKeyModal = false;
						apiKeysStore.setCreatedKeyValue(null);
					}}
				>
					Done
				</Button>
			</div>
		</div>
	</div>
{/if}

