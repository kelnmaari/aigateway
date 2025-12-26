<script lang="ts">
	import { cn } from '$lib/utils';
	import { Tooltip } from '$lib/components/ui/tooltip';

	interface Props {
		tooltip?: string;
		tooltipTitle?: string;
		tooltipPosition?: 'top' | 'bottom' | 'left' | 'right';
		variant?: 'default' | 'ghost' | 'destructive' | 'success';
		size?: 'sm' | 'md' | 'lg';
		disabled?: boolean;
		class?: string;
		onclick?: (e: MouseEvent) => void;
		children?: import('svelte').Snippet;
	}

	let {
		tooltip = '',
		tooltipTitle = '',
		tooltipPosition = 'top',
		variant = 'ghost',
		size = 'md',
		disabled = false,
		class: className = '',
		onclick,
		children
	}: Props = $props();

	const sizeClasses = {
		sm: 'h-7 w-7',
		md: 'h-8 w-8',
		lg: 'h-10 w-10'
	};

	const iconSizeClasses = {
		sm: '[&>svg]:h-3.5 [&>svg]:w-3.5',
		md: '[&>svg]:h-4 [&>svg]:w-4',
		lg: '[&>svg]:h-5 [&>svg]:w-5'
	};

	const variantClasses = {
		default: 'text-foreground hover:bg-accent hover:text-accent-foreground',
		ghost: 'text-muted-foreground hover:bg-accent hover:text-foreground',
		destructive: 'text-muted-foreground hover:bg-destructive/10 hover:text-destructive',
		success: 'text-muted-foreground hover:bg-green-500/10 hover:text-green-500'
	};
</script>

<Tooltip text={tooltip} title={tooltipTitle} position={tooltipPosition}>
	<button
		type="button"
		{disabled}
		{onclick}
		class={cn(
			'inline-flex items-center justify-center rounded-lg transition-colors',
			'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
			'disabled:pointer-events-none disabled:opacity-50',
			sizeClasses[size],
			iconSizeClasses[size],
			variantClasses[variant],
			className
		)}
	>
		{@render children?.()}
	</button>
</Tooltip>

