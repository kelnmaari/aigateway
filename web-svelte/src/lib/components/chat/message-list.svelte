<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { User, Bot, Copy, Check } from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import type { Message } from '$lib/stores/chat.svelte';

	interface Props {
		messages: Message[];
		streamingContent?: string;
		isStreaming?: boolean;
	}

	let { messages, streamingContent = '', isStreaming = false }: Props = $props();

	let container: HTMLDivElement;
	let copiedId: string | null = $state(null);

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
		try {
			await navigator.clipboard.writeText(content);
			copiedId = id;
			setTimeout(() => {
				copiedId = null;
			}, 2000);
		} catch {
			console.error('Failed to copy');
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
	<div class="mx-auto max-w-3xl space-y-6">
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
					<div class="prose prose-sm dark:prose-invert max-w-none">
						{#if message.role === 'user'}
							<p class="m-0 whitespace-pre-wrap">{message.content}</p>
						{:else}
							{@html renderMarkdown(message.content)}
						{/if}
					</div>

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
			<div class="flex gap-4">
				<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
					<Bot class="h-4 w-4" />
				</div>
				<div class="max-w-[80%] rounded-2xl bg-muted px-4 py-3">
					<div class="prose prose-sm dark:prose-invert max-w-none">
						{@html renderMarkdown(streamingContent)}
					</div>
					<span class="ml-1 inline-block h-4 w-1 animate-pulse bg-foreground"></span>
				</div>
			</div>
		{/if}
	</div>
</div>

