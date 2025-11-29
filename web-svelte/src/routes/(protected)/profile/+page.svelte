<script lang="ts">
	import { onMount } from 'svelte';
	import {
		User,
		Mail,
		Lock,
		Save,
		Loader2,
		Eye,
		EyeOff,
		CheckCircle,
		Smartphone,
		Monitor,
		Tablet,
		LogOut,
		Shield
	} from 'lucide-svelte';
	import { profileApi, type UserProfile, type Device } from '$lib/api/profile';
	import { authStore } from '$lib/stores/auth.svelte';
	import { cn, formatRelativeTime } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	// Profile state
	let profile = $state<UserProfile | null>(null);
	let isLoading = $state(true);
	let fullName = $state('');
	let email = $state('');
	let isSavingProfile = $state(false);
	let profileSuccess = $state(false);
	let profileError = $state('');

	// Password state
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let showPasswords = $state(false);
	let isChangingPassword = $state(false);
	let passwordSuccess = $state(false);
	let passwordError = $state('');

	// Devices state
	let devices = $state<Device[]>([]);
	let loadingDevices = $state(false);

	onMount(async () => {
		await Promise.all([loadProfile(), loadDevices()]);
	});

	async function loadProfile() {
		isLoading = true;
		try {
			profile = await profileApi.getProfile();
			fullName = profile.full_name || '';
			email = profile.email;
		} catch (error) {
			console.error('Failed to load profile:', error);
		} finally {
			isLoading = false;
		}
	}

	async function loadDevices() {
		loadingDevices = true;
		try {
			const response = await profileApi.getDevices();
			devices = response.devices || [];
		} catch (error) {
			console.error('Failed to load devices:', error);
			devices = [];
		} finally {
			loadingDevices = false;
		}
	}

	async function handleSaveProfile() {
		isSavingProfile = true;
		profileError = '';
		profileSuccess = false;

		try {
			await profileApi.updateProfile({
				full_name: fullName.trim() || undefined,
				email: email.trim()
			});
			profileSuccess = true;
			// Update auth store
			authStore.updateUser({ full_name: fullName.trim(), email: email.trim() });
			setTimeout(() => (profileSuccess = false), 3000);
		} catch (error) {
			profileError = error instanceof Error ? error.message : 'Failed to update profile';
		} finally {
			isSavingProfile = false;
		}
	}

	async function handleChangePassword() {
		if (newPassword !== confirmPassword) {
			passwordError = 'Passwords do not match';
			return;
		}

		if (newPassword.length < 8) {
			passwordError = 'Password must be at least 8 characters';
			return;
		}

		isChangingPassword = true;
		passwordError = '';
		passwordSuccess = false;

		try {
			await profileApi.changePassword({
				current_password: currentPassword,
				new_password: newPassword
			});
			passwordSuccess = true;
			currentPassword = '';
			newPassword = '';
			confirmPassword = '';
			setTimeout(() => (passwordSuccess = false), 3000);
		} catch (error) {
			passwordError = error instanceof Error ? error.message : 'Failed to change password';
		} finally {
			isChangingPassword = false;
		}
	}

	async function handleRevokeDevice(deviceId: string) {
		if (!confirm('Revoke access for this device?')) return;

		try {
			await profileApi.revokeDevice(deviceId);
			devices = devices.filter((d) => d.id !== deviceId);
		} catch (error) {
			console.error('Failed to revoke device:', error);
			alert('Failed to revoke device');
		}
	}

	async function handleRevokeAll() {
		if (!confirm('Revoke access for all other devices? You will need to log in again on those devices.')) return;

		try {
			await profileApi.revokeAllDevices();
			devices = devices.filter((d) => d.is_current);
		} catch (error) {
			console.error('Failed to revoke devices:', error);
			alert('Failed to revoke devices');
		}
	}

	function getDeviceIcon(type: string) {
		switch (type.toLowerCase()) {
			case 'mobile':
				return Smartphone;
			case 'tablet':
				return Tablet;
			default:
				return Monitor;
		}
	}
</script>

<svelte:head>
	<title>{m.nav_profile()} | AI Gateway</title>
</svelte:head>

<div class="mx-auto max-w-3xl px-4 py-8 sm:px-6 lg:px-8">
	<h1 class="mb-8 text-2xl font-bold text-foreground">{m.nav_profile()}</h1>

	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
		</div>
	{:else if profile}
		<div class="space-y-8">
			<!-- Profile Info -->
			<section class="rounded-xl border border-border bg-card p-6">
				<div class="mb-6 flex items-center gap-4">
					<div class="flex h-16 w-16 items-center justify-center rounded-full bg-primary/10 text-primary">
						<User class="h-8 w-8" />
					</div>
					<div>
						<h2 class="text-lg font-semibold">{profile.full_name || profile.username}</h2>
						<p class="text-sm text-muted-foreground">@{profile.username}</p>
						{#if profile.is_admin}
							<span class="mt-1 inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
								<Shield class="h-3 w-3" />
								Admin
							</span>
						{/if}
					</div>
				</div>

				<form onsubmit={(e) => { e.preventDefault(); handleSaveProfile(); }} class="space-y-4">
					{#if profileError}
						<div class="rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
							{profileError}
						</div>
					{/if}
					{#if profileSuccess}
						<div class="flex items-center gap-2 rounded-lg bg-green-500/10 p-3 text-sm text-green-500">
							<CheckCircle class="h-4 w-4" />
							Profile updated successfully
						</div>
					{/if}

					<div class="space-y-2">
						<label for="full-name" class="text-sm font-medium">Full Name</label>
						<div class="relative">
							<User class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
							<input
								id="full-name"
								type="text"
								bind:value={fullName}
								placeholder="Your full name"
								class="w-full rounded-lg border border-input bg-background py-2 pl-10 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
							/>
						</div>
					</div>

					<div class="space-y-2">
						<label for="email" class="text-sm font-medium">Email</label>
						<div class="relative">
							<Mail class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
							<input
								id="email"
								type="email"
								bind:value={email}
								required
								class="w-full rounded-lg border border-input bg-background py-2 pl-10 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
							/>
						</div>
					</div>

					<div class="flex justify-end">
						<Button type="submit" disabled={isSavingProfile}>
							{#if isSavingProfile}
								<Loader2 class="mr-2 h-4 w-4 animate-spin" />
							{:else}
								<Save class="mr-2 h-4 w-4" />
							{/if}
							{m.common_save()}
						</Button>
					</div>
				</form>
			</section>

			<!-- Change Password -->
			<section class="rounded-xl border border-border bg-card p-6">
				<h2 class="mb-4 flex items-center gap-2 text-lg font-semibold">
					<Lock class="h-5 w-5" />
					Change Password
				</h2>

				<form onsubmit={(e) => { e.preventDefault(); handleChangePassword(); }} class="space-y-4">
					{#if passwordError}
						<div class="rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
							{passwordError}
						</div>
					{/if}
					{#if passwordSuccess}
						<div class="flex items-center gap-2 rounded-lg bg-green-500/10 p-3 text-sm text-green-500">
							<CheckCircle class="h-4 w-4" />
							Password changed successfully
						</div>
					{/if}

					<div class="space-y-2">
						<label for="current-password" class="text-sm font-medium">Current Password</label>
						<div class="relative">
							<Lock class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
							<input
								id="current-password"
								type={showPasswords ? 'text' : 'password'}
								bind:value={currentPassword}
								required
								class="w-full rounded-lg border border-input bg-background py-2 pl-10 pr-10 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
							/>
							<button
								type="button"
								onclick={() => (showPasswords = !showPasswords)}
								class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
							>
								{#if showPasswords}
									<EyeOff class="h-4 w-4" />
								{:else}
									<Eye class="h-4 w-4" />
								{/if}
							</button>
						</div>
					</div>

					<div class="space-y-2">
						<label for="new-password" class="text-sm font-medium">New Password</label>
						<input
							id="new-password"
							type={showPasswords ? 'text' : 'password'}
							bind:value={newPassword}
							required
							minlength="8"
							class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						/>
					</div>

					<div class="space-y-2">
						<label for="confirm-password" class="text-sm font-medium">Confirm New Password</label>
						<input
							id="confirm-password"
							type={showPasswords ? 'text' : 'password'}
							bind:value={confirmPassword}
							required
							class={cn(
								'w-full rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring',
								confirmPassword && newPassword !== confirmPassword ? 'border-destructive' : 'border-input'
							)}
						/>
					</div>

					<div class="flex justify-end">
						<Button type="submit" disabled={isChangingPassword}>
							{#if isChangingPassword}
								<Loader2 class="mr-2 h-4 w-4 animate-spin" />
							{/if}
							Change Password
						</Button>
					</div>
				</form>
			</section>

			<!-- Active Devices -->
			<section class="rounded-xl border border-border bg-card p-6">
				<div class="mb-4 flex items-center justify-between">
					<h2 class="flex items-center gap-2 text-lg font-semibold">
						<Monitor class="h-5 w-5" />
						Active Sessions
					</h2>
					{#if devices.length > 1}
						<Button variant="outline" size="sm" onclick={handleRevokeAll}>
							<LogOut class="mr-2 h-4 w-4" />
							Revoke All Others
						</Button>
					{/if}
				</div>

				{#if loadingDevices}
					<div class="flex items-center justify-center py-8">
						<Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
					</div>
				{:else if devices.length === 0}
					<p class="py-4 text-center text-muted-foreground">No active sessions</p>
				{:else}
					<div class="space-y-3">
						{#each devices as device (device.id)}
							{@const DeviceIcon = getDeviceIcon(device.device_type)}
							<div class="flex items-center justify-between rounded-lg border border-border p-4">
								<div class="flex items-center gap-4">
									<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
										<DeviceIcon class="h-5 w-5 text-muted-foreground" />
									</div>
									<div>
										<p class="font-medium text-foreground">
											{device.device_name}
											{#if device.is_current}
												<span class="ml-2 rounded bg-primary/10 px-1.5 py-0.5 text-xs text-primary">
													Current
												</span>
											{/if}
										</p>
										<p class="text-xs text-muted-foreground">
											{device.ip_address} • Last active {formatRelativeTime(device.last_active_at)}
										</p>
									</div>
								</div>
								{#if !device.is_current}
									<button
										onclick={() => handleRevokeDevice(device.id)}
										class="rounded p-2 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
										title="Revoke access"
									>
										<LogOut class="h-4 w-4" />
									</button>
								{/if}
							</div>
						{/each}
					</div>
				{/if}
			</section>

			<!-- Account Info -->
			<section class="rounded-xl border border-border bg-card p-6">
				<h2 class="mb-4 text-lg font-semibold">Account Information</h2>
				<dl class="space-y-3 text-sm">
					<div class="flex justify-between">
						<dt class="text-muted-foreground">Username</dt>
						<dd class="font-medium">@{profile.username}</dd>
					</div>
					<div class="flex justify-between">
						<dt class="text-muted-foreground">Account Status</dt>
						<dd>
							<span class={cn(
								'rounded-full px-2 py-0.5 text-xs font-medium',
								profile.status === 'active' ? 'bg-green-500/10 text-green-500' : 'bg-red-500/10 text-red-500'
							)}>
								{profile.status}
							</span>
						</dd>
					</div>
					<div class="flex justify-between">
						<dt class="text-muted-foreground">Member Since</dt>
						<dd class="font-medium">{new Date(profile.created_at).toLocaleDateString()}</dd>
					</div>
					{#if profile.last_login_at}
						<div class="flex justify-between">
							<dt class="text-muted-foreground">Last Login</dt>
							<dd class="font-medium">{formatRelativeTime(profile.last_login_at)}</dd>
						</div>
					{/if}
					{#if profile.auth_provider}
						<div class="flex justify-between">
							<dt class="text-muted-foreground">Auth Provider</dt>
							<dd class="font-medium capitalize">{profile.auth_provider}</dd>
						</div>
					{/if}
				</dl>
			</section>
		</div>
	{/if}
</div>

