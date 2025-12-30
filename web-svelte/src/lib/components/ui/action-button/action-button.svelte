<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props {
		label: string;
		onclick?: () => void;
		disabled?: boolean;
		loading?: boolean;
		class?: string;
		iconClass?: string;
		children: Snippet;
		loadingIcon?: Snippet;
	}

	let {
		label,
		onclick,
		disabled = false,
		loading = false,
		class: className = '',
		iconClass = '',
		children,
		loadingIcon
	}: Props = $props();
</script>

<button
	{onclick}
	{disabled}
	class={cn(
		'group relative flex items-center gap-0 rounded-md p-1.5 transition-all duration-200',
		'hover:bg-muted hover:pr-2',
		disabled && 'opacity-50 cursor-not-allowed',
		className
	)}
>
	<!-- Icon -->
	<span class={cn('flex-shrink-0 transition-transform duration-200', iconClass)}>
		{#if loading && loadingIcon}
			{@render loadingIcon()}
		{:else}
			{@render children()}
		{/if}
	</span>

	<!-- Expanding label -->
	<span
		class={cn(
			'overflow-hidden whitespace-nowrap text-xs font-medium',
			'max-w-0 opacity-0 transition-all duration-200 ease-out',
			'group-hover:max-w-[150px] group-hover:opacity-100 group-hover:ml-1.5'
		)}
	>
		{label}
	</span>
</button>

<style>
	/* Smooth width transition */
	button:hover span:last-child {
		max-width: 150px;
	}
</style>

