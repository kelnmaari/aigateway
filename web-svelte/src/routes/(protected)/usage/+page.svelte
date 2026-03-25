<script lang="ts">
	import { onMount } from 'svelte';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import {
		faChartColumn,
		faSpinner,
		faArrowsRotate,
		faArrowTrendUp,
		faBolt,
		faClockRotateLeft,
		faCalendar
	} from '@fortawesome/free-solid-svg-icons';
	import { usageApi, type UsageStats, type UsageRecord } from '$lib/api/usage';
	import { cn, formatNumber, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	let stats = $state<UsageStats | null>(null);
	let records = $state<UsageRecord[]>([]);
	let isLoading = $state(true);
	let period = $state<'7d' | '30d' | '90d'>('30d');

	onMount(async () => {
		await loadUsage();
	});

	async function loadUsage() {
		isLoading = true;
		try {
			const response = await usageApi.getUsage(period);
			stats = response.stats;
			records = response.records || [];
		} catch (error) {
			console.error('Failed to load usage:', error);
		} finally {
			isLoading = false;
		}
	}

	function handlePeriodChange(newPeriod: '7d' | '30d' | '90d') {
		period = newPeriod;
		loadUsage();
	}

	function getPeriodLabel(p: string): string {
		switch (p) {
			case '7d': return 'Last 7 days';
			case '30d': return 'Last 30 days';
			case '90d': return 'Last 90 days';
			default: return p;
		}
	}
</script>

<svelte:head>
	<title>{m.nav_usage()} | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-[1600px] px-4 py-8 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-foreground">{m.usage_title()}</h1>
			<p class="mt-1 text-muted-foreground">{m.usage_subtitle()}</p>
		</div>
		<div class="flex items-center gap-2">
			<!-- Period Selector -->
			<div class="flex rounded-lg border border-input">
				{#each ['7d', '30d', '90d'] as p}
					<button
						onclick={() => handlePeriodChange(p as '7d' | '30d' | '90d')}
						class={cn(
							'px-3 py-1.5 text-sm transition-colors',
							period === p ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'
						)}
					>
						{p}
					</button>
				{/each}
			</div>
			<Button variant="outline" onclick={loadUsage} disabled={isLoading}>
				<FontAwesomeIcon icon={faArrowsRotate} class={cn('h-4 w-4', isLoading && 'animate-spin')} />
			</Button>
		</div>
	</div>

	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<FontAwesomeIcon icon={faSpinner} class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if stats}
		<!-- Stats Cards -->
		<div class="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10 text-blue-500">
						<FontAwesomeIcon icon={faBolt} class="h-5 w-5" />
					</div>
					<div>
						<p class="text-sm text-muted-foreground">Total Requests</p>
						<p class="text-2xl font-bold">{formatNumber(stats.total_requests)}</p>
					</div>
				</div>
			</div>

			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/10 text-green-500">
						<FontAwesomeIcon icon={faChartColumn} class="h-5 w-5" />
					</div>
					<div>
						<p class="text-sm text-muted-foreground">Total Tokens</p>
						<p class="text-2xl font-bold">{formatNumber(stats.total_tokens)}</p>
					</div>
				</div>
			</div>

			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-500/10 text-amber-500">
						<FontAwesomeIcon icon={faArrowTrendUp} class="h-5 w-5" />
					</div>
					<div>
						<p class="text-sm text-muted-foreground">Input Tokens</p>
						<p class="text-2xl font-bold">{formatNumber(stats.total_input_tokens)}</p>
					</div>
				</div>
			</div>

			<div class="rounded-xl border border-border bg-card p-5">
				<div class="flex items-center gap-3">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/10 text-purple-500">
						<FontAwesomeIcon icon={faArrowTrendUp} class="h-5 w-5" />
					</div>
					<div>
						<p class="text-sm text-muted-foreground">Output Tokens</p>
						<p class="text-2xl font-bold">{formatNumber(stats.total_output_tokens)}</p>
					</div>
				</div>
			</div>
		</div>

		<!-- Usage by Model -->
		{#if stats.by_model && stats.by_model.length > 0}
			<div class="mb-8 rounded-xl border border-border bg-card p-6">
				<h2 class="mb-4 font-semibold text-foreground">Usage by Model</h2>
				<div class="space-y-3">
					{#each stats.by_model as model}
						{@const percentage = stats.total_requests > 0 ? (model.requests / stats.total_requests) * 100 : 0}
						<div>
							<div class="mb-1 flex items-center justify-between text-sm">
								<span class="font-medium">{model.model}</span>
								<span class="text-muted-foreground">
									{formatNumber(model.requests)} requests • {formatNumber(model.tokens)} tokens
								</span>
							</div>
							<div class="h-2 overflow-hidden rounded-full bg-muted">
								<div class="h-full bg-primary transition-all" style="width: {percentage}%"></div>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Recent Usage -->
		{#if records.length > 0}
			<div class="rounded-xl border border-border bg-card">
				<div class="border-b border-border p-4">
					<h2 class="font-semibold text-foreground">Recent Activity</h2>
				</div>
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead class="border-b border-border bg-muted/50">
							<tr>
								<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Time</th>
								<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Model</th>
								<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">Input</th>
								<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">Output</th>
								<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">Latency</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border">
							{#each records.slice(0, 20) as record}
								<tr class="hover:bg-muted/30">
									<td class="px-4 py-3 text-sm text-muted-foreground">
										{formatRelativeTime(record.timestamp)}
									</td>
									<td class="px-4 py-3 text-sm font-medium">{record.model}</td>
									<td class="px-4 py-3 text-right text-sm">{formatNumber(record.input_tokens)}</td>
									<td class="px-4 py-3 text-right text-sm">{formatNumber(record.output_tokens)}</td>
									<td class="px-4 py-3 text-right text-sm text-muted-foreground">{record.latency_ms}ms</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}
	{:else}
		<div class="rounded-lg border border-dashed border-border py-16 text-center">
			<FontAwesomeIcon icon={faChartColumn} class="mx-auto h-12 w-12 text-muted-foreground/40" />
			<p class="mt-4 text-lg font-medium">No usage data</p>
			<p class="mt-1 text-muted-foreground">Start using the API to see statistics here</p>
		</div>
	{/if}
</div>

