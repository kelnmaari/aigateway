<script lang="ts">
	import { Settings, ChevronDown, Sparkles, Scale, Target, Code, Search } from 'lucide-svelte';
	import { cn } from '$lib/utils';
	import type { ChatParams } from '$lib/stores/chat.svelte';
	import * as m from '$lib/paraglide/messages';

	interface Props {
		models: Array<{ id: string; name?: string }>;
		selectedModel: string | null;
		params: ChatParams;
		onModelSelect: (model: string) => void;
		onParamsChange: (params: Partial<ChatParams>) => void;
		onPreset: (preset: 'creative' | 'balanced' | 'precise' | 'coding') => void;
	}

	let {
		models,
		selectedModel,
		params,
		onModelSelect,
		onParamsChange,
		onPreset
	}: Props = $props();

	let showParams = $state(false);

	const presets = [
		{ id: 'creative', label: m.chat_preset_creative, desc: m.chat_preset_creative_desc, icon: Sparkles, color: 'text-purple-500' },
		{ id: 'balanced', label: m.chat_preset_balanced, desc: m.chat_preset_balanced_desc, icon: Scale, color: 'text-blue-500' },
		{ id: 'precise', label: m.chat_preset_precise, desc: m.chat_preset_precise_desc, icon: Target, color: 'text-green-500' },
		{ id: 'coding', label: m.chat_preset_coding, desc: m.chat_preset_coding_desc, icon: Code, color: 'text-orange-500' }
	] as const;

	let activePreset = $state<string | null>(null);
</script>

<div class="border-b border-border bg-card/50 px-4 py-3">
	<div class="mx-auto flex max-w-5xl items-center gap-4">
		<!-- Model Select -->
		<div class="flex-1">
			<label for="model-select" class="sr-only">{m.chat_model()}</label>
			<select
				id="model-select"
				value={selectedModel || ''}
				onchange={(e) => onModelSelect(e.currentTarget.value)}
				class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
			>
				<option value="" disabled>Select a model...</option>
				{#each models as model (model.id)}
					<option value={model.id}>{model.name || model.id}</option>
				{/each}
			</select>
		</div>

		<!-- Presets -->
		<div class="hidden items-center gap-1 md:flex">
			{#each presets as preset}
				<button
					onclick={() => { onPreset(preset.id); activePreset = preset.id; }}
					class={cn(
						'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
						'hover:bg-accent',
						activePreset === preset.id ? 'bg-accent ring-1 ring-border' : '',
						preset.color
					)}
					title={preset.desc()}
				>
					<preset.icon class="h-3.5 w-3.5" />
					<span class="hidden lg:inline">{preset.label()}</span>
				</button>
			{/each}
		</div>

		<!-- Settings Toggle -->
		<button
			onclick={() => (showParams = !showParams)}
			class={cn(
				'flex items-center gap-1 rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent',
				showParams && 'bg-accent'
			)}
		>
			<Settings class="h-4 w-4" />
			<ChevronDown class={cn('h-3 w-3 transition-transform', showParams && 'rotate-180')} />
		</button>
	</div>

	<!-- Parameters Panel -->
	{#if showParams}
		<div class="mx-auto mt-4 max-w-5xl rounded-lg border border-border bg-background p-4">
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<!-- Temperature -->
				<div class="space-y-2">
					<div class="flex items-center justify-between">
						<label for="temperature" class="text-sm font-medium">{m.chat_temperature()}</label>
						<span class="text-xs text-muted-foreground">{params.temperature.toFixed(2)}</span>
					</div>
					<input
						id="temperature"
						type="range"
						min="0"
						max="2"
						step="0.1"
						value={params.temperature}
						oninput={(e) => onParamsChange({ temperature: parseFloat(e.currentTarget.value) })}
						class="w-full"
					/>
				</div>

				<!-- Top P -->
				<div class="space-y-2">
					<div class="flex items-center justify-between">
						<label for="top_p" class="text-sm font-medium">{m.chat_topP()}</label>
						<span class="text-xs text-muted-foreground">{params.top_p.toFixed(2)}</span>
					</div>
					<input
						id="top_p"
						type="range"
						min="0"
						max="1"
						step="0.05"
						value={params.top_p}
						oninput={(e) => onParamsChange({ top_p: parseFloat(e.currentTarget.value) })}
						class="w-full"
					/>
				</div>

				<!-- Max Tokens -->
				<div class="space-y-2">
					<div class="flex items-center justify-between">
						<label for="max_tokens" class="text-sm font-medium">{m.chat_maxTokens()}</label>
						<span class="text-xs text-muted-foreground">{params.max_tokens}</span>
					</div>
					<input
						id="max_tokens"
						type="range"
						min="256"
						max="32768"
						step="256"
						value={params.max_tokens}
						oninput={(e) => onParamsChange({ max_tokens: parseInt(e.currentTarget.value) })}
						class="w-full"
					/>
				</div>

				<!-- Context Window -->
				<div class="space-y-2">
					<div class="flex items-center justify-between">
						<label for="num_ctx" class="text-sm font-medium">{m.chat_contextWindow()}</label>
						<span class="text-xs text-muted-foreground">{params.num_ctx}</span>
					</div>
					<input
						id="num_ctx"
						type="range"
						min="2048"
						max="131072"
						step="2048"
						value={params.num_ctx}
						oninput={(e) => onParamsChange({ num_ctx: parseInt(e.currentTarget.value) })}
						class="w-full"
					/>
				</div>
			</div>
			
			<!-- Web Search Toggle -->
			<div class="mt-4 border-t border-border pt-4">
				<label class="flex items-center gap-3 cursor-pointer">
					<input
						type="checkbox"
						checked={params.use_tools ?? true}
						onchange={(e) => onParamsChange({ use_tools: e.currentTarget.checked })}
						class="h-4 w-4 rounded border-border text-primary focus:ring-primary"
					/>
					<Search class="h-4 w-4 text-blue-500" />
					<div class="flex-1">
						<span class="text-sm font-medium">{m.chat_use_tools()}</span>
						<p class="text-xs text-muted-foreground">{m.chat_use_tools_hint()}</p>
					</div>
				</label>
			</div>

			<!-- Mobile Presets -->
			<div class="mt-4 flex flex-wrap gap-2 md:hidden">
				{#each presets as preset}
					<button
						onclick={() => { onPreset(preset.id); activePreset = preset.id; }}
						class={cn(
							'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
							'border border-border hover:bg-accent',
							activePreset === preset.id ? 'bg-accent ring-1 ring-primary' : '',
							preset.color
						)}
						title={preset.desc()}
					>
						<preset.icon class="h-3.5 w-3.5" />
						{preset.label()}
					</button>
				{/each}
			</div>
		</div>
	{/if}
</div>

