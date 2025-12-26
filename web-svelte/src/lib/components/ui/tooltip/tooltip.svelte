<script lang="ts">
	import { cn } from '$lib/utils';

	interface Props {
		text?: string;
		title?: string;
		position?: 'top' | 'bottom' | 'left' | 'right';
		class?: string;
		children?: import('svelte').Snippet;
		content?: import('svelte').Snippet;
	}

	let {
		text = '',
		title = '',
		position = 'top',
		class: className = '',
		children,
		content
	}: Props = $props();

	let visible = $state(false);
	let tooltipEl: HTMLDivElement | null = $state(null);

	const positionClasses = {
		top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
		bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
		left: 'right-full top-1/2 -translate-y-1/2 mr-2',
		right: 'left-full top-1/2 -translate-y-1/2 ml-2'
	};

	const arrowClasses = {
		top: 'top-full left-1/2 -translate-x-1/2 border-t-card border-x-transparent border-b-transparent',
		bottom: 'bottom-full left-1/2 -translate-x-1/2 border-b-card border-x-transparent border-t-transparent',
		left: 'left-full top-1/2 -translate-y-1/2 border-l-card border-y-transparent border-r-transparent',
		right: 'right-full top-1/2 -translate-y-1/2 border-r-card border-y-transparent border-l-transparent'
	};
</script>

<div
	class={cn('relative inline-flex', className)}
	onmouseenter={() => (visible = true)}
	onmouseleave={() => (visible = false)}
	onfocus={() => (visible = true)}
	onblur={() => (visible = false)}
	role="tooltip"
>
	{@render children?.()}

	{#if visible && (text || title || content)}
		<div
			bind:this={tooltipEl}
			class={cn(
				'absolute z-50 pointer-events-none',
				'animate-in fade-in-0 zoom-in-95 duration-150',
				positionClasses[position]
			)}
		>
			<div
				class={cn(
					'relative rounded-lg border border-border/50 bg-card px-3 py-2 shadow-xl',
					'text-sm text-foreground',
					'min-w-max max-w-xs'
				)}
			>
				{#if title}
					<div class="flex items-center gap-1.5 font-semibold text-primary mb-1">
						{title}
					</div>
				{/if}
				{#if content}
					{@render content()}
				{:else if text}
					<p class="text-muted-foreground text-xs leading-relaxed whitespace-pre-line">{text}</p>
				{/if}
				<!-- Arrow -->
				<div
					class={cn(
						'absolute w-0 h-0 border-4',
						arrowClasses[position]
					)}
				></div>
			</div>
		</div>
	{/if}
</div>

