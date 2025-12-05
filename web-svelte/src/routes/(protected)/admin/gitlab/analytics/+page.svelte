<script lang="ts">
	import { onMount } from 'svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';

	interface ReviewStats {
		total_reviews: number;
		completed_reviews: number;
		failed_reviews: number;
		pending_reviews: number;
		total_files_reviewed: number;
		total_lines_changed: number;
		total_issues_found: number;
		total_tokens_used: number;
		avg_score: number;
		avg_processing_time_ms: number;
		success_rate: number;
	}

	interface ProjectStats {
		project_id: string;
		project_name: string;
		total_reviews: number;
		completed_reviews: number;
		total_issues_found: number;
		avg_score: number;
	}

	interface ModelStats {
		model_id: string;
		model_name: string;
		total_reviews: number;
		total_tokens_used: number;
		avg_tokens_per_review: number;
		avg_processing_time_ms: number;
		avg_score: number;
	}

	interface CategoryStats {
		category: string;
		count: number;
		percentage: number;
	}

	let stats: ReviewStats | null = null;
	let projectStats: ProjectStats[] = [];
	let modelStats: ModelStats[] = [];
	let categoryStats: CategoryStats[] = [];
	let loading = true;
	let error = '';
	let timeRange = '7d';
	let activeTab = 'projects';

	const timeRanges = [
		{ value: '24h', label: 'Last 24 hours' },
		{ value: '7d', label: 'Last 7 days' },
		{ value: '30d', label: 'Last 30 days' },
		{ value: '90d', label: 'Last 90 days' }
	];

	onMount(() => {
		loadAnalytics();
	});

	async function loadAnalytics() {
		loading = true;
		error = '';
		try {
			const response = await fetch(`/api/admin/gitlab/analytics?range=${timeRange}`);
			if (!response.ok) throw new Error('Failed to load analytics');
			const data = await response.json();
			stats = data.overview;
			projectStats = data.top_projects || [];
			modelStats = data.top_models || [];
			categoryStats = data.issues_by_category || [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Unknown error';
		} finally {
			loading = false;
		}
	}

	function formatNumber(n: number): string {
		if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M';
		if (n >= 1000) return (n / 1000).toFixed(1) + 'K';
		return n.toString();
	}

	function formatDuration(ms: number): string {
		if (ms < 1000) return ms + 'ms';
		if (ms < 60000) return (ms / 1000).toFixed(1) + 's';
		return (ms / 60000).toFixed(1) + 'm';
	}

	function getScoreColor(score: number): string {
		if (score >= 80) return 'text-green-500';
		if (score >= 60) return 'text-yellow-500';
		return 'text-red-500';
	}

	$: if (timeRange) loadAnalytics();
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">GitLab Analytics</h1>
			<p class="text-muted-foreground">Code review statistics and insights</p>
		</div>
		<div class="flex items-center gap-4">
			<select 
				bind:value={timeRange}
				class="h-10 rounded-md border bg-background px-3 text-sm"
			>
				{#each timeRanges as range}
					<option value={range.value}>{range.label}</option>
				{/each}
			</select>
			<Button variant="outline" onclick={loadAnalytics}>
				Refresh
			</Button>
		</div>
	</div>

	{#if loading}
		<div class="flex items-center justify-center h-64">
			<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
		</div>
	{:else if error}
		<div class="rounded-lg border border-destructive bg-destructive/10 p-6">
			<p class="text-destructive">{error}</p>
		</div>
	{:else if stats}
		<!-- Overview Cards -->
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Total Reviews</CardDescription>
					<CardTitle class="text-3xl">{formatNumber(stats.total_reviews)}</CardTitle>
				</CardHeader>
				<CardContent>
					<div class="text-sm text-muted-foreground">
						<span class="text-green-500">{stats.completed_reviews} completed</span>
						{#if stats.failed_reviews > 0}
							<span class="text-red-500 ml-2">{stats.failed_reviews} failed</span>
						{/if}
					</div>
				</CardContent>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Average Score</CardDescription>
					<CardTitle class="text-3xl {getScoreColor(stats.avg_score)}">
						{stats.avg_score.toFixed(1)}
					</CardTitle>
				</CardHeader>
				<CardContent>
					<div class="h-2 bg-muted rounded-full overflow-hidden">
						<div class="h-full bg-primary" style="width: {stats.avg_score}%"></div>
					</div>
				</CardContent>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Issues Found</CardDescription>
					<CardTitle class="text-3xl">{formatNumber(stats.total_issues_found)}</CardTitle>
				</CardHeader>
				<CardContent>
					<div class="text-sm text-muted-foreground">
						{stats.total_files_reviewed} files • {formatNumber(stats.total_lines_changed)} lines
					</div>
				</CardContent>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Tokens Used</CardDescription>
					<CardTitle class="text-3xl">{formatNumber(stats.total_tokens_used)}</CardTitle>
				</CardHeader>
				<CardContent>
					<div class="text-sm text-muted-foreground">
						Avg: {formatDuration(stats.avg_processing_time_ms)} per review
					</div>
				</CardContent>
			</Card>
		</div>

		<!-- Tabs -->
		<div class="space-y-4">
			<div class="flex gap-2 border-b">
				<button
					class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'projects' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
					onclick={() => activeTab = 'projects'}
				>
					By Project
				</button>
				<button
					class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'models' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
					onclick={() => activeTab = 'models'}
				>
					By Model
				</button>
				<button
					class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'categories' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
					onclick={() => activeTab = 'categories'}
				>
					Issue Categories
				</button>
			</div>

			{#if activeTab === 'projects'}
				<Card>
					<CardHeader>
						<CardTitle>Top Projects</CardTitle>
						<CardDescription>Most active projects by review count</CardDescription>
					</CardHeader>
					<CardContent>
						<div class="space-y-4">
							{#each projectStats as project}
								<div class="flex items-center justify-between p-3 bg-muted/50 rounded-lg">
									<div>
										<div class="font-medium">{project.project_name}</div>
										<div class="text-sm text-muted-foreground">
											{project.total_reviews} reviews • {project.total_issues_found} issues
										</div>
									</div>
									<div class="text-right">
										<div class="text-lg font-semibold {getScoreColor(project.avg_score)}">
											{project.avg_score.toFixed(1)}
										</div>
										<div class="text-xs text-muted-foreground">avg score</div>
									</div>
								</div>
							{:else}
								<p class="text-muted-foreground text-center py-8">No project data available</p>
							{/each}
						</div>
					</CardContent>
				</Card>
			{:else if activeTab === 'models'}
				<Card>
					<CardHeader>
						<CardTitle>Model Performance</CardTitle>
						<CardDescription>Comparison of models used for reviews</CardDescription>
					</CardHeader>
					<CardContent>
						<div class="space-y-4">
							{#each modelStats as model}
								<div class="flex items-center justify-between p-3 bg-muted/50 rounded-lg">
									<div>
										<div class="font-medium">{model.model_name}</div>
										<div class="text-sm text-muted-foreground">
											{model.total_reviews} reviews • {formatNumber(model.total_tokens_used)} tokens
										</div>
									</div>
									<div class="flex items-center gap-4">
										<div class="text-center">
											<div class="text-sm font-medium">{formatDuration(model.avg_processing_time_ms)}</div>
											<div class="text-xs text-muted-foreground">avg time</div>
										</div>
										<div class="text-center">
											<div class="text-lg font-semibold {getScoreColor(model.avg_score)}">
												{model.avg_score.toFixed(1)}
											</div>
											<div class="text-xs text-muted-foreground">avg score</div>
										</div>
									</div>
								</div>
							{:else}
								<p class="text-muted-foreground text-center py-8">No model data available</p>
							{/each}
						</div>
					</CardContent>
				</Card>
			{:else}
				<Card>
					<CardHeader>
						<CardTitle>Issue Categories</CardTitle>
						<CardDescription>Distribution of issues by category</CardDescription>
					</CardHeader>
					<CardContent>
						<div class="space-y-3">
							{#each categoryStats as cat}
								<div class="space-y-1">
									<div class="flex justify-between text-sm">
										<span class="font-medium">{cat.category}</span>
										<span class="text-muted-foreground">{cat.count} ({cat.percentage.toFixed(1)}%)</span>
									</div>
									<div class="h-2 bg-muted rounded-full overflow-hidden">
										<div class="h-full bg-primary" style="width: {cat.percentage}%"></div>
									</div>
								</div>
							{:else}
								<p class="text-muted-foreground text-center py-8">No category data available</p>
							{/each}
						</div>
					</CardContent>
				</Card>
			{/if}
		</div>

		<!-- Success Rate Card -->
		<Card>
			<CardHeader>
				<CardTitle>Success Rate</CardTitle>
				<CardDescription>Percentage of reviews completed successfully</CardDescription>
			</CardHeader>
			<CardContent>
				<div class="flex items-center gap-4">
					<div class="text-4xl font-bold {stats.success_rate >= 0.9 ? 'text-green-500' : stats.success_rate >= 0.7 ? 'text-yellow-500' : 'text-red-500'}">
						{(stats.success_rate * 100).toFixed(1)}%
					</div>
					<div class="flex-1 h-3 bg-muted rounded-full overflow-hidden">
						<div class="h-full bg-primary" style="width: {stats.success_rate * 100}%"></div>
					</div>
				</div>
			</CardContent>
		</Card>
	{/if}
</div>
