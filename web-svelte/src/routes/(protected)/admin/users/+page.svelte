<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Search,
		Plus,
		MoreVertical,
		UserCheck,
		UserX,
		Trash2,
		Key,
		Loader2,
		Shield,
		X,
		Eye,
		EyeOff
	} from 'lucide-svelte';
	import { adminApi, type AdminUser } from '$lib/api/admin';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	// State
	let users = $state<AdminUser[]>([]);
	let total = $state(0);
	let page = $state(1);
	let perPage = $state(20);
	let search = $state('');
	let isLoading = $state(true);

	// Modals
	let showCreateModal = $state(false);
	let showMenuFor = $state<string | null>(null);

	// Create form
	let newUsername = $state('');
	let newEmail = $state('');
	let newPassword = $state('');
	let newFullName = $state('');
	let newIsAdmin = $state(false);
	let showPassword = $state(false);
	let isCreating = $state(false);
	let createError = $state('');

	onMount(async () => {
		await loadUsers();
	});

	async function loadUsers() {
		isLoading = true;
		try {
			const response = await adminApi.getUsers(page, perPage, search || undefined);
			users = response.users || [];
			total = response.total;
		} catch (error) {
			console.error('Failed to load users:', error);
		} finally {
			isLoading = false;
		}
	}

	async function handleSearch() {
		page = 1;
		await loadUsers();
	}

	function openCreateModal() {
		newUsername = '';
		newEmail = '';
		newPassword = '';
		newFullName = '';
		newIsAdmin = false;
		createError = '';
		showCreateModal = true;
	}

	async function handleCreateUser() {
		if (!newUsername.trim() || !newEmail.trim() || !newPassword) {
			createError = 'Username, email, and password are required';
			return;
		}

		isCreating = true;
		createError = '';

		try {
			const newUser = await adminApi.createUser({
				username: newUsername.trim(),
				email: newEmail.trim(),
				password: newPassword,
				full_name: newFullName.trim() || undefined,
				is_admin: newIsAdmin
			});
			users = [newUser, ...users];
			total += 1;
			showCreateModal = false;
		} catch (error) {
			createError = error instanceof Error ? error.message : 'Failed to create user';
		} finally {
			isCreating = false;
		}
	}

	async function toggleUserStatus(user: AdminUser) {
		const newStatus = user.status === 'active' ? 'disabled' : 'active';
		try {
			await adminApi.toggleUserStatus(user.id, newStatus === 'active');
			users = users.map((u) => (u.id === user.id ? { ...u, status: newStatus } : u));
		} catch (error) {
			console.error('Failed to toggle user status:', error);
			alert('Failed to update user status');
		}
		showMenuFor = null;
	}

	async function handleDeleteUser(user: AdminUser) {
		if (!confirm(`Are you sure you want to delete "${user.username}"? This cannot be undone.`)) return;

		try {
			await adminApi.deleteUser(user.id);
			users = users.filter((u) => u.id !== user.id);
			total -= 1;
		} catch (error) {
			console.error('Failed to delete user:', error);
			alert('Failed to delete user');
		}
		showMenuFor = null;
	}

	function getStatusBadgeClass(status: string) {
		switch (status) {
			case 'active':
				return 'bg-green-500/10 text-green-500';
			case 'disabled':
				return 'bg-red-500/10 text-red-500';
			default:
				return 'bg-muted text-muted-foreground';
		}
	}
</script>

<div class="space-y-6">
	<!-- Header -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<h2 class="text-lg font-semibold">{m.admin_users()}</h2>
		<div class="flex gap-3">
			<!-- Search -->
			<form onsubmit={(e) => { e.preventDefault(); handleSearch(); }} class="relative">
				<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
				<input
					type="text"
					bind:value={search}
					placeholder="Search users..."
					class="w-64 rounded-lg border border-input bg-background py-2 pl-9 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
				/>
			</form>
			<Button onclick={openCreateModal}>
				<Plus class="mr-2 h-4 w-4" />
				Add User
			</Button>
		</div>
	</div>

	<!-- Table -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if users.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<p class="text-muted-foreground">No users found</p>
		</div>
	{:else}
		<div class="overflow-hidden rounded-lg border border-border">
			<table class="w-full">
				<thead class="border-b border-border bg-muted/50">
					<tr>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							User
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							Role
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							Auth
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.common_status()}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							Last Login
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.common_created()}
						</th>
						<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">
							{m.common_actions()}
						</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border">
					{#each users as user (user.id)}
						<tr class="hover:bg-muted/30">
							<td class="px-4 py-3">
								<div>
									<p class="font-medium text-foreground">
										{user.full_name || user.username}
									</p>
									<p class="text-xs text-muted-foreground">{user.email}</p>
								</div>
							</td>
							<td class="px-4 py-3">
								{#if user.is_admin}
									<span class="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
										<Shield class="h-3 w-3" />
										Admin
									</span>
								{:else}
									<span class="text-sm text-muted-foreground">User</span>
								{/if}
							</td>
							<td class="px-4 py-3">
								{#if user.oidc_subject}
									<!-- User has OIDC linked (may also have local password) -->
									<span class="inline-flex items-center gap-1 rounded-full bg-blue-500/10 px-2 py-0.5 text-xs font-medium text-blue-600 dark:text-blue-400" title="OIDC linked{user.auth_provider === 'local' ? ' + Local password' : ''}">
										<Key class="h-3 w-3" />
										OIDC
									</span>
								{:else if user.auth_provider === 'ldap'}
									<span class="inline-flex items-center gap-1 rounded-full bg-purple-500/10 px-2 py-0.5 text-xs font-medium text-purple-600 dark:text-purple-400">
										<Key class="h-3 w-3" />
										LDAP
									</span>
								{:else}
									<span class="inline-flex items-center gap-1 rounded-full bg-gray-500/10 px-2 py-0.5 text-xs font-medium text-gray-600 dark:text-gray-400">
										Local
									</span>
								{/if}
							</td>
							<td class="px-4 py-3">
								<span
									class={cn(
										'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
										getStatusBadgeClass(user.status)
									)}
								>
									{user.status}
								</span>
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{user.last_login_at ? formatRelativeTime(user.last_login_at) : 'Never'}
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{formatRelativeTime(user.created_at)}
							</td>
							<td class="relative px-4 py-3 text-right">
								<button
									onclick={() => (showMenuFor = showMenuFor === user.id ? null : user.id)}
									class="rounded p-1.5 text-muted-foreground hover:bg-accent"
								>
									<MoreVertical class="h-4 w-4" />
								</button>

								{#if showMenuFor === user.id}
									<div class="absolute right-4 top-full z-10 mt-1 w-48 rounded-lg border border-border bg-popover py-1 shadow-lg">
										<button
											onclick={() => toggleUserStatus(user)}
											class="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
										>
											{#if user.status === 'active'}
												<UserX class="h-4 w-4" />
												Disable User
											{:else}
												<UserCheck class="h-4 w-4" />
												Enable User
											{/if}
										</button>
										<button
											onclick={() => handleDeleteUser(user)}
											class="flex w-full items-center gap-2 px-3 py-2 text-sm text-destructive hover:bg-destructive/10"
										>
											<Trash2 class="h-4 w-4" />
											Delete User
										</button>
									</div>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<!-- Pagination -->
		{#if total > perPage}
			<div class="flex items-center justify-between">
				<p class="text-sm text-muted-foreground">
					Showing {(page - 1) * perPage + 1} to {Math.min(page * perPage, total)} of {total} users
				</p>
				<div class="flex gap-2">
					<Button
						variant="outline"
						size="sm"
						disabled={page === 1}
						onclick={() => { page -= 1; loadUsers(); }}
					>
						{m.common_previous()}
					</Button>
					<Button
						variant="outline"
						size="sm"
						disabled={page * perPage >= total}
						onclick={() => { page += 1; loadUsers(); }}
					>
						{m.common_next()}
					</Button>
				</div>
			</div>
		{/if}
	{/if}
</div>

<!-- Create User Modal -->
{#if showCreateModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showCreateModal = false)}
		role="dialog"
		tabindex="-1"
	>
		<div class="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl">
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Create User</h2>
				<button
					onclick={() => (showCreateModal = false)}
					class="rounded p-1 text-muted-foreground hover:bg-accent"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			<form onsubmit={(e) => { e.preventDefault(); handleCreateUser(); }} class="space-y-4">
				{#if createError}
					<div class="rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
						{createError}
					</div>
				{/if}

				<div class="space-y-2">
					<label for="username" class="text-sm font-medium">Username *</label>
					<input
						id="username"
						type="text"
						bind:value={newUsername}
						placeholder="johndoe"
						required
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>

				<div class="space-y-2">
					<label for="email" class="text-sm font-medium">Email *</label>
					<input
						id="email"
						type="email"
						bind:value={newEmail}
						placeholder="john@example.com"
						required
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>

				<div class="space-y-2">
					<label for="password" class="text-sm font-medium">Password *</label>
					<div class="relative">
						<input
							id="password"
							type={showPassword ? 'text' : 'password'}
							bind:value={newPassword}
							required
							minlength="8"
							class="w-full rounded-lg border border-input bg-background py-2 pl-3 pr-10 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						/>
						<button
							type="button"
							onclick={() => (showPassword = !showPassword)}
							class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
						>
							{#if showPassword}
								<EyeOff class="h-4 w-4" />
							{:else}
								<Eye class="h-4 w-4" />
							{/if}
						</button>
					</div>
				</div>

				<div class="space-y-2">
					<label for="fullname" class="text-sm font-medium">Full Name</label>
					<input
						id="fullname"
						type="text"
						bind:value={newFullName}
						placeholder="John Doe"
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>

				<label class="flex items-center gap-2">
					<input type="checkbox" bind:checked={newIsAdmin} class="h-4 w-4 rounded border-input" />
					<span class="text-sm font-medium">Administrator</span>
				</label>

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

