<script lang="ts">
	import { onMount } from 'svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';

	interface FeedbackStats {
		total_feedback: number;
		approved_count: number;
		rejected_count: number;
		edited_count: number;
		ignored_count: number;
		approval_rate: number;
		accuracy_rate: number;
	}

	interface CategoryAccuracy {
		category: string;
		total_issues: number;
		approved_count: number;
		rejected_count: number;
		accuracy_rate: number;
	}

	interface ModelAccuracy {
		model_id: string;
		model_name: string;
		total_issues: number;
		approved_count: number;
		rejected_count: number;
		accuracy_rate: number;
	}

	interface FeedbackItem {
		id: string;
		review_id: string;
		issue_category: string;
		issue_severity: string;
		issue_message: string;
		issue_file: string;
		issue_line: number;
		type: string;
		comment: string;
		username: string;
		created_at: string;
	}

	let stats: FeedbackStats | null = null;
	let categoryAccuracy: CategoryAccuracy[] = [];
	let modelAccuracy: ModelAccuracy[] = [];
	let recentFeedback: FeedbackItem[] = [];
	let loading = true;
	let error = '';
	let filterType = 'all';
	let activeTab = 'categories';

	const feedbackTypes = [
		{ value: 'all', label: 'All Feedback' },
		{ value: 'approve', label: 'Approved' },
		{ value: 'reject', label: 'Rejected' },
		{ value: 'edit', label: 'Edited' }
	];

	onMount(() => {
		loadFeedback();
	});

	async function loadFeedback() {
		loading = true;
		try {
			const params = new URLSearchParams();
			if (filterType !== 'all') params.append('type', filterType);

			const response = await fetch(`/api/admin/gitlab/feedback?${params}`);
			if (!response.ok) throw new Error('Failed to load feedback');
			const data = await response.json();
			stats = data.stats;
			categoryAccuracy = data.category_accuracy || [];
			modelAccuracy = data.model_accuracy || [];
			recentFeedback = data.recent || [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Unknown error';
		} finally {
			loading = false;
		}
	}

	async function exportTrainingData() {
		try {
			const response = await fetch('/api/admin/gitlab/feedback/export', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' }
			});
			if (!response.ok) throw new Error('Export failed');
			const blob = await response.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `training_data_${new Date().toISOString().split('T')[0]}.json`;
			a.click();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Export failed';
		}
	}

	function getFeedbackIcon(type: string): string {
		switch (type) {
			case 'approve': return '✓';
			case 'reject': return '✗';
			case 'edit': return '✎';
			case 'ignore': return '−';
			default: return '?';
		}
	}

	function getFeedbackColor(type: string): string {
		switch (type) {
			case 'approve': return 'bg-green-500';
			case 'reject': return 'bg-red-500';
			case 'edit': return 'bg-yellow-500';
			case 'ignore': return 'bg-gray-500';
			default: return 'bg-gray-500';
		}
	}

	function getSeverityColor(severity: string): string {
		switch (severity) {
			case 'critical': return 'text-red-500';
			case 'high': return 'text-orange-500';
			case 'medium': return 'text-yellow-500';
			case 'low': return 'text-green-500';
			default: return 'text-gray-500';
		}
	}

	$: if (filterType) loadFeedback();
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">Review Feedback</h1>
			<p class="text-muted-foreground">Track accuracy and improve AI reviews</p>
		</div>
		<div class="flex items-center gap-4">
			<select
				bind:value={filterType}
				class="h-10 rounded-md border bg-background px-3 text-sm"
			>
				{#each feedbackTypes as type}
					<option value={type.value}>{type.label}</option>
				{/each}
			</select>
			<Button variant="outline" onclick={exportTrainingData}>
				Export Training Data
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
		<!-- Stats Overview -->
		<div class="grid grid-cols-1 md:grid-cols-4 gap-4">
			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Total Feedback</CardDescription>
					<CardTitle class="text-3xl">{stats.total_feedback}</CardTitle>
				</CardHeader>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Approval Rate</CardDescription>
					<CardTitle class="text-3xl text-green-500">
						{(stats.approval_rate * 100).toFixed(1)}%
					</CardTitle>
				</CardHeader>
				<CardContent>
					<div class="h-2 bg-muted rounded-full overflow-hidden">
						<div class="h-full bg-green-500" style="width: {stats.approval_rate * 100}%"></div>
					</div>
				</CardContent>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Accuracy Rate</CardDescription>
					<CardTitle class="text-3xl {stats.accuracy_rate >= 0.8 ? 'text-green-500' : stats.accuracy_rate >= 0.6 ? 'text-yellow-500' : 'text-red-500'}">
						{(stats.accuracy_rate * 100).toFixed(1)}%
					</CardTitle>
				</CardHeader>
				<CardContent>
					<div class="h-2 bg-muted rounded-full overflow-hidden">
						<div class="h-full bg-primary" style="width: {stats.accuracy_rate * 100}%"></div>
					</div>
				</CardContent>
			</Card>

			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Feedback Breakdown</CardDescription>
				</CardHeader>
				<CardContent>
					<div class="flex gap-2 text-sm">
						<span class="px-2 py-1 rounded bg-green-500/10 text-green-500">✓ {stats.approved_count}</span>
						<span class="px-2 py-1 rounded bg-red-500/10 text-red-500">✗ {stats.rejected_count}</span>
						<span class="px-2 py-1 rounded bg-yellow-500/10 text-yellow-500">✎ {stats.edited_count}</span>
					</div>
				</CardContent>
			</Card>
		</div>

		<!-- Tabs -->
		<div class="space-y-4">
			<div class="flex gap-2 border-b">
				<button
					class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'categories' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
					onclick={() => activeTab = 'categories'}
				>
					By Category
				</button>
				<button
					class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'models' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
					onclick={() => activeTab = 'models'}
				>
					By Model
				</button>
				<button
					class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'recent' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
					onclick={() => activeTab = 'recent'}
				>
					Recent Feedback
				</button>
			</div>

			{#if activeTab === 'categories'}
				<Card>
					<CardHeader>
						<CardTitle>Accuracy by Issue Category</CardTitle>
						<CardDescription>How accurate is AI for each type of issue</CardDescription>
					</CardHeader>
					<CardContent>
						<div class="space-y-3">
							{#each categoryAccuracy as cat}
								<div class="space-y-1">
									<div class="flex justify-between text-sm">
										<span class="font-medium capitalize">{cat.category}</span>
										<span class="text-muted-foreground">
											{cat.approved_count}/{cat.total_issues} approved ({(cat.accuracy_rate * 100).toFixed(0)}%)
										</span>
									</div>
									<div class="h-2 bg-muted rounded-full overflow-hidden">
										<div class="h-full bg-primary" style="width: {cat.accuracy_rate * 100}%"></div>
									</div>
								</div>
							{:else}
								<p class="text-muted-foreground text-center py-8">No category data available</p>
							{/each}
						</div>
					</CardContent>
				</Card>
			{:else if activeTab === 'models'}
				<Card>
					<CardHeader>
						<CardTitle>Accuracy by Model</CardTitle>
						<CardDescription>Compare model performance based on user feedback</CardDescription>
					</CardHeader>
					<CardContent>
						<div class="space-y-4">
							{#each modelAccuracy as model}
								<div class="flex items-center justify-between p-3 bg-muted/50 rounded-lg">
									<div>
										<div class="font-medium">{model.model_name}</div>
										<div class="text-sm text-muted-foreground">
											{model.total_issues} issues reviewed
										</div>
									</div>
									<div class="flex items-center gap-4">
										<div class="text-center">
											<div class="text-sm text-green-500">{model.approved_count} ✓</div>
											<div class="text-sm text-red-500">{model.rejected_count} ✗</div>
										</div>
										<div class="text-center">
											<div class="text-2xl font-bold {model.accuracy_rate >= 0.8 ? 'text-green-500' : model.accuracy_rate >= 0.6 ? 'text-yellow-500' : 'text-red-500'}">
												{(model.accuracy_rate * 100).toFixed(0)}%
											</div>
											<div class="text-xs text-muted-foreground">accuracy</div>
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
						<CardTitle>Recent Feedback</CardTitle>
						<CardDescription>Latest user feedback on AI reviews</CardDescription>
					</CardHeader>
					<CardContent>
						<div class="space-y-3">
							{#each recentFeedback as item}
								<div class="p-3 bg-muted/50 rounded-lg">
									<div class="flex items-start justify-between">
										<div class="flex items-center gap-2">
											<span class="px-2 py-1 text-xs font-medium text-white rounded {getFeedbackColor(item.type)}">
												{getFeedbackIcon(item.type)}
											</span>
											<span class="font-medium capitalize">{item.issue_category}</span>
											<span class={getSeverityColor(item.issue_severity)}>
												{item.issue_severity}
											</span>
										</div>
										<div class="text-xs text-muted-foreground">
											by {item.username} • {new Date(item.created_at).toLocaleDateString()}
										</div>
									</div>
									<div class="mt-2 text-sm">
										<code class="text-xs bg-muted px-1 rounded">{item.issue_file}:{item.issue_line}</code>
									</div>
									<p class="mt-1 text-sm text-muted-foreground">{item.issue_message}</p>
									{#if item.comment}
										<p class="mt-2 text-sm italic border-l-2 border-primary pl-2">
											"{item.comment}"
										</p>
									{/if}
								</div>
							{:else}
								<p class="text-muted-foreground text-center py-8">No feedback yet</p>
							{/each}
						</div>
					</CardContent>
				</Card>
			{/if}
		</div>
	{/if}
</div>
