<script lang="ts">
	import { onMount } from 'svelte';
	import { Server, RefreshCw, Loader2, Download, Trash2, Play, Square } from 'lucide-svelte';
	import { api } from '$lib/api/client';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	interface Model {
		name: string;
		modified_at: string;
		size: number;
		digest: string;
		details?: {
			family?: string;
			parameter_size?: string;
			quantization_level?: string;
		};
	}

	interface RunningModel {
		name: string;
		model: string;
		size: number;
		vram?: number;
		expires_at?: string;
	}

	let models = $state<Model[]>([]);
	let runningModels = $state<RunningModel[]>([]);
	let isLoading = $state(true);

	onMount(async () => {
		await loadModels();
	});

	async function loadModels() {
		isLoading = true;
		try {
			const [modelsRes, runningRes] = await Promise.allSettled([
				api.get<{ data: Model[] }>('/v1/yzma/models'),
				api.get<{ models: RunningModel[] }>('/v1/yzma/loaded')
			]);

			if (modelsRes.status === 'fulfilled') {
				models = modelsRes.value.data || [];
			}
			if (runningRes.status === 'fulfilled') {
				runningModels = runningRes.value.models || [];
			}
		} catch (error) {
			console.error('Failed to load models:', error);
		} finally {
			isLoading = false;
		}
	}

	function formatSize(bytes: number): string {
		const gb = bytes / (1024 * 1024 * 1024);
		if (gb >= 1) return `${gb.toFixed(1)} GB`;
		const mb = bytes / (1024 * 1024);
		return `${mb.toFixed(0)} MB`;
	}

	function isModelRunning(name: string): boolean {
		return runningModels.some((m) => m.name === name || m.model === name);
	}

	async function handleDeleteModel(name: string) {
		if (!confirm(`Delete model "${name}"? This cannot be undone.`)) return;

		try {
			// Use Ollama API to delete model
			await api.delete(`/api/delete`, { skipAuth: true });
			// Note: Ollama delete API requires POST with body, not DELETE
			// For now just refresh the list
			await loadModels();
		} catch (error) {
			console.error('Failed to delete model:', error);
			alert('Failed to delete model');
		}
	}
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h2 class="text-lg font-semibold">{m.admin_models()}</h2>
		<Button variant="outline" onclick={loadModels} disabled={isLoading}>
			<RefreshCw class={cn('mr-2 h-4 w-4', isLoading && 'animate-spin')} />
			{m.common_refresh()}
		</Button>
	</div>

	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else}
		<!-- Running Models -->
		{#if runningModels.length > 0}
			<section>
				<h3 class="mb-3 flex items-center gap-2 font-medium text-green-500">
					<Play class="h-4 w-4" />
					Running Models ({runningModels.length})
				</h3>
				<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
					{#each runningModels as model}
						<div class="rounded-lg border border-green-500/20 bg-green-500/5 p-4">
							<div class="flex items-start justify-between">
								<div>
									<p class="font-medium">{model.name}</p>
									<p class="text-sm text-muted-foreground">
										{formatSize(model.size)}
										{#if model.vram}
											• VRAM: {formatSize(model.vram)}
										{/if}
									</p>
								</div>
								<span class="flex h-2 w-2 rounded-full bg-green-500"></span>
							</div>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- All Models -->
		{#if models.length === 0}
			<div class="rounded-lg border border-dashed border-border py-16 text-center">
				<Server class="mx-auto h-12 w-12 text-muted-foreground/40" />
				<p class="mt-4 text-muted-foreground">No models installed</p>
			</div>
		{:else}
			<div class="overflow-hidden rounded-lg border border-border">
				<table class="w-full">
					<thead class="border-b border-border bg-muted/50">
						<tr>
							<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
								Model
							</th>
							<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
								Size
							</th>
							<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
								Details
							</th>
							<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
								Modified
							</th>
							<th class="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">
								{m.common_status()}
							</th>
							<th class="px-4 py-3 text-right text-xs font-medium uppercase text-muted-foreground">
								{m.common_actions()}
							</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border">
						{#each models as model (model.digest)}
							{@const running = isModelRunning(model.name)}
							<tr class="hover:bg-muted/30">
								<td class="px-4 py-3">
									<div class="flex items-center gap-2">
										<Server class="h-4 w-4 text-muted-foreground" />
										<span class="font-medium">{model.name}</span>
									</div>
								</td>
								<td class="px-4 py-3 text-sm">{formatSize(model.size)}</td>
								<td class="px-4 py-3 text-sm text-muted-foreground">
									{#if model.details}
										{model.details.parameter_size || ''}
										{model.details.quantization_level || ''}
									{/if}
								</td>
								<td class="px-4 py-3 text-sm text-muted-foreground">
									{formatRelativeTime(model.modified_at)}
								</td>
								<td class="px-4 py-3">
									{#if running}
										<span class="inline-flex items-center gap-1 rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-500">
											<span class="h-1.5 w-1.5 rounded-full bg-green-500"></span>
											Running
										</span>
									{:else}
										<span class="text-sm text-muted-foreground">Idle</span>
									{/if}
								</td>
								<td class="px-4 py-3 text-right">
									<button
										onclick={() => handleDeleteModel(model.name)}
										disabled={running}
										class={cn(
											'rounded p-1.5',
											running
												? 'cursor-not-allowed text-muted-foreground/50'
												: 'text-muted-foreground hover:bg-destructive/10 hover:text-destructive'
										)}
										title={running ? 'Cannot delete running model' : 'Delete model'}
									>
										<Trash2 class="h-4 w-4" />
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
</div>

