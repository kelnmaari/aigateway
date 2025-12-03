<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { authStore } from '$lib';
	import { authApi } from '$lib/api';
	import { cn } from '$lib/utils';
	import { Eye, EyeOff, Loader2, Shield, CheckCircle, XCircle } from 'lucide-svelte';

	let bootstrapToken = $state('');
	let username = $state('');
	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let showPassword = $state(false);
	let loading = $state(false);
	let error = $state('');
	let checkingStatus = $state(true);
	let alreadyInitialized = $state(false);

	// Password validation
	const passwordRequirements = $derived({
		minLength: password.length >= 8,
		hasUppercase: /[A-Z]/.test(password),
		hasLowercase: /[a-z]/.test(password),
		hasNumber: /\d/.test(password)
	});

	const isPasswordValid = $derived(
		passwordRequirements.minLength &&
			passwordRequirements.hasUppercase &&
			passwordRequirements.hasLowercase &&
			passwordRequirements.hasNumber
	);

	const passwordsMatch = $derived(password === confirmPassword && password.length > 0);

	onMount(async () => {
		try {
			const status = await authApi.getInitStatus();
			if (status.initialized || status.has_users) {
				alreadyInitialized = true;
			}
		} catch (err) {
			// If we can't check, assume we need bootstrap
		} finally {
			checkingStatus = false;
		}
	});

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';

		if (!isPasswordValid) {
			error = 'Password does not meet requirements';
			return;
		}

		if (!passwordsMatch) {
			error = 'Passwords do not match';
			return;
		}

		loading = true;

		try {
			const response = await authApi.bootstrap({
				bootstrap_token: bootstrapToken,
				username,
				email,
				password
			});

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
			error = err instanceof Error ? err.message : 'Bootstrap failed';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Setup Administrator - AIGateway</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-background p-4">
	<div class="w-full max-w-md">
		<!-- Logo -->
		<div class="mb-8 text-center">
			<div
				class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-primary text-3xl text-primary-foreground"
			>
				<Shield class="h-8 w-8" />
			</div>
			<h1 class="text-2xl font-bold text-foreground">Initial Setup</h1>
			<p class="mt-2 text-muted-foreground">Create the first administrator account</p>
		</div>

		<!-- Bootstrap Form -->
		<div class="rounded-xl border border-border bg-card p-6 shadow-sm">
			{#if checkingStatus}
				<div class="flex items-center justify-center gap-2 py-8 text-muted-foreground">
					<Loader2 class="h-5 w-5 animate-spin" />
					Checking system status...
				</div>
			{:else if alreadyInitialized}
				<div class="py-8 text-center">
					<CheckCircle class="mx-auto mb-4 h-12 w-12 text-green-500" />
					<h2 class="mb-2 text-lg font-semibold text-foreground">Already Initialized</h2>
					<p class="mb-6 text-muted-foreground">
						This system has already been set up. Please sign in with your credentials.
					</p>
					<a
						href="/login"
						class={cn(
							'inline-flex items-center justify-center rounded-lg bg-primary px-6 py-2 text-sm font-medium text-primary-foreground',
							'hover:bg-primary/90'
						)}
					>
						Go to Login
					</a>
				</div>
			{:else}
				<div class="mb-6 rounded-lg bg-amber-500/10 p-4 text-sm text-amber-600 dark:text-amber-400">
					<p class="font-medium">⚠️ Important</p>
					<p class="mt-1">
						This will create the first administrator account. The bootstrap token can be found in
						the server configuration file or environment variables.
					</p>
				</div>

				<form onsubmit={handleSubmit} class="space-y-4">
					{#if error}
						<div class="rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
							{error}
						</div>
					{/if}

					<div class="space-y-2">
						<label for="token" class="text-sm font-medium text-foreground"> Bootstrap Token </label>
						<input
							id="token"
							type="password"
							bind:value={bootstrapToken}
							placeholder="Enter the bootstrap token"
							required
							class={cn(
								'w-full rounded-lg border border-input bg-background px-3 py-2 text-sm',
								'placeholder:text-muted-foreground',
								'focus:outline-none focus:ring-2 focus:ring-ring'
							)}
						/>
						<p class="text-xs text-muted-foreground">
							Found in configs/bootstrap or BOOTSTRAP_TOKEN env var
						</p>
					</div>

					<div class="space-y-2">
						<label for="username" class="text-sm font-medium text-foreground">
							Admin Username
						</label>
						<input
							id="username"
							type="text"
							bind:value={username}
							placeholder="admin"
							required
							minlength="3"
							maxlength="50"
							pattern="[a-zA-Z0-9_\-]+"
							class={cn(
								'w-full rounded-lg border border-input bg-background px-3 py-2 text-sm',
								'placeholder:text-muted-foreground',
								'focus:outline-none focus:ring-2 focus:ring-ring'
							)}
						/>
					</div>

					<div class="space-y-2">
						<label for="email" class="text-sm font-medium text-foreground"> Admin Email </label>
						<input
							id="email"
							type="email"
							bind:value={email}
							placeholder="admin@example.com"
							required
							class={cn(
								'w-full rounded-lg border border-input bg-background px-3 py-2 text-sm',
								'placeholder:text-muted-foreground',
								'focus:outline-none focus:ring-2 focus:ring-ring'
							)}
						/>
					</div>

					<div class="space-y-2">
						<label for="password" class="text-sm font-medium text-foreground">
							Admin Password
						</label>
						<div class="relative">
							<input
								id="password"
								type={showPassword ? 'text' : 'password'}
								bind:value={password}
								placeholder="Create a strong password"
								required
								minlength="8"
								class={cn(
									'w-full rounded-lg border border-input bg-background px-3 py-2 pr-10 text-sm',
									'placeholder:text-muted-foreground',
									'focus:outline-none focus:ring-2 focus:ring-ring'
								)}
							/>
							<button
								type="button"
								onclick={() => (showPassword = !showPassword)}
								class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
							>
								{#if showPassword}
									<EyeOff class="h-4 w-4" />
								{:else}
									<Eye class="h-4 w-4" />
								{/if}
							</button>
						</div>

						<!-- Password requirements -->
						<div class="mt-2 space-y-1 text-xs">
							<div class="flex items-center gap-1.5">
								{#if passwordRequirements.minLength}
									<CheckCircle class="h-3.5 w-3.5 text-green-500" />
								{:else}
									<XCircle class="h-3.5 w-3.5 text-muted-foreground" />
								{/if}
								<span class={passwordRequirements.minLength ? 'text-green-500' : 'text-muted-foreground'}>
									At least 8 characters
								</span>
							</div>
							<div class="flex items-center gap-1.5">
								{#if passwordRequirements.hasUppercase}
									<CheckCircle class="h-3.5 w-3.5 text-green-500" />
								{:else}
									<XCircle class="h-3.5 w-3.5 text-muted-foreground" />
								{/if}
								<span class={passwordRequirements.hasUppercase ? 'text-green-500' : 'text-muted-foreground'}>
									One uppercase letter
								</span>
							</div>
							<div class="flex items-center gap-1.5">
								{#if passwordRequirements.hasLowercase}
									<CheckCircle class="h-3.5 w-3.5 text-green-500" />
								{:else}
									<XCircle class="h-3.5 w-3.5 text-muted-foreground" />
								{/if}
								<span class={passwordRequirements.hasLowercase ? 'text-green-500' : 'text-muted-foreground'}>
									One lowercase letter
								</span>
							</div>
							<div class="flex items-center gap-1.5">
								{#if passwordRequirements.hasNumber}
									<CheckCircle class="h-3.5 w-3.5 text-green-500" />
								{:else}
									<XCircle class="h-3.5 w-3.5 text-muted-foreground" />
								{/if}
								<span class={passwordRequirements.hasNumber ? 'text-green-500' : 'text-muted-foreground'}>
									One number
								</span>
							</div>
						</div>
					</div>

					<div class="space-y-2">
						<label for="confirm-password" class="text-sm font-medium text-foreground">
							Confirm Password
						</label>
						<input
							id="confirm-password"
							type="password"
							bind:value={confirmPassword}
							placeholder="Confirm your password"
							required
							class={cn(
								'w-full rounded-lg border bg-background px-3 py-2 text-sm',
								'placeholder:text-muted-foreground',
								'focus:outline-none focus:ring-2 focus:ring-ring',
								confirmPassword.length > 0 && !passwordsMatch
									? 'border-destructive'
									: 'border-input'
							)}
						/>
						{#if confirmPassword.length > 0 && !passwordsMatch}
							<p class="text-xs text-destructive">Passwords do not match</p>
						{/if}
					</div>

					<button
						type="submit"
						disabled={loading || !isPasswordValid || !passwordsMatch || !bootstrapToken}
						class={cn(
							'flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground',
							'hover:bg-primary/90 focus:outline-none focus:ring-2 focus:ring-ring',
							'disabled:cursor-not-allowed disabled:opacity-50'
						)}
					>
						{#if loading}
							<Loader2 class="h-4 w-4 animate-spin" />
							Setting up...
						{:else}
							<Shield class="h-4 w-4" />
							Create Administrator
						{/if}
					</button>
				</form>
			{/if}
		</div>

		<!-- Footer -->
		<p class="mt-8 text-center text-xs text-muted-foreground">
			This page is only available during initial setup
		</p>
	</div>
</div>

