<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import { faLock, faShieldHalved, faChartBar, faBox, faSkull, faFileLines, faFlask, faMagnifyingGlass, faGear, faTriangleExclamation, faFolder, faStopwatch, faCalendar, faArrowRight } from '@fortawesome/free-solid-svg-icons';
	import {
		type UserJob,
		type JobStatus,
		type JobType,
		getJobTypeName,
		getJobStatusColor,
		streamJobProgress,
		cancelJob,
		isJobActive
	} from '$lib/api/gitlab-jobs';

	export let job: UserJob;
	export let showCancel: boolean = true;
	export let autoStream: boolean = true;

	let progress = job.progress;
	let progressMsg = job.progress_msg || '';
	let status: JobStatus = job.status;
	let error = job.error;
	let cleanup: (() => void) | null = null;
	let cancelling = false;

	$: statusColor = getJobStatusColor(status);
	$: isActive = isJobActive(status);
	$: jobTypeName = getJobTypeName(job.job_type);

	onMount(() => {
		if (autoStream && isJobActive(job.status)) {
			startStreaming();
		}
	});

	onDestroy(() => {
		if (cleanup) {
			cleanup();
		}
	});

	function startStreaming() {
		cleanup = streamJobProgress(job.id, {
			onInit: (j) => {
				progress = j.progress;
				progressMsg = j.progress_msg || '';
				status = j.status;
			},
			onProgress: (event) => {
				progress = event.progress;
				progressMsg = event.progress_msg || '';
				status = event.status;
				if (event.error) {
					error = event.error;
				}
			},
			onComplete: (j) => {
				progress = j.progress;
				progressMsg = j.progress_msg || '';
				status = j.status;
				error = j.error;
			},
			onError: (err) => {
				error = err;
			}
		});
	}

	async function handleCancel() {
		if (cancelling) return;
		cancelling = true;
		try {
			await cancelJob(job.id);
			status = 'cancelled';
		} catch (e) {
			console.error('Failed to cancel job:', e);
		} finally {
			cancelling = false;
		}
	}

	function formatTime(dateStr?: string): string {
		if (!dateStr) return '-';
		const date = new Date(dateStr);
		return date.toLocaleString();
	}

	function getElapsedTime(): string {
		const start = job.started_at ? new Date(job.started_at) : new Date(job.created_at);
		const end = job.completed_at ? new Date(job.completed_at) : new Date();
		const diff = Math.floor((end.getTime() - start.getTime()) / 1000);

		if (diff < 60) return `${diff}s`;
		if (diff < 3600) return `${Math.floor(diff / 60)}m ${diff % 60}s`;
		return `${Math.floor(diff / 3600)}h ${Math.floor((diff % 3600) / 60)}m`;
	}
</script>

<div
	class="job-card"
	class:active={isActive}
	class:failed={status === 'failed'}
	class:completed={status === 'completed'}
>
	<div class="job-header">
		<div class="job-type">
			<span class="type-icon">
				{#if job.job_type.includes('secrets')}<FontAwesomeIcon icon={faLock} />
				{:else if job.job_type.includes('sast')}<FontAwesomeIcon icon={faShieldHalved} />
				{:else if job.job_type.includes('quality')}<FontAwesomeIcon icon={faChartBar} />
				{:else if job.job_type.includes('dependency')}<FontAwesomeIcon icon={faBox} />
				{:else if job.job_type.includes('deadcode')}<FontAwesomeIcon icon={faSkull} />
				{:else if job.job_type.includes('docs')}<FontAwesomeIcon icon={faFileLines} />
				{:else if job.job_type.includes('test')}<FontAwesomeIcon icon={faFlask} />
				{:else if job.job_type.includes('index')}<FontAwesomeIcon icon={faMagnifyingGlass} />
				{:else}<FontAwesomeIcon icon={faGear} />
				{/if}
			</span>
			<span class="type-name">{jobTypeName}</span>
		</div>
		<div class="job-status" style="--status-color: var(--color-{statusColor}, gray)">
			<span class="status-dot"></span>
			<span class="status-text">{status}</span>
		</div>
	</div>

	<div class="job-progress">
		<div class="progress-bar">
			<div class="progress-fill" style="width: {progress}%"></div>
		</div>
		<div class="progress-info">
			<span class="progress-percent">{progress}%</span>
			{#if progressMsg}
				<span class="progress-msg">{progressMsg}</span>
			{/if}
		</div>
	</div>

	{#if error}
		<div class="job-error">
			<span class="error-icon"><FontAwesomeIcon icon={faTriangleExclamation} /></span>
			<span class="error-text">{error}</span>
		</div>
	{/if}

	<div class="job-meta">
		{#if job.project_name}
			<span class="meta-item"><FontAwesomeIcon icon={faFolder} class="inline" /> {job.project_name}</span>
		{/if}
		<span class="meta-item"><FontAwesomeIcon icon={faStopwatch} class="inline" /> {getElapsedTime()}</span>
		<span class="meta-item" title={formatTime(job.created_at)}>
			<FontAwesomeIcon icon={faCalendar} class="inline" /> {new Date(job.created_at).toLocaleDateString()}
		</span>
	</div>

	{#if showCancel && isActive}
		<div class="job-actions">
			<button class="cancel-btn" on:click={handleCancel} disabled={cancelling}>
				{cancelling ? 'Cancelling...' : 'Cancel'}
			</button>
		</div>
	{/if}

	{#if job.result_url}
		<div class="job-result">
			<a href={job.result_url} target="_blank" rel="noopener noreferrer"> View Result <FontAwesomeIcon icon={faArrowRight} class="inline h-3 w-3" /> </a>
		</div>
	{/if}
</div>

<style>
	.job-card {
		background: var(--card-bg, #fff);
		border: 1px solid var(--border-color, #e0e0e0);
		border-radius: 12px;
		padding: 16px;
		transition: all 0.2s ease;
	}

	.job-card.active {
		border-color: var(--color-blue, #3b82f6);
		box-shadow: 0 0 0 1px var(--color-blue, #3b82f6);
	}

	.job-card.failed {
		border-color: var(--color-red, #ef4444);
	}

	.job-card.completed {
		border-color: var(--color-green, #22c55e);
	}

	.job-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 12px;
	}

	.job-type {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.type-icon {
		font-size: 1.2rem;
	}

	.type-name {
		font-weight: 600;
		color: var(--text-primary, #1a1a1a);
	}

	.job-status {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 0.85rem;
		text-transform: capitalize;
	}

	.status-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--status-color);
	}

	.job-card.active .status-dot {
		animation: pulse 1.5s ease-in-out infinite;
	}

	@keyframes pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.5;
		}
	}

	.job-progress {
		margin-bottom: 12px;
	}

	.progress-bar {
		height: 8px;
		background: var(--progress-bg, #e5e7eb);
		border-radius: 4px;
		overflow: hidden;
	}

	.progress-fill {
		height: 100%;
		background: var(--color-blue, #3b82f6);
		border-radius: 4px;
		transition: width 0.3s ease;
	}

	.job-card.completed .progress-fill {
		background: var(--color-green, #22c55e);
	}

	.job-card.failed .progress-fill {
		background: var(--color-red, #ef4444);
	}

	.progress-info {
		display: flex;
		justify-content: space-between;
		margin-top: 6px;
		font-size: 0.8rem;
		color: var(--text-secondary, #666);
	}

	.progress-msg {
		flex: 1;
		text-align: right;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		margin-left: 12px;
	}

	.job-error {
		background: var(--error-bg, #fef2f2);
		color: var(--color-red, #ef4444);
		padding: 8px 12px;
		border-radius: 6px;
		font-size: 0.85rem;
		margin-bottom: 12px;
		display: flex;
		align-items: flex-start;
		gap: 8px;
	}

	.error-text {
		flex: 1;
		word-break: break-word;
	}

	.job-meta {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;
		font-size: 0.8rem;
		color: var(--text-secondary, #666);
	}

	.meta-item {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.job-actions {
		margin-top: 12px;
		padding-top: 12px;
		border-top: 1px solid var(--border-color, #e0e0e0);
	}

	.cancel-btn {
		padding: 6px 16px;
		background: var(--color-orange, #f97316);
		color: white;
		border: none;
		border-radius: 6px;
		cursor: pointer;
		font-size: 0.85rem;
		transition: background 0.2s ease;
	}

	.cancel-btn:hover:not(:disabled) {
		background: var(--color-orange-dark, #ea580c);
	}

	.cancel-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.job-result {
		margin-top: 12px;
		padding-top: 12px;
		border-top: 1px solid var(--border-color, #e0e0e0);
	}

	.job-result a {
		color: var(--color-blue, #3b82f6);
		text-decoration: none;
		font-weight: 500;
	}

	.job-result a:hover {
		text-decoration: underline;
	}
</style>
