<script lang="ts">
	import { MessageSquare, Plus, Trash2, MoreVertical } from 'lucide-svelte';
	import { cn, formatRelativeTime } from '$lib/utils';
	import type { Conversation } from '$lib/stores/chat.svelte';
	import * as m from '$lib/paraglide/messages';

	interface Props {
		conversations: Conversation[];
		currentId: string | null;
		onSelect: (id: string) => void;
		onNew: () => void;
		onDelete: (id: string) => void;
	}

	let { conversations, currentId, onSelect, onNew, onDelete }: Props = $props();

	let menuOpenId = $state<string | null>(null);

	function toggleMenu(id: string, e: MouseEvent) {
		e.stopPropagation();
		menuOpenId = menuOpenId === id ? null : id;
	}

	function handleDelete(id: string, e: MouseEvent) {
		e.stopPropagation();
		menuOpenId = null;
		onDelete(id);
	}
</script>

<svelte:window
	onclick={() => {
		menuOpenId = null;
	}}
/>

<div class="flex h-full flex-col">
	<!-- New Chat Button -->
	<div class="p-3">
		<button
			onclick={onNew}
			class="flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-border bg-background px-4 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:border-primary hover:bg-accent hover:text-foreground"
		>
			<Plus class="h-4 w-4" />
			{m.chat_newChat()}
		</button>
	</div>

	<!-- Conversations List -->
	<div class="flex-1 overflow-y-auto px-2">
		{#if conversations.length === 0}
			<div class="flex flex-col items-center justify-center py-12 text-center">
				<MessageSquare class="h-10 w-10 text-muted-foreground/40" />
				<p class="mt-3 text-sm text-muted-foreground">No conversations yet</p>
			</div>
		{:else}
			<div class="space-y-1 pb-4">
				{#each conversations as conversation (conversation.id)}
					<div
						role="button"
						tabindex="0"
						onclick={() => onSelect(conversation.id)}
						onkeydown={(e) => e.key === 'Enter' && onSelect(conversation.id)}
						class={cn(
							'group relative flex cursor-pointer items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-colors',
							currentId === conversation.id
								? 'bg-accent text-accent-foreground'
								: 'text-muted-foreground hover:bg-accent/50 hover:text-foreground'
						)}
					>
						<MessageSquare class="h-4 w-4 shrink-0" />
						<div class="min-w-0 flex-1">
							<p class="truncate font-medium">{conversation.title || 'Untitled'}</p>
							<p class="text-xs opacity-60">{formatRelativeTime(conversation.updated_at)}</p>
						</div>

						<!-- Actions Menu -->
						<div class="relative">
							<button
								onclick={(e) => toggleMenu(conversation.id, e)}
								class="rounded p-1 opacity-0 transition-opacity hover:bg-background/50 group-hover:opacity-100"
								class:opacity-100={menuOpenId === conversation.id}
							>
								<MoreVertical class="h-4 w-4" />
							</button>

							{#if menuOpenId === conversation.id}
								<div
									class="absolute right-0 top-full z-10 mt-1 w-36 rounded-md border border-border bg-card py-1 shadow-lg"
								>
									<button
										onclick={(e) => handleDelete(conversation.id, e)}
										class="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-destructive hover:bg-destructive/10"
									>
										<Trash2 class="h-4 w-4" />
										{m.common_delete()}
									</button>
								</div>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

