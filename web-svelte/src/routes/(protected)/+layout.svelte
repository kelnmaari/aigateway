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
	<div class="flex min-h-screen items-center justify-center bg-background">
		<div class="animate-pulse text-muted-foreground">Loading...</div>
	</div>
{/if}
