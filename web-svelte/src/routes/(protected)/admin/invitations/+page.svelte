<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Mail,
		Plus,
		Copy,
		Trash2,
		Loader2,
		RefreshCw,
		Check,
		X,
		Calendar,
		User
	} from 'lucide-svelte';
	import { adminApi, type Invitation } from '$lib/api/admin';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import { toast } from 'svelte-sonner';
	import * as m from '$lib/paraglide/messages';

	let invitations = $state<Invitation[]>([]);
	let isLoading = $state(true);

	// Create modal
	let showCreateModal = $state(false);
	let newEmail = $state('');
	let newRole = $state<'user' | 'admin'>('user');
	let newExpiresIn = $state('7d');
	let isCreating = $state(false);

	// Copied state
	let copiedId = $state<string | null>(null);

	onMount(async () => {
		await loadInvitations();
	});

	async function loadInvitations() {
		isLoading = true;
		try {
			const response = await adminApi.getInvitations();
			invitations = response.invitations || [];
		} catch (error) {
			console.error('Failed to load invitations:', error);
			toast.error('Failed to load invitations');
		} finally {
			isLoading = false;
		}
	}

	function openCreateModal() {
		newEmail = '';
		newRole = 'user';
		newExpiresIn = '7d';
		showCreateModal = true;
	}

	async function handleCreate() {
		isCreating = true;
		try {
			const invitation = await adminApi.createInvitation({
				email: newEmail || undefined,
				role: newRole,
				expires_in: newExpiresIn
			});
			invitations = [invitation, ...invitations];
			showCreateModal = false;
			toast.success('Invitation created');
		} catch (error) {
			console.error('Failed to create invitation:', error);
			toast.error('Failed to create invitation');
		} finally {
			isCreating = false;
		}
	}

	async function handleRevoke(invitation: Invitation) {
		if (!confirm('Revoke this invitation?')) return;

		try {
			await adminApi.revokeInvitation(invitation.id);
			invitations = invitations.filter((i) => i.id !== invitation.id);
			toast.success('Invitation revoked');
		} catch (error) {
			console.error('Failed to revoke:', error);
			toast.error('Failed to revoke invitation');
		}
	}

	async function copyLink(invitation: Invitation) {
		const link = `${window.location.origin}/register?token=${invitation.token}`;
		await navigator.clipboard.writeText(link);
		copiedId = invitation.id;
		toast.success('Link copied to clipboard');
		setTimeout(() => (copiedId = null), 2000);
	}

	function getStatusBadge(invitation: Invitation) {
		if (invitation.used_at) {
			return { text: 'Used', class: 'bg-green-500/10 text-green-500' };
		}
		if (new Date(invitation.expires_at) < new Date()) {
			return { text: 'Expired', class: 'bg-red-500/10 text-red-500' };
		}
		return { text: 'Active', class: 'bg-blue-500/10 text-blue-500' };
	}
</script>

<svelte:head>
	<title>Invitations | Admin | AI Gateway</title>
</svelte:head>

<div class="p-6">
	<!-- Header -->
	<div class="mb-6 flex items-center justify-between">
		<div>
			<h2 class="text-xl font-semibold">Invitations</h2>
			<p class="text-sm text-muted-foreground">Manage user invitation links</p>
		</div>
		<div class="flex gap-2">
			<Button variant="outline" onclick={loadInvitations} disabled={isLoading}>
				<RefreshCw class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
				{m.common_refresh()}
			</Button>
			<Button onclick={openCreateModal}>
				<Plus class="mr-2 h-4 w-4" />
				Create Invitation
			</Button>
		</div>
	</div>

	<!-- Invitations List -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if invitations.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Mail class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-lg font-medium">No invitations</p>
			<p class="mt-1 text-muted-foreground">Create an invitation to allow new users to register</p>
			<Button class="mt-6" onclick={openCreateModal}>
				<Plus class="mr-2 h-4 w-4" />
				Create Invitation
			</Button>
		</div>
	{:else}
		<div class="overflow-hidden rounded-xl border border-border">
			<table class="w-full">
				<thead class="border-b border-border bg-muted/50">
					<tr>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Email / Token</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Role</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Status</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Expires</th>
						<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">{m.common_actions()}</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border">
					{#each invitations as invitation (invitation.id)}
						{@const status = getStatusBadge(invitation)}
						<tr class="hover:bg-muted/30">
							<td class="px-4 py-3">
								{#if invitation.email}
									<div class="flex items-center gap-2">
										<Mail class="h-4 w-4 text-muted-foreground" />
										<span>{invitation.email}</span>
									</div>
								{:else}
									<span class="font-mono text-sm text-muted-foreground">
										{invitation.token.substring(0, 16)}...
									</span>
								{/if}
							</td>
							<td class="px-4 py-3">
								<span class="capitalize">{invitation.role}</span>
							</td>
							<td class="px-4 py-3">
								<span class={cn('rounded-full px-2 py-0.5 text-xs font-medium', status.class)}>
									{status.text}
								</span>
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{formatRelativeTime(invitation.expires_at)}
							</td>
							<td class="px-4 py-3 text-right">
								<div class="flex justify-end gap-1">
									{#if !invitation.used_at && new Date(invitation.expires_at) > new Date()}
										<button
											onclick={() => copyLink(invitation)}
											class="rounded p-1.5 text-muted-foreground hover:bg-accent"
											title="Copy link"
										>
											{#if copiedId === invitation.id}
												<Check class="h-4 w-4 text-green-500" />
											{:else}
												<Copy class="h-4 w-4" />
											{/if}
										</button>
									{/if}
									<button
										onclick={() => handleRevoke(invitation)}
										class="rounded p-1.5 text-destructive hover:bg-destructive/10"
										title="Revoke"
									>
										<Trash2 class="h-4 w-4" />
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
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
				<h2 class="text-lg font-semibold">Create Invitation</h2>
				<button onclick={() => (showCreateModal = false)} class="rounded p-1 text-muted-foreground hover:bg-accent">
					<X class="h-5 w-5" />
				</button>
			</div>

			<form onsubmit={(e) => { e.preventDefault(); handleCreate(); }} class="space-y-4">
				<div class="space-y-2">
					<label for="email" class="text-sm font-medium">Email (optional)</label>
					<input
						id="email"
						type="email"
						bind:value={newEmail}
						placeholder="user@example.com"
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					/>
					<p class="text-xs text-muted-foreground">Leave empty to create a generic invitation link</p>
				</div>

				<div class="space-y-2">
					<label class="text-sm font-medium">Role</label>
					<div class="flex gap-2">
						<button
							type="button"
							onclick={() => (newRole = 'user')}
							class={cn(
								'flex-1 rounded-lg border px-3 py-2 text-sm transition-colors',
								newRole === 'user' ? 'border-primary bg-primary/10 text-primary' : 'border-input'
							)}
						>
							User
						</button>
						<button
							type="button"
							onclick={() => (newRole = 'admin')}
							class={cn(
								'flex-1 rounded-lg border px-3 py-2 text-sm transition-colors',
								newRole === 'admin' ? 'border-primary bg-primary/10 text-primary' : 'border-input'
							)}
						>
							Admin
						</button>
					</div>
				</div>

				<div class="space-y-2">
					<label class="text-sm font-medium">Expires In</label>
					<select
						bind:value={newExpiresIn}
						class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
					>
						<option value="1d">1 day</option>
						<option value="7d">7 days</option>
						<option value="30d">30 days</option>
						<option value="90d">90 days</option>
					</select>
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

