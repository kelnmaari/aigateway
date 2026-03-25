<script lang="ts">
	import { themeStore } from '$lib/stores/theme.svelte';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import { faSun, faMoon } from '@fortawesome/free-solid-svg-icons';
	import { cn } from '$lib/utils';

	interface Props {
		class?: string;
	}

	let { class: className }: Props = $props();
</script>

<button
	onclick={() => themeStore.toggle()}
	class={cn(
		'inline-flex h-10 w-10 items-center justify-center rounded-md text-muted-foreground transition-colors',
		'hover:bg-accent hover:text-accent-foreground',
		'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
		className
	)}
	title={themeStore.isDark ? 'Switch to light mode' : 'Switch to dark mode'}
>
	{#if themeStore.isDark}
		<FontAwesomeIcon icon={faSun} class="h-5 w-5" />
	{:else}
		<FontAwesomeIcon icon={faMoon} class="h-5 w-5" />
	{/if}
	<span class="sr-only">Toggle theme</span>
</button>
