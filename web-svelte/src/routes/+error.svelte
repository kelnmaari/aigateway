<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import { faTriangleExclamation, faHouse, faArrowLeft } from '@fortawesome/free-solid-svg-icons';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';
</script>

<svelte:head>
	<title>{$page.status === 404 ? m.error_notFound() : m.common_error()} | AI Gateway</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-background px-4">
	<div class="text-center">
		<div class="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-destructive/10">
			<FontAwesomeIcon icon={faTriangleExclamation} class="h-10 w-10 text-destructive" />
		</div>

		<h1 class="mb-2 text-6xl font-bold text-foreground">{$page.status}</h1>

		<p class="mb-2 text-xl font-medium text-foreground">
			{#if $page.status === 404}
				{m.error_notFound()}
			{:else if $page.status === 401}
				{m.error_unauthorized()}
			{:else if $page.status >= 500}
				{m.error_serverError()}
			{:else}
				{m.common_error()}
			{/if}
		</p>

		{#if $page.error?.message}
			<p class="mb-8 text-muted-foreground">{$page.error.message}</p>
		{:else}
			<p class="mb-8 text-muted-foreground">
				{#if $page.status === 404}
					The page you're looking for doesn't exist or has been moved.
				{:else}
					Something went wrong. Please try again later.
				{/if}
			</p>
		{/if}

		<div class="flex justify-center gap-4">
			<Button variant="outline" onclick={() => history.back()}>
				<FontAwesomeIcon icon={faArrowLeft} class="mr-2 h-4 w-4" />
				{m.common_back()}
			</Button>
			<Button onclick={() => goto('/dashboard')}>
				<FontAwesomeIcon icon={faHouse} class="mr-2 h-4 w-4" />
				{m.nav_dashboard()}
			</Button>
		</div>
	</div>
</div>
