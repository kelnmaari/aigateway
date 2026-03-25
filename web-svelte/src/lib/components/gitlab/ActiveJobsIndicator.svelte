<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { getActiveJobs, type UserJob, getJobTypeName } from '$lib/api/gitlab-jobs';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import { faGear, faArrowRight, faWandMagicSparkles } from '@fortawesome/free-solid-svg-icons';

	export let refreshInterval: number = 5000; // 5 seconds

	let activeJobs: UserJob[] = [];
	let isOpen = false;
	let loading = false;
	let intervalId: number | null = null;

	$: hasActiveJobs = activeJobs.length > 0;

	onMount(() => {
		fetchJobs();
		intervalId = window.setInterval(fetchJobs, refreshInterval);
	});

	onDestroy(() => {
		if (intervalId) {
			clearInterval(intervalId);
		}
	});

	async function fetchJobs() {
		try {
			loading = true;
			const result = await getActiveJobs();
			activeJobs = result.jobs || [];
		} catch (e) {
			console.error('Failed to fetch active jobs:', e);
		} finally {
			loading = false;
		}
	}

	function toggleDropdown() {
		isOpen = !isOpen;
	}

	function closeDropdown() {
		isOpen = false;
	}
</script>

<svelte:window on:click={closeDropdown} />

<div class="jobs-indicator" on:click|stopPropagation={toggleDropdown}>
	<button class="indicator-btn" class:has-jobs={hasActiveJobs} aria-label="Active jobs">
		<span class="icon"><FontAwesomeIcon icon={faGear} /></span>
		{#if hasActiveJobs}
			<span class="badge">{activeJobs.length}</span>
		{/if}
		{#if loading && hasActiveJobs}
			<span class="spinner"></span>
		{/if}
	</button>

	{#if isOpen}
		<div class="dropdown" on:click|stopPropagation>
			<div class="dropdown-header">
				<span class="header-title">Active Jobs</span>
				<a href="/gitlab/jobs" class="view-all">View All <FontAwesomeIcon icon={faArrowRight} class="inline h-3 w-3" /></a>
			</div>

			{#if activeJobs.length === 0}
				<div class="empty-state">
					<span class="empty-icon"><FontAwesomeIcon icon={faWandMagicSparkles} /></span>
					<span class="empty-text">No active jobs</span>
				</div>
			{:else}
				<ul class="job-list">
					{#each activeJobs.slice(0, 5) as job}
						<li class="job-item">
							<a href="/gitlab/jobs?id={job.id}">
								<div class="job-info">
									<span class="job-type">{getJobTypeName(job.job_type)}</span>
									{#if job.project_name}
										<span class="job-project">{job.project_name}</span>
									{/if}
								</div>
								<div class="job-progress">
									<div class="progress-bar">
										<div class="progress-fill" style="width: {job.progress}%"></div>
									</div>
									<span class="progress-text">{job.progress}%</span>
								</div>
							</a>
						</li>
					{/each}
				</ul>
				{#if activeJobs.length > 5}
					<div class="more-jobs">
						+{activeJobs.length - 5} more jobs
					</div>
				{/if}
			{/if}
		</div>
	{/if}
</div>

<style>
	.jobs-indicator {
		position: relative;
	}

	.indicator-btn {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		background: transparent;
		border: none;
		border-radius: 8px;
		cursor: pointer;
		transition: background 0.2s ease;
	}

	.indicator-btn:hover {
		background: var(--hover-bg, rgba(0, 0, 0, 0.05));
	}

	.indicator-btn.has-jobs {
		animation: pulse-bg 2s ease-in-out infinite;
	}

	@keyframes pulse-bg {
		0%,
		100% {
			background: transparent;
		}
		50% {
			background: rgba(59, 130, 246, 0.1);
		}
	}

	.icon {
		font-size: 1.2rem;
	}

	.badge {
		position: absolute;
		top: 2px;
		right: 2px;
		min-width: 18px;
		height: 18px;
		padding: 0 5px;
		background: var(--color-blue, #3b82f6);
		color: white;
		font-size: 0.7rem;
		font-weight: 600;
		border-radius: 9px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.spinner {
		position: absolute;
		bottom: 4px;
		right: 4px;
		width: 8px;
		height: 8px;
		border: 2px solid var(--color-blue, #3b82f6);
		border-top-color: transparent;
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.dropdown {
		position: absolute;
		top: calc(100% + 8px);
		right: 0;
		width: 320px;
		background: var(--card-bg, #fff);
		border: 1px solid var(--border-color, #e0e0e0);
		border-radius: 12px;
		box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
		overflow: hidden;
		z-index: 1000;
	}

	.dropdown-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 12px 16px;
		border-bottom: 1px solid var(--border-color, #e0e0e0);
	}

	.header-title {
		font-weight: 600;
		color: var(--text-primary, #1a1a1a);
	}

	.view-all {
		font-size: 0.85rem;
		color: var(--color-blue, #3b82f6);
		text-decoration: none;
	}

	.view-all:hover {
		text-decoration: underline;
	}

	.empty-state {
		padding: 24px;
		text-align: center;
		color: var(--text-secondary, #666);
	}

	.empty-icon {
		display: block;
		font-size: 2rem;
		margin-bottom: 8px;
	}

	.job-list {
		list-style: none;
		margin: 0;
		padding: 0;
		max-height: 300px;
		overflow-y: auto;
	}

	.job-item a {
		display: block;
		padding: 12px 16px;
		text-decoration: none;
		color: inherit;
		transition: background 0.2s ease;
	}

	.job-item a:hover {
		background: var(--hover-bg, rgba(0, 0, 0, 0.02));
	}

	.job-info {
		display: flex;
		justify-content: space-between;
		margin-bottom: 8px;
	}

	.job-type {
		font-weight: 500;
		color: var(--text-primary, #1a1a1a);
		font-size: 0.9rem;
	}

	.job-project {
		font-size: 0.8rem;
		color: var(--text-secondary, #666);
		max-width: 120px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.job-progress {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.progress-bar {
		flex: 1;
		height: 6px;
		background: var(--progress-bg, #e5e7eb);
		border-radius: 3px;
		overflow: hidden;
	}

	.progress-fill {
		height: 100%;
		background: var(--color-blue, #3b82f6);
		border-radius: 3px;
		transition: width 0.3s ease;
	}

	.progress-text {
		font-size: 0.75rem;
		color: var(--text-secondary, #666);
		min-width: 32px;
		text-align: right;
	}

	.more-jobs {
		padding: 8px 16px;
		text-align: center;
		font-size: 0.8rem;
		color: var(--text-secondary, #666);
		border-top: 1px solid var(--border-color, #e0e0e0);
	}
</style>
