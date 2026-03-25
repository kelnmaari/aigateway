<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import { faFileLines, faArrowsRotate, faSpinner, faCircleExclamation, faCircleInfo, faTriangleExclamation, faBug, faDownload, faPlay, faSquare, faTowerBroadcast } from '@fortawesome/free-solid-svg-icons';
	import { adminApi, type LogEntry, type AuditEntry } from '$lib/api/admin';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	// State
	let activeTab = $state<'logs' | 'audit'>('logs');
	let logs = $state<LogEntry[]>([]);
	let auditLogs = $state<AuditEntry[]>([]);
	let isLoading = $state(true);
	let logFiles = $state<Array<{ name: string; size: number; modified: string; is_current: boolean }>>([]);
	let selectedFile = $state('');
	let selectedLevel = $state('');
	let logsError = $state('');

	// Real-time SSE
	let eventSource: EventSource | null = null;
	let isRealtime = $state(false);
	let logsContainer = $state<HTMLDivElement | null>(null);

	// Filter levels
	const levels = ['', 'debug', 'info', 'warn', 'error'] as const;

	// Filtered logs
	let filteredLogs = $derived(
		selectedLevel ? logs.filter((log) => log.level.toLowerCase() === selectedLevel) : logs
	);

	onMount(async () => {
		await loadLogFiles();
	});

	onDestroy(() => {
		stopRealtime();
	});

	async function loadLogFiles() {
		isLoading = true;
		try {
			const response = await adminApi.getLogFiles();
			logFiles = response.files || [];
			// Select current log or first file
			const currentLog = logFiles.find((f) => f.is_current);
			if (currentLog) {
				selectedFile = currentLog.name;
			} else if (logFiles.length > 0) {
				selectedFile = logFiles[0].name;
			}
			if (selectedFile) {
				await loadLogs();
			} else {
				isLoading = false;
				logsError = 'No log files found. File-based logging may not be configured.';
			}
		} catch (error) {
			logFiles = [];
			isLoading = false;
			logsError = 'Failed to load log files. File-based logging may not be configured.';
			console.error('Failed to load log files:', error);
		}
	}

	async function loadLogs() {
		if (!selectedFile) {
			logs = [];
			return;
		}
		isLoading = true;
		logsError = '';
		stopRealtime();
		try {
			const response = await adminApi.getLogs(selectedFile);
			logs = response.logs || [];
			// Auto-scroll to bottom
			setTimeout(() => scrollToBottom(), 100);
		} catch (error) {
			console.error('Failed to load logs:', error);
			logs = [];
			logsError = 'Failed to load logs from file';
		} finally {
			isLoading = false;
		}
	}

	async function loadAuditLogs() {
		isLoading = true;
		try {
			const response = await adminApi.getAuditLogs();
			auditLogs = response.events || [];
		} catch (error) {
			console.error('Failed to load audit logs:', error);
			auditLogs = [];
		} finally {
			isLoading = false;
		}
	}

	function switchTab(tab: 'logs' | 'audit') {
		activeTab = tab;
		if (tab === 'audit' && auditLogs.length === 0) {
			loadAuditLogs();
		}
		if (tab === 'logs') {
			stopRealtime();
		}
	}

	// Real-time SSE
	function toggleRealtime() {
		if (isRealtime) {
			stopRealtime();
		} else {
			startRealtime();
		}
	}

	function startRealtime() {
		if (!selectedFile) {
			alert(m.alert_select_log_file());
			return;
		}

		if (eventSource) {
			eventSource.close();
		}

		const token = localStorage.getItem('access_token');
		if (!token) {
			alert(m.alert_auth_required());
			return;
		}

		const url = `/api/admin/logs/stream?token=${encodeURIComponent(token)}&file=${encodeURIComponent(selectedFile)}`;
		eventSource = new EventSource(url);

		eventSource.addEventListener('log', (e) => {
			const entry = JSON.parse(e.data) as LogEntry;
			logs = [...logs, entry];
			// Keep only last 1000 entries
			if (logs.length > 1000) {
				logs = logs.slice(-1000);
			}
			setTimeout(() => scrollToBottom(), 50);
		});

		eventSource.addEventListener('error', () => {
			console.error('SSE error');
			stopRealtime();
		});

		isRealtime = true;
	}

	function stopRealtime() {
		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}
		isRealtime = false;
	}

	function scrollToBottom() {
		if (logsContainer) {
			logsContainer.scrollTop = logsContainer.scrollHeight;
		}
	}

	function downloadLog() {
		if (selectedFile) {
			window.open(`/api/admin/logs/${selectedFile}/download`, '_blank');
		}
	}

	function getLevelIcon(level: string) {
		switch (level.toLowerCase()) {
			case 'error':
			case 'fatal':
				return faCircleExclamation;
			case 'warn':
			case 'warning':
				return faTriangleExclamation;
			case 'debug':
			case 'trace':
				return faBug;
			default:
				return faCircleInfo;
		}
	}

	function getLevelClass(level: string) {
		switch (level.toLowerCase()) {
			case 'error':
			case 'fatal':
				return 'text-red-500 bg-red-500/10';
			case 'warn':
			case 'warning':
				return 'text-amber-500 bg-amber-500/10';
			case 'debug':
			case 'trace':
				return 'text-purple-500 bg-purple-500/10';
			default:
				return 'text-blue-500 bg-blue-500/10';
		}
	}

	function formatTimestamp(ts: string): string {
		try {
			const date = new Date(ts);
			const hours = String(date.getHours()).padStart(2, '0');
			const minutes = String(date.getMinutes()).padStart(2, '0');
			const seconds = String(date.getSeconds()).padStart(2, '0');
			const ms = String(date.getMilliseconds()).padStart(3, '0');
			return `${hours}:${minutes}:${seconds}.${ms}`;
		} catch {
			return ts;
		}
	}

	// Parse and format log message - split by | and highlight key=value
	function formatMessage(message: string): { main: string; context: Array<{ key: string; value: string }> } {
		if (!message.includes(' | ')) {
			// Try to parse key=value pairs from the whole message
			const fields = parseKeyValuePairs(message);
			if (fields.length > 0) {
				return { main: '', context: fields };
			}
			return { main: message, context: [] };
		}

		const parts = message.split(' | ');
		const mainMsg = parts[0];
		const contextStr = parts.slice(1).join(' | ');
		const context = parseKeyValuePairs(contextStr);

		return { main: mainMsg, context };
	}

	function parseKeyValuePairs(text: string): Array<{ key: string; value: string }> {
		const result: Array<{ key: string; value: string }> = [];
		const regex = /(\w+)=([^\s]+)/g;
		let match;
		while ((match = regex.exec(text)) !== null) {
			result.push({ key: match[1], value: match[2] });
		}
		return result;
	}

	function getLevelLabel(level: string): string {
		switch (level) {
			case '':
				return 'All';
			case 'debug':
				return 'Debug';
			case 'info':
				return 'Info';
			case 'warn':
				return 'Warn';
			case 'error':
				return 'Error';
			default:
				return level;
		}
	}
</script>

<div class="space-y-4">
	<!-- Tabs -->
	<div class="flex items-center justify-between">
		<div class="flex gap-2 border-b border-border">
			<button
				onclick={() => switchTab('logs')}
				class={cn(
					'border-b-2 px-4 py-2 text-sm font-medium transition-colors',
					activeTab === 'logs'
						? 'border-primary text-primary'
						: 'border-transparent text-muted-foreground hover:text-foreground'
				)}
			>
				{m.admin_logs_app()}
			</button>
			<button
				onclick={() => switchTab('audit')}
				class={cn(
					'border-b-2 px-4 py-2 text-sm font-medium transition-colors',
					activeTab === 'audit'
						? 'border-primary text-primary'
						: 'border-transparent text-muted-foreground hover:text-foreground'
				)}
			>
				{m.admin_audit()}
			</button>
		</div>

		{#if activeTab === 'logs'}
			<div class="flex items-center gap-2">
				<!-- File selector -->
				<select
					bind:value={selectedFile}
					onchange={loadLogs}
					class="rounded-lg border border-input bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
				>
					{#each logFiles as file}
						<option value={file.name}>
							{file.name}
							{file.is_current ? ` (${m.admin_logs_current()})` : ''}
						</option>
					{/each}
				</select>

				<!-- Real-time toggle -->
				<Button
					variant={isRealtime ? 'destructive' : 'outline'}
					size="sm"
					onclick={toggleRealtime}
					class="gap-1.5"
					title={m.tooltip_live()}
				>
					{#if isRealtime}
						<FontAwesomeIcon icon={faTowerBroadcast} class="h-3.5 w-3.5 animate-pulse" />
						<FontAwesomeIcon icon={faSquare} class="h-3.5 w-3.5" />
						{m.admin_logs_stop()}
					{:else}
						<FontAwesomeIcon icon={faPlay} class="h-3.5 w-3.5" />
						{m.admin_logs_realtime()}
					{/if}
				</Button>

				<!-- Download -->
				<Button variant="outline" size="sm" onclick={downloadLog} disabled={!selectedFile} title={m.admin_logs_download()}>
					<FontAwesomeIcon icon={faDownload} class="h-4 w-4" />
				</Button>

				<!-- Refresh -->
				<Button variant="outline" size="sm" onclick={loadLogs} disabled={isLoading || isRealtime} title={m.common_refresh()}>
					<FontAwesomeIcon icon={faArrowsRotate} class={cn('h-4 w-4', isLoading && 'animate-spin')} />
				</Button>
			</div>
		{:else}
			<Button variant="outline" size="sm" onclick={loadAuditLogs} disabled={isLoading}>
				<FontAwesomeIcon icon={faArrowsRotate} class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
				{m.common_refresh()}
			</Button>
		{/if}
	</div>

	<!-- Level filter buttons (for logs tab) -->
	{#if activeTab === 'logs' && !logsError}
		<div class="flex items-center gap-1">
			{#each levels as level}
				<button
					onclick={() => (selectedLevel = level)}
					class={cn(
						'rounded-md px-3 py-1 text-xs font-medium transition-colors',
						selectedLevel === level
							? level === ''
								? 'bg-primary text-primary-foreground'
								: level === 'error'
									? 'bg-red-500 text-white'
									: level === 'warn'
										? 'bg-amber-500 text-white'
										: level === 'info'
											? 'bg-blue-500 text-white'
											: 'bg-purple-500 text-white'
							: 'bg-muted text-muted-foreground hover:bg-muted/80'
					)}
				>
					{getLevelLabel(level)}
				</button>
			{/each}
			<span class="ml-2 text-xs text-muted-foreground">
				{filteredLogs.length} / {logs.length} entries
			</span>
		</div>
	{/if}

	<!-- Content -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<FontAwesomeIcon icon={faSpinner} class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if activeTab === 'logs'}
		{#if logsError}
			<div class="rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-8 text-center">
				<FontAwesomeIcon icon={faTriangleExclamation} class="mx-auto h-10 w-10 text-amber-500" />
				<p class="mt-3 font-medium text-amber-600 dark:text-amber-400">{logsError}</p>
				<p class="mt-1 text-sm text-muted-foreground">
					Application logs are available via stdout/stderr or your log aggregation system.
				</p>
			</div>
		{:else if filteredLogs.length === 0}
			<div class="rounded-lg border border-dashed border-border py-16 text-center">
				<FontAwesomeIcon icon={faFileLines} class="mx-auto h-12 w-12 text-muted-foreground/40" />
				<p class="mt-4 text-muted-foreground">
					{logs.length === 0 ? 'No logs found in selected file' : 'No logs match selected filter'}
				</p>
			</div>
		{:else}
			<div
				bind:this={logsContainer}
				class="max-h-[calc(100vh-320px)] space-y-0.5 overflow-y-auto rounded-lg border border-border bg-card p-2 font-mono text-xs"
			>
				{#each filteredLogs as log, i (i)}
					{@const LevelIcon = getLevelIcon(log.level)}
					{@const parsed = formatMessage(log.message)}
					<div
						class={cn(
							'flex items-start gap-2 rounded px-2 py-1.5 hover:bg-muted/50',
							log.level.toLowerCase() === 'error' && 'bg-red-500/5'
						)}
					>
						<!-- Timestamp -->
						<span class="shrink-0 text-muted-foreground">{formatTimestamp(log.timestamp)}</span>

						<!-- Level badge -->
						<span
							class={cn('shrink-0 rounded px-1.5 py-0.5 text-[10px] font-semibold uppercase', getLevelClass(log.level))}
						>
							{log.level}
						</span>

						<!-- Message -->
						<div class="min-w-0 flex-1">
							{#if parsed.main}
								<span class="text-foreground">{parsed.main}</span>
							{/if}
							{#if parsed.context.length > 0}
								<span class="ml-1 text-muted-foreground">
									{#each parsed.context as field, idx}
										<span class="text-blue-400">{field.key}</span>=<span class="text-orange-400"
											>{field.value}</span
										>{idx < parsed.context.length - 1 ? ' ' : ''}
									{/each}
								</span>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	{:else}
		<!-- Audit Logs -->
		{#if auditLogs.length === 0}
			<div class="rounded-lg border border-dashed border-border py-16 text-center">
				<FontAwesomeIcon icon={faFileLines} class="mx-auto h-12 w-12 text-muted-foreground/40" />
				<p class="mt-4 text-muted-foreground">No audit events found</p>
			</div>
		{:else}
			<div class="overflow-hidden rounded-lg border border-border">
				<table class="w-full">
					<thead class="border-b border-border bg-muted/50">
						<tr>
							<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
								Time
							</th>
							<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
								User
							</th>
							<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
								Action
							</th>
							<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
								Resource
							</th>
							<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
								{m.common_status()}
							</th>
							<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
								IP
							</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border">
						{#each auditLogs as event (event.id)}
							<tr class="hover:bg-muted/30">
								<td class="px-4 py-3 text-sm text-muted-foreground">
									{formatRelativeTime(event.created_at)}
								</td>
								<td class="px-4 py-3 text-sm">
									{event.username || event.user_id || 'System'}
								</td>
								<td class="px-4 py-3">
									<code class="rounded bg-muted px-1.5 py-0.5 text-xs">{event.action}</code>
								</td>
								<td class="px-4 py-3 text-sm">
									{event.resource}
									{#if event.resource_id}
										<span class="text-muted-foreground">#{event.resource_id.slice(0, 8)}</span>
									{/if}
								</td>
								<td class="px-4 py-3">
									<span
										class={cn(
											'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
											event.status === 'success'
												? 'bg-green-500/10 text-green-500'
												: 'bg-red-500/10 text-red-500'
										)}
									>
										{event.status}
									</span>
								</td>
								<td class="px-4 py-3 font-mono text-xs text-muted-foreground">
									{event.ip_address}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
</div>
