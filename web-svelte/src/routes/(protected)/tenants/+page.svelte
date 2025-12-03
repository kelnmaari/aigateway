<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Building2,
		Plus,
		Users,
		Settings,
		Trash2,
		X,
		Loader2,
		Crown,
		Shield,
		User
	} from 'lucide-svelte';
	import { tenantsApi, type Tenant, type TenantMember } from '$lib/api/tenants';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	// State
	let tenants = $state<Tenant[]>([]);
	let isLoading = $state(true);
	let selectedTenant = $state<Tenant | null>(null);
	let members = $state<TenantMember[]>([]);
	let loadingMembers = $state(false);

	// Modals
	let showCreateModal = $state(false);
	let showDetailModal = $state(false);

	// Create form
	let tenantName = $state('');
	let tenantSlug = $state('');
	let tenantDescription = $state('');
	let isCreating = $state(false);
	let createError = $state('');

	onMount(async () => {
		await loadTenants();
	});

	async function loadTenants() {
		isLoading = true;
		try {
			const response = await tenantsApi.getUserTenants();
			tenants = response.tenants || [];
		} catch (error) {
			console.error('Failed to load tenants:', error);
		} finally {
			isLoading = false;
		}
	}

	async function loadTenantDetails(tenant: Tenant) {
		selectedTenant = tenant;
		showDetailModal = true;
		loadingMembers = true;

		try {
			const response = await tenantsApi.getMembers(tenant.id);
			members = response.members || [];
		} catch (error) {
			console.error('Failed to load members:', error);
			members = [];
		} finally {
			loadingMembers = false;
		}
	}

	function openCreateModal() {
		tenantName = '';
		tenantSlug = '';
		tenantDescription = '';
		createError = '';
		showCreateModal = true;
	}

	function generateSlug(name: string): string {
		return name
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-|-$/g, '');
	}

	function handleNameChange() {
		if (!tenantSlug || tenantSlug === generateSlug(tenantName.slice(0, -1))) {
			tenantSlug = generateSlug(tenantName);
		}
	}

	async function handleCreateTenant() {
		if (!tenantName.trim() || !tenantSlug.trim()) {
			createError = 'Name and slug are required';
			return;
		}

		isCreating = true;
		createError = '';

		try {
			const newTenant = await tenantsApi.createTenant({
				name: tenantName.trim(),
				slug: tenantSlug.trim(),
				description: tenantDescription.trim() || undefined,
				type: 'organization'
			});

			tenants = [newTenant, ...tenants];
			showCreateModal = false;
		} catch (error) {
			createError = error instanceof Error ? error.message : 'Failed to create organization';
		} finally {
			isCreating = false;
		}
	}

	async function handleDeleteTenant(tenant: Tenant) {
		if (!confirm(`Are you sure you want to delete "${tenant.name}"? This cannot be undone.`)) {
			return;
		}

		try {
			await tenantsApi.deleteTenant(tenant.id);
			tenants = tenants.filter((t) => t.id !== tenant.id);
			if (selectedTenant?.id === tenant.id) {
				showDetailModal = false;
				selectedTenant = null;
			}
		} catch (error) {
			console.error('Failed to delete tenant:', error);
			alert('Failed to delete organization');
		}
	}

	async function handleRemoveMember(member: TenantMember) {
		if (!selectedTenant) return;
		if (!confirm(`Remove ${member.username} from this organization?`)) return;

		try {
			await tenantsApi.removeMember(selectedTenant.id, member.id);
			members = members.filter((m) => m.id !== member.id);
		} catch (error) {
			console.error('Failed to remove member:', error);
			alert('Failed to remove member');
		}
	}

	function getRoleIcon(role: string) {
		switch (role) {
			case 'owner':
				return Crown;
			case 'admin':
				return Shield;
			default:
				return User;
		}
	}

	function getRoleBadgeClass(role: string) {
		switch (role) {
			case 'owner':
				return 'bg-amber-500/10 text-amber-500';
			case 'admin':
				return 'bg-blue-500/10 text-blue-500';
			default:
				return 'bg-muted text-muted-foreground';
		}
	}
</script>

<svelte:head>
	<title>{m.nav_tenants()} | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-foreground">{m.nav_tenants()}</h1>
			<p class="mt-1 text-muted-foreground">Manage your organizations and teams</p>
		</div>
		<Button onclick={openCreateModal}>
			<Plus class="mr-2 h-4 w-4" />
			{m.common_create()} Organization
		</Button>
	</div>

	<!-- Tenants Grid -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if tenants.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Building2 class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-lg font-medium text-foreground">No organizations yet</p>
			<p class="mt-1 text-muted-foreground">Create your first organization to collaborate with your team</p>
			<Button variant="outline" class="mt-6" onclick={openCreateModal}>
				<Plus class="mr-2 h-4 w-4" />
				Create Organization
			</Button>
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each tenants as tenant (tenant.id)}
				<div
					class="group cursor-pointer rounded-xl border border-border bg-card p-5 transition-all hover:border-primary/50 hover:shadow-md"
					onclick={() => loadTenantDetails(tenant)}
					role="button"
					tabindex="0"
					onkeydown={(e) => e.key === 'Enter' && loadTenantDetails(tenant)}
				>
					<div class="mb-4 flex items-start justify-between">
						<div class="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10 text-primary">
							<Building2 class="h-6 w-6" />
						</div>
						<span
							class={cn(
								'rounded-full px-2 py-0.5 text-xs font-medium',
								tenant.status === 'active'
									? 'bg-green-500/10 text-green-500'
									: 'bg-muted text-muted-foreground'
							)}
						>
							{tenant.status}
						</span>
					</div>

					<h3 class="font-semibold text-foreground">{tenant.name}</h3>
					<p class="mt-1 text-sm text-muted-foreground">@{tenant.slug}</p>

					{#if tenant.description}
						<p class="mt-2 line-clamp-2 text-sm text-muted-foreground">{tenant.description}</p>
					{/if}

					<div class="mt-4 flex items-center gap-4 text-xs text-muted-foreground">
						<span class="flex items-center gap-1">
							<Users class="h-3.5 w-3.5" />
							{tenant.member_count || 1} members
						</span>
						<span>{formatRelativeTime(tenant.created_at)}</span>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Create Modal -->
{#if showCreateModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showCreateModal = false)}
		role="dialog"
		tabindex="-1"
	>
		<div class="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl">
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Create Organization</h2>
				<button
					onclick={() => (showCreateModal = false)}
					class="rounded p-1 text-muted-foreground hover:bg-accent"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			<form onsubmit={(e) => { e.preventDefault(); handleCreateTenant(); }} class="space-y-4">
				{#if createError}
					<div class="rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
						{createError}
					</div>
				{/if}

				<div class="space-y-2">
					<label for="tenant-name" class="text-sm font-medium">Name *</label>
					<input
						id="tenant-name"
						type="text"
						bind:value={tenantName}
						oninput={handleNameChange}
						placeholder="My Organization"
						required
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>

				<div class="space-y-2">
					<label for="tenant-slug" class="text-sm font-medium">Slug *</label>
					<div class="flex items-center gap-1">
						<span class="text-sm text-muted-foreground">@</span>
						<input
							id="tenant-slug"
							type="text"
							bind:value={tenantSlug}
							placeholder="my-organization"
							required
							pattern="[a-z0-9-]+"
							class="flex-1 rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						/>
					</div>
					<p class="text-xs text-muted-foreground">Lowercase letters, numbers, and hyphens only</p>
				</div>

				<div class="space-y-2">
					<label for="tenant-desc" class="text-sm font-medium">{m.common_description()}</label>
					<textarea
						id="tenant-desc"
						bind:value={tenantDescription}
						placeholder="Optional description"
						rows="3"
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					></textarea>
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

<!-- Detail Modal -->
{#if showDetailModal && selectedTenant}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showDetailModal = false)}
		role="dialog"
		tabindex="-1"
	>
		<div class="w-full max-w-2xl rounded-xl border border-border bg-card shadow-xl">
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-border p-6">
				<div class="flex items-center gap-4">
					<div class="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10 text-primary">
						<Building2 class="h-6 w-6" />
					</div>
					<div>
						<h2 class="text-lg font-semibold">{selectedTenant.name}</h2>
						<p class="text-sm text-muted-foreground">@{selectedTenant.slug}</p>
					</div>
				</div>
				<div class="flex items-center gap-2">
					<button
						onclick={() => selectedTenant && handleDeleteTenant(selectedTenant)}
						class="rounded p-2 text-destructive hover:bg-destructive/10"
						title="Delete organization"
					>
						<Trash2 class="h-5 w-5" />
					</button>
					<button
						onclick={() => (showDetailModal = false)}
						class="rounded p-2 text-muted-foreground hover:bg-accent"
					>
						<X class="h-5 w-5" />
					</button>
				</div>
			</div>

			<!-- Content -->
			<div class="max-h-[60vh] overflow-y-auto p-6">
				{#if selectedTenant.description}
					<p class="mb-6 text-muted-foreground">{selectedTenant.description}</p>
				{/if}

				<!-- Members -->
				<div>
					<h3 class="mb-4 flex items-center gap-2 font-semibold">
						<Users class="h-5 w-5" />
						Members
					</h3>

					{#if loadingMembers}
						<div class="flex items-center justify-center py-8">
							<Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
						</div>
					{:else if members.length === 0}
						<p class="py-4 text-center text-muted-foreground">No members yet</p>
					{:else}
						<div class="space-y-2">
							{#each members as member (member.id)}
								{@const RoleIcon = getRoleIcon(member.role)}
								<div class="flex items-center justify-between rounded-lg border border-border p-3">
									<div class="flex items-center gap-3">
										<div class="flex h-9 w-9 items-center justify-center rounded-full bg-muted">
											<User class="h-4 w-4 text-muted-foreground" />
										</div>
										<div>
											<p class="font-medium text-foreground">
												{member.full_name || member.username}
											</p>
											<p class="text-xs text-muted-foreground">{member.email}</p>
										</div>
									</div>
									<div class="flex items-center gap-2">
										<span class={cn('flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium', getRoleBadgeClass(member.role))}>
											<RoleIcon class="h-3 w-3" />
											{member.role}
										</span>
										{#if member.role !== 'owner'}
											<button
												onclick={() => handleRemoveMember(member)}
												class="rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
												title="Remove member"
											>
												<X class="h-4 w-4" />
											</button>
										{/if}
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}

