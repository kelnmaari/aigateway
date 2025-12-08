<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		Activity,
		RefreshCw,
		Loader2,
		X,
		CheckCircle,
		XCircle,
		AlertCircle,
		Clock,
		Play,
		Pause,
		Server,
		RotateCcw,
		Wifi,
		WifiOff
	} from 'lucide-svelte';
	import { gitlabApi, type GitLabQueueStats, type GitLabJob } from '$lib/api/gitlab';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import GitLabNav from '$lib/components/gitlab-nav.svelte';

	let stats = $state<GitLabQueueStats | null>(null);
	let jobs = $state<GitLabJob[]>([]);
	let totalJobs = $state(0);
	let isLoading = $state(true);
	let autoRefresh = $state(true);
	let refreshInterval: ReturnType<typeof setInterval> | null = null;

	// WebSocket (disabled by default - requires full GitLab integration setup)
	let ws: WebSocket | null = null;
	let wsConnected = $state(false);
	let wsEnabled = $state(false);
	let recentEvents = $state<{ type: string; data: any; time: Date }[]>([]);

	// Filters
	let statusFilter = $state('');

	// Pagination
	let currentPage = $state(0);
	let pageSize = 20;

	onMount(async () => {
		await loadAll();
		startAutoRefresh();
		if (wsEnabled) connectWebSocket();
	});

	onDestroy(() => {
		stopAutoRefresh();
		disconnectWebSocket();
	});

	function connectWebSocket() {
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const wsUrl = `${protocol}//${window.location.host}/api/admin/gitlab/ws`;
		
		try {
			ws = new WebSocket(wsUrl);
			
			ws.onopen = () => {
				wsConnected = true;
				// Subscribe to queue events
				ws?.send(JSON.stringify({
					type: 'subscribe',
					filters: { event_types: ['job_created', 'job_started', 'job_completed', 'job_failed', 'stats'] }
				}));
			};
			
			ws.onmessage = (event) => {
				try {
					const msg = JSON.parse(event.data);
					handleWebSocketMessage(msg);
				} catch (e) {
					console.error('Failed to parse WebSocket message:', e);
				}
			};
			
			ws.onclose = () => {
				wsConnected = false;
				// Reconnect after 5 seconds
				if (wsEnabled) {
					setTimeout(connectWebSocket, 5000);
				}
			};
			
			ws.onerror = () => {
				wsConnected = false;
			};
		} catch (e) {
			console.error('WebSocket connection failed:', e);
		}
	}

	function disconnectWebSocket() {
		if (ws) {
			ws.close();
			ws = null;
		}
	}

	function handleWebSocketMessage(msg: { type: string; event: string; data: any }) {
		// Add to recent events
		recentEvents = [{ type: msg.event, data: msg.data, time: new Date() }, ...recentEvents.slice(0, 9)];
		
		switch (msg.type) {
			case 'stats':
				if (msg.data) {
					stats = { ...stats, ...msg.data };
				}
				break;
			case 'job_created':
			case 'job_started':
			case 'job_completed':
			case 'job_failed':
				// Update job in list or add it
				if (msg.data?.job_id) {
					const idx = jobs.findIndex(j => j.id === msg.data.job_id);
					if (idx >= 0) {
						jobs[idx] = { ...jobs[idx], ...msg.data, status: msg.data.status };
						jobs = [...jobs];
					} else if (msg.type === 'job_created') {
						loadJobs(); // Reload to get new job
					}
				}
				break;
		}
	}

	function toggleWebSocket() {
		wsEnabled = !wsEnabled;
		if (wsEnabled) {
			connectWebSocket();
		} else {
			disconnectWebSocket();
		}
	}

	function startAutoRefresh() {
		if (refreshInterval) return;
		refreshInterval = setInterval(() => {
			if (autoRefresh) {
				loadAll();
			}
		}, 5000);
	}

	function stopAutoRefresh() {
		if (refreshInterval) {
			clearInterval(refreshInterval);
			refreshInterval = null;
		}
	}

	async function loadAll() {
		await Promise.all([loadStats(), loadJobs()]);
	}

	async function loadStats() {
		try {
			stats = await gitlabApi.getQueueStatus();
		} catch (error) {
			console.error('Failed to load queue stats:', error);
		}
	}

	async function loadJobs() {
		isLoading = true;
		try {
			const response = await gitlabApi.listJobs({
				limit: pageSize,
				offset: currentPage * pageSize,
				status: statusFilter || undefined
			});
			jobs = response.data || [];
			totalJobs = response.total || jobs.length;
		} catch (error) {
			console.error('Failed to load jobs:', error);
			jobs = [];
		} finally {
			isLoading = false;
		}
	}

	async function handleCancelJob(job: GitLabJob) {
		if (!confirm(`Cancel job for MR !${job.mr_iid}?`)) return;
		
		try {
			await gitlabApi.cancelJob(job.id);
			await loadJobs();
		} catch (error) {
			alert('Failed to cancel job');
		}
	}

	async function handleRetryJob(job: GitLabJob) {
		try {
			await gitlabApi.retryJob(job.id);
			await loadJobs();
		} catch (error) {
			alert('Failed to retry job');
		}
	}

	function getStatusIcon(status: string) {
		switch (status) {
			case 'completed':
				return CheckCircle;
			case 'failed':
			case 'cancelled':
				return XCircle;
			case 'processing':
				return Loader2;
			case 'pending':
			case 'retrying':
				return Clock;
			default:
				return AlertCircle;
		}
	}

	function getStatusColor(status: string) {
		switch (status) {
			case 'completed':
				return 'text-green-500';
			case 'failed':
				return 'text-red-500';
			case 'cancelled':
				return 'text-gray-500';
			case 'processing':
				return 'text-blue-500';
			case 'pending':
			case 'retrying':
				return 'text-yellow-500';
			default:
				return 'text-muted-foreground';
		}
	}

	const totalPages = $derived(Math.ceil(totalJobs / pageSize));
</script>

<div class="space-y-6">
	<!-- Sub Navigation -->
	<GitLabNav />

	<!-- Header -->
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-4">
			<div>
				<h2 class="text-2xl font-bold text-foreground">Analysis Queue</h2>
				<p class="text-muted-foreground">Monitor and manage MR analysis jobs</p>
			</div>
		</div>
		<div class="flex items-center gap-2">
			<!-- WebSocket Status -->
			<Button
				variant={wsEnabled ? (wsConnected ? 'default' : 'outline') : 'outline'}
				size="sm"
				onclick={toggleWebSocket}
				title={wsConnected ? 'Connected - Click to disconnect' : 'Disconnected - Click to connect'}
			>
				{#if wsConnected}
					<Wifi class="mr-2 h-4 w-4 text-green-500" />
					Live
				{:else}
					<WifiOff class="mr-2 h-4 w-4" />
					{wsEnabled ? 'Connecting...' : 'Offline'}
				{/if}
			</Button>
			<Button
				variant={autoRefresh ? 'default' : 'outline'}
				size="sm"
				onclick={() => { autoRefresh = !autoRefresh; }}
			>
				{#if autoRefresh}
					<Pause class="mr-2 h-4 w-4" />
					Auto-refresh ON
				{:else}
					<Play class="mr-2 h-4 w-4" />
					Auto-refresh OFF
				{/if}
			</Button>
			<Button variant="outline" size="sm" onclick={loadAll}>
				<RefreshCw class="mr-2 h-4 w-4" />
				Refresh
			</Button>
		</div>
	</div>

	<!-- Stats Cards -->
	{#if stats}
		<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
			<div class="rounded-lg border bg-card p-4">
				<div class="flex items-center gap-2">
					<Clock class="h-4 w-4 text-yellow-500" />
					<span class="text-sm text-muted-foreground">Pending</span>
				</div>
				<p class="mt-1 text-3xl font-bold">{stats.pending}</p>
			</div>
			<div class="rounded-lg border bg-card p-4">
				<div class="flex items-center gap-2">
					<Loader2 class="h-4 w-4 animate-spin text-blue-500" />
					<span class="text-sm text-muted-foreground">Processing</span>
				</div>
				<p class="mt-1 text-3xl font-bold">{stats.processing}</p>
			</div>
			<div class="rounded-lg border bg-card p-4">
				<div class="flex items-center gap-2">
					<CheckCircle class="h-4 w-4 text-green-500" />
					<span class="text-sm text-muted-foreground">Completed</span>
				</div>
				<p class="mt-1 text-3xl font-bold">{stats.completed}</p>
			</div>
			<div class="rounded-lg border bg-card p-4">
				<div class="flex items-center gap-2">
					<XCircle class="h-4 w-4 text-red-500" />
					<span class="text-sm text-muted-foreground">Failed</span>
				</div>
				<p class="mt-1 text-3xl font-bold">{stats.failed}</p>
			</div>
			<div class="rounded-lg border bg-card p-4">
				<div class="flex items-center gap-2">
					<Server class="h-4 w-4 text-purple-500" />
					<span class="text-sm text-muted-foreground">Workers</span>
				</div>
				<p class="mt-1 text-3xl font-bold">
					<span class="text-green-500">{stats.active_workers}</span>
					<span class="text-muted-foreground text-lg">/{stats.total_workers}</span>
				</p>
			</div>
			<div class="rounded-lg border bg-card p-4">
				<div class="flex items-center gap-2">
					<Activity class="h-4 w-4 text-cyan-500" />
					<span class="text-sm text-muted-foreground">Avg Time</span>
				</div>
				<p class="mt-1 text-3xl font-bold">
					{stats.avg_processing_time_ms ? `${(stats.avg_processing_time_ms / 1000).toFixed(1)}s` : '-'}
				</p>
			</div>
		</div>

		<!-- Additional Stats -->
		<div class="flex gap-4 text-sm text-muted-foreground">
			<span>Last hour: <strong class="text-foreground">{stats.jobs_last_hour}</strong> jobs</span>
			<span>Last 24h: <strong class="text-foreground">{stats.jobs_last_24_hours}</strong> jobs</span>
			<span>Completed today: <strong class="text-foreground">{stats.completed_today}</strong></span>
		</div>
	{/if}

	<!-- Filters -->
	<div class="flex items-center gap-4">
		<select
			bind:value={statusFilter}
			onchange={() => { currentPage = 0; loadJobs(); }}
			class="h-10 rounded-md border bg-background px-3 text-sm"
		>
			<option value="">All Statuses</option>
			<option value="pending">Pending</option>
			<option value="processing">Processing</option>
			<option value="completed">Completed</option>
			<option value="failed">Failed</option>
			<option value="cancelled">Cancelled</option>
			<option value="retrying">Retrying</option>
		</select>
	</div>

	<!-- Jobs Table -->
	<div class="rounded-lg border bg-card">
		{#if isLoading && jobs.length === 0}
			<div class="flex items-center justify-center py-12">
				<Loader2 class="h-8 w-8 animate-spin text-primary" />
			</div>
		{:else if jobs.length === 0}
			<div class="flex flex-col items-center justify-center py-12 text-center">
				<Activity class="h-12 w-12 text-muted-foreground/50" />
				<p class="mt-4 text-lg font-medium text-muted-foreground">No jobs found</p>
				<p class="text-sm text-muted-foreground">Jobs will appear here when MRs are queued for analysis</p>
			</div>
		{:else}
			<table class="w-full">
				<thead>
					<tr class="border-b text-left text-sm text-muted-foreground">
						<th class="px-4 py-3 font-medium">MR</th>
						<th class="px-4 py-3 font-medium">Status</th>
						<th class="px-4 py-3 font-medium">Priority</th>
						<th class="px-4 py-3 font-medium">Worker</th>
						<th class="px-4 py-3 font-medium">Retries</th>
						<th class="px-4 py-3 font-medium">Created</th>
						<th class="px-4 py-3 font-medium">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each jobs as job}
						<tr class="border-b last:border-0 hover:bg-muted/50">
							<td class="px-4 py-3">
								<div class="font-medium">!{job.mr_iid}</div>
								<div class="text-sm text-muted-foreground truncate max-w-[200px]" title={job.mr_title}>
									{job.mr_title}
								</div>
							</td>
							<td class="px-4 py-3">
								<div class="flex items-center gap-2">
									<svelte:component
										this={getStatusIcon(job.status)}
										class={cn(
											'h-4 w-4',
											getStatusColor(job.status),
											job.status === 'processing' && 'animate-spin'
										)}
									/>
									<span class="text-sm capitalize">{job.status}</span>
								</div>
								{#if job.last_error}
									<p class="mt-1 text-xs text-red-500 truncate max-w-[200px]" title={job.last_error}>
										{job.last_error}
									</p>
								{/if}
							</td>
							<td class="px-4 py-3">
								<span class={cn(
									'text-sm capitalize',
									job.priority === 'urgent' && 'text-red-500 font-medium',
									job.priority === 'high' && 'text-orange-500',
									job.priority === 'normal' && 'text-foreground',
									job.priority === 'low' && 'text-muted-foreground'
								)}>
									{job.priority}
								</span>
							</td>
							<td class="px-4 py-3">
								<span class="text-sm font-mono text-muted-foreground">
									{job.worker_id || '-'}
								</span>
							</td>
							<td class="px-4 py-3">
								<span class="text-sm">
									{job.retry_count}/{job.max_retries}
								</span>
								{#if job.next_retry_at}
									<p class="text-xs text-muted-foreground">
										Next: {formatRelativeTime(job.next_retry_at)}
									</p>
								{/if}
							</td>
							<td class="px-4 py-3">
								<span class="text-sm text-muted-foreground">
									{formatRelativeTime(job.created_at)}
								</span>
								{#if job.started_at}
									<p class="text-xs text-muted-foreground">
										Started: {formatRelativeTime(job.started_at)}
									</p>
								{/if}
							</td>
							<td class="px-4 py-3">
								<div class="flex items-center gap-2">
									{#if job.status === 'failed' || job.status === 'cancelled'}
										<button
											onclick={() => handleRetryJob(job)}
											class="rounded p-1.5 text-primary hover:bg-primary/10"
											title="Retry"
										>
											<RotateCcw class="h-4 w-4" />
										</button>
									{/if}
									{#if job.status === 'pending' || job.status === 'processing'}
										<button
											onclick={() => handleCancelJob(job)}
											class="rounded p-1.5 text-red-500 hover:bg-red-500/10"
											title="Cancel"
										>
											<X class="h-4 w-4" />
										</button>
									{/if}
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>

			<!-- Pagination -->
			{#if totalPages > 1}
				<div class="flex items-center justify-between border-t px-4 py-3">
					<p class="text-sm text-muted-foreground">
						Showing {currentPage * pageSize + 1} to {Math.min((currentPage + 1) * pageSize, totalJobs)} of {totalJobs}
					</p>
					<div class="flex items-center gap-2">
						<Button
							variant="outline"
							size="sm"
							disabled={currentPage === 0}
							onclick={() => { currentPage--; loadJobs(); }}
						>
							Previous
						</Button>
						<Button
							variant="outline"
							size="sm"
							disabled={currentPage >= totalPages - 1}
							onclick={() => { currentPage++; loadJobs(); }}
						>
							Next
						</Button>
					</div>
				</div>
			{/if}
		{/if}
	</div>

	<!-- Real-time Events -->
	{#if wsConnected && recentEvents.length > 0}
		<div class="rounded-lg border bg-card p-4">
			<h3 class="text-sm font-medium text-muted-foreground mb-3 flex items-center gap-2">
				<Wifi class="h-4 w-4 text-green-500" />
				Live Events
			</h3>
			<div class="space-y-2 max-h-48 overflow-y-auto">
				{#each recentEvents as event}
					<div class="flex items-center gap-3 text-sm py-1 border-b border-muted last:border-0">
						<span class={cn(
							'w-2 h-2 rounded-full',
							event.type === 'job_completed' && 'bg-green-500',
							event.type === 'job_failed' && 'bg-red-500',
							event.type === 'job_started' && 'bg-blue-500',
							event.type === 'job_created' && 'bg-yellow-500'
						)}></span>
						<span class="capitalize">{event.type.replace('_', ' ')}</span>
						{#if event.data?.mr_iid}
							<span class="text-muted-foreground">MR !{event.data.mr_iid}</span>
						{/if}
						<span class="text-xs text-muted-foreground ml-auto">
							{event.time.toLocaleTimeString()}
						</span>
					</div>
				{/each}
			</div>
		</div>
	{/if}
</div>

