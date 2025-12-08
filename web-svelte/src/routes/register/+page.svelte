<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { authStore } from '$lib';
	import { authApi } from '$lib/api';
	import { cn } from '$lib/utils';
	import { Eye, EyeOff, Loader2, CheckCircle, XCircle } from 'lucide-svelte';

	// Get invitation token from URL if present
	let invitationToken = $state('');

	let username = $state('');
	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let fullName = $state('');
	let showPassword = $state(false);
	let loading = $state(false);
	let error = $state('');
	let invitationEmail = $state<string | null>(null);
	let validatingInvitation = $state(false);

	// Password validation
	const passwordRequirements = $derived({
		minLength: password.length >= 8,
		hasUppercase: /[A-Z]/.test(password),
		hasLowercase: /[a-z]/.test(password),
		hasNumber: /\d/.test(password),
		hasSpecial: /[!@#$%^&*(),.?":{}|<>]/.test(password)
	});

	const isPasswordValid = $derived(
		passwordRequirements.minLength &&
			passwordRequirements.hasUppercase &&
			passwordRequirements.hasLowercase &&
			passwordRequirements.hasNumber
	);

	const passwordsMatch = $derived(password === confirmPassword && password.length > 0);

	onMount(() => {
		// Check for invitation token in URL
		const urlToken = $page.url.searchParams.get('token');
		if (urlToken) {
			invitationToken = urlToken;
			validateInvitation(urlToken);
		}
	});

	async function validateInvitation(token: string) {
		validatingInvitation = true;
		try {
			const result = await authApi.validateInvitation(token);
			if (result.valid) {
				if (result.email) {
					invitationEmail = result.email;
					email = result.email;
				}
			} else {
				error = 'Invalid or expired invitation';
			}
		} catch (err) {
			error = 'Failed to validate invitation';
		} finally {
			validatingInvitation = false;
		}
	}

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
			const response = await authApi.register({
				username,
				email,
				password,
				display_name: fullName || undefined,
				invitation_token: invitationToken || undefined
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
			error = err instanceof Error ? err.message : 'Registration failed';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Register - AIGateway</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-background p-4">
	<div class="w-full max-w-md">
		<!-- Logo -->
		<div class="mb-8 text-center">
			<div
				class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-primary text-3xl text-primary-foreground"
			>
				🤖
			</div>
			<h1 class="text-2xl font-bold text-foreground">Create Account</h1>
			<p class="mt-2 text-muted-foreground">Join AIGateway</p>
		</div>

		<!-- Register Form -->
		<div class="rounded-xl border border-border bg-card p-6 shadow-sm">
			{#if validatingInvitation}
				<div class="flex items-center justify-center gap-2 py-8 text-muted-foreground">
					<Loader2 class="h-5 w-5 animate-spin" />
					Validating invitation...
				</div>
			{:else}
				<form onsubmit={handleSubmit} class="space-y-4">
					{#if error}
						<div class="rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
							{error}
						</div>
					{/if}

					{#if invitationEmail}
						<div class="rounded-lg bg-primary/10 p-3 text-sm text-primary">
							Registering with invitation for: {invitationEmail}
						</div>
					{/if}

					<div class="space-y-2">
						<label for="username" class="text-sm font-medium text-foreground"> Username </label>
						<input
							id="username"
							type="text"
							bind:value={username}
							placeholder="johndoe"
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
						<p class="text-xs text-muted-foreground">
							Letters, numbers, underscores and dashes only
						</p>
					</div>

					<div class="space-y-2">
						<label for="email" class="text-sm font-medium text-foreground"> Email </label>
						<input
							id="email"
							type="email"
							bind:value={email}
							placeholder="john@example.com"
							required
							disabled={!!invitationEmail}
							class={cn(
								'w-full rounded-lg border border-input bg-background px-3 py-2 text-sm',
								'placeholder:text-muted-foreground',
								'focus:outline-none focus:ring-2 focus:ring-ring',
								'disabled:cursor-not-allowed disabled:opacity-50'
							)}
						/>
					</div>

					<div class="space-y-2">
						<label for="fullname" class="text-sm font-medium text-foreground">
							Full Name <span class="text-muted-foreground">(optional)</span>
						</label>
						<input
							id="fullname"
							type="text"
							bind:value={fullName}
							placeholder="John Doe"
							class={cn(
								'w-full rounded-lg border border-input bg-background px-3 py-2 text-sm',
								'placeholder:text-muted-foreground',
								'focus:outline-none focus:ring-2 focus:ring-ring'
							)}
						/>
					</div>

					<div class="space-y-2">
						<label for="password" class="text-sm font-medium text-foreground"> Password </label>
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

					{#if !invitationToken}
						<div class="space-y-2">
							<label for="invitation" class="text-sm font-medium text-foreground">
								Invitation Code <span class="text-muted-foreground">(if required)</span>
							</label>
							<input
								id="invitation"
								type="text"
								bind:value={invitationToken}
								placeholder="Enter invitation code"
								class={cn(
									'w-full rounded-lg border border-input bg-background px-3 py-2 text-sm',
									'placeholder:text-muted-foreground',
									'focus:outline-none focus:ring-2 focus:ring-ring'
								)}
							/>
						</div>
					{/if}

					<button
						type="submit"
						disabled={loading || !isPasswordValid || !passwordsMatch}
						class={cn(
							'flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground',
							'hover:bg-primary/90 focus:outline-none focus:ring-2 focus:ring-ring',
							'disabled:cursor-not-allowed disabled:opacity-50'
						)}
					>
						{#if loading}
							<Loader2 class="h-4 w-4 animate-spin" />
							Creating account...
						{:else}
							Create Account
						{/if}
					</button>
				</form>
			{/if}

			<div class="mt-6 text-center text-sm text-muted-foreground">
				Already have an account?
				<a href="/login" class="font-medium text-primary hover:underline">Sign in</a>
			</div>
		</div>

		<!-- Footer -->
		<p class="mt-8 text-center text-xs text-muted-foreground">
			By creating an account, you agree to our Terms of Service
		</p>
	</div>
</div>

