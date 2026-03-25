<script lang="ts">
	import { onMount } from 'svelte';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import {
		faBuilding,
		faPlus,
		faUsers,
		faGear,
		faTrash,
		faXmark,
		faSpinner,
		faCrown,
		faShield,
		faUser
	} from '@fortawesome/free-solid-svg-icons';
	import {
		tenantsApi,
		type Tenant,
		type TenantMember,
		type UserSearchResult
	} from '$lib/api/tenants';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import { IconButton } from '$lib/components/ui/icon-button';
	import { FormLabel } from '$lib/components/ui/form-label';
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
	let showAddMember = $state(false);
	let memberSearchQuery = $state('');
	let searchResult = $state<UserSearchResult | null>(null);
	let isAlreadyMember = $state(false);
	let isSearching = $state(false);
	let selectedRole = $state<'admin' | 'member'>('member');
	let isAddingMember = $state(false);
	let addMemberError = $state('');

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
		if (!confirm(m.confirm_delete_tenant({ name: tenant.name }))) {
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
			alert(m.alert_failed_delete_org());
		}
	}

	async function handleRemoveMember(member: TenantMember) {
		if (!selectedTenant) return;
		if (!confirm(m.confirm_remove_member({ name: member.username }))) return;

		try {
			await tenantsApi.removeMember(selectedTenant.id, member.id);
			members = members.filter((m) => m.id !== member.id);
		} catch (error) {
			console.error('Failed to remove member:', error);
			alert(m.alert_failed_remove_member());
		}
	}

	async function searchUser() {
		if (!selectedTenant || memberSearchQuery.length < 2) {
			searchResult = null;
			return;
		}

		isSearching = true;
		try {
			const response = await tenantsApi.searchUsers(selectedTenant.id, memberSearchQuery);
			searchResult = response.user;
			isAlreadyMember = response.already_member;
		} catch (error) {
			console.error('Failed to search user:', error);
			searchResult = null;
			isAlreadyMember = false;
		} finally {
			isSearching = false;
		}
	}

	async function handleAddMember() {
		if (!selectedTenant || !searchResult) return;

		isAddingMember = true;
		addMemberError = '';

		try {
			await tenantsApi.addMember(selectedTenant.id, {
				user_id: searchResult.id,
				role: selectedRole
			});

			// Refresh members list
			await loadTenantDetails(selectedTenant);

			// Reset
			showAddMember = false;
			searchResult = null;
			memberSearchQuery = '';
		} catch (error) {
			addMemberError = error instanceof Error ? error.message : 'Failed to add member';
		} finally {
			isAddingMember = false;
		}
	}

	async function handleUpdateMemberRole(member: TenantMember) {
		if (!selectedTenant) return;

		const roles = ['viewer', 'member', 'admin'];
		const currentRole = member.role;
		const newRole = prompt(
			`Change role for ${member.username}\nCurrent: ${currentRole}\nExclude owner role. Enter new role (viewer, member, admin):`,
			currentRole
		);

		if (!newRole || newRole === currentRole) return;
		if (!roles.includes(newRole.toLowerCase())) {
			alert('Invalid role. Must be viewer, member, or admin');
			return;
		}

		try {
			await tenantsApi.updateMemberRole(selectedTenant.id, member.user_id, newRole.toLowerCase());
			await loadTenantDetails(selectedTenant);
		} catch (error) {
			console.error('Failed to update member role:', error);
			alert('Failed to update member role');
		}
	}

	function getRoleIcon(role: string) {
		switch (role) {
			case 'owner':
				return faCrown;
			case 'admin':
				return faShield;
			default:
				return faUser;
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

<div class="mx-auto max-w-[1600px] px-4 py-8 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-foreground text-2xl font-bold">{m.tenants_title()}</h1>
			<p class="text-muted-foreground mt-1">{m.tenants_subtitle()}</p>
		</div>
		<Button onclick={openCreateModal}>
			<FontAwesomeIcon icon={faPlus} class="mr-2 h-4 w-4" />
			{m.tenants_create()}
		</Button>
	</div>

	<!-- Tenants Grid -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<FontAwesomeIcon icon={faSpinner} class="text-muted-foreground h-8 w-8 animate-spin" />
		</div>
	{:else if tenants.length === 0}
		<div class="border-border rounded-lg border border-dashed py-16 text-center">
			<FontAwesomeIcon icon={faBuilding} class="text-muted-foreground/40 mx-auto h-12 w-12" />
			<p class="text-foreground mt-4 text-lg font-medium">{m.tenants_noTenants()}</p>
			<p class="text-muted-foreground mt-1">{m.tenants_createFirst()}</p>
			<Button variant="outline" class="mt-6" onclick={openCreateModal}>
				<FontAwesomeIcon icon={faPlus} class="mr-2 h-4 w-4" />
				{m.tenants_create()}
			</Button>
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each tenants as tenant (tenant.id)}
				<div
					class="group border-border bg-card hover:border-primary/50 cursor-pointer rounded-xl border p-5 transition-all hover:shadow-md"
					onclick={() => loadTenantDetails(tenant)}
					role="button"
					tabindex="0"
					onkeydown={(e) => e.key === 'Enter' && loadTenantDetails(tenant)}
				>
					<div class="mb-4 flex items-start justify-between">
						<div
							class="bg-primary/10 text-primary flex h-12 w-12 items-center justify-center rounded-lg"
						>
							<FontAwesomeIcon icon={faBuilding} class="h-6 w-6" />
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

					<h3 class="text-foreground font-semibold">{tenant.name}</h3>
					<p class="text-muted-foreground mt-1 text-sm">@{tenant.slug}</p>

					{#if tenant.description}
						<p class="text-muted-foreground mt-2 line-clamp-2 text-sm">{tenant.description}</p>
					{/if}

					<div class="text-muted-foreground mt-4 flex items-center gap-4 text-xs">
						<span class="flex items-center gap-1">
							<FontAwesomeIcon icon={faUsers} class="h-3.5 w-3.5" />
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
		onkeydown={(e) => e.key === 'Escape' && (showCreateModal = false)}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div class="border-border bg-card w-full max-w-md rounded-xl border p-6 shadow-xl">
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Create Organization</h2>
				<button
					onclick={() => (showCreateModal = false)}
					class="text-muted-foreground hover:bg-accent rounded p-1"
				>
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			<form
				onsubmit={(e) => {
					e.preventDefault();
					handleCreateTenant();
				}}
				class="space-y-4"
			>
				{#if createError}
					<div class="bg-destructive/10 text-destructive rounded-lg p-3 text-sm">
						{createError}
					</div>
				{/if}

				<div>
					<FormLabel
						label={m.form_tenant_name()}
						description={m.form_tenant_name_desc()}
						required
						for="tenant-name"
					/>
					<input
						id="tenant-name"
						type="text"
						bind:value={tenantName}
						oninput={handleNameChange}
						placeholder={m.placeholder_org_name()}
						required
						class="border-input bg-background focus:ring-ring w-full rounded-lg border px-3 py-2 text-sm focus:ring-2 focus:outline-none"
					/>
				</div>

				<div>
					<FormLabel
						label={m.form_tenant_slug()}
						description={m.form_tenant_slug_desc()}
						required
						for="tenant-slug"
					/>
					<div class="flex items-center gap-1">
						<span class="text-muted-foreground text-sm">@</span>
						<input
							id="tenant-slug"
							type="text"
							bind:value={tenantSlug}
							placeholder="my-organization"
							required
							pattern="[a-z0-9-]+"
							class="border-input bg-background focus:ring-ring flex-1 rounded-lg border px-3 py-2 text-sm focus:ring-2 focus:outline-none"
						/>
					</div>
				</div>

				<div>
					<FormLabel
						label={m.form_tenant_description()}
						description={m.form_tenant_description_desc()}
						for="tenant-desc"
					/>
					<textarea
						id="tenant-desc"
						bind:value={tenantDescription}
						placeholder={m.placeholder_org_desc()}
						rows="3"
						class="border-input bg-background focus:ring-ring w-full rounded-lg border px-3 py-2 text-sm focus:ring-2 focus:outline-none"
					></textarea>
				</div>

				<div class="flex justify-end gap-3 pt-2">
					<Button variant="outline" type="button" onclick={() => (showCreateModal = false)}>
						{m.common_cancel()}
					</Button>
					<Button type="submit" disabled={isCreating}>
						{#if isCreating}
							<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
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
		onkeydown={(e) => e.key === 'Escape' && (showDetailModal = false)}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div class="border-border bg-card w-full max-w-2xl rounded-xl border shadow-xl">
			<!-- Header -->
			<div class="border-border flex items-center justify-between border-b p-6">
				<div class="flex items-center gap-4">
					<div
						class="bg-primary/10 text-primary flex h-12 w-12 items-center justify-center rounded-lg"
					>
						<FontAwesomeIcon icon={faBuilding} class="h-6 w-6" />
					</div>
					<div>
						<h2 class="text-lg font-semibold">{selectedTenant.name}</h2>
						<p class="text-muted-foreground text-sm">@{selectedTenant.slug}</p>
					</div>
				</div>
				<div class="flex items-center gap-2">
					<button
						onclick={() => selectedTenant && handleDeleteTenant(selectedTenant)}
						class="text-destructive hover:bg-destructive/10 rounded p-2"
						title="Delete organization"
					>
						<FontAwesomeIcon icon={faTrash} class="h-5 w-5" />
					</button>
					<button
						onclick={() => (showDetailModal = false)}
						class="text-muted-foreground hover:bg-accent rounded p-2"
					>
						<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
					</button>
				</div>
			</div>

			<!-- Content -->
			<div class="max-h-[60vh] overflow-y-auto p-6">
				{#if selectedTenant.description}
					<p class="text-muted-foreground mb-6">{selectedTenant.description}</p>
				{/if}

				<!-- Members -->
				<div>
					<h3 class="mb-0 flex items-center gap-2 font-semibold">
						<FontAwesomeIcon icon={faUsers} class="h-5 w-5" />
						Members
					</h3>
					<Button
						variant="outline"
						size="sm"
						onclick={() => {
							showAddMember = true;
							memberSearchQuery = '';
							searchResult = null;
							addMemberError = '';
						}}
					>
						<FontAwesomeIcon icon={faPlus} class="mr-2 h-4 w-4" />
						Add Member
					</Button>
				</div>

				<div class="mt-4">
					{#if loadingMembers}
						<div class="flex items-center justify-center py-8">
							<FontAwesomeIcon icon={faSpinner} class="text-muted-foreground h-6 w-6 animate-spin" />
						</div>
					{:else if members.length === 0}
						<p class="text-muted-foreground py-4 text-center">No members yet</p>
					{:else}
						<div class="space-y-2">
							{#each members as member (member.id)}
								{@const RoleIcon = getRoleIcon(member.role)}
								<div class="border-border flex items-center justify-between rounded-lg border p-3">
									<div class="flex items-center gap-3">
										<div class="bg-muted flex h-9 w-9 items-center justify-center rounded-full">
											<FontAwesomeIcon icon={faUser} class="text-muted-foreground h-4 w-4" />
										</div>
										<div>
											<p class="text-foreground font-medium">
												{member.full_name || member.username}
											</p>
											<p class="text-muted-foreground text-xs">{member.email}</p>
										</div>
									</div>
									<div class="flex items-center gap-2">
										<span
											class={cn(
												'flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium',
												getRoleBadgeClass(member.role)
											)}
										>
											<FontAwesomeIcon icon={RoleIcon} class="h-3 w-3" />
											{member.role}
										</span>
										{#if member.role !== 'owner'}
											<button
												onclick={() => handleUpdateMemberRole(member)}
												class="text-muted-foreground hover:bg-accent rounded p-1"
												title="Change role"
											>
												<FontAwesomeIcon icon={faGear} class="h-4 w-4" />
											</button>
											<button
												onclick={() => handleRemoveMember(member)}
												class="text-muted-foreground hover:bg-destructive/10 hover:text-destructive rounded p-1"
												title="Remove member"
											>
												<FontAwesomeIcon icon={faXmark} class="h-4 w-4" />
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

<!-- Add Member Modal -->
{#if showAddMember && selectedTenant}
	<div
		class="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showAddMember = false)}
		onkeydown={(e) => e.key === 'Escape' && (showAddMember = false)}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div class="border-border bg-card w-full max-w-md rounded-xl border p-6 shadow-xl">
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Add Member</h2>
				<button
					onclick={() => (showAddMember = false)}
					class="text-muted-foreground hover:bg-accent rounded p-1"
				>
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			<div class="space-y-4">
				{#if addMemberError}
					<div class="bg-destructive/10 text-destructive rounded-lg p-3 text-sm">
						{addMemberError}
					</div>
				{/if}

				<div>
					<FormLabel
						label="Search User"
						description="Enter username or email"
						required
						for="member-search"
					/>
					<div class="flex gap-2">
						<input
							id="member-search"
							type="text"
							bind:value={memberSearchQuery}
							placeholder="username or email"
							class="border-input bg-background focus:ring-ring flex-1 rounded-lg border px-3 py-2 text-sm focus:ring-2 focus:outline-none"
							onkeydown={(e) => e.key === 'Enter' && searchUser()}
						/>
						<Button
							variant="secondary"
							onclick={searchUser}
							disabled={isSearching || memberSearchQuery.length < 2}
						>
							{#if isSearching}
								<FontAwesomeIcon icon={faSpinner} class="h-4 w-4 animate-spin" />
							{:else}
								Search
							{/if}
						</Button>
					</div>
				</div>

				{#if searchResult}
					<div class="border-border bg-accent/30 rounded-lg border p-4">
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-3">
								<div class="bg-muted flex h-10 w-10 items-center justify-center rounded-full">
									<FontAwesomeIcon icon={faUser} class="text-muted-foreground h-5 w-5" />
								</div>
								<div>
									<p class="text-foreground font-medium">
										{searchResult.full_name || searchResult.username}
									</p>
									<p class="text-muted-foreground text-xs">{searchResult.email}</p>
								</div>
							</div>
							{#if isAlreadyMember}
								<span
									class="bg-muted text-muted-foreground rounded px-2 py-1 text-[10px] font-bold uppercase"
								>
									Already Member
								</span>
							{/if}
						</div>

						{#if !isAlreadyMember}
							<div class="mt-4">
								<FormLabel label="Role" for="member-role" required />
								<select
									id="member-role"
									bind:value={selectedRole}
									class="border-input bg-background focus:ring-ring w-full rounded-lg border px-3 py-2 text-sm focus:ring-2 focus:outline-none"
								>
									<option value="member">Member (Can use resources)</option>
									<option value="admin">Admin (Can manage tenant)</option>
								</select>
							</div>

							<div class="mt-6 flex justify-end gap-3">
								<Button variant="outline" onclick={() => (searchResult = null)}>Clear</Button>
								<Button onclick={handleAddMember} disabled={isAddingMember}>
									{#if isAddingMember}
										<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
									{/if}
									Add to Organization
								</Button>
							</div>
						{/if}
					</div>
				{/if}
			</div>
		</div>
	</div>
{/if}
