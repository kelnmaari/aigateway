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
		Code,
		Plus,
		RefreshCw,
		Bug,
		Shield,
		Wrench,
		Trash2,
		AlertTriangle
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

	interface ParsedSection {
		title: string;
		type: 'added' | 'changed' | 'fixed' | 'security' | 'technical' | 'removed' | 'deprecated' | 'other';
		items: string[];
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

	function parseChangelog(content: string): ParsedSection[] {
		const sections: ParsedSection[] = [];
		const lines = content.split('\n');
		let currentSection: ParsedSection | null = null;
		let currentItem = '';

		for (const line of lines) {
			// Check for section headers (### Added, ### Changed, etc.)
			const headerMatch = line.match(/^###\s+(.+)$/);
			if (headerMatch) {
				// Save previous section
				if (currentSection) {
					if (currentItem.trim()) {
						currentSection.items.push(currentItem.trim());
					}
					sections.push(currentSection);
				}

				const title = headerMatch[1].trim();
				const type = getSectionType(title);
				currentSection = { title, type, items: [] };
				currentItem = '';
				continue;
			}

			// Skip version header lines (## [x.x.x] - date)
			if (line.match(/^##\s+\[/)) {
				continue;
			}

			// Check for list items
			const itemMatch = line.match(/^-\s+(.+)$/);
			if (itemMatch && currentSection) {
				// Save previous item if exists
				if (currentItem.trim()) {
					currentSection.items.push(currentItem.trim());
				}
				currentItem = itemMatch[1];
				continue;
			}

			// Continuation of previous item (indented lines)
			if (line.match(/^\s+/) && currentItem && currentSection) {
				currentItem += '\n' + line.trim();
				continue;
			}
		}

		// Save last section and item
		if (currentSection) {
			if (currentItem.trim()) {
				currentSection.items.push(currentItem.trim());
			}
			sections.push(currentSection);
		}

		return sections;
	}

	function getSectionType(title: string): ParsedSection['type'] {
		const lower = title.toLowerCase();
		if (lower.includes('added') || lower.includes('new')) return 'added';
		if (lower.includes('changed') || lower.includes('updated')) return 'changed';
		if (lower.includes('fixed') || lower.includes('bug')) return 'fixed';
		if (lower.includes('security')) return 'security';
		if (lower.includes('technical') || lower.includes('internal')) return 'technical';
		if (lower.includes('removed') || lower.includes('deleted')) return 'removed';
		if (lower.includes('deprecated')) return 'deprecated';
		return 'other';
	}

	function getSectionConfig(type: ParsedSection['type']) {
		const configs = {
			added: {
				bg: 'bg-emerald-500/10 dark:bg-emerald-500/20',
				border: 'border-emerald-500/30',
				text: 'text-emerald-700 dark:text-emerald-400',
				icon: Plus
			},
			changed: {
				bg: 'bg-blue-500/10 dark:bg-blue-500/20',
				border: 'border-blue-500/30',
				text: 'text-blue-700 dark:text-blue-400',
				icon: RefreshCw
			},
			fixed: {
				bg: 'bg-amber-500/10 dark:bg-amber-500/20',
				border: 'border-amber-500/30',
				text: 'text-amber-700 dark:text-amber-400',
				icon: Bug
			},
			security: {
				bg: 'bg-red-500/10 dark:bg-red-500/20',
				border: 'border-red-500/30',
				text: 'text-red-700 dark:text-red-400',
				icon: Shield
			},
			technical: {
				bg: 'bg-purple-500/10 dark:bg-purple-500/20',
				border: 'border-purple-500/30',
				text: 'text-purple-700 dark:text-purple-400',
				icon: Wrench
			},
			removed: {
				bg: 'bg-gray-500/10 dark:bg-gray-500/20',
				border: 'border-gray-500/30',
				text: 'text-gray-700 dark:text-gray-400',
				icon: Trash2
			},
			deprecated: {
				bg: 'bg-orange-500/10 dark:bg-orange-500/20',
				border: 'border-orange-500/30',
				text: 'text-orange-700 dark:text-orange-400',
				icon: AlertTriangle
			},
			other: {
				bg: 'bg-slate-500/10 dark:bg-slate-500/20',
				border: 'border-slate-500/30',
				text: 'text-slate-700 dark:text-slate-400',
				icon: Info
			}
		};
		return configs[type];
	}

	function formatItemText(text: string): string {
		// Convert **bold** to <strong>
		let result = text.replace(/\*\*(.+?)\*\*/g, '<strong class="font-semibold">$1</strong>');
		// Convert `code` to <code>
		result = result.replace(/`([^`]+)`/g, '<code class="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">$1</code>');
		// Convert newlines in item to proper breaks
		result = result.replace(/\n/g, '<br>');
		return result;
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
									<div class="space-y-4">
										{#each parseChangelog(changelog.content) as section}
											{@const config = getSectionConfig(section.type)}
											{@const Icon = config.icon}
											<div class="rounded-lg border {config.border} {config.bg} overflow-hidden">
												<div class="flex items-center gap-2 px-4 py-2.5 border-b {config.border}">
													<Icon class="h-4 w-4 {config.text}" />
													<h4 class="font-semibold {config.text}">{section.title}</h4>
												</div>
												<div class="px-4 py-3">
													<ul class="space-y-2 text-sm text-foreground/90">
														{#each section.items as item}
															<li class="flex gap-2">
																<span class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full {config.text.replace('text-', 'bg-')}"></span>
																<span>{@html formatItemText(item)}</span>
															</li>
														{/each}
													</ul>
												</div>
											</div>
										{/each}
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

