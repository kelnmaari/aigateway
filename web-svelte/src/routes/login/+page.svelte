<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import { faEye, faEyeSlash, faSpinner, faKey } from '@fortawesome/free-solid-svg-icons';
	import { goto } from '$app/navigation';
	import { authApi } from '$lib/api';
	import { authStore } from '$lib/stores/auth.svelte';
	import * as m from '$lib/paraglide/messages';
	import { onMount } from 'svelte';

	let username = $state('');
	let password = $state('');
	let showPassword = $state(false);
	let isLoading = $state(false);
	let error = $state('');
	let oidcEnabled = $state(false);
	let ldapEnabled = $state(false);

	onMount(async () => {
		try {
			const resp = await fetch('/api/auth/providers');
			if (resp.ok) {
				const data = await resp.json();
				oidcEnabled = data.oidc_enabled ?? false;
				ldapEnabled = data.ldap_enabled ?? false;
			}
		} catch {
			// Ignore - just don't show SSO buttons
		}
	});

	function loginWithSSO() {
		window.location.href = '/api/auth/oidc/login';
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		isLoading = true;

		try {
			const response = await authApi.login({ username, password });
			authStore.login(
				{
					access_token: response.token.access_token,
					refresh_token: response.token.refresh_token,
					expires_at: new Date(response.token.expires_at).getTime()
				},
				response.user
			);
			goto('/dashboard');
		} catch (err) {
			error = err instanceof Error ? err.message : m.error_invalidCredentials();
		} finally {
			isLoading = false;
		}
	}
</script>

<svelte:head>
	<title>{m.auth_login()} | AI Gateway</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center bg-background p-4">
	<Card class="w-full max-w-md">
		<CardHeader class="space-y-1 text-center">
			<CardTitle class="text-2xl font-bold">{m.auth_welcomeBack()}</CardTitle>
			<CardDescription>{m.auth_signIn()}</CardDescription>
		</CardHeader>
		<CardContent>
			<form onsubmit={handleSubmit} class="space-y-4">
				{#if error}
					<div class="p-3 text-sm text-destructive bg-destructive/10 rounded-md">
						{error}
					</div>
				{/if}

				<div class="space-y-2">
					<label for="username" class="text-sm font-medium">{m.auth_username()}</label>
					<input
						id="username"
						type="text"
						bind:value={username}
						required
						disabled={isLoading}
						class="w-full px-3 py-2 border rounded-md bg-background text-foreground border-input focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50"
						placeholder="admin"
					/>
				</div>

				<div class="space-y-2">
					<label for="password" class="text-sm font-medium">{m.auth_password()}</label>
					<div class="relative">
						<input
							id="password"
							type={showPassword ? 'text' : 'password'}
							bind:value={password}
							required
							disabled={isLoading}
							class="w-full px-3 py-2 pr-10 border rounded-md bg-background text-foreground border-input focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50"
							placeholder="••••••••"
						/>
						<button
							type="button"
							onclick={() => showPassword = !showPassword}
							class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
						>
							{#if showPassword}
								<FontAwesomeIcon icon={faEyeSlash} class="h-4 w-4" />
							{:else}
								<FontAwesomeIcon icon={faEye} class="h-4 w-4" />
							{/if}
						</button>
					</div>
				</div>

				<Button type="submit" class="w-full" disabled={isLoading}>
					{#if isLoading}
						<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
						{m.common_loading()}
					{:else}
						{m.auth_login()}
					{/if}
				</Button>

				{#if oidcEnabled}
					<div class="relative my-4">
						<div class="absolute inset-0 flex items-center">
							<span class="w-full border-t"></span>
						</div>
						<div class="relative flex justify-center text-xs uppercase">
							<span class="bg-background px-2 text-muted-foreground">{m.auth_or()}</span>
						</div>
					</div>

					<Button
						type="button"
						variant="outline"
						class="w-full"
						onclick={loginWithSSO}
						disabled={isLoading}
					>
						<FontAwesomeIcon icon={faKey} class="mr-2 h-4 w-4" />
						{m.auth_login_with_sso()}
					</Button>
				{/if}

				<div class="text-center text-sm">
					<span class="text-muted-foreground">{m.auth_noAccount()} </span>
					<a href="/register" class="text-primary hover:underline">{m.auth_signUp()}</a>
				</div>

				<div class="text-center text-sm">
					<a href="/bootstrap" class="text-muted-foreground hover:text-foreground">
						{m.auth_setupAdmin()}
					</a>
				</div>
			</form>
		</CardContent>
	</Card>
</div>
