<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';

	onMount(() => {
		// Initialize auth store
		authStore.init();
		
		// Redirect based on auth status
		if (authStore.isAuthenticated) {
			goto('/dashboard');
		} else {
			goto('/login');
		}
	});
</script>

<div class="flex min-h-screen items-center justify-center bg-background">
	<div class="flex flex-col items-center gap-4">
		<div class="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent"></div>
		<p class="text-muted-foreground">Loading...</p>
	</div>
</div>

