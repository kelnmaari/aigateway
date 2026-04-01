<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib';
	import Navbar from '$lib/components/navbar.svelte';

	let { children } = $props();
	let initialized = $state(false);

	onMount(() => {
		// Initialize stores
		authStore.init();

		// Check auth status
		if (!authStore.isAuthenticated) {
			goto('/login');
			return;
		}

		initialized = true;
	});
</script>

{#if initialized}
	<div class="min-h-screen bg-background">
		<Navbar />
		<main>
			{@render children()}
		</main>
	</div>
{:else}
	<div class="min-h-screen bg-background">
		<!-- Skeleton navbar -->
		<div class="h-16 border-b border-border bg-background/95"></div>
		<!-- Skeleton content -->
		<div class="mx-auto max-w-[1600px] px-4 py-8 sm:px-6 lg:px-8">
			<div class="mb-8 space-y-2">
				<div class="h-8 w-48 animate-pulse rounded-lg bg-muted"></div>
				<div class="h-4 w-72 animate-pulse rounded-lg bg-muted"></div>
			</div>
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				{#each Array(4) as _}
					<div class="h-24 animate-pulse rounded-lg border border-border bg-card"></div>
				{/each}
			</div>
		</div>
	</div>
{/if}
