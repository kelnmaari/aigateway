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
	import { cn, formatRelativeTime, copyToClipboard } from '$lib/utils';
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
			toast.error(m.toast_failed_load({ item: m.admin_invitations_title() }));
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

	function getExpiresAt(expiresIn: string): string {
		const days = parseInt(expiresIn.replace('d', ''));
		const date = new Date();
		date.setDate(date.getDate() + days);
		return date.toISOString();
	}

	async function handleCreate() {
		isCreating = true;
		try {
			const response = await adminApi.createInvitation({
				email: newEmail || undefined,
				expires_at: getExpiresAt(newExpiresIn),
				max_uses: 1
			});
			invitations = [response.invitation, ...invitations];
			showCreateModal = false;
			toast.success(m.toast_invitation_created());
			// Copy link to clipboard
			if (response.invitation_link) {
				const copied = await copyToClipboard(response.invitation_link);
				if (copied) {
					toast.success(m.toast_link_copied());
				}
			}
		} catch (error) {
			console.error('Failed to create invitation:', error);
			toast.error(m.toast_failed_create({ item: m.admin_invitations_title() }));
		} finally {
			isCreating = false;
		}
	}

	async function handleRevoke(invitation: Invitation) {
		if (!confirm(m.confirm_revoke_invitation())) return;

		try {
			await adminApi.revokeInvitation(invitation.id);
			invitations = invitations.filter((i) => i.id !== invitation.id);
			toast.success(m.toast_invitation_revoked());
		} catch (error) {
			console.error('Failed to revoke:', error);
			toast.error(m.toast_failed_revoke({ item: m.admin_invitations_title() }));
		}
	}

	async function copyLink(invitation: Invitation) {
		if (!invitation.token) {
			toast.error(m.toast_no_token());
			return;
		}
		const link = `${window.location.origin}/register?token=${invitation.token}`;
		const copied = await copyToClipboard(link);
		if (copied) {
			copiedId = invitation.id;
			toast.success(m.toast_link_copied());
			setTimeout(() => (copiedId = null), 2000);
		} else {
			toast.error(m.toast_failed_copy());
		}
	}

	function getStatusBadge(invitation: Invitation) {
		if (invitation.revoked_at) {
			return { text: m.admin_invitations_revoked(), class: 'bg-gray-500/10 text-gray-500' };
		}
		if (invitation.used_at || invitation.current_uses >= invitation.max_uses) {
			return { text: m.admin_invitations_used(), class: 'bg-green-500/10 text-green-500' };
		}
		if (invitation.expires_at && new Date(invitation.expires_at) < new Date()) {
			return { text: m.admin_invitations_expired(), class: 'bg-red-500/10 text-red-500' };
		}
		return { text: m.common_active(), class: 'bg-blue-500/10 text-blue-500' };
	}
	
	function isInvitationActive(invitation: Invitation): boolean {
		if (invitation.revoked_at) return false;
		if (invitation.used_at || invitation.current_uses >= invitation.max_uses) return false;
		if (invitation.expires_at && new Date(invitation.expires_at) < new Date()) return false;
		return true;
	}
</script>

<svelte:head>
	<title>Invitations | Admin | AI Gateway</title>
</svelte:head>

<div class="p-6">
	<!-- Header -->
	<div class="mb-6 flex items-center justify-between">
		<div>
			<h2 class="text-xl font-semibold">{m.admin_invitations_title()}</h2>
			<p class="text-sm text-muted-foreground">{m.admin_invitations_subtitle()}</p>
		</div>
		<div class="flex gap-2">
			<Button variant="outline" onclick={loadInvitations} disabled={isLoading}>
				<RefreshCw class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
				{m.common_refresh()}
			</Button>
			<Button onclick={openCreateModal}>
				<Plus class="mr-2 h-4 w-4" />
				{m.admin_invitations_create()}
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
			<p class="mt-4 text-lg font-medium">{m.admin_invitations_empty()}</p>
			<p class="mt-1 text-muted-foreground">{m.admin_invitations_empty_desc()}</p>
			<Button class="mt-6" onclick={openCreateModal}>
				<Plus class="mr-2 h-4 w-4" />
				{m.admin_invitations_create()}
			</Button>
		</div>
	{:else}
		<div class="overflow-hidden rounded-xl border border-border">
			<table class="w-full">
				<thead class="border-b border-border bg-muted/50">
					<tr>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">{m.admin_invitations_email()}</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">{m.common_status()}</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">{m.admin_invitations_uses()}</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">{m.admin_invitations_expires()}</th>
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
								{:else if invitation.token}
									<span class="font-mono text-sm text-muted-foreground">
										{invitation.token.substring(0, 16)}...
									</span>
								{:else}
									<span class="text-sm text-muted-foreground">—</span>
								{/if}
							</td>
							<td class="px-4 py-3">
								<span class={cn('rounded-full px-2 py-0.5 text-xs font-medium', status.class)}>
									{status.text}
								</span>
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{invitation.current_uses || 0} / {invitation.max_uses || 1}
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{invitation.expires_at ? formatRelativeTime(invitation.expires_at) : m.admin_invitations_never()}
							</td>
							<td class="px-4 py-3 text-right">
								<div class="flex justify-end gap-1">
									{#if isInvitationActive(invitation)}
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
									{#if !invitation.revoked_at}
										<button
											onclick={() => handleRevoke(invitation)}
											class="rounded p-1.5 text-destructive hover:bg-destructive/10"
											title="Revoke"
										>
											<Trash2 class="h-4 w-4" />
										</button>
									{/if}
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
		onkeydown={(e) => e.key === 'Escape' && (showCreateModal = false)}
		role="dialog"
		aria-modal="true"
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
					<span class="text-sm font-medium">Role</span>
					<div class="flex gap-2" role="radiogroup" aria-label="Role selection">
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
					<label for="invite-expires" class="text-sm font-medium">Expires In</label>
					<select
						id="invite-expires"
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

