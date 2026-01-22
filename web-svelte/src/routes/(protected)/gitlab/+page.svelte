<script lang="ts">
	import { onMount } from 'svelte';
	import {
		listMyIntegrations,
		createMyIntegration,
		deleteMyIntegration,
		type GitLabIntegration
	} from '$lib/api/gitlab-user';
	import * as m from '$lib/paraglide/messages';

	let integrations: GitLabIntegration[] = [];
	let loading = true;
	let error = '';
	let showCreateModal = false;

	// Create form
	let newIntegration = {
		name: '',
		base_url: 'https://gitlab.com',
		access_token: '',
		webhook_secret: ''
	};
	let creating = false;

	onMount(async () => {
		await loadIntegrations();
	});

	async function loadIntegrations() {
		loading = true;
		error = '';
		try {
			const response = await listMyIntegrations();
			integrations = response.data || [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load integrations';
		} finally {
			loading = false;
		}
	}

	async function handleCreate() {
		if (!newIntegration.name || !newIntegration.base_url || !newIntegration.access_token) {
			error = 'Please fill in all required fields';
			return;
		}

		creating = true;
		error = '';
		try {
			await createMyIntegration(newIntegration);
			showCreateModal = false;
			newIntegration = {
				name: '',
				base_url: 'https://gitlab.com',
				access_token: '',
				webhook_secret: ''
			};
			await loadIntegrations();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create integration';
		} finally {
			creating = false;
		}
	}

	async function handleDelete(id: string, name: string) {
		if (!confirm(m.confirm_delete_integration({ name }))) {
			return;
		}

		try {
			await deleteMyIntegration(id);
			await loadIntegrations();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete integration';
		}
	}

	function getStatusColor(status: string) {
		switch (status) {
			case 'active':
				return 'bg-green-500';
			case 'disabled':
				return 'bg-gray-500';
			case 'error':
				return 'bg-red-500';
			default:
				return 'bg-gray-500';
		}
	}
</script>

<svelte:head>
	<title>{m.gitlab_title()} - AIGateway</title>
</svelte:head>

<div class="container mx-auto max-w-[1600px] px-4 py-8">
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-bold text-gray-100">{m.gitlab_title()}</h1>
			<p class="mt-1 text-gray-400">{m.gitlab_subtitle()}</p>
		</div>
		<button
			on:click={() => (showCreateModal = true)}
			class="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 font-medium text-white transition-colors hover:bg-indigo-700"
		>
			<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
			</svg>
			{m.gitlab_add_integration()}
		</button>
	</div>

	{#if error}
		<div class="mb-6 rounded-lg border border-red-700 bg-red-900/50 p-4 text-red-200">
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="flex justify-center py-12">
			<div
				class="h-12 w-12 animate-spin rounded-full border-4 border-indigo-500 border-t-transparent"
			></div>
		</div>
	{:else if integrations.length === 0}
		<div class="rounded-xl border border-gray-700 bg-gray-800/50 py-16 text-center">
			<svg
				class="mx-auto mb-4 h-16 w-16 text-gray-600"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="1.5"
					d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
				/>
			</svg>
			<h3 class="mb-2 text-xl font-semibold text-gray-300">{m.gitlab_no_integrations()}</h3>
			<p class="mb-6 text-gray-500">{m.gitlab_no_integrations_desc()}</p>
			<button
				on:click={() => (showCreateModal = true)}
				class="rounded-lg bg-indigo-600 px-6 py-3 font-medium text-white transition-colors hover:bg-indigo-700"
			>
				{m.gitlab_add_integration()}
			</button>
		</div>
	{:else}
		<div class="grid gap-4">
			{#each integrations as integration}
				<div
					class="rounded-xl border border-gray-700 bg-gray-800 p-6 transition-colors hover:border-gray-600"
				>
					<div class="flex items-start justify-between">
						<div class="flex items-start gap-4">
							<div class="flex h-12 w-12 items-center justify-center rounded-lg bg-orange-600">
								<svg class="h-7 w-7 text-white" viewBox="0 0 24 24" fill="currentColor">
									<path
										d="M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 01-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 014.82 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.49h8.1l2.44-7.51A.42.42 0 0118.6 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.51L23 13.45a.84.84 0 01-.35.94z"
									/>
								</svg>
							</div>
							<div>
								<h3 class="text-lg font-semibold text-gray-100">{integration.name}</h3>
								<p class="text-sm text-gray-400">{integration.base_url}</p>
								<div class="mt-2 flex items-center gap-3">
									<span class="flex items-center gap-1.5">
										<span class="h-2 w-2 rounded-full {getStatusColor(integration.status)}"></span>
										<span class="text-sm text-gray-400 capitalize">{integration.status}</span>
									</span>
									{#if integration.project_count !== undefined}
										<span class="text-sm text-gray-500">
											{integration.project_count} project{integration.project_count !== 1
												? 's'
												: ''}
										</span>
									{/if}
								</div>
							</div>
						</div>
						<div class="flex items-center gap-2">
							<a
								href="/gitlab/{integration.id}"
								class="rounded-lg bg-gray-700 px-4 py-2 text-sm font-medium text-gray-200 transition-colors hover:bg-gray-600"
							>
								Manage
							</a>
							<button
								on:click={() => handleDelete(integration.id, integration.name)}
								class="rounded-lg p-2 text-gray-400 transition-colors hover:bg-red-900/30 hover:text-red-400"
								title="Delete integration"
							>
								<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
									/>
								</svg>
							</button>
						</div>
					</div>
					{#if integration.last_error}
						<div class="mt-4 rounded-lg border border-red-800/50 bg-red-900/30 p-3">
							<p class="text-sm text-red-300">{integration.last_error}</p>
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Create Modal -->
{#if showCreateModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
		<div class="w-full max-w-lg rounded-xl border border-gray-700 bg-gray-800">
			<div class="flex items-center justify-between border-b border-gray-700 p-6">
				<h2 class="text-xl font-semibold text-gray-100">Add GitLab Integration</h2>
				<button
					on:click={() => (showCreateModal = false)}
					class="text-gray-400 hover:text-gray-200"
					title={m.common_close()}
					aria-label={m.common_close()}
				>
					<svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>
			</div>

			<form on:submit|preventDefault={handleCreate} class="space-y-4 p-6">
				<div>
					<label class="mb-2 block text-sm font-medium text-gray-300" for="name">
						Integration Name *
					</label>
					<input
						id="name"
						type="text"
						bind:value={newIntegration.name}
						placeholder="e.g., Company GitLab"
						class="w-full rounded-lg border border-gray-700 bg-gray-900 px-4 py-2 text-gray-100 placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
						required
					/>
				</div>

				<div>
					<label class="mb-2 block text-sm font-medium text-gray-300" for="base_url">
						GitLab URL *
					</label>
					<input
						id="base_url"
						type="url"
						bind:value={newIntegration.base_url}
						placeholder="https://gitlab.com"
						class="w-full rounded-lg border border-gray-700 bg-gray-900 px-4 py-2 text-gray-100 placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
						required
					/>
					<p class="mt-1 text-xs text-gray-500">
						Use https://gitlab.com for GitLab.com or your self-hosted instance URL
					</p>
				</div>

				<div>
					<label class="mb-2 block text-sm font-medium text-gray-300" for="access_token">
						Personal Access Token *
					</label>
					<input
						id="access_token"
						type="password"
						bind:value={newIntegration.access_token}
						placeholder="glpat-..."
						class="w-full rounded-lg border border-gray-700 bg-gray-900 px-4 py-2 text-gray-100 placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
						required
					/>
					<p class="mt-1 text-xs text-gray-500">
						Required scopes: api, read_api, read_repository, write_repository
					</p>
				</div>

				<div>
					<label class="mb-2 block text-sm font-medium text-gray-300" for="webhook_secret">
						Webhook Secret (optional)
					</label>
					<input
						id="webhook_secret"
						type="password"
						bind:value={newIntegration.webhook_secret}
						placeholder={m.placeholder_optional_secret()}
						class="w-full rounded-lg border border-gray-700 bg-gray-900 px-4 py-2 text-gray-100 placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
					/>
				</div>

				<div class="flex justify-end gap-3 pt-4">
					<button
						type="button"
						on:click={() => (showCreateModal = false)}
						class="px-4 py-2 font-medium text-gray-300 transition-colors hover:text-gray-100"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={creating}
						class="flex items-center gap-2 rounded-lg bg-indigo-600 px-6 py-2 font-medium text-white transition-colors hover:bg-indigo-700 disabled:cursor-not-allowed disabled:bg-indigo-800"
					>
						{#if creating}
							<div
								class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"
							></div>
							Creating...
						{:else}
							Create Integration
						{/if}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	:global(body) {
		background-color: #111827;
	}
</style>
