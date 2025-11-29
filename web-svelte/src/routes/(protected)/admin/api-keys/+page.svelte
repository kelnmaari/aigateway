<script lang="ts">
	import { onMount } from 'svelte';
	import { Key, Search, Loader2, Trash2, Eye, EyeOff, MoreVertical, X } from 'lucide-svelte';
	import { api } from '$lib/api/client';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	interface AdminAPIKey {
		id: string;
		name: string;
		key_prefix: string;
		user_id?: string;
		username?: string;
		tenant_id?: string;
		tenant_name?: string;
		status: 'active' | 'disabled' | 'revoked';
		all_models: boolean;
		models?: string[];
		created_at: string;
		last_used_at?: string;
	}

	let keys = $state<AdminAPIKey[]>([]);
	let isLoading = $state(true);
	let search = $state('');
	let showMenuFor = $state<string | null>(null);

	onMount(async () => {
		await loadKeys();
	});

	async function loadKeys() {
		isLoading = true;
		try {
			const params = new URLSearchParams();
			if (search) params.set('search', search);
			const response = await api.get<{ api_keys: AdminAPIKey[] }>(`/api/admin/api-keys?${params}`);
			keys = response.api_keys || [];
		} catch (error) {
			console.error('Failed to load API keys:', error);
		} finally {
			isLoading = false;
		}
	}

	async function handleSearch() {
		await loadKeys();
	}

	async function handleRevokeKey(key: AdminAPIKey) {
		if (!confirm(`Revoke API key "${key.name}"?`)) return;

		try {
			await api.delete(`/api/admin/api-keys/${key.id}`);
			keys = keys.filter((k) => k.id !== key.id);
		} catch (error) {
			console.error('Failed to revoke key:', error);
			alert('Failed to revoke API key');
		}
		showMenuFor = null;
	}

	function maskKey(prefix: string): string {
		return prefix + '••••••••••••••••';
	}
</script>

<div class="space-y-6">
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<h2 class="text-lg font-semibold">All API Keys</h2>
		<form onsubmit={(e) => { e.preventDefault(); handleSearch(); }} class="relative">
			<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
			<input
				type="text"
				bind:value={search}
				placeholder="Search keys..."
				class="w-64 rounded-lg border border-input bg-background py-2 pl-9 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
			/>
		</form>
	</div>

	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if keys.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Key class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-muted-foreground">No API keys found</p>
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
							Key
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							Owner
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.common_status()}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							Last Used
						</th>
						<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">
							{m.common_actions()}
						</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border">
					{#each keys as key (key.id)}
						<tr class="hover:bg-muted/30">
							<td class="px-4 py-3">
								<p class="font-medium">{key.name}</p>
							</td>
							<td class="px-4 py-3">
								<code class="rounded bg-muted px-2 py-1 font-mono text-xs">
									{maskKey(key.key_prefix)}
								</code>
							</td>
							<td class="px-4 py-3 text-sm">
								{#if key.tenant_name}
									<span class="text-muted-foreground">{key.tenant_name}</span>
								{:else if key.username}
									{key.username}
								{:else}
									<span class="text-muted-foreground">Unknown</span>
								{/if}
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
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{key.last_used_at ? formatRelativeTime(key.last_used_at) : 'Never'}
							</td>
							<td class="relative px-4 py-3 text-right">
								<button
									onclick={() => (showMenuFor = showMenuFor === key.id ? null : key.id)}
									class="rounded p-1.5 text-muted-foreground hover:bg-accent"
								>
									<MoreVertical class="h-4 w-4" />
								</button>

								{#if showMenuFor === key.id}
									<div class="absolute right-4 top-full z-10 mt-1 w-40 rounded-lg border border-border bg-popover py-1 shadow-lg">
										<button
											onclick={() => handleRevokeKey(key)}
											class="flex w-full items-center gap-2 px-3 py-2 text-sm text-destructive hover:bg-destructive/10"
										>
											<Trash2 class="h-4 w-4" />
											Revoke Key
										</button>
									</div>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

