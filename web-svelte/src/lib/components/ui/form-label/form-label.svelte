<script lang="ts">
	import { cn } from '$lib/utils';
	import { HelpCircle } from 'lucide-svelte';
	import { Tooltip } from '$lib/components/ui/tooltip';

	interface Props {
		label: string;
		description?: string;
		required?: boolean;
		for?: string;
		class?: string;
	}

	let {
		label,
		description = '',
		required = false,
		for: htmlFor = '',
		class: className = ''
	}: Props = $props();
</script>

<div class={cn('flex items-center gap-1.5 mb-1.5', className)}>
	<label for={htmlFor} class="text-sm font-medium text-foreground">
		{label}
		{#if required}
			<span class="text-destructive">*</span>
		{/if}
	</label>
	{#if description}
		<Tooltip text={description} position="right">
			<HelpCircle class="h-3.5 w-3.5 text-muted-foreground/60 hover:text-primary cursor-help transition-colors" />
		</Tooltip>
	{/if}
</div>

