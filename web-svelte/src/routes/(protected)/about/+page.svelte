<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Info,
		Loader2,
		ExternalLink,
		Github,
		ChevronDown,
		ChevronRight,
		Tag,
		Calendar,
		Code
	} from 'lucide-svelte';
	import { api } from '$lib/api/client';
	import { cn } from '$lib/utils';
	import * as m from '$lib/paraglide/messages';

	interface SystemInfo {
		version: string;
		git_commit: string;
		build_date: string;
		go_version: string;
	}

	interface ChangelogEntry {
		version: string;
		release_date: string;
		content: string;
	}

	let systemInfo = $state<SystemInfo | null>(null);
	let changelogs = $state<ChangelogEntry[]>([]);
	let isLoading = $state(true);
	let expandedVersions = $state<Set<string>>(new Set());

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		isLoading = true;
		try {
			const [infoRes, changelogsRes] = await Promise.allSettled([
				api.get<SystemInfo>('/api/system/info'),
				api.get<{ changelogs: ChangelogEntry[] }>('/api/system/changelogs')
			]);

			if (infoRes.status === 'fulfilled') {
				systemInfo = infoRes.value;
			}
			if (changelogsRes.status === 'fulfilled') {
				changelogs = changelogsRes.value.changelogs || [];
				// Expand latest version by default
				if (changelogs.length > 0) {
					expandedVersions.add(changelogs[0].version);
				}
			}
		} catch (error) {
			console.error('Failed to load data:', error);
		} finally {
			isLoading = false;
		}
	}

	function toggleVersion(version: string) {
		const newSet = new Set(expandedVersions);
		if (newSet.has(version)) {
			newSet.delete(version);
		} else {
			newSet.add(version);
		}
		expandedVersions = newSet;
	}

	function formatDate(dateStr: string): string {
		try {
			return new Date(dateStr).toLocaleDateString(undefined, {
				year: 'numeric',
				month: 'long',
				day: 'numeric'
			});
		} catch {
			return dateStr;
		}
	}
</script>

<svelte:head>
	<title>About | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-[1600px] px-4 py-8 sm:px-6 lg:px-8">
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else}
		<!-- Header -->
		<div class="mb-8 text-center">
			<div class="mx-auto mb-4 flex h-20 w-20 items-center justify-center rounded-2xl bg-primary text-4xl text-primary-foreground">
				🤖
			</div>
			<h1 class="text-3xl font-bold text-foreground">AI Gateway</h1>
			<p class="mt-2 text-muted-foreground">OpenAI-compatible API Gateway for Local LLMs</p>
		</div>

		<!-- System Info -->
		{#if systemInfo}
			<div class="mb-8 rounded-xl border border-border bg-card p-6">
				<h2 class="mb-4 flex items-center gap-2 font-semibold">
					<Info class="h-5 w-5" />
					System Information
				</h2>

				<div class="grid gap-4 sm:grid-cols-2">
					<div class="flex items-center gap-3">
						<Tag class="h-5 w-5 text-muted-foreground" />
						<div>
							<p class="text-sm text-muted-foreground">Version</p>
							<p class="font-mono font-medium">{systemInfo.version}</p>
						</div>
					</div>

					<div class="flex items-center gap-3">
						<Code class="h-5 w-5 text-muted-foreground" />
						<div>
							<p class="text-sm text-muted-foreground">Git Commit</p>
							<p class="font-mono font-medium">{systemInfo.git_commit || 'N/A'}</p>
						</div>
					</div>

					<div class="flex items-center gap-3">
						<Calendar class="h-5 w-5 text-muted-foreground" />
						<div>
							<p class="text-sm text-muted-foreground">Build Date</p>
							<p class="font-mono font-medium">{systemInfo.build_date || 'N/A'}</p>
						</div>
					</div>

					<div class="flex items-center gap-3">
						<Code class="h-5 w-5 text-muted-foreground" />
						<div>
							<p class="text-sm text-muted-foreground">Go Version</p>
							<p class="font-mono font-medium">{systemInfo.go_version || 'N/A'}</p>
						</div>
					</div>
				</div>
			</div>
		{/if}

		<!-- Links -->
		<div class="mb-8 flex flex-wrap justify-center gap-4">
			<a
				href="https://gitlab.alexue4.dev/KelnMaari/ollama-openai-proxy"
				target="_blank"
				rel="noopener noreferrer"
				class="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm transition-colors hover:bg-accent"
			>
				<Github class="h-4 w-4" />
				GitLab
				<ExternalLink class="h-3 w-3" />
			</a>
			<a
				href="https://gitlab.alexue4.dev/KelnMaari/ollama-openai-proxy/-/issues"
				target="_blank"
				rel="noopener noreferrer"
				class="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm transition-colors hover:bg-accent"
			>
				Report Issue
				<ExternalLink class="h-3 w-3" />
			</a>
		</div>

		<!-- Changelog -->
		{#if changelogs.length > 0}
			<div class="rounded-xl border border-border bg-card">
				<div class="border-b border-border p-4">
					<h2 class="font-semibold">Changelog</h2>
				</div>

				<div class="divide-y divide-border">
					{#each changelogs as changelog}
						<div>
							<button
								onclick={() => toggleVersion(changelog.version)}
								class="flex w-full items-center justify-between px-4 py-3 text-left transition-colors hover:bg-accent/50"
							>
								<div class="flex items-center gap-3">
									<span class="rounded bg-primary/10 px-2 py-0.5 font-mono text-sm font-medium text-primary">
										v{changelog.version}
									</span>
									<span class="text-sm text-muted-foreground">
										{formatDate(changelog.release_date)}
									</span>
								</div>
								{#if expandedVersions.has(changelog.version)}
									<ChevronDown class="h-4 w-4 text-muted-foreground" />
								{:else}
									<ChevronRight class="h-4 w-4 text-muted-foreground" />
								{/if}
							</button>

							{#if expandedVersions.has(changelog.version)}
								<div class="border-t border-border bg-muted/30 px-4 py-4">
									<div class="prose prose-sm dark:prose-invert max-w-none">
										{@html changelog.content.replace(/\n/g, '<br>')}
									</div>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		{/if}
	{/if}
</div>

