<script lang="ts">
	import { onMount } from 'svelte';
	import {
		History,
		RefreshCw,
		Loader2,
		Search,
		Eye,
		CheckCircle,
		XCircle,
		AlertCircle,
		AlertTriangle,
		Clock,
		Filter,
		ChevronLeft,
		ChevronRight,
		FileCode,
		Shield,
		Bug,
		BookOpen,
		TestTube,
		LayoutGrid,
		Package,
		Activity
	} from 'lucide-svelte';
	import { gitlabApi, type ScanHistoryItem, type ScanTypeInfo } from '$lib/api/gitlab';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import GitLabNav from '$lib/components/gitlab-nav.svelte';
	import * as m from '$lib/paraglide/messages';

	let results = $state<ScanHistoryItem[]>([]);
	let scanTypes = $state<ScanTypeInfo[]>([]);
	let total = $state(0);
	let isLoading = $state(true);

	// Filters
	let scanTypeFilter = $state('');
	let statusFilter = $state('');
	let projectFilter = $state('');

	// Pagination
	let currentPage = $state(0);
	let pageSize = 20;

	// Detail modal
	let showDetailModal = $state(false);
	let selectedResult = $state<ScanHistoryItem | null>(null);
	let isLoadingDetail = $state(false);

	onMount(async () => {
		await Promise.all([loadResults(), loadScanTypes()]);
	});

	async function loadResults() {
		isLoading = true;
		try {
			const response = await gitlabApi.listScanHistory({
				scan_type: scanTypeFilter || undefined,
				status: statusFilter || undefined,
				project_id: projectFilter || undefined,
				limit: pageSize,
				offset: currentPage * pageSize,
			});
			results = response.results || [];
			total = response.total || 0;
		} catch (error) {
			console.error('Failed to load scan history:', error);
			results = [];
			total = 0;
		} finally {
			isLoading = false;
		}
	}

	async function loadScanTypes() {
		try {
			const response = await gitlabApi.getScanTypes();
			scanTypes = response.types || [];
		} catch (error) {
			console.error('Failed to load scan types:', error);
		}
	}

	async function viewDetails(item: ScanHistoryItem) {
		selectedResult = item;
		showDetailModal = true;
		
		if (!item.results_json) {
			isLoadingDetail = true;
			try {
				const fullResult = await gitlabApi.getScanHistoryItem(item.id);
				selectedResult = fullResult;
			} catch (error) {
				console.error('Failed to load result details:', error);
			} finally {
				isLoadingDetail = false;
			}
		}
	}

	function getScanTypeIcon(type: string) {
		switch (type) {
			case 'secrets':
			case 'secrets_deep':
				return Shield;
			case 'dependencies':
				return Package;
			case 'quality':
				return FileCode;
			case 'deadcode':
				return Bug;
			case 'autodocs':
				return BookOpen;
			case 'testgen':
				return TestTube;
			case 'architecture':
				return LayoutGrid;
			default:
				return History;
		}
	}

	function getScanTypeLabel(type: string): string {
		const found = scanTypes.find(t => t.value === type);
		if (found) return found.label;
		
		const labels: Record<string, string> = {
			secrets: 'Secrets Scan',
			secrets_deep: 'Deep Secrets Scan',
			dependencies: 'Dependencies',
			quality: 'Code Quality',
			deadcode: 'Dead Code',
			autodocs: 'Auto-Docs',
			testgen: 'Test Gen',
			architecture: 'Architecture',
		};
		return labels[type] || type;
	}

	function getStatusColor(status: string): string {
		switch (status) {
			case 'completed': return 'text-green-500';
			case 'failed': return 'text-red-500';
			case 'running': return 'text-blue-500';
			case 'pending': return 'text-yellow-500';
			case 'cancelled': return 'text-gray-500';
			default: return 'text-muted-foreground';
		}
	}

	function getStatusIcon(status: string) {
		switch (status) {
			case 'completed': return CheckCircle;
			case 'failed': return XCircle;
			case 'running': return Loader2;
			case 'pending': return Clock;
			default: return AlertCircle;
		}
	}

	function formatDuration(ms: number): string {
		if (ms < 1000) return `${ms}ms`;
		if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
		return `${(ms / 60000).toFixed(1)}m`;
	}

	function clearFilters() {
		scanTypeFilter = '';
		statusFilter = '';
		projectFilter = '';
		currentPage = 0;
		loadResults();
	}

	function handlePageChange(direction: 'prev' | 'next') {
		if (direction === 'prev' && currentPage > 0) {
			currentPage--;
		} else if (direction === 'next' && (currentPage + 1) * pageSize < total) {
			currentPage++;
		}
		loadResults();
	}

	$effect(() => {
		// Reload when filters change
		if (scanTypeFilter !== undefined || statusFilter !== undefined) {
			currentPage = 0;
		}
	});

	const totalPages = $derived(Math.ceil(total / pageSize));
	const hasFilters = $derived(scanTypeFilter || statusFilter || projectFilter);
</script>

<div class="container mx-auto py-6">
	<GitLabNav />

	<div class="mb-6">
		<h1 class="text-2xl font-bold flex items-center gap-2">
			<History class="h-6 w-6" />
			{m.admin_gitlab_scan_history?.() || 'Scan History'}
		</h1>
		<p class="text-muted-foreground mt-1">
			View all scan results with historical data
		</p>
	</div>

	<!-- Filters -->
	<div class="bg-card rounded-lg border p-4 mb-6">
		<div class="flex flex-wrap items-center gap-4">
			<div class="flex items-center gap-2">
				<Filter class="h-4 w-4 text-muted-foreground" />
				<span class="text-sm font-medium">Filters:</span>
			</div>

			<select
				bind:value={scanTypeFilter}
				onchange={() => loadResults()}
				class="px-3 py-2 rounded-md border bg-background text-sm"
			>
				<option value="">All Scan Types</option>
				{#each scanTypes as type}
					<option value={type.value}>{type.label}</option>
				{/each}
			</select>

			<select
				bind:value={statusFilter}
				onchange={() => loadResults()}
				class="px-3 py-2 rounded-md border bg-background text-sm"
			>
				<option value="">All Statuses</option>
				<option value="completed">Completed</option>
				<option value="failed">Failed</option>
				<option value="running">Running</option>
				<option value="pending">Pending</option>
			</select>

			{#if hasFilters}
				<Button variant="ghost" size="sm" onclick={clearFilters}>
					Clear Filters
				</Button>
			{/if}

			<div class="ml-auto">
				<Button variant="outline" size="sm" onclick={loadResults} disabled={isLoading}>
					{#if isLoading}
						<Loader2 class="h-4 w-4 mr-2 animate-spin" />
					{:else}
						<RefreshCw class="h-4 w-4 mr-2" />
					{/if}
					Refresh
				</Button>
			</div>
		</div>
	</div>

	<!-- Results Table -->
	<div class="bg-card rounded-lg border overflow-hidden">
		{#if isLoading && results.length === 0}
			<div class="p-12 text-center">
				<Loader2 class="h-8 w-8 animate-spin mx-auto text-muted-foreground" />
				<p class="mt-2 text-muted-foreground">Loading scan history...</p>
			</div>
		{:else if results.length === 0}
			<div class="p-12 text-center">
				<History class="h-12 w-12 mx-auto text-muted-foreground mb-4" />
				<h3 class="text-lg font-medium">No scan results found</h3>
				<p class="text-muted-foreground mt-1">
					{hasFilters ? 'Try adjusting your filters' : 'Run some scans to see results here'}
				</p>
			</div>
		{:else}
			<table class="w-full">
				<thead class="bg-muted/50">
					<tr class="text-left text-sm">
						<th class="px-4 py-3 font-medium">Scan Type</th>
						<th class="px-4 py-3 font-medium">Project</th>
						<th class="px-4 py-3 font-medium">Status</th>
						<th class="px-4 py-3 font-medium">Findings</th>
						<th class="px-4 py-3 font-medium">Duration</th>
						<th class="px-4 py-3 font-medium">Started</th>
						<th class="px-4 py-3 font-medium">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y">
					{#each results as result (result.id)}
						{@const TypeIcon = getScanTypeIcon(result.scan_type)}
						{@const StatusIcon = getStatusIcon(result.status)}
						<tr class="hover:bg-muted/50 transition-colors">
							<td class="px-4 py-3">
								<div class="flex items-center gap-2">
									<TypeIcon class="h-4 w-4 text-muted-foreground" />
									<span class="font-medium">{getScanTypeLabel(result.scan_type)}</span>
								</div>
							</td>
							<td class="px-4 py-3">
								<span class="text-sm">{result.project_name || result.project_id}</span>
							</td>
							<td class="px-4 py-3">
								<div class="flex items-center gap-2">
									<StatusIcon class={cn("h-4 w-4", getStatusColor(result.status), result.status === 'running' && 'animate-spin')} />
									<span class={cn("text-sm capitalize", getStatusColor(result.status))}>
										{result.status}
									</span>
								</div>
							</td>
							<td class="px-4 py-3">
								<span class={cn(
									"text-sm font-medium",
									result.findings_count > 0 ? "text-orange-500" : "text-green-500"
								)}>
									{result.findings_count}
								</span>
								{#if result.files_affected > 0}
									<span class="text-xs text-muted-foreground ml-1">
										({result.files_affected} files)
									</span>
								{/if}
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{formatDuration(result.duration_ms)}
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{formatRelativeTime(result.started_at)}
							</td>
							<td class="px-4 py-3">
								<Button variant="ghost" size="sm" onclick={() => viewDetails(result)}>
									<Eye class="h-4 w-4 mr-1" />
									View
								</Button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>

			<!-- Pagination -->
			{#if total > pageSize}
				<div class="flex items-center justify-between px-4 py-3 border-t bg-muted/30">
					<span class="text-sm text-muted-foreground">
						Showing {currentPage * pageSize + 1}-{Math.min((currentPage + 1) * pageSize, total)} of {total}
					</span>
					<div class="flex items-center gap-2">
						<Button
							variant="outline"
							size="sm"
							onclick={() => handlePageChange('prev')}
							disabled={currentPage === 0}
						>
							<ChevronLeft class="h-4 w-4" />
							Previous
						</Button>
						<span class="text-sm">
							Page {currentPage + 1} of {totalPages}
						</span>
						<Button
							variant="outline"
							size="sm"
							onclick={() => handlePageChange('next')}
							disabled={(currentPage + 1) * pageSize >= total}
						>
							Next
							<ChevronRight class="h-4 w-4" />
						</Button>
					</div>
				</div>
			{/if}
		{/if}
	</div>
</div>

<!-- Detail Modal -->
{#if showDetailModal && selectedResult}
	{@const ModalTypeIcon = getScanTypeIcon(selectedResult.scan_type)}
	<div class="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
		<div class="bg-card border rounded-lg shadow-lg max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col">
			<div class="flex items-center justify-between p-4 border-b">
				<div class="flex items-center gap-3">
					<ModalTypeIcon class="h-5 w-5" />
					<div>
						<h2 class="font-semibold">{getScanTypeLabel(selectedResult.scan_type)} Results</h2>
						<p class="text-sm text-muted-foreground">
							{selectedResult.project_name || selectedResult.project_id}
						</p>
					</div>
				</div>
				<Button variant="ghost" size="sm" onclick={() => showDetailModal = false}>×</Button>
			</div>

			<div class="flex-1 overflow-auto p-4">
				{#if isLoadingDetail}
					<div class="flex items-center justify-center py-8">
						<Loader2 class="h-6 w-6 animate-spin" />
					</div>
				{:else}
					<!-- Summary -->
					<div class="grid grid-cols-4 gap-4 mb-6">
						<div class="bg-muted/50 rounded-lg p-3">
							<div class="text-sm text-muted-foreground">Status</div>
							<div class={cn("text-lg font-semibold capitalize", getStatusColor(selectedResult.status))}>
								{selectedResult.status}
							</div>
						</div>
						<div class="bg-muted/50 rounded-lg p-3">
							<div class="text-sm text-muted-foreground">Findings</div>
							<div class="text-lg font-semibold">{selectedResult.findings_count}</div>
						</div>
						<div class="bg-muted/50 rounded-lg p-3">
							<div class="text-sm text-muted-foreground">Duration</div>
							<div class="text-lg font-semibold">{formatDuration(selectedResult.duration_ms)}</div>
						</div>
						<div class="bg-muted/50 rounded-lg p-3">
							<div class="text-sm text-muted-foreground">Tokens Used</div>
							<div class="text-lg font-semibold">{selectedResult.tokens_used.toLocaleString()}</div>
						</div>
					</div>

					<!-- Error if any -->
					{#if selectedResult.error}
						<div class="bg-red-500/10 border border-red-500/20 rounded-lg p-4 mb-6">
							<div class="font-medium text-red-500 mb-1">Error</div>
							<pre class="text-sm text-red-400 whitespace-pre-wrap">{selectedResult.error}</pre>
						</div>
					{/if}

					<!-- Formatted Results -->
					{#if selectedResult.results_json}
						{@const parsedResults = JSON.parse(selectedResult.results_json)}
						
						<!-- Summary Section -->
						{#if parsedResults.summary}
							<div class="mb-6">
								<h3 class="font-medium mb-3 flex items-center gap-2">
									<Activity class="h-4 w-4" />
									Summary
								</h3>
								<div class="grid grid-cols-2 gap-4">
									<!-- By Severity -->
									{#if parsedResults.summary.by_severity}
										<div class="bg-muted/30 rounded-lg p-4">
											<div class="text-sm text-muted-foreground mb-2">By Severity</div>
											<div class="space-y-2">
												{#if parsedResults.summary.by_severity.critical > 0}
													<div class="flex items-center justify-between">
														<span class="text-sm flex items-center gap-2">
															<span class="w-2 h-2 rounded-full bg-red-500"></span>
															Critical
														</span>
														<span class="font-semibold text-red-500">{parsedResults.summary.by_severity.critical}</span>
													</div>
												{/if}
												{#if parsedResults.summary.by_severity.high > 0}
													<div class="flex items-center justify-between">
														<span class="text-sm flex items-center gap-2">
															<span class="w-2 h-2 rounded-full bg-orange-500"></span>
															High
														</span>
														<span class="font-semibold text-orange-500">{parsedResults.summary.by_severity.high}</span>
													</div>
												{/if}
												{#if parsedResults.summary.by_severity.medium > 0}
													<div class="flex items-center justify-between">
														<span class="text-sm flex items-center gap-2">
															<span class="w-2 h-2 rounded-full bg-yellow-500"></span>
															Medium
														</span>
														<span class="font-semibold text-yellow-500">{parsedResults.summary.by_severity.medium}</span>
													</div>
												{/if}
												{#if parsedResults.summary.by_severity.low > 0}
													<div class="flex items-center justify-between">
														<span class="text-sm flex items-center gap-2">
															<span class="w-2 h-2 rounded-full bg-blue-500"></span>
															Low
														</span>
														<span class="font-semibold text-blue-500">{parsedResults.summary.by_severity.low}</span>
													</div>
												{/if}
											</div>
										</div>
									{/if}
									
									<!-- By Category -->
									{#if parsedResults.summary.by_category}
										<div class="bg-muted/30 rounded-lg p-4">
											<div class="text-sm text-muted-foreground mb-2">By Category</div>
											<div class="space-y-2">
												{#each Object.entries(parsedResults.summary.by_category) as [category, count]}
													<div class="flex items-center justify-between">
														<span class="text-sm capitalize">{category}</span>
														<span class="font-semibold">{count}</span>
													</div>
												{/each}
											</div>
										</div>
									{/if}
								</div>
							</div>
						{/if}
						
						<!-- Findings List -->
						{#if parsedResults.findings && parsedResults.findings.length > 0}
							<div class="mb-6">
								<h3 class="font-medium mb-3 flex items-center gap-2">
									<AlertTriangle class="h-4 w-4" />
									Findings ({parsedResults.findings.length})
								</h3>
								<div class="space-y-2 max-h-72 overflow-auto">
									{#each parsedResults.findings.slice(0, 50) as finding}
										<div class="bg-muted/30 rounded p-3 text-sm">
											<div class="flex items-center gap-2 mb-1">
												<span class={cn(
													"px-2 py-0.5 rounded text-xs font-medium",
													finding.severity === 'critical' && "bg-red-500/20 text-red-500",
													finding.severity === 'high' && "bg-orange-500/20 text-orange-500",
													finding.severity === 'medium' && "bg-yellow-500/20 text-yellow-500",
													finding.severity === 'low' && "bg-blue-500/20 text-blue-500"
												)}>
													{finding.severity}
												</span>
												<span class="text-muted-foreground capitalize">{finding.category || finding.type}</span>
											</div>
											<div class="font-mono text-xs text-muted-foreground mb-1">
												{finding.file}:{finding.line}
											</div>
											{#if finding.match || finding.description}
												<div class="text-xs text-muted-foreground">
													{finding.description || finding.match}
												</div>
											{/if}
										</div>
									{/each}
									{#if parsedResults.findings.length > 50}
										<div class="text-sm text-muted-foreground text-center py-2">
											...and {parsedResults.findings.length - 50} more findings
										</div>
									{/if}
								</div>
							</div>
						{/if}
						
						<!-- Issues List (for Quality/DeadCode scans) -->
						{#if parsedResults.issues && parsedResults.issues.length > 0}
							<div class="mb-6">
								<h3 class="font-medium mb-3 flex items-center gap-2">
									<AlertTriangle class="h-4 w-4" />
									Issues ({parsedResults.issues.length})
								</h3>
								<div class="space-y-2 max-h-72 overflow-auto">
									{#each parsedResults.issues.slice(0, 50) as issue}
										<div class="bg-muted/30 rounded p-3 text-sm">
											<div class="flex items-center gap-2 mb-1">
												<span class={cn(
													"px-2 py-0.5 rounded text-xs font-medium",
													issue.severity === 'critical' && "bg-red-500/20 text-red-500",
													issue.severity === 'high' && "bg-orange-500/20 text-orange-500",
													issue.severity === 'medium' && "bg-yellow-500/20 text-yellow-500",
													issue.severity === 'low' && "bg-blue-500/20 text-blue-500",
													!issue.severity && "bg-gray-500/20 text-gray-500"
												)}>
													{issue.severity || 'info'}
												</span>
												<span class="text-muted-foreground">{issue.category || issue.type}</span>
											</div>
											<div class="font-medium mb-1">{issue.message || issue.title || issue.name}</div>
											{#if issue.file || issue.file_path}
												<div class="font-mono text-xs text-muted-foreground">
													{issue.file || issue.file_path}{issue.line ? `:${issue.line}` : ''}
												</div>
											{/if}
											{#if issue.suggestion}
												<div class="text-xs text-green-500 mt-1">
													💡 {issue.suggestion}
												</div>
											{/if}
										</div>
									{/each}
								</div>
							</div>
						{/if}
						
						<!-- Raw JSON (collapsible) -->
						<details class="group">
							<summary class="cursor-pointer text-sm text-muted-foreground hover:text-foreground flex items-center gap-2">
								<ChevronRight class="h-4 w-4 transition-transform group-open:rotate-90" />
								View Raw JSON
							</summary>
							<pre class="bg-muted/50 rounded-lg p-4 text-xs overflow-auto max-h-64 mt-2 whitespace-pre-wrap">{JSON.stringify(parsedResults, null, 2)}</pre>
						</details>
					{/if}
				{/if}
			</div>

			<div class="flex justify-end gap-2 p-4 border-t">
				<Button variant="outline" onclick={() => showDetailModal = false}>
					Close
				</Button>
			</div>
		</div>
	</div>
{/if}

