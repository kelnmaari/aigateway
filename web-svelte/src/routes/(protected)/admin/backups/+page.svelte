<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Database,
		Plus,
		Download,
		Trash2,
		RefreshCw,
		Loader2,
		Upload,
		CheckCircle,
		Clock
	} from 'lucide-svelte';
	import { api } from '$lib/api/client';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	interface Backup {
		id: string;
		filename: string;
		size: number;
		created_at: string;
		type: 'manual' | 'auto' | 'pre-migration';
		status: 'completed' | 'in_progress' | 'failed';
	}

	let backups = $state<Backup[]>([]);
	let isLoading = $state(true);
	let isCreating = $state(false);

	onMount(async () => {
		await loadBackups();
	});

	async function loadBackups() {
		isLoading = true;
		try {
			const response = await api.get<{ backups: Backup[] }>('/api/admin/backups');
			backups = response.backups || [];
		} catch (error) {
			console.error('Failed to load backups:', error);
		} finally {
			isLoading = false;
		}
	}

	async function createBackup() {
		isCreating = true;
		try {
			const newBackup = await api.post<Backup>('/api/admin/backup', {});
			backups = [newBackup, ...backups];
		} catch (error) {
			console.error('Failed to create backup:', error);
			alert('Failed to create backup');
		} finally {
			isCreating = false;
		}
	}

	async function downloadBackup(backup: Backup) {
		try {
			window.open(`/api/admin/backup/${backup.filename}`, '_blank');
		} catch (error) {
			console.error('Failed to download backup:', error);
		}
	}

	async function deleteBackup(backup: Backup) {
		if (!confirm(`Delete backup "${backup.filename}"?`)) return;

		try {
			await api.delete(`/api/admin/backup/${backup.filename}`);
			backups = backups.filter((b) => b.id !== backup.id);
		} catch (error) {
			console.error('Failed to delete backup:', error);
			alert('Failed to delete backup');
		}
	}

	async function restoreBackup(backup: Backup) {
		if (!confirm(`Restore from backup "${backup.filename}"? Current data will be replaced.`)) return;

		try {
			await api.post(`/api/admin/restore/${backup.filename}`, {});
			alert('Backup restored successfully. Please restart the server.');
		} catch (error) {
			console.error('Failed to restore backup:', error);
			alert('Failed to restore backup');
		}
	}

	function formatSize(bytes: number): string {
		const mb = bytes / (1024 * 1024);
		if (mb >= 1) return `${mb.toFixed(1)} MB`;
		const kb = bytes / 1024;
		return `${kb.toFixed(0)} KB`;
	}

	function getTypeLabel(type: string): string {
		switch (type) {
			case 'auto':
				return 'Automatic';
			case 'pre-migration':
				return 'Pre-Migration';
			default:
				return 'Manual';
		}
	}

	function getTypeClass(type: string): string {
		switch (type) {
			case 'auto':
				return 'bg-blue-500/10 text-blue-500';
			case 'pre-migration':
				return 'bg-amber-500/10 text-amber-500';
			default:
				return 'bg-muted text-muted-foreground';
		}
	}
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h2 class="text-lg font-semibold">{m.admin_backups()}</h2>
		<div class="flex gap-3">
			<Button variant="outline" onclick={loadBackups} disabled={isLoading}>
				<RefreshCw class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
				{m.common_refresh()}
			</Button>
			<Button onclick={createBackup} disabled={isCreating}>
				{#if isCreating}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{:else}
					<Plus class="mr-2 h-4 w-4" />
				{/if}
				Create Backup
			</Button>
		</div>
	</div>

	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if backups.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Database class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-lg font-medium text-foreground">No backups yet</p>
			<p class="mt-1 text-muted-foreground">Create your first backup to secure your data</p>
			<Button class="mt-6" onclick={createBackup} disabled={isCreating}>
				{#if isCreating}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{:else}
					<Plus class="mr-2 h-4 w-4" />
				{/if}
				Create Backup
			</Button>
		</div>
	{:else}
		<div class="overflow-hidden rounded-lg border border-border">
			<table class="w-full">
				<thead class="border-b border-border bg-muted/50">
					<tr>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							Backup
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							Type
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							Size
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
					{#each backups as backup (backup.id)}
						<tr class="hover:bg-muted/30">
							<td class="px-4 py-3">
								<div class="flex items-center gap-2">
									<Database class="h-4 w-4 text-muted-foreground" />
									<span class="font-mono text-sm">{backup.filename}</span>
								</div>
							</td>
							<td class="px-4 py-3">
								<span
									class={cn(
										'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
										getTypeClass(backup.type)
									)}
								>
									{getTypeLabel(backup.type)}
								</span>
							</td>
							<td class="px-4 py-3 text-sm">{formatSize(backup.size)}</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{formatRelativeTime(backup.created_at)}
							</td>
							<td class="px-4 py-3">
								{#if backup.status === 'completed'}
									<span class="inline-flex items-center gap-1 text-sm text-green-500">
										<CheckCircle class="h-4 w-4" />
										Completed
									</span>
								{:else if backup.status === 'in_progress'}
									<span class="inline-flex items-center gap-1 text-sm text-amber-500">
										<Loader2 class="h-4 w-4 animate-spin" />
										In Progress
									</span>
								{:else}
									<span class="text-sm text-red-500">Failed</span>
								{/if}
							</td>
							<td class="px-4 py-3 text-right">
								<div class="flex items-center justify-end gap-1">
									<button
										onclick={() => downloadBackup(backup)}
										class="rounded p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
										title="Download"
									>
										<Download class="h-4 w-4" />
									</button>
									<button
										onclick={() => restoreBackup(backup)}
										class="rounded p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
										title="Restore"
									>
										<Upload class="h-4 w-4" />
									</button>
									<button
										onclick={() => deleteBackup(backup)}
										class="rounded p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
										title="Delete"
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

