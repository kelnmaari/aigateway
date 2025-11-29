<script lang="ts">
	import { Send, Square, Loader2 } from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import * as m from '$lib/paraglide/messages';

	interface Props {
		onSend: (content: string) => void;
		onStop?: () => void;
		isStreaming?: boolean;
		disabled?: boolean;
	}

	let { onSend, onStop, isStreaming = false, disabled = false }: Props = $props();

	let content = $state('');
	let textarea: HTMLTextAreaElement;

	function handleSubmit(e: Event) {
		e.preventDefault();
		if (!content.trim() || disabled || isStreaming) return;
		onSend(content.trim());
		content = '';
		resetTextareaHeight();
	}

	function handleKeydown(e: KeyboardEvent) {
		// Submit on Enter (but not Shift+Enter for new line)
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleSubmit(e);
		}
	}

	function handleInput() {
		// Auto-resize textarea
		if (textarea) {
			textarea.style.height = 'auto';
			textarea.style.height = Math.min(textarea.scrollHeight, 200) + 'px';
		}
	}

	function resetTextareaHeight() {
		if (textarea) {
			textarea.style.height = 'auto';
		}
	}

	function handleStop() {
		if (onStop) {
			onStop();
		}
	}
</script>

<form onsubmit={handleSubmit} class="border-t border-border bg-background p-4">
	<div class="mx-auto max-w-3xl">
		<div class="relative flex items-end gap-2">
			<div class="relative flex-1">
				<textarea
					bind:this={textarea}
					bind:value={content}
					onkeydown={handleKeydown}
					oninput={handleInput}
					placeholder={m.chat_typeMessage()}
					disabled={disabled}
					rows="1"
					class={cn(
						'w-full resize-none rounded-xl border border-input bg-background px-4 py-3 pr-12 text-sm',
						'placeholder:text-muted-foreground',
						'focus:outline-none focus:ring-2 focus:ring-ring',
						'disabled:cursor-not-allowed disabled:opacity-50',
						'min-h-[48px] max-h-[200px]'
					)}
				></textarea>
			</div>

			<!-- Send / Stop Button -->
			{#if isStreaming}
				<button
					type="button"
					onclick={handleStop}
					class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-destructive text-destructive-foreground transition-colors hover:bg-destructive/90"
					title="Stop generating"
				>
					<Square class="h-4 w-4" />
				</button>
			{:else}
				<button
					type="submit"
					disabled={!content.trim() || disabled}
					class={cn(
						'flex h-10 w-10 shrink-0 items-center justify-center rounded-lg transition-colors',
						content.trim() && !disabled
							? 'bg-primary text-primary-foreground hover:bg-primary/90'
							: 'bg-muted text-muted-foreground cursor-not-allowed'
					)}
					title={m.chat_sendMessage()}
				>
					{#if disabled && !isStreaming}
						<Loader2 class="h-4 w-4 animate-spin" />
					{:else}
						<Send class="h-4 w-4" />
					{/if}
				</button>
			{/if}
		</div>

		<p class="mt-2 text-center text-xs text-muted-foreground">
			Press Enter to send, Shift+Enter for new line
		</p>
	</div>
</form>

