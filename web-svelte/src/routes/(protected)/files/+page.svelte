<script lang="ts">
	import { onMount } from 'svelte';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import {
		faFile,
		faFileLines,
		faUpload,
		faDownload,
		faTrash,
		faArrowsRotate,
		faSpinner,
		faEllipsisVertical,
		faCircleCheck,
		faClock,
		faCircleExclamation,
		faChevronLeft,
		faChevronRight,
		faEye,
		faMagnifyingGlass,
		faXmark
	} from '@fortawesome/free-solid-svg-icons';
	import { filesApi, type FileItem } from '$lib/api/files';
	import { cn, formatRelativeTime, copyToClipboard } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import { IconButton } from '$lib/components/ui/icon-button';
	import * as m from '$lib/paraglide/messages';

	let files = $state<FileItem[]>([]);
	let totalFiles = $state(0);
	let isLoading = $state(true);
	let showMenuFor = $state<string | null>(null);

	// Pagination
	let currentPage = $state(0);
	let pageSize = 20;

	// Filters
	let typeFilter = $state('');
	let statusFilter = $state('');
	let searchQuery = $state('');
	let searchTimeout: ReturnType<typeof setTimeout>;

	// Modals
	let showUploadModal = $state(false);
	let uploadingFiles = $state<File[]>([]);
	let isUploading = $state(false);

	// View Text Modal
	let showTextModal = $state(false);
	let viewingFile = $state<FileItem | null>(null);
	let extractedText = $state('');
	let isLoadingText = $state(false);

	// Drag & Drop
	let isDragging = $state(false);

	onMount(() => {
		loadFiles();

		// Close dropdown when clicking outside
		const handleClickOutside = (e: MouseEvent) => {
			if (showMenuFor && !(e.target as Element).closest('.dropdown-menu-container')) {
				showMenuFor = null;
			}
		};
		document.addEventListener('click', handleClickOutside);

		return () => {
			document.removeEventListener('click', handleClickOutside);
		};
	});

	async function loadFiles() {
		isLoading = true;
		try {
			const response = await filesApi.list({
				limit: pageSize,
				offset: currentPage * pageSize,
				mimeType: typeFilter || undefined,
				extractionStatus: statusFilter || undefined
			});
			files = response.files || [];
			totalFiles = response.total || 0;
		} catch (error) {
			console.error('Failed to load files:', error);
			files = [];
			totalFiles = 0;
		} finally {
			isLoading = false;
		}
	}

	function handleSearchInput(e: Event) {
		const value = (e.target as HTMLInputElement).value;
		searchQuery = value;

		// Debounce search
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => {
			currentPage = 0;
			if (searchQuery.trim()) {
				searchFiles();
			} else {
				loadFiles();
			}
		}, 300);
	}

	async function searchFiles() {
		if (!searchQuery.trim()) {
			await loadFiles();
			return;
		}

		isLoading = true;
		try {
			const response = await filesApi.search(searchQuery);
			files = response.files || [];
			totalFiles = response.total || files.length;
		} catch (error) {
			console.error('Failed to search files:', error);
			// Fallback to regular load
			await loadFiles();
		} finally {
			isLoading = false;
		}
	}

	async function handleUpload() {
		if (uploadingFiles.length === 0) return;

		isUploading = true;
		try {
			for (const file of uploadingFiles) {
				await filesApi.upload(file);
			}
			await loadFiles();
			showUploadModal = false;
			uploadingFiles = [];
		} catch (error) {
			console.error('Failed to upload:', error);
			alert(m.alert_failed_upload());
		} finally {
			isUploading = false;
		}
	}

	async function handleDelete(item: FileItem) {
		const name = item.original_name || item.filename || String(item.id);
		if (!confirm(m.confirm_delete_file({ name }))) return;

		try {
			await filesApi.delete(String(item.id));
			files = files.filter((f) => String(f.id) !== String(item.id));
			totalFiles--;
		} catch (error) {
			console.error('Failed to delete:', error);
			alert(m.alert_failed_delete());
		}
		showMenuFor = null;
	}

	async function handleViewText(item: FileItem) {
		viewingFile = item;
		showTextModal = true;
		isLoadingText = true;
		extractedText = '';
		showMenuFor = null;

		try {
			const response = await filesApi.getText(String(item.id));
			extractedText = response.text || 'No text extracted';
		} catch (error) {
			console.error('Failed to load text:', error);
			extractedText = 'Failed to load extracted text. The file may not have been processed yet.';
		} finally {
			isLoadingText = false;
		}
	}

	function closeTextModal() {
		showTextModal = false;
		viewingFile = null;
		extractedText = '';
	}

	function formatSize(bytes: number): string {
		if (!bytes || bytes === 0) return '—';
		const units = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(1024));
		return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
	}

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files) {
			uploadingFiles = Array.from(input.files);
		}
	}

	// Drag & Drop handlers
	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		isDragging = true;
	}

	function handleDragLeave(e: DragEvent) {
		e.preventDefault();
		isDragging = false;
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		isDragging = false;

		if (e.dataTransfer?.files) {
			uploadingFiles = Array.from(e.dataTransfer.files);
			showUploadModal = true;
		}
	}

	function getFileIcon(mimeType: string) {
		if (mimeType.includes('pdf') || mimeType.includes('document') || mimeType.includes('text')) {
			return faFileLines;
		}
		return faFile;
	}

	function getStatusBadge(status: string) {
		switch (status) {
			case 'completed':
				return { icon: faCircleCheck, class: 'text-green-500', label: 'Extracted' };
			case 'pending':
				return { icon: faClock, class: 'text-yellow-500', label: 'Pending' };
			case 'failed':
				return { icon: faCircleExclamation, class: 'text-red-500', label: 'Failed' };
			default:
				return { icon: faClock, class: 'text-muted-foreground', label: status || 'Unknown' };
		}
	}

	function nextPage() {
		if ((currentPage + 1) * pageSize < totalFiles) {
			currentPage++;
			loadFiles();
		}
	}

	function prevPage() {
		if (currentPage > 0) {
			currentPage--;
			loadFiles();
		}
	}

	const totalPages = $derived(Math.ceil(totalFiles / pageSize));
</script>

<svelte:head>
	<title>{m.nav_files()} | AI Gateway</title>
</svelte:head>

<!-- Drop Zone Overlay -->
{#if isDragging}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-primary/10 backdrop-blur-sm"
		ondragover={handleDragOver}
		ondragleave={handleDragLeave}
		ondrop={handleDrop}
		role="region"
		aria-label="Drop zone"
	>
		<div class="rounded-xl border-2 border-dashed border-primary bg-background p-12 text-center">
			<FontAwesomeIcon icon={faUpload} class="mx-auto mb-4 h-16 w-16 text-primary" />
			<p class="text-xl font-semibold">{m.files_drop_here()}</p>
		</div>
	</div>
{/if}

<div
	class="mx-auto max-w-[1600px] px-4 py-8 sm:px-6 lg:px-8"
	ondragover={handleDragOver}
	role="main"
>
	<!-- Header -->
	<div class="mb-6 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-foreground">{m.files_title()}</h1>
			<p class="mt-1 text-sm text-muted-foreground">
				{m.files_count({ count: totalFiles.toString() })}
			</p>
		</div>
		<div class="flex gap-2">
			<Button variant="outline" onclick={() => loadFiles()}>
				<FontAwesomeIcon icon={faArrowsRotate} class="mr-2 h-4 w-4" />
				{m.common_refresh()}
			</Button>
			<Button onclick={() => (showUploadModal = true)}>
				<FontAwesomeIcon icon={faUpload} class="mr-2 h-4 w-4" />
				{m.files_upload()}
			</Button>
		</div>
	</div>

	<!-- Filters -->
	<div class="mb-4 flex flex-wrap gap-4">
		<!-- Search -->
		<div class="relative flex-1 min-w-[200px]">
			<FontAwesomeIcon icon={faMagnifyingGlass} class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
			<input
				type="text"
				placeholder={m.files_search()}
				value={searchQuery}
				oninput={handleSearchInput}
				class="w-full rounded-lg border border-input bg-background py-2 pl-10 pr-4 text-sm placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
			/>
		</div>

		<select
			bind:value={typeFilter}
			onchange={() => { currentPage = 0; loadFiles(); }}
			class="rounded-lg border border-input bg-background px-3 py-2 text-sm"
		>
			<option value="">{m.files_all_types()}</option>
			<option value="application/pdf">PDF</option>
			<option value="text/plain">Text</option>
			<option value="application/vnd.openxmlformats-officedocument.wordprocessingml.document">DOCX</option>
			<option value="text/csv">CSV</option>
			<option value="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet">XLSX</option>
		</select>
		<select
			bind:value={statusFilter}
			onchange={() => { currentPage = 0; loadFiles(); }}
			class="rounded-lg border border-input bg-background px-3 py-2 text-sm"
		>
			<option value="">{m.files_all_statuses()}</option>
			<option value="completed">{m.files_status_extracted()}</option>
			<option value="pending">{m.files_status_pending()}</option>
			<option value="failed">{m.files_status_failed()}</option>
		</select>
	</div>

	<!-- Files Table -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<FontAwesomeIcon icon={faSpinner} class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if files.length === 0}
		<div
			class={cn(
				"rounded-lg border-2 border-dashed border-border py-16 text-center transition-colors",
				isDragging && "border-primary bg-primary/5"
			)}
			ondragover={handleDragOver}
			ondragleave={handleDragLeave}
			ondrop={handleDrop}
			role="region"
		>
			<FontAwesomeIcon icon={faFile} class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-muted-foreground">{m.files_no_files()}</p>
			<p class="mt-2 text-sm text-muted-foreground">{m.files_drag_drop()}</p>
			<Button variant="outline" class="mt-4" onclick={() => (showUploadModal = true)}>
				<FontAwesomeIcon icon={faUpload} class="mr-2 h-4 w-4" />
				{m.files_upload_files()}
			</Button>
		</div>
	{:else}
		<div class="rounded-lg border border-border">
			<table class="w-full">
				<thead class="border-b border-border bg-muted/50">
					<tr>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.common_name()}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.files_type()}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.files_size()}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.files_status()}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
							{m.files_uploaded_at()}
						</th>
						<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">
							{m.common_actions()}
						</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border">
					{#each files as item (item.id)}
						{@const FileIcon = getFileIcon(item.mime_type)}
						{@const status = getStatusBadge(item.extraction_status)}
						<tr class="hover:bg-muted/30">
							<td class="px-4 py-3">
								<div class="flex items-center gap-3">
									<FontAwesomeIcon icon={FileIcon} class="h-5 w-5 text-muted-foreground" />
									<span class="font-medium">{item.original_name || item.filename}</span>
								</div>
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{item.mime_type.split('/').pop()?.toUpperCase() || 'Unknown'}
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{formatSize(item.size)}
							</td>
							<td class="px-4 py-3">
								{#if true}
									{@const StatusIcon = status.icon}
									<div class="flex items-center gap-1.5">
										<FontAwesomeIcon icon={StatusIcon} class={cn('h-4 w-4', status.class)} />
										<span class="text-sm">{status.label}</span>
									</div>
								{/if}
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{formatRelativeTime(item.created_at)}
							</td>
							<td class="relative px-4 py-3 text-right">
								<div class="dropdown-menu-container inline-block">
									<button
										onclick={(e) => { e.stopPropagation(); const id = String(item.id); showMenuFor = showMenuFor === id ? null : id; }}
										class="rounded p-1.5 text-muted-foreground hover:bg-accent"
									>
										<FontAwesomeIcon icon={faEllipsisVertical} class="h-4 w-4" />
									</button>

									{#if showMenuFor === String(item.id)}
										<div class="absolute right-0 top-full z-50 mt-1 w-44 rounded-lg border border-border bg-popover py-1 shadow-lg">
											{#if item.extraction_status === 'completed'}
												<button
													onclick={() => handleViewText(item)}
													class="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
												>
													<FontAwesomeIcon icon={faEye} class="h-4 w-4" />
													{m.files_view_text()}
												</button>
											{/if}
											<button
												onclick={() => { filesApi.download(String(item.id)); showMenuFor = null; }}
												class="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
											>
												<FontAwesomeIcon icon={faDownload} class="h-4 w-4" />
												{m.tooltip_download()}
											</button>
											<button
												onclick={() => handleDelete(item)}
												class="flex w-full items-center gap-2 px-3 py-2 text-sm text-destructive hover:bg-destructive/10"
											>
												<FontAwesomeIcon icon={faTrash} class="h-4 w-4" />
												{m.common_delete()}
											</button>
										</div>
									{/if}
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<!-- Pagination -->
		{#if totalPages > 1}
			<div class="mt-4 flex items-center justify-between">
				<p class="text-sm text-muted-foreground">
					Showing {currentPage * pageSize + 1} - {Math.min((currentPage + 1) * pageSize, totalFiles)} of {totalFiles}
				</p>
				<div class="flex gap-2">
					<Button variant="outline" size="sm" onclick={prevPage} disabled={currentPage === 0}>
						<FontAwesomeIcon icon={faChevronLeft} class="h-4 w-4" />
					</Button>
					<span class="flex items-center px-3 text-sm">
						Page {currentPage + 1} of {totalPages}
					</span>
					<Button variant="outline" size="sm" onclick={nextPage} disabled={currentPage >= totalPages - 1}>
						<FontAwesomeIcon icon={faChevronRight} class="h-4 w-4" />
					</Button>
				</div>
			</div>
		{/if}
	{/if}
</div>

<!-- Upload Modal -->
{#if showUploadModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showUploadModal = false)}
		role="dialog"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && (showUploadModal = false)}
	>
		<div class="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl">
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Upload Files</h2>
				<button onclick={() => { showUploadModal = false; uploadingFiles = []; }} class="text-muted-foreground hover:text-foreground">
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>
			<p class="mb-4 text-sm text-muted-foreground">
				Supported formats: PDF, DOCX, TXT, CSV, XLSX, MD (Max 100MB)
			</p>

			<div class="mb-4">
				<label
					class="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-border py-8 transition-colors hover:border-primary hover:bg-primary/5"
					ondragover={(e) => e.preventDefault()}
					ondrop={(e) => { e.preventDefault(); if (e.dataTransfer?.files) uploadingFiles = Array.from(e.dataTransfer.files); }}
				>
					<FontAwesomeIcon icon={faUpload} class="mb-2 h-8 w-8 text-muted-foreground" />
					<span class="text-sm text-muted-foreground">Click to select or drag files here</span>
					<input
						type="file"
						multiple
						class="hidden"
						onchange={handleFileSelect}
						accept=".pdf,.docx,.doc,.txt,.csv,.xlsx,.xls,.md,.rtf"
					/>
				</label>
			</div>

			{#if uploadingFiles.length > 0}
				<div class="mb-4 max-h-48 space-y-2 overflow-y-auto">
					{#each uploadingFiles as file, idx}
						<div class="flex items-center gap-2 rounded bg-muted px-3 py-2 text-sm">
							<FontAwesomeIcon icon={faFile} class="h-4 w-4 shrink-0" />
							<span class="truncate">{file.name}</span>
							<span class="ml-auto shrink-0 text-muted-foreground">{formatSize(file.size)}</span>
							<button
								onclick={() => { uploadingFiles = uploadingFiles.filter((_, i) => i !== idx); }}
								class="text-muted-foreground hover:text-destructive"
							>
								<FontAwesomeIcon icon={faXmark} class="h-4 w-4" />
							</button>
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
						<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					Upload {uploadingFiles.length > 0 ? `(${uploadingFiles.length})` : ''}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- View Text Modal -->
{#if showTextModal && viewingFile}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && closeTextModal()}
		role="dialog"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && closeTextModal()}
	>
		<div class="flex max-h-[80vh] w-full max-w-3xl flex-col rounded-xl border border-border bg-card shadow-xl">
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-border px-6 py-4">
				<div>
					<h2 class="text-lg font-semibold">Extracted Text</h2>
					<p class="text-sm text-muted-foreground">{viewingFile.original_name || viewingFile.filename}</p>
				</div>
				<button onclick={closeTextModal} class="text-muted-foreground hover:text-foreground">
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			<!-- Content -->
			<div class="flex-1 overflow-y-auto p-6">
				{#if isLoadingText}
					<div class="flex items-center justify-center py-12">
						<FontAwesomeIcon icon={faSpinner} class="h-8 w-8 animate-spin text-muted-foreground" />
					</div>
				{:else}
					<pre class="whitespace-pre-wrap rounded-lg border border-border bg-muted/50 p-4 font-mono text-sm leading-relaxed">{extractedText}</pre>
				{/if}
			</div>

			<!-- Footer -->
			<div class="flex justify-end gap-3 border-t border-border px-6 py-4">
				<Button variant="outline" onclick={() => { copyToClipboard(extractedText); }}>
					Copy Text
				</Button>
				<Button onclick={closeTextModal}>Close</Button>
			</div>
		</div>
	</div>
{/if}
