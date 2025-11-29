<script lang="ts">
	import { onMount } from 'svelte';
	import {
		File,
		Folder,
		Upload,
		FolderPlus,
		Download,
		Trash2,
		RefreshCw,
		Loader2,
		ChevronRight,
		Home,
		MoreVertical
	} from 'lucide-svelte';
	import { filesApi, type FileItem } from '$lib/api/files';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	let files = $state<FileItem[]>([]);
	let currentPath = $state('/');
	let isLoading = $state(true);
	let showMenuFor = $state<string | null>(null);

	// Modals
	let showUploadModal = $state(false);
	let showNewFolderModal = $state(false);
	let newFolderName = $state('');
	let isCreatingFolder = $state(false);
	let uploadingFiles = $state<File[]>([]);
	let isUploading = $state(false);

	onMount(async () => {
		await loadFiles();
	});

	async function loadFiles() {
		isLoading = true;
		try {
			const response = await filesApi.list(currentPath);
			files = response.files || [];
		} catch (error) {
			console.error('Failed to load files:', error);
		} finally {
			isLoading = false;
		}
	}

	function navigateTo(path: string) {
		currentPath = path;
		loadFiles();
	}

	function openItem(item: FileItem) {
		if (item.is_directory) {
			navigateTo(item.path);
		} else {
			filesApi.download(item.id);
		}
	}

	async function handleCreateFolder() {
		if (!newFolderName.trim()) return;

		isCreatingFolder = true;
		try {
			await filesApi.createDirectory(newFolderName.trim(), currentPath);
			await loadFiles();
			showNewFolderModal = false;
			newFolderName = '';
		} catch (error) {
			console.error('Failed to create folder:', error);
			alert('Failed to create folder');
		} finally {
			isCreatingFolder = false;
		}
	}

	async function handleUpload() {
		if (uploadingFiles.length === 0) return;

		isUploading = true;
		try {
			for (const file of uploadingFiles) {
				await filesApi.upload(file, currentPath);
			}
			await loadFiles();
			showUploadModal = false;
			uploadingFiles = [];
		} catch (error) {
			console.error('Failed to upload:', error);
			alert('Failed to upload files');
		} finally {
			isUploading = false;
		}
	}

	async function handleDelete(item: FileItem) {
		if (!confirm(`Delete "${item.name}"?`)) return;

		try {
			await filesApi.delete(item.id);
			files = files.filter((f) => f.id !== item.id);
		} catch (error) {
			console.error('Failed to delete:', error);
			alert('Failed to delete');
		}
		showMenuFor = null;
	}

	function formatSize(bytes: number): string {
		if (bytes === 0) return '—';
		const units = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(1024));
		return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
	}

	function getBreadcrumbs(): Array<{ name: string; path: string }> {
		const parts = currentPath.split('/').filter(Boolean);
		const breadcrumbs = [{ name: 'Home', path: '/' }];
		let path = '';
		for (const part of parts) {
			path += '/' + part;
			breadcrumbs.push({ name: part, path });
		}
		return breadcrumbs;
	}

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files) {
			uploadingFiles = Array.from(input.files);
		}
	}
</script>

<svelte:head>
	<title>{m.nav_files()} | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="mb-6 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-foreground">{m.nav_files()}</h1>
			<!-- Breadcrumbs -->
			<nav class="mt-2 flex items-center gap-1 text-sm">
				{#each getBreadcrumbs() as crumb, i}
					{#if i > 0}
						<ChevronRight class="h-4 w-4 text-muted-foreground" />
					{/if}
					<button
						onclick={() => navigateTo(crumb.path)}
						class={cn(
							'flex items-center gap-1 hover:text-foreground',
							crumb.path === currentPath ? 'text-foreground' : 'text-muted-foreground'
						)}
					>
						{#if i === 0}
							<Home class="h-4 w-4" />
						{/if}
						{crumb.name}
					</button>
				{/each}
			</nav>
		</div>
		<div class="flex gap-2">
			<Button variant="outline" onclick={() => (showNewFolderModal = true)}>
				<FolderPlus class="mr-2 h-4 w-4" />
				New Folder
			</Button>
			<Button onclick={() => (showUploadModal = true)}>
				<Upload class="mr-2 h-4 w-4" />
				Upload
			</Button>
		</div>
	</div>

	<!-- Files Table -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if files.length === 0}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<Folder class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-muted-foreground">This folder is empty</p>
			<Button variant="outline" class="mt-4" onclick={() => (showUploadModal = true)}>
				<Upload class="mr-2 h-4 w-4" />
				Upload files
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
							Size
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							Modified
						</th>
						<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">
							{m.common_actions()}
						</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border">
					{#each files as item (item.id)}
						<tr class="cursor-pointer hover:bg-muted/30" ondblclick={() => openItem(item)}>
							<td class="px-4 py-3">
								<div class="flex items-center gap-3">
									{#if item.is_directory}
										<Folder class="h-5 w-5 text-amber-500" />
									{:else}
										<File class="h-5 w-5 text-muted-foreground" />
									{/if}
									<span class="font-medium">{item.name}</span>
								</div>
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{item.is_directory ? '—' : formatSize(item.size)}
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{formatRelativeTime(item.modified_at)}
							</td>
							<td class="relative px-4 py-3 text-right">
								<button
									onclick={(e) => { e.stopPropagation(); showMenuFor = showMenuFor === item.id ? null : item.id; }}
									class="rounded p-1.5 text-muted-foreground hover:bg-accent"
								>
									<MoreVertical class="h-4 w-4" />
								</button>

								{#if showMenuFor === item.id}
									<div class="absolute right-4 top-full z-10 mt-1 w-40 rounded-lg border border-border bg-popover py-1 shadow-lg">
										{#if !item.is_directory}
											<button
												onclick={() => filesApi.download(item.id)}
												class="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
											>
												<Download class="h-4 w-4" />
												Download
											</button>
										{/if}
										<button
											onclick={() => handleDelete(item)}
											class="flex w-full items-center gap-2 px-3 py-2 text-sm text-destructive hover:bg-destructive/10"
										>
											<Trash2 class="h-4 w-4" />
											{m.common_delete()}
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

<!-- Upload Modal -->
{#if showUploadModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showUploadModal = false)}
		role="dialog"
		tabindex="-1"
	>
		<div class="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl">
			<h2 class="mb-4 text-lg font-semibold">Upload Files</h2>

			<div class="mb-4">
				<label
					class="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-border py-8 transition-colors hover:border-primary"
				>
					<Upload class="mb-2 h-8 w-8 text-muted-foreground" />
					<span class="text-sm text-muted-foreground">Click to select files</span>
					<input type="file" multiple class="hidden" onchange={handleFileSelect} />
				</label>
			</div>

			{#if uploadingFiles.length > 0}
				<div class="mb-4 space-y-2">
					{#each uploadingFiles as file}
						<div class="flex items-center gap-2 rounded bg-muted px-3 py-2 text-sm">
							<File class="h-4 w-4" />
							{file.name}
							<span class="ml-auto text-muted-foreground">{formatSize(file.size)}</span>
						</div>
					{/each}
				</div>
			{/if}

			<div class="flex justify-end gap-3">
				<Button variant="outline" onclick={() => { showUploadModal = false; uploadingFiles = []; }}>
					{m.common_cancel()}
				</Button>
				<Button onclick={handleUpload} disabled={uploadingFiles.length === 0 || isUploading}>
					{#if isUploading}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					Upload
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- New Folder Modal -->
{#if showNewFolderModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showNewFolderModal = false)}
		role="dialog"
		tabindex="-1"
	>
		<div class="w-full max-w-sm rounded-xl border border-border bg-card p-6 shadow-xl">
			<h2 class="mb-4 text-lg font-semibold">New Folder</h2>

			<form onsubmit={(e) => { e.preventDefault(); handleCreateFolder(); }}>
				<input
					type="text"
					bind:value={newFolderName}
					placeholder="Folder name"
					required
					class="mb-4 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
				/>

				<div class="flex justify-end gap-3">
					<Button variant="outline" type="button" onclick={() => (showNewFolderModal = false)}>
						{m.common_cancel()}
					</Button>
					<Button type="submit" disabled={isCreatingFolder}>
						{#if isCreatingFolder}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						{m.common_create()}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}

