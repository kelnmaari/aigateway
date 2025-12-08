<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { User, Bot, Copy, Check, Brain, ChevronDown, ChevronRight } from 'lucide-svelte';
	import { cn, copyToClipboard as copyText } from '$lib/utils';
	import type { Message } from '$lib/stores/chat.svelte';

	interface Props {
		messages: Message[];
		streamingContent?: string;
		isStreaming?: boolean;
	}

	let { messages, streamingContent = '', isStreaming = false }: Props = $props();

	let container: HTMLDivElement;
	let copiedId: string | null = $state(null);
	let expandedThinking = $state<Set<string>>(new Set());

	// Toggle thinking block visibility
	function toggleThinking(id: string) {
		const newSet = new Set(expandedThinking);
		if (newSet.has(id)) {
			newSet.delete(id);
		} else {
			newSet.add(id);
		}
		expandedThinking = newSet;
	}

	// Parse thinking blocks from content
	// Supports multiple formats including models that output special characters around tags
	interface ParsedContent {
		thinking: string | null;
		response: string;
		isThinking: boolean; // Currently in thinking phase (no closing tag)
	}

	function parseThinkingContent(content: string): ParsedContent {
		if (!content) return { thinking: null, response: '', isThinking: false };

		// Normalize content - replace various unicode brackets/arrows with standard chars
		// This handles encoding issues where ◁▷ become garbled
		let normalized = content
			.replace(/[◁◀⟨〈❮‹«<＜]/g, '<')
			.replace(/[▷▶⟩〉❯›»>＞]/g, '>')
			.replace(/\uFFFD+/g, ''); // Remove replacement characters

		// Try regex first - most flexible
		// Match <think> or variations with any brackets
		const thinkRegex = /[<\[{(]+\s*think\s*[>\]})]+\s*\n?([\s\S]*?)[<\[{(]+\s*\/\s*think\s*[>\]})]+/i;
		const match = normalized.match(thinkRegex);
		
		if (match) {
			const thinking = match[1].trim();
			const beforeThink = normalized.slice(0, match.index).trim();
			const afterThink = normalized.slice((match.index || 0) + match[0].length).trim();
			const response = (beforeThink + ' ' + afterThink).trim();
			return { thinking, response, isThinking: false };
		}

		// Check for incomplete thinking (open tag but no close)
		const openRegex = /[<\[{(]+\s*think\s*[>\]})]+/i;
		const closeRegex = /[<\[{(]+\s*\/\s*think\s*[>\]})]+/i;
		const openMatch = normalized.match(openRegex);
		
		if (openMatch && !normalized.match(closeRegex)) {
			const openIdx = openMatch.index || 0;
			const thinking = normalized.slice(openIdx + openMatch[0].length).trim();
			const beforeThink = normalized.slice(0, openIdx).trim();
			return { thinking, response: beforeThink, isThinking: true };
		}

		// Fallback - return original content
		return { thinking: null, response: content, isThinking: false };
	}

	// Auto-scroll to bottom on new messages
	$effect(() => {
		if (messages.length || streamingContent) {
			tick().then(() => {
				if (container) {
					container.scrollTop = container.scrollHeight;
				}
			});
		}
	});

	async function copyToClipboard(content: string, id: string) {
		const success = await copyText(content);
		if (success) {
			copiedId = id;
			setTimeout(() => {
				copiedId = null;
			}, 2000);
		}
	}

	// Simple markdown to HTML (basic implementation)
	function renderMarkdown(text: string): string {
		if (!text) return '';
		
		// Escape HTML first
		let html = text
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;');

		// Code blocks (```lang\ncode```)
		html = html.replace(/```(\w*)\n([\s\S]*?)```/g, (_, lang, code) => {
			return `<pre class="bg-muted rounded-lg p-4 my-2 overflow-x-auto"><code class="text-sm">${code.trim()}</code></pre>`;
		});

		// Inline code (`code`)
		html = html.replace(/`([^`]+)`/g, '<code class="bg-muted px-1.5 py-0.5 rounded text-sm">$1</code>');

		// Bold (**text** or __text__)
		html = html.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
		html = html.replace(/__([^_]+)__/g, '<strong>$1</strong>');

		// Italic (*text* or _text_)
		html = html.replace(/\*([^*]+)\*/g, '<em>$1</em>');
		html = html.replace(/_([^_]+)_/g, '<em>$1</em>');

		// Links [text](url)
		html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener" class="text-primary hover:underline">$1</a>');

		// Line breaks
		html = html.replace(/\n/g, '<br>');

		return html;
	}
</script>

<div bind:this={container} class="flex-1 overflow-y-auto px-4 py-6">
	<div class="mx-auto max-w-5xl space-y-6">
		{#each messages as message (message.id)}
			<div
				class={cn(
					'flex gap-4',
					message.role === 'user' ? 'flex-row-reverse' : 'flex-row'
				)}
			>
				<!-- Avatar -->
				<div
					class={cn(
						'flex h-8 w-8 shrink-0 items-center justify-center rounded-full',
						message.role === 'user'
							? 'bg-primary text-primary-foreground'
							: 'bg-muted text-muted-foreground'
					)}
				>
					{#if message.role === 'user'}
						<User class="h-4 w-4" />
					{:else}
						<Bot class="h-4 w-4" />
					{/if}
				</div>

				<!-- Message Content -->
				<div
					class={cn(
						'group relative max-w-[80%] rounded-2xl px-4 py-3',
						message.role === 'user'
							? 'bg-primary text-primary-foreground'
							: 'bg-muted text-foreground'
					)}
				>
					{#if message.role === 'user'}
						<div class="prose prose-sm dark:prose-invert max-w-none">
							<p class="m-0 whitespace-pre-wrap">{message.content}</p>
						</div>
					{:else}
						{@const parsed = parseThinkingContent(message.content)}
						
						<!-- Thinking Block (collapsible) -->
						{#if parsed.thinking}
							<div class="mb-3">
								<button
									onclick={() => toggleThinking(message.id)}
									class="flex w-full items-center gap-2 rounded-lg border border-purple-500/30 bg-purple-500/10 px-3 py-2 text-left text-sm transition-colors hover:bg-purple-500/20"
								>
									<Brain class="h-4 w-4 text-purple-500" />
									<span class="font-medium text-purple-500">Thinking</span>
									{#if expandedThinking.has(message.id)}
										<ChevronDown class="ml-auto h-4 w-4 text-purple-500" />
									{:else}
										<ChevronRight class="ml-auto h-4 w-4 text-purple-500" />
									{/if}
								</button>
								
								{#if expandedThinking.has(message.id)}
									<div class="mt-2 max-h-64 overflow-y-auto rounded-lg border border-purple-500/20 bg-purple-500/5 p-3 text-sm text-muted-foreground">
										<pre class="whitespace-pre-wrap font-sans">{parsed.thinking}</pre>
									</div>
								{/if}
							</div>
						{/if}
						
						<!-- Response -->
						{#if parsed.response}
							<div class="prose prose-sm dark:prose-invert max-w-none">
								{@html renderMarkdown(parsed.response)}
							</div>
						{:else if parsed.isThinking}
							<div class="flex items-center gap-2 text-sm text-muted-foreground">
								<Brain class="h-4 w-4 animate-pulse text-purple-500" />
								<span>Thinking...</span>
							</div>
						{/if}
					{/if}

					<!-- Copy Button -->
					<button
						onclick={() => copyToClipboard(message.content, message.id)}
						class={cn(
							'absolute -bottom-2 right-2 rounded-md bg-background/80 p-1.5 opacity-0 shadow-sm backdrop-blur transition-opacity group-hover:opacity-100',
							message.role === 'user' ? '-left-2 right-auto' : ''
						)}
						title="Copy"
					>
						{#if copiedId === message.id}
							<Check class="h-3.5 w-3.5 text-green-500" />
						{:else}
							<Copy class="h-3.5 w-3.5 text-muted-foreground" />
						{/if}
					</button>
				</div>
			</div>
		{/each}

		<!-- Streaming Message -->
		{#if isStreaming && streamingContent}
			{@const streamParsed = parseThinkingContent(streamingContent)}
			<div class="flex gap-4">
				<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
					{#if streamParsed.isThinking}
						<Brain class="h-4 w-4 animate-pulse text-purple-500" />
					{:else}
						<Bot class="h-4 w-4" />
					{/if}
				</div>
				<div class="max-w-[80%] rounded-2xl bg-muted px-4 py-3">
					<!-- Thinking indicator while streaming -->
					{#if streamParsed.isThinking}
						<div class="mb-3">
							<div class="flex items-center gap-2 rounded-lg border border-purple-500/30 bg-purple-500/10 px-3 py-2 text-sm">
								<Brain class="h-4 w-4 animate-pulse text-purple-500" />
								<span class="font-medium text-purple-500">Thinking...</span>
							</div>
							<div class="mt-2 max-h-48 overflow-y-auto rounded-lg border border-purple-500/20 bg-purple-500/5 p-3 text-sm text-muted-foreground">
								<pre class="whitespace-pre-wrap font-sans">{streamParsed.thinking}</pre>
							</div>
						</div>
					{/if}
					
					<!-- Completed thinking + response -->
					{#if streamParsed.thinking && !streamParsed.isThinking}
						<div class="mb-3">
							<div class="flex items-center gap-2 rounded-lg border border-green-500/30 bg-green-500/10 px-3 py-2 text-sm">
								<Brain class="h-4 w-4 text-green-500" />
								<span class="font-medium text-green-500">Thought complete</span>
							</div>
						</div>
					{/if}
					
					<!-- Response content -->
					{#if streamParsed.response}
						<div class="prose prose-sm dark:prose-invert max-w-none">
							{@html renderMarkdown(streamParsed.response)}
						</div>
					{/if}
					
					<span class="ml-1 inline-block h-4 w-1 animate-pulse bg-foreground"></span>
				</div>
			</div>
		{/if}
	</div>
</div>

