<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { JobProgressCard } from '$lib/components/gitlab';
	import {
		listMyJobs,
		getJobTypes,
		getJobStatuses,
		type UserJob,
		type JobType,
		type JobStatus,
		type JobTypeInfo,
		type JobStatusInfo,
		isJobActive
	} from '$lib/api/gitlab-jobs';

	let jobs: UserJob[] = [];
	let jobTypes: JobTypeInfo[] = [];
	let jobStatuses: JobStatusInfo[] = [];
	let total = 0;
	let loading = true;
	let error = '';

	// Filters
	let filterStatus: JobStatus | '' = '';
	let filterType: JobType | '' = '';
	let page_num = 1;
	const limit = 20;

	// Tabs
	let activeTab: 'all' | 'active' | 'completed' = 'all';

	$: offset = (page_num - 1) * limit;
	$: totalPages = Math.ceil(total / limit);

	onMount(async () => {
		await Promise.all([fetchJobs(), fetchMeta()]);

		// Check for job ID in URL
		const id = $page.url.searchParams.get('id');
		if (id) {
			// Scroll to or highlight specific job
			setTimeout(() => {
				const element = document.getElementById(`job-${id}`);
				if (element) {
					element.scrollIntoView({ behavior: 'smooth', block: 'center' });
					element.classList.add('highlighted');
				}
			}, 100);
		}
	});

	async function fetchMeta() {
		try {
			const [typesRes, statusesRes] = await Promise.all([getJobTypes(), getJobStatuses()]);
			jobTypes = typesRes.types;
			jobStatuses = statusesRes.statuses;
		} catch (e) {
			console.error('Failed to fetch job meta:', e);
		}
	}

	async function fetchJobs() {
		loading = true;
		error = '';
		try {
			let statusFilter: JobStatus | undefined = filterStatus || undefined;

			// Apply tab filter
			if (activeTab === 'active') {
				statusFilter = 'running'; // Pending and running
			} else if (activeTab === 'completed') {
				statusFilter = 'completed';
			}

			// Get integration_id from URL if present
			const integrationId = $page.url.searchParams.get('integration_id') || undefined;

			const result = await listMyJobs({
				status: statusFilter,
				job_type: filterType || undefined,
				integration_id: integrationId,
				limit,
				offset
			});
			jobs = result.jobs || [];
			total = result.total;
		} catch (e: any) {
			error = e.message || 'Failed to fetch jobs';
			console.error('Failed to fetch jobs:', e);
		} finally {
			loading = false;
		}
	}

	function handleTabChange(tab: 'all' | 'active' | 'completed') {
		activeTab = tab;
		page_num = 1;
		filterStatus = '';
		fetchJobs();
	}

	function handleFilterChange() {
		page_num = 1;
		fetchJobs();
	}

	function handlePageChange(newPage: number) {
		if (newPage >= 1 && newPage <= totalPages) {
			page_num = newPage;
			fetchJobs();
		}
	}

	function refresh() {
		fetchJobs();
	}
</script>

<svelte:head>
	<title>Jobs - GitLab Integration</title>
</svelte:head>

<div class="jobs-page">
	<header class="page-header">
		<div class="header-content">
			<h1>Background Jobs</h1>
			<p class="subtitle">Monitor and manage your scanning tasks</p>
		</div>
		<button class="refresh-btn" on:click={refresh} disabled={loading}>
			{loading ? '⏳' : '🔄'} Refresh
		</button>
	</header>

	<div class="tabs">
		<button class="tab" class:active={activeTab === 'all'} on:click={() => handleTabChange('all')}>
			All Jobs
		</button>
		<button
			class="tab"
			class:active={activeTab === 'active'}
			on:click={() => handleTabChange('active')}
		>
			Active
		</button>
		<button
			class="tab"
			class:active={activeTab === 'completed'}
			on:click={() => handleTabChange('completed')}
		>
			Completed
		</button>
	</div>

	<div class="filters">
		<div class="filter-group">
			<label for="type-filter">Type:</label>
			<select id="type-filter" bind:value={filterType} on:change={handleFilterChange}>
				<option value="">All Types</option>
				{#each jobTypes as type}
					<option value={type.type}>{type.name}</option>
				{/each}
			</select>
		</div>

		{#if activeTab === 'all'}
			<div class="filter-group">
				<label for="status-filter">Status:</label>
				<select id="status-filter" bind:value={filterStatus} on:change={handleFilterChange}>
					<option value="">All Statuses</option>
					{#each jobStatuses as status}
						<option value={status.status}>{status.name}</option>
					{/each}
				</select>
			</div>
		{/if}

		<div class="result-count">
			{total} job{total !== 1 ? 's' : ''} found
		</div>
	</div>

	{#if error}
		<div class="error-banner">
			<span class="error-icon">⚠️</span>
			<span>{error}</span>
		</div>
	{/if}

	{#if loading}
		<div class="loading-state">
			<div class="spinner"></div>
			<span>Loading jobs...</span>
		</div>
	{:else if jobs.length === 0}
		<div class="empty-state">
			<span class="empty-icon">📭</span>
			<h3>No jobs found</h3>
			<p>Start a scan from a project page to see it here.</p>
			<a href="/gitlab" class="btn-primary">Go to Projects</a>
		</div>
	{:else}
		<div class="jobs-grid">
			{#each jobs as job (job.id)}
				<div id="job-{job.id}" class="job-wrapper">
					<JobProgressCard {job} showCancel={isJobActive(job.status)} />
				</div>
			{/each}
		</div>

		{#if totalPages > 1}
			<div class="pagination">
				<button
					class="page-btn"
					disabled={page_num === 1}
					on:click={() => handlePageChange(page_num - 1)}
				>
					← Previous
				</button>
				<span class="page-info">
					Page {page_num} of {totalPages}
				</span>
				<button
					class="page-btn"
					disabled={page_num === totalPages}
					on:click={() => handlePageChange(page_num + 1)}
				>
					Next →
				</button>
			</div>
		{/if}
	{/if}
</div>

<style>
	.jobs-page {
		max-width: 1200px;
		margin: 0 auto;
		padding: 24px;
	}

	.page-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 24px;
	}

	.header-content h1 {
		font-size: 1.75rem;
		font-weight: 700;
		color: var(--text-primary, #1a1a1a);
		margin: 0;
	}

	.subtitle {
		color: var(--text-secondary, #666);
		margin: 4px 0 0;
	}

	.refresh-btn {
		padding: 8px 16px;
		background: var(--bg-secondary, #f5f5f5);
		border: 1px solid var(--border-color, #e0e0e0);
		border-radius: 8px;
		cursor: pointer;
		font-size: 0.9rem;
		display: flex;
		align-items: center;
		gap: 6px;
		transition: background 0.2s ease;
	}

	.refresh-btn:hover:not(:disabled) {
		background: var(--bg-hover, #eee);
	}

	.refresh-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.tabs {
		display: flex;
		gap: 4px;
		padding: 4px;
		background: var(--bg-secondary, #f5f5f5);
		border-radius: 10px;
		margin-bottom: 20px;
		width: fit-content;
	}

	.tab {
		padding: 8px 20px;
		background: transparent;
		border: none;
		border-radius: 8px;
		cursor: pointer;
		font-size: 0.9rem;
		color: var(--text-secondary, #666);
		transition: all 0.2s ease;
	}

	.tab:hover {
		color: var(--text-primary, #1a1a1a);
	}

	.tab.active {
		background: var(--card-bg, #fff);
		color: var(--text-primary, #1a1a1a);
		font-weight: 500;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	.filters {
		display: flex;
		gap: 16px;
		align-items: center;
		margin-bottom: 20px;
		flex-wrap: wrap;
	}

	.filter-group {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.filter-group label {
		font-size: 0.9rem;
		color: var(--text-secondary, #666);
	}

	.filter-group select {
		padding: 6px 12px;
		border: 1px solid var(--border-color, #e0e0e0);
		border-radius: 6px;
		background: var(--card-bg, #fff);
		font-size: 0.9rem;
		cursor: pointer;
	}

	.result-count {
		margin-left: auto;
		font-size: 0.85rem;
		color: var(--text-secondary, #666);
	}

	.error-banner {
		background: var(--error-bg, #fef2f2);
		color: var(--color-red, #ef4444);
		padding: 12px 16px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 20px;
	}

	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 60px;
		color: var(--text-secondary, #666);
		gap: 16px;
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid var(--border-color, #e0e0e0);
		border-top-color: var(--color-blue, #3b82f6);
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.empty-state {
		text-align: center;
		padding: 60px;
		color: var(--text-secondary, #666);
	}

	.empty-icon {
		font-size: 3rem;
		display: block;
		margin-bottom: 16px;
	}

	.empty-state h3 {
		margin: 0 0 8px;
		color: var(--text-primary, #1a1a1a);
	}

	.empty-state p {
		margin: 0 0 20px;
	}

	.btn-primary {
		display: inline-block;
		padding: 10px 24px;
		background: var(--color-blue, #3b82f6);
		color: white;
		text-decoration: none;
		border-radius: 8px;
		font-weight: 500;
		transition: background 0.2s ease;
	}

	.btn-primary:hover {
		background: var(--color-blue-dark, #2563eb);
	}

	.jobs-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
		gap: 16px;
	}

	.job-wrapper {
		transition: transform 0.2s ease;
	}

	.job-wrapper:global(.highlighted) {
		animation: highlight 2s ease;
	}

	@keyframes highlight {
		0%,
		100% {
			transform: scale(1);
		}
		10% {
			transform: scale(1.02);
			box-shadow: 0 0 0 3px var(--color-blue, #3b82f6);
		}
		30% {
			transform: scale(1);
		}
	}

	.pagination {
		display: flex;
		justify-content: center;
		align-items: center;
		gap: 16px;
		margin-top: 32px;
		padding-top: 24px;
		border-top: 1px solid var(--border-color, #e0e0e0);
	}

	.page-btn {
		padding: 8px 16px;
		background: var(--bg-secondary, #f5f5f5);
		border: 1px solid var(--border-color, #e0e0e0);
		border-radius: 6px;
		cursor: pointer;
		font-size: 0.9rem;
		transition: background 0.2s ease;
	}

	.page-btn:hover:not(:disabled) {
		background: var(--bg-hover, #eee);
	}

	.page-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.page-info {
		font-size: 0.9rem;
		color: var(--text-secondary, #666);
	}
</style>
