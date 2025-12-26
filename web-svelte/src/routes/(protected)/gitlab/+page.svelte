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
      newIntegration = { name: '', base_url: 'https://gitlab.com', access_token: '', webhook_secret: '' };
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
      case 'active': return 'bg-green-500';
      case 'disabled': return 'bg-gray-500';
      case 'error': return 'bg-red-500';
      default: return 'bg-gray-500';
    }
  }
</script>

<svelte:head>
  <title>{m.gitlab_title()} - AIGateway</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-[1600px]">
  <div class="flex justify-between items-center mb-8">
    <div>
      <h1 class="text-3xl font-bold text-gray-100">{m.gitlab_title()}</h1>
      <p class="text-gray-400 mt-1">{m.gitlab_subtitle()}</p>
    </div>
    <button
      on:click={() => showCreateModal = true}
      class="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-medium transition-colors flex items-center gap-2"
    >
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
      </svg>
      {m.gitlab_add_integration()}
    </button>
  </div>

  {#if error}
    <div class="mb-6 p-4 bg-red-900/50 border border-red-700 rounded-lg text-red-200">
      {error}
    </div>
  {/if}

  {#if loading}
    <div class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-4 border-indigo-500 border-t-transparent"></div>
    </div>
  {:else if integrations.length === 0}
    <div class="text-center py-16 bg-gray-800/50 rounded-xl border border-gray-700">
      <svg class="w-16 h-16 mx-auto text-gray-600 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
      </svg>
      <h3 class="text-xl font-semibold text-gray-300 mb-2">{m.gitlab_no_integrations()}</h3>
      <p class="text-gray-500 mb-6">{m.gitlab_no_integrations_desc()}</p>
      <button
        on:click={() => showCreateModal = true}
        class="px-6 py-3 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-medium transition-colors"
      >
        {m.gitlab_add_integration()}
      </button>
    </div>
  {:else}
    <div class="grid gap-4">
      {#each integrations as integration}
        <div class="bg-gray-800 rounded-xl border border-gray-700 p-6 hover:border-gray-600 transition-colors">
          <div class="flex items-start justify-between">
            <div class="flex items-start gap-4">
              <div class="w-12 h-12 bg-orange-600 rounded-lg flex items-center justify-center">
                <svg class="w-7 h-7 text-white" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 01-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 014.82 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.49h8.1l2.44-7.51A.42.42 0 0118.6 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.51L23 13.45a.84.84 0 01-.35.94z"/>
                </svg>
              </div>
              <div>
                <h3 class="text-lg font-semibold text-gray-100">{integration.name}</h3>
                <p class="text-gray-400 text-sm">{integration.base_url}</p>
                <div class="flex items-center gap-3 mt-2">
                  <span class="flex items-center gap-1.5">
                    <span class="w-2 h-2 rounded-full {getStatusColor(integration.status)}"></span>
                    <span class="text-sm text-gray-400 capitalize">{integration.status}</span>
                  </span>
                  {#if integration.project_count !== undefined}
                    <span class="text-sm text-gray-500">
                      {integration.project_count} project{integration.project_count !== 1 ? 's' : ''}
                    </span>
                  {/if}
                </div>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <a
                href="/gitlab/{integration.id}"
                class="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-200 rounded-lg text-sm font-medium transition-colors"
              >
                Manage
              </a>
              <button
                on:click={() => handleDelete(integration.id, integration.name)}
                class="p-2 text-gray-400 hover:text-red-400 hover:bg-red-900/30 rounded-lg transition-colors"
                title="Delete integration"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
              </button>
            </div>
          </div>
          {#if integration.last_error}
            <div class="mt-4 p-3 bg-red-900/30 border border-red-800/50 rounded-lg">
              <p class="text-red-300 text-sm">{integration.last_error}</p>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Create Modal -->
{#if showCreateModal}
  <div class="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
    <div class="bg-gray-800 rounded-xl border border-gray-700 w-full max-w-lg">
      <div class="flex items-center justify-between p-6 border-b border-gray-700">
        <h2 class="text-xl font-semibold text-gray-100">Add GitLab Integration</h2>
        <button
          on:click={() => showCreateModal = false}
          class="text-gray-400 hover:text-gray-200"
          title={m.common_close()}
          aria-label={m.common_close()}
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>
      
      <form on:submit|preventDefault={handleCreate} class="p-6 space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-300 mb-2" for="name">
            Integration Name *
          </label>
          <input
            id="name"
            type="text"
            bind:value={newIntegration.name}
            placeholder="e.g., Company GitLab"
            class="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
            required
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-300 mb-2" for="base_url">
            GitLab URL *
          </label>
          <input
            id="base_url"
            type="url"
            bind:value={newIntegration.base_url}
            placeholder="https://gitlab.com"
            class="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
            required
          />
          <p class="text-xs text-gray-500 mt-1">Use https://gitlab.com for GitLab.com or your self-hosted instance URL</p>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-300 mb-2" for="access_token">
            Personal Access Token *
          </label>
          <input
            id="access_token"
            type="password"
            bind:value={newIntegration.access_token}
            placeholder="glpat-..."
            class="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
            required
          />
          <p class="text-xs text-gray-500 mt-1">
            Required scopes: api, read_api, read_repository, write_repository
          </p>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-300 mb-2" for="webhook_secret">
            Webhook Secret (optional)
          </label>
          <input
            id="webhook_secret"
            type="password"
            bind:value={newIntegration.webhook_secret}
            placeholder={m.placeholder_optional_secret()}
            class="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
          />
        </div>

        <div class="flex justify-end gap-3 pt-4">
          <button
            type="button"
            on:click={() => showCreateModal = false}
            class="px-4 py-2 text-gray-300 hover:text-gray-100 font-medium transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={creating}
            class="px-6 py-2 bg-indigo-600 hover:bg-indigo-700 disabled:bg-indigo-800 disabled:cursor-not-allowed text-white rounded-lg font-medium transition-colors flex items-center gap-2"
          >
            {#if creating}
              <div class="animate-spin rounded-full h-4 w-4 border-2 border-white border-t-transparent"></div>
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

