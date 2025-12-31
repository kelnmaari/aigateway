<script lang="ts">
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import {
    getMyIntegration,
    updateMyIntegration,
    listMyProjects,
    addMyProject,
    deleteMyProject,
    listMyReviews,
    startMyProjectIndexing,
    getMyProjectIndexStatus,
    type GitLabIntegration,
    type GitLabProject,
    type GitLabReview,
    type GitLabIndexStatus
  } from '$lib/api/gitlab-user';
  import * as m from '$lib/paraglide/messages';

  const integrationId = $page.params.id ?? '';
  
  let integration: GitLabIntegration | null = null;
  let projects: GitLabProject[] = [];
  let reviews: GitLabReview[] = [];
  let loading = true;
  let error = '';
  let activeTab: 'projects' | 'reviews' | 'settings' = 'projects';

  // Add project modal
  let showAddProject = false;
  let newProject = {
    gitlab_project_id: 0,
    name: '',
    path_with_namespace: '',
    analysis_model_id: '',
    embedding_model_id: ''
  };
  let addingProject = false;

  // Indexing state
  let indexingProjects: Set<string> = new Set();

  onMount(async () => {
    await loadData();
  });

  async function loadData() {
    loading = true;
    error = '';
    try {
      [integration, { data: projects }, { data: reviews }] = await Promise.all([
        getMyIntegration(integrationId),
        listMyProjects(integrationId),
        listMyReviews()
      ]);
      // Filter reviews for this integration
      reviews = reviews.filter(r => r.integration_id === integrationId);
    } catch (e) {
      if (e instanceof Error && e.message.includes('403')) {
        error = 'Access denied. You do not own this integration.';
      } else {
        error = e instanceof Error ? e.message : 'Failed to load data';
      }
    } finally {
      loading = false;
    }
  }

  async function handleAddProject() {
    if (!newProject.gitlab_project_id || !newProject.name || !newProject.analysis_model_id) {
      error = 'Please fill in required fields';
      return;
    }

    addingProject = true;
    try {
      await addMyProject(integrationId, newProject);
      showAddProject = false;
      newProject = { gitlab_project_id: 0, name: '', path_with_namespace: '', analysis_model_id: '', embedding_model_id: '' };
      const response = await listMyProjects(integrationId);
      projects = response.data;
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to add project';
    } finally {
      addingProject = false;
    }
  }

  async function handleDeleteProject(projectId: string, projectName: string) {
    if (!confirm(m.confirm_delete_project({ name: projectName }))) return;
    
    try {
      await deleteMyProject(projectId);
      projects = projects.filter(p => p.id !== projectId);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to delete project';
    }
  }

  async function handleStartIndexing(project: GitLabProject) {
    if (indexingProjects.has(project.id)) return;

    const force = project.index_status === 'completed';
    if (force && !confirm(m.confirm_reindex_project({ name: project.name }))) {
      return;
    }

    indexingProjects = new Set([...indexingProjects, project.id]);

    try {
      await startMyProjectIndexing(project.id, { 
        branch: project.default_branch || 'main',
        force 
      });
      
      // Poll for status updates
      pollIndexStatus(project.id);
    } catch (e) {
      console.error('Failed to start indexing:', e);
      indexingProjects = new Set([...indexingProjects].filter(id => id !== project.id));
      error = e instanceof Error ? e.message : 'Failed to start indexing';
    }
  }

  async function pollIndexStatus(projectId: string) {
    const maxAttempts = 60;
    let attempts = 0;

    const poll = async () => {
      try {
        const status = await getMyProjectIndexStatus(projectId);
        
        // Update project in list
        projects = projects.map(p => {
          if (p.id === projectId) {
            return { 
              ...p, 
              index_status: status.status,
              last_indexed_at: status.last_indexed 
            };
          }
          return p;
        });

        if (status.status === 'in_progress' && attempts < maxAttempts) {
          attempts++;
          setTimeout(poll, 5000);
        } else {
          indexingProjects = new Set([...indexingProjects].filter(id => id !== projectId));
        }
      } catch {
        indexingProjects = new Set([...indexingProjects].filter(id => id !== projectId));
      }
    };

    poll();
  }

  async function toggleIntegrationStatus() {
    if (!integration) return;
    
    const newStatus = integration.status === 'active' ? 'disabled' : 'active';
    try {
      await updateMyIntegration(integrationId, { status: newStatus });
      integration = await getMyIntegration(integrationId);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to update integration';
    }
  }

  function getStatusColor(status: string) {
    switch (status) {
      case 'active': return 'bg-green-500';
      case 'disabled': return 'bg-gray-500';
      case 'error': return 'bg-red-500';
      case 'completed': return 'bg-green-500';
      case 'failed': return 'bg-red-500';
      case 'pending': case 'queued': return 'bg-yellow-500';
      case 'processing': return 'bg-blue-500';
      default: return 'bg-gray-500';
    }
  }
</script>

<svelte:head>
  <title>{integration?.name || 'Integration'} - GitLab - AIGateway</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-[1600px]">
  <!-- Back button -->
  <a href="/gitlab" class="inline-flex items-center gap-2 text-gray-400 hover:text-gray-200 mb-6">
    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"/>
    </svg>
    {m.common_back()} {m.gitlab_title()}
  </a>

  {#if loading}
    <div class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-4 border-indigo-500 border-t-transparent"></div>
    </div>
  {:else if error}
    <div class="p-6 bg-red-900/50 border border-red-700 rounded-xl text-red-200">
      {error}
    </div>
  {:else if integration}
    <!-- Header -->
    <div class="bg-gray-800 rounded-xl border border-gray-700 p-6 mb-6">
      <div class="flex items-start justify-between">
        <div class="flex items-start gap-4">
          <div class="w-14 h-14 bg-orange-600 rounded-xl flex items-center justify-center">
            <svg class="w-8 h-8 text-white" viewBox="0 0 24 24" fill="currentColor">
              <path d="M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 01-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 014.82 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.49h8.1l2.44-7.51A.42.42 0 0118.6 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.51L23 13.45a.84.84 0 01-.35.94z"/>
            </svg>
          </div>
          <div>
            <h1 class="text-2xl font-bold text-gray-100">{integration.name}</h1>
            <p class="text-gray-400">{integration.base_url}</p>
            <div class="flex items-center gap-3 mt-2">
              <span class="flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full {getStatusColor(integration.status)}"></span>
                <span class="text-sm text-gray-400 capitalize">{integration.status}</span>
              </span>
              <span class="text-sm text-gray-500">
                {projects.length} project{projects.length !== 1 ? 's' : ''}
              </span>
            </div>
          </div>
        </div>
        <button
          on:click={toggleIntegrationStatus}
          class="px-4 py-2 rounded-lg font-medium transition-colors {integration.status === 'active' 
            ? 'bg-gray-700 hover:bg-gray-600 text-gray-200' 
            : 'bg-green-600 hover:bg-green-700 text-white'}"
        >
          {integration.status === 'active' ? m.common_disable() : m.common_enable()}
        </button>
      </div>
    </div>

    <!-- Tabs -->
    <div class="flex gap-1 mb-6 bg-gray-800/50 p-1 rounded-lg w-fit">
      <button
        on:click={() => activeTab = 'projects'}
        class="px-4 py-2 rounded-md font-medium transition-colors {activeTab === 'projects'
          ? 'bg-indigo-600 text-white'
          : 'text-gray-400 hover:text-gray-200'}"
      >
        {m.admin_project_projects_tab()} ({projects.length})
      </button>
      <button
        on:click={() => activeTab = 'reviews'}
        class="px-4 py-2 rounded-md font-medium transition-colors {activeTab === 'reviews'
          ? 'bg-indigo-600 text-white'
          : 'text-gray-400 hover:text-gray-200'}"
      >
        {m.admin_project_reviews_tab()} ({reviews.length})
      </button>
      <button
        on:click={() => activeTab = 'settings'}
        class="px-4 py-2 rounded-md font-medium transition-colors {activeTab === 'settings'
          ? 'bg-indigo-600 text-white'
          : 'text-gray-400 hover:text-gray-200'}"
      >
        {m.settings_title()}
      </button>
    </div>

    <!-- Tab content -->
    {#if activeTab === 'projects'}
      <div class="space-y-4">
        <div class="flex justify-end">
          <button
            on:click={() => showAddProject = true}
            class="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-medium transition-colors flex items-center gap-2"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
            </svg>
            {m.modal_add_project()}
          </button>
        </div>

        {#if projects.length === 0}
          <div class="text-center py-12 bg-gray-800/50 rounded-xl border border-gray-700">
            <p class="text-gray-400">{m.gitlab_no_projects()}</p>
            <p class="text-gray-500 text-sm mt-1">{m.gitlab_no_projects_desc()}</p>
          </div>
        {:else}
          {#each projects as project}
            <div class="bg-gray-800 rounded-xl border border-gray-700 p-5">
              <div class="flex items-start justify-between">
                <div>
                  <h3 class="font-semibold text-gray-100">{project.name}</h3>
                  <p class="text-gray-500 text-sm">{project.path_with_namespace}</p>
                  <div class="flex items-center gap-4 mt-2 text-sm">
                    <span class="flex items-center gap-1.5">
                      <span class="w-2 h-2 rounded-full {getStatusColor(project.status)}"></span>
                      <span class="text-gray-400 capitalize">{project.status}</span>
                    </span>
                    <span class="text-gray-500">
                      Model: {project.analysis_model_id}
                    </span>
                    {#if project.auto_review}
                      <span class="text-green-400">Auto-review ON</span>
                    {:else}
                      <span class="text-gray-500">Auto-review OFF</span>
                    {/if}
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <!-- Index Status & Button -->
                  <div class="flex items-center gap-2 mr-2">
                    {#if indexingProjects.has(project.id)}
                      <span class="text-yellow-400 text-sm flex items-center gap-1">
                        <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        Indexing...
                      </span>
                    {:else if project.index_status === 'completed'}
                      <span class="text-green-400 text-sm">✓ Indexed</span>
                    {:else if project.index_status === 'failed'}
                      <span class="text-red-400 text-sm">✗ Failed</span>
                    {:else}
                      <span class="text-gray-500 text-sm">Not indexed</span>
                    {/if}
                  </div>
                  <button
                    on:click={() => handleStartIndexing(project)}
                    class="p-2 text-gray-400 hover:text-indigo-400 hover:bg-indigo-900/30 rounded-lg transition-colors"
                    title={project.index_status === 'completed' ? 'Re-index project' : 'Index project'}
                    disabled={indexingProjects.has(project.id)}
                  >
                    <svg class="w-5 h-5 {indexingProjects.has(project.id) ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                    </svg>
                  </button>
                  <button
                    on:click={() => handleDeleteProject(project.id, project.name)}
                    class="p-2 text-gray-400 hover:text-red-400 hover:bg-red-900/30 rounded-lg transition-colors"
                    title="Delete project"
                  >
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    {:else if activeTab === 'reviews'}
      <div class="space-y-3">
        {#if reviews.length === 0}
          <div class="text-center py-12 bg-gray-800/50 rounded-xl border border-gray-700">
            <p class="text-gray-400">No reviews yet</p>
            <p class="text-gray-500 text-sm mt-1">Reviews will appear here when MRs are analyzed</p>
          </div>
        {:else}
          {#each reviews as review}
            <div class="bg-gray-800 rounded-xl border border-gray-700 p-4">
              <div class="flex items-start justify-between">
                <div>
                  <h3 class="font-medium text-gray-100">
                    <a href={review.mr_url} target="_blank" class="hover:text-indigo-400">
                      !{review.mr_iid}: {review.mr_title}
                    </a>
                  </h3>
                  <div class="flex items-center gap-3 mt-1 text-sm text-gray-500">
                    <span>{review.source_branch} → {review.target_branch}</span>
                    <span>by {review.mr_author}</span>
                  </div>
                </div>
                <span class="flex items-center gap-1.5 px-2 py-1 rounded text-xs font-medium {getStatusColor(review.status)} bg-opacity-20">
                  <span class="w-1.5 h-1.5 rounded-full {getStatusColor(review.status)}"></span>
                  {review.status}
                </span>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    {:else if activeTab === 'settings'}
      <div class="bg-gray-800 rounded-xl border border-gray-700 p-6">
        <h3 class="text-lg font-semibold text-gray-100 mb-4">Integration Settings</h3>
        <div class="space-y-4">
          <div>
            <span class="block text-sm font-medium text-gray-300 mb-1">API Version</span>
            <p class="text-gray-400">{integration.settings.api_version || 'v4'}</p>
          </div>
          <div>
            <span class="block text-sm font-medium text-gray-300 mb-1">Request Timeout</span>
            <p class="text-gray-400">{integration.settings.request_timeout || 30} seconds</p>
          </div>
          <div>
            <span class="block text-sm font-medium text-gray-300 mb-1">Rate Limit</span>
            <p class="text-gray-400">{integration.settings.rate_limit_per_min || 30} requests/minute</p>
          </div>
          <div>
            <span class="block text-sm font-medium text-gray-300 mb-1">Webhook URL</span>
            <code class="text-sm text-indigo-400 bg-gray-900 px-2 py-1 rounded">
              {window.location.origin}/webhook/gitlab?integration={integration.id}
            </code>
          </div>
        </div>
      </div>
    {/if}
  {/if}
</div>

<!-- Add Project Modal -->
{#if showAddProject}
  <div class="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
    <div class="bg-gray-800 rounded-xl border border-gray-700 w-full max-w-lg">
      <div class="flex items-center justify-between p-6 border-b border-gray-700">
        <h2 class="text-xl font-semibold text-gray-100">Add Project</h2>
        <button on:click={() => showAddProject = false} class="text-gray-400 hover:text-gray-200" title={m.common_close()} aria-label={m.common_close()}>
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>
      
      <form on:submit|preventDefault={handleAddProject} class="p-6 space-y-4">
        <div>
          <label for="user-project-id" class="block text-sm font-medium text-gray-300 mb-2">GitLab Project ID *</label>
          <input
            id="user-project-id"
            type="number"
            bind:value={newProject.gitlab_project_id}
            placeholder="e.g., 12345"
            class="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-100"
            required
          />
          <p class="text-xs text-gray-500 mt-1">Find this in GitLab: Settings → General → Project ID</p>
        </div>

        <div>
          <label for="user-project-name" class="block text-sm font-medium text-gray-300 mb-2">Project Name *</label>
          <input
            id="user-project-name"
            type="text"
            bind:value={newProject.name}
            placeholder="e.g., My Project"
            class="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-100"
            required
          />
        </div>

        <div>
          <label for="user-project-path" class="block text-sm font-medium text-gray-300 mb-2">Path with Namespace</label>
          <input
            id="user-project-path"
            type="text"
            bind:value={newProject.path_with_namespace}
            placeholder="e.g., group/project"
            class="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-100"
          />
        </div>

        <div>
          <label for="user-analysis-model" class="block text-sm font-medium text-gray-300 mb-2">Analysis Model *</label>
          <input
            id="user-analysis-model"
            type="text"
            bind:value={newProject.analysis_model_id}
            placeholder="e.g., gpt-4"
            class="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-100"
            required
          />
        </div>

        <div>
          <label for="user-embedding-model" class="block text-sm font-medium text-gray-300 mb-2">Embedding Model</label>
          <input
            id="user-embedding-model"
            type="text"
            bind:value={newProject.embedding_model_id}
            placeholder="e.g., text-embedding-3-small"
            class="w-full px-4 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-100"
          />
        </div>

        <div class="flex justify-end gap-3 pt-4">
          <button
            type="button"
            on:click={() => showAddProject = false}
            class="px-4 py-2 text-gray-300 hover:text-gray-100 font-medium"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={addingProject}
            class="px-6 py-2 bg-indigo-600 hover:bg-indigo-700 disabled:bg-indigo-800 text-white rounded-lg font-medium flex items-center gap-2"
          >
            {#if addingProject}
              <div class="animate-spin rounded-full h-4 w-4 border-2 border-white border-t-transparent"></div>
              Adding...
            {:else}
              Add Project
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

