<script lang="ts">
	import { onMount } from 'svelte';
	import { FileText, RefreshCw, Loader2, AlertCircle, Info, AlertTriangle, Bug } from 'lucide-svelte';
	import { adminApi, type LogEntry, type AuditEntry } from '$lib/api/admin';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	// State
	let activeTab = $state<'logs' | 'audit'>('logs');
	let logs = $state<LogEntry[]>([]);
	let auditLogs = $state<AuditEntry[]>([]);
	let isLoading = $state(true);
	let logFiles = $state<string[]>([]);
	let selectedFile = $state('');
	let selectedLevel = $state('');
	let logsError = $state('');

	onMount(async () => {
		await loadLogFiles();
	});

	async function loadLogFiles() {
		isLoading = true;
		try {
			const response = await adminApi.getLogFiles();
			const files = response.files?.map(f => f.name) || [];
			logFiles = files;
			// Select first available file
			if (files.length > 0 && !selectedFile) {
				selectedFile = files[0];
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
		try {
			const response = await adminApi.getLogs(selectedFile);
			logs = response.logs || [];
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
	}

	function getLevelIcon(level: string) {
		switch (level.toLowerCase()) {
			case 'error':
			case 'fatal':
				return AlertCircle;
			case 'warn':
			case 'warning':
				return AlertTriangle;
			case 'debug':
			case 'trace':
				return Bug;
			default:
				return Info;
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
			return new Date(ts).toLocaleString();
		} catch {
			return ts;
		}
	}
</script>

<div class="space-y-6">
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
				Application Logs
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
			<div class="flex gap-3">
				<select
					bind:value={selectedFile}
					onchange={loadLogs}
					class="rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
				>
					{#each logFiles as file}
						<option value={file}>{file}</option>
					{/each}
				</select>
				<select
					bind:value={selectedLevel}
					onchange={loadLogs}
					class="rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
				>
					<option value="">All Levels</option>
					<option value="debug">Debug</option>
					<option value="info">Info</option>
					<option value="warn">Warning</option>
					<option value="error">Error</option>
				</select>
				<Button variant="outline" onclick={loadLogs} disabled={isLoading}>
					<RefreshCw class={cn('h-4 w-4', isLoading && 'animate-spin')} />
				</Button>
			</div>
		{:else}
			<Button variant="outline" onclick={loadAuditLogs} disabled={isLoading}>
				<RefreshCw class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
				{m.common_refresh()}
			</Button>
		{/if}
	</div>

	<!-- Content -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if activeTab === 'logs'}
		{#if logsError}
			<div class="rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-8 text-center">
				<AlertTriangle class="mx-auto h-10 w-10 text-amber-500" />
				<p class="mt-3 font-medium text-amber-600 dark:text-amber-400">{logsError}</p>
				<p class="mt-1 text-sm text-muted-foreground">
					Application logs are available via stdout/stderr or your log aggregation system.
				</p>
			</div>
		{:else if logs.length === 0}
			<div class="rounded-lg border border-dashed border-border py-16 text-center">
				<FileText class="mx-auto h-12 w-12 text-muted-foreground/40" />
				<p class="mt-4 text-muted-foreground">No logs found in selected file</p>
			</div>
		{:else}
			<div class="space-y-1 rounded-lg border border-border bg-card p-2">
				{#each logs as log, i (i)}
					{@const LevelIcon = getLevelIcon(log.level)}
					<div class="flex items-start gap-3 rounded px-3 py-2 hover:bg-muted/50">
						<span
							class={cn(
								'mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded',
								getLevelClass(log.level)
							)}
						>
							<LevelIcon class="h-3 w-3" />
						</span>
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-2">
								<span class="text-xs text-muted-foreground">
									{formatTimestamp(log.timestamp)}
								</span>
								<span
									class={cn(
										'rounded px-1 py-0.5 text-xs font-medium uppercase',
										getLevelClass(log.level)
									)}
								>
									{log.level}
								</span>
							</div>
							<p class="mt-0.5 break-words text-sm">{log.message}</p>
							{#if log.fields && Object.keys(log.fields).length > 0}
								<pre class="mt-1 overflow-x-auto rounded bg-muted p-2 text-xs">{JSON.stringify(log.fields, null, 2)}</pre>
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
				<FileText class="mx-auto h-12 w-12 text-muted-foreground/40" />
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

