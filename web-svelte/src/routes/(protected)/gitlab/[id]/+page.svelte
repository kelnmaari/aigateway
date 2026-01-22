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
		scanMySecrets,
		deepScanMySecrets,
		analyzeMyQuality,
		detectMyDuplication,
		checkMyDependencies,
		detectMyDeadCode,
		scanMyUndocumented,
		generateMyDocs,
		scanMyTestable,
		generateMyTests,
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

	// Analysis state
	let selectedProjectForAnalysis: GitLabProject | null = null;
	let showAnalysisModal = false;
	let analysisLoading = false;
	let analysisType: string = '';
	let analysisResult: any = null;
	let analysisError: string = '';

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		loading = true;
		error = '';
		try {
			const [intResponse, projResponse, revResponse] = await Promise.all([
				getMyIntegration(integrationId),
				listMyProjects(integrationId),
				listMyReviews()
			]);
			integration = intResponse;
			projects = projResponse.data || [];
			const allReviews = revResponse.data || [];
			// Filter reviews for this integration
			reviews = allReviews.filter((r) => r.integration_id === integrationId);
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
			newProject = {
				gitlab_project_id: 0,
				name: '',
				path_with_namespace: '',
				analysis_model_id: '',
				embedding_model_id: ''
			};
			const response = await listMyProjects(integrationId);
			projects = response.data || [];
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
			projects = projects.filter((p) => p.id !== projectId);
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
			indexingProjects = new Set([...indexingProjects].filter((id) => id !== project.id));
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
				projects = projects.map((p) => {
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
					indexingProjects = new Set([...indexingProjects].filter((id) => id !== projectId));
				}
			} catch {
				indexingProjects = new Set([...indexingProjects].filter((id) => id !== projectId));
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

	function openAnalysis(project: GitLabProject) {
		selectedProjectForAnalysis = project;
		showAnalysisModal = true;
		analysisResult = null;
		analysisError = '';
	}

	async function runAnalysis(type: string) {
		if (!selectedProjectForAnalysis) return;

		analysisLoading = true;
		analysisType = type;
		analysisResult = null;
		analysisError = '';

		try {
			let result;
			switch (type) {
				case 'secrets':
					result = await scanMySecrets(selectedProjectForAnalysis.id);
					break;
				case 'deep-scan':
					result = await deepScanMySecrets(selectedProjectForAnalysis.id);
					break;
				case 'quality':
					result = await analyzeMyQuality(selectedProjectForAnalysis.id);
					break;
				case 'duplication':
					result = await detectMyDuplication(selectedProjectForAnalysis.id);
					break;
				case 'dependencies':
					result = await checkMyDependencies(selectedProjectForAnalysis.id);
					break;
				case 'dead-code':
					result = await detectMyDeadCode(selectedProjectForAnalysis.id);
					break;
				case 'docs':
					result = await scanMyUndocumented(selectedProjectForAnalysis.id);
					break;
				case 'tests':
					result = await scanMyTestable(selectedProjectForAnalysis.id);
					break;
			}
			analysisResult = result;
		} catch (e) {
			analysisError = e instanceof Error ? e.message : 'Analysis failed';
		} finally {
			analysisLoading = false;
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
			case 'completed':
				return 'bg-green-500';
			case 'failed':
				return 'bg-red-500';
			case 'pending':
			case 'queued':
				return 'bg-yellow-500';
			case 'processing':
				return 'bg-blue-500';
			default:
				return 'bg-gray-500';
		}
	}
</script>

<svelte:head>
	<title>{integration?.name || 'Integration'} - GitLab - AIGateway</title>
</svelte:head>

<div class="container mx-auto max-w-[1600px] px-4 py-8">
	<!-- Back button -->
	<a href="/gitlab" class="mb-6 inline-flex items-center gap-2 text-gray-400 hover:text-gray-200">
		<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M10 19l-7-7m0 0l7-7m-7 7h18"
			/>
		</svg>
		{m.common_back()}
		{m.gitlab_title()}
	</a>

	{#if loading}
		<div class="flex justify-center py-12">
			<div
				class="h-12 w-12 animate-spin rounded-full border-4 border-indigo-500 border-t-transparent"
			></div>
		</div>
	{:else if error}
		<div class="rounded-xl border border-red-700 bg-red-900/50 p-6 text-red-200">
			{error}
		</div>
	{:else if integration}
		<!-- Header -->
		<div class="mb-6 rounded-xl border border-gray-700 bg-gray-800 p-6">
			<div class="flex items-start justify-between">
				<div class="flex items-start gap-4">
					<div class="flex h-14 w-14 items-center justify-center rounded-xl bg-orange-600">
						<svg class="h-8 w-8 text-white" viewBox="0 0 24 24" fill="currentColor">
							<path
								d="M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 01-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 014.82 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.49h8.1l2.44-7.51A.42.42 0 0118.6 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.51L23 13.45a.84.84 0 01-.35.94z"
							/>
						</svg>
					</div>
					<div>
						<h1 class="text-2xl font-bold text-gray-100">{integration.name}</h1>
						<p class="text-gray-400">{integration.base_url}</p>
						<div class="mt-2 flex items-center gap-3">
							<span class="flex items-center gap-1.5">
								<span class="h-2 w-2 rounded-full {getStatusColor(integration.status)}"></span>
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
					class="rounded-lg px-4 py-2 font-medium transition-colors {integration.status === 'active'
						? 'bg-gray-700 text-gray-200 hover:bg-gray-600'
						: 'bg-green-600 text-white hover:bg-green-700'}"
				>
					{integration.status === 'active' ? m.common_disable() : m.common_enable()}
				</button>
			</div>
		</div>

		<!-- Tabs -->
		<div class="mb-6 flex w-fit gap-1 rounded-lg bg-gray-800/50 p-1">
			<button
				on:click={() => (activeTab = 'projects')}
				class="rounded-md px-4 py-2 font-medium transition-colors {activeTab === 'projects'
					? 'bg-indigo-600 text-white'
					: 'text-gray-400 hover:text-gray-200'}"
			>
				{m.admin_project_projects_tab()} ({projects.length})
			</button>
			<button
				on:click={() => (activeTab = 'reviews')}
				class="rounded-md px-4 py-2 font-medium transition-colors {activeTab === 'reviews'
					? 'bg-indigo-600 text-white'
					: 'text-gray-400 hover:text-gray-200'}"
			>
				{m.admin_project_reviews_tab()} ({reviews.length})
			</button>
			<button
				on:click={() => (activeTab = 'settings')}
				class="rounded-md px-4 py-2 font-medium transition-colors {activeTab === 'settings'
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
						on:click={() => (showAddProject = true)}
						class="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 font-medium text-white transition-colors hover:bg-indigo-700"
					>
						<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 4v16m8-8H4"
							/>
						</svg>
						{m.modal_add_project()}
					</button>
				</div>

				{#if projects.length === 0}
					<div class="rounded-xl border border-gray-700 bg-gray-800/50 py-12 text-center">
						<p class="text-gray-400">{m.gitlab_no_projects()}</p>
						<p class="mt-1 text-sm text-gray-500">{m.gitlab_no_projects_desc()}</p>
					</div>
				{:else}
					{#each projects as project}
						<div class="rounded-xl border border-gray-700 bg-gray-800 p-5">
							<div class="flex items-start justify-between">
								<div>
									<h3 class="font-semibold text-gray-100">{project.name}</h3>
									<p class="text-sm text-gray-500">{project.path_with_namespace}</p>
									<div class="mt-2 flex items-center gap-4 text-sm">
										<span class="flex items-center gap-1.5">
											<span class="h-2 w-2 rounded-full {getStatusColor(project.status)}"></span>
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
									<div class="mr-2 flex items-center gap-2">
										{#if indexingProjects.has(project.id)}
											<span class="flex items-center gap-1 text-sm text-yellow-400">
												<svg class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
													<circle
														class="opacity-25"
														cx="12"
														cy="12"
														r="10"
														stroke="currentColor"
														stroke-width="4"
													></circle>
													<path
														class="opacity-75"
														fill="currentColor"
														d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
													></path>
												</svg>
												Indexing...
											</span>
										{:else if project.index_status === 'completed'}
											<span class="text-sm text-green-400">✓ Indexed</span>
										{:else if project.index_status === 'failed'}
											<span class="text-sm text-red-400">✗ Failed</span>
										{:else}
											<span class="text-sm text-gray-500">Not indexed</span>
										{/if}
									</div>
									<button
										on:click={() => handleStartIndexing(project)}
										class="rounded-lg p-2 text-gray-400 transition-colors hover:bg-indigo-900/30 hover:text-indigo-400"
										title={project.index_status === 'completed'
											? 'Re-index project'
											: 'Index project'}
										disabled={indexingProjects.has(project.id)}
									>
										<svg
											class="h-5 w-5 {indexingProjects.has(project.id) ? 'animate-spin' : ''}"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
											/>
										</svg>
									</button>
									<button
										on:click={() => openAnalysis(project)}
										class="rounded-lg p-2 text-gray-400 transition-colors hover:bg-indigo-900/30 hover:text-indigo-400"
										title="Analyze project"
									>
										<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
											/>
										</svg>
									</button>
									<button
										on:click={() => handleDeleteProject(project.id, project.name)}
										class="rounded-lg p-2 text-gray-400 transition-colors hover:bg-red-900/30 hover:text-red-400"
										title="Delete project"
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
						</div>
					{/each}
				{/if}
			</div>
		{:else if activeTab === 'reviews'}
			<div class="space-y-3">
				{#if reviews.length === 0}
					<div class="rounded-xl border border-gray-700 bg-gray-800/50 py-12 text-center">
						<p class="text-gray-400">No reviews yet</p>
						<p class="mt-1 text-sm text-gray-500">Reviews will appear here when MRs are analyzed</p>
					</div>
				{:else}
					{#each reviews as review}
						<div class="rounded-xl border border-gray-700 bg-gray-800 p-4">
							<div class="flex items-start justify-between">
								<div>
									<h3 class="font-medium text-gray-100">
										<a href={review.mr_url} target="_blank" class="hover:text-indigo-400">
											!{review.mr_iid}: {review.mr_title}
										</a>
									</h3>
									<div class="mt-1 flex items-center gap-3 text-sm text-gray-500">
										<span>{review.source_branch} → {review.target_branch}</span>
										<span>by {review.mr_author}</span>
									</div>
								</div>
								<span
									class="flex items-center gap-1.5 rounded px-2 py-1 text-xs font-medium {getStatusColor(
										review.status
									)} bg-opacity-20"
								>
									<span class="h-1.5 w-1.5 rounded-full {getStatusColor(review.status)}"></span>
									{review.status}
								</span>
							</div>
						</div>
					{/each}
				{/if}
			</div>
		{:else if activeTab === 'settings'}
			<div class="rounded-xl border border-gray-700 bg-gray-800 p-6">
				<h3 class="mb-4 text-lg font-semibold text-gray-100">Integration Settings</h3>
				<div class="space-y-4">
					<div>
						<span class="mb-1 block text-sm font-medium text-gray-300">API Version</span>
						<p class="text-gray-400">{integration.settings.api_version || 'v4'}</p>
					</div>
					<div>
						<span class="mb-1 block text-sm font-medium text-gray-300">Request Timeout</span>
						<p class="text-gray-400">{integration.settings.request_timeout || 30} seconds</p>
					</div>
					<div>
						<span class="mb-1 block text-sm font-medium text-gray-300">Rate Limit</span>
						<p class="text-gray-400">
							{integration.settings.rate_limit_per_min || 30} requests/minute
						</p>
					</div>
					<div>
						<span class="mb-1 block text-sm font-medium text-gray-300">Webhook URL</span>
						<code class="rounded bg-gray-900 px-2 py-1 text-sm text-indigo-400">
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
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
		<div class="w-full max-w-lg rounded-xl border border-gray-700 bg-gray-800">
			<div class="flex items-center justify-between border-b border-gray-700 p-6">
				<h2 class="text-xl font-semibold text-gray-100">Add Project</h2>
				<button
					on:click={() => (showAddProject = false)}
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

			<form on:submit|preventDefault={handleAddProject} class="space-y-4 p-6">
				<div>
					<label for="user-project-id" class="mb-2 block text-sm font-medium text-gray-300"
						>GitLab Project ID *</label
					>
					<input
						id="user-project-id"
						type="number"
						bind:value={newProject.gitlab_project_id}
						placeholder="e.g., 12345"
						class="w-full rounded-lg border border-gray-700 bg-gray-900 px-4 py-2 text-gray-100"
						required
					/>
					<p class="mt-1 text-xs text-gray-500">
						Find this in GitLab: Settings → General → Project ID
					</p>
				</div>

				<div>
					<label for="user-project-name" class="mb-2 block text-sm font-medium text-gray-300"
						>Project Name *</label
					>
					<input
						id="user-project-name"
						type="text"
						bind:value={newProject.name}
						placeholder="e.g., My Project"
						class="w-full rounded-lg border border-gray-700 bg-gray-900 px-4 py-2 text-gray-100"
						required
					/>
				</div>

				<div>
					<label for="user-project-path" class="mb-2 block text-sm font-medium text-gray-300"
						>Path with Namespace</label
					>
					<input
						id="user-project-path"
						type="text"
						bind:value={newProject.path_with_namespace}
						placeholder="e.g., group/project"
						class="w-full rounded-lg border border-gray-700 bg-gray-900 px-4 py-2 text-gray-100"
					/>
				</div>

				<div>
					<label for="user-analysis-model" class="mb-2 block text-sm font-medium text-gray-300"
						>Analysis Model *</label
					>
					<input
						id="user-analysis-model"
						type="text"
						bind:value={newProject.analysis_model_id}
						placeholder="e.g., gpt-4"
						class="w-full rounded-lg border border-gray-700 bg-gray-900 px-4 py-2 text-gray-100"
						required
					/>
				</div>

				<div>
					<label for="user-embedding-model" class="mb-2 block text-sm font-medium text-gray-300"
						>Embedding Model</label
					>
					<input
						id="user-embedding-model"
						type="text"
						bind:value={newProject.embedding_model_id}
						placeholder="e.g., text-embedding-3-small"
						class="w-full rounded-lg border border-gray-700 bg-gray-900 px-4 py-2 text-gray-100"
					/>
				</div>

				<div class="flex justify-end gap-3 pt-4">
					<button
						type="button"
						on:click={() => (showAddProject = false)}
						class="px-4 py-2 font-medium text-gray-300 hover:text-gray-100"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={addingProject}
						class="flex items-center gap-2 rounded-lg bg-indigo-600 px-6 py-2 font-medium text-white hover:bg-indigo-700 disabled:bg-indigo-800"
					>
						{#if addingProject}
							<div
								class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"
							></div>
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

<!-- Analysis Modal -->
{#if showAnalysisModal && selectedProjectForAnalysis}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
		<div
			class="flex max-h-[90vh] w-full max-w-4xl flex-col overflow-y-auto rounded-xl border border-gray-700 bg-gray-800"
		>
			<div
				class="sticky top-0 z-10 flex items-center justify-between border-b border-gray-700 bg-gray-800 p-6"
			>
				<div>
					<h2 class="text-xl font-semibold text-gray-100">
						Analyze Project: {selectedProjectForAnalysis.name}
					</h2>
					<p class="text-sm text-gray-400">{selectedProjectForAnalysis.path_with_namespace}</p>
				</div>
				<button
					on:click={() => (showAnalysisModal = false)}
					class="text-gray-400 hover:text-gray-200"
					title={m.common_close()}
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

			<div class="p-6">
				{#if !analysisResult && !analysisLoading}
					<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
						<!-- Security Section -->
						<div class="col-span-1 mb-2 border-b border-gray-700 pb-2 md:col-span-2 lg:col-span-3">
							<h3 class="text-sm font-bold tracking-wider text-gray-400 uppercase">
								Security & Secrets
							</h3>
						</div>
						<button
							on:click={() => runAnalysis('secrets')}
							class="group flex flex-col items-start rounded-xl border border-gray-700 bg-gray-900/50 p-4 text-left transition-colors hover:bg-gray-700"
						>
							<div
								class="mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-orange-500/20 text-orange-500 transition-colors group-hover:bg-orange-500 group-hover:text-white"
							>
								<svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
									/>
								</svg>
							</div>
							<span class="font-bold text-gray-100">Secrets Scan</span>
							<span class="mt-1 text-xs text-gray-400"
								>Search for leaked credentials, tokens and keys</span
							>
						</button>

						<button
							on:click={() => runAnalysis('deep-scan')}
							class="group flex flex-col items-start rounded-xl border border-gray-700 bg-gray-900/50 p-4 text-left transition-colors hover:bg-gray-700"
						>
							<div
								class="mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-red-500/20 text-red-500 transition-colors group-hover:bg-red-500 group-hover:text-white"
							>
								<svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM10 7v3m0 0v3m0-3h3m-3 0H7"
									/>
								</svg>
							</div>
							<span class="font-bold text-gray-100">Deep Security Scan</span>
							<span class="mt-1 text-xs text-gray-400"
								>Deep analysis of vulnerable code patterns</span
							>
						</button>

						<!-- Quality Section -->
						<div
							class="col-span-1 mt-4 mb-2 border-b border-gray-700 pb-2 md:col-span-2 lg:col-span-3"
						>
							<h3 class="text-sm font-bold tracking-wider text-gray-400 uppercase">
								Quality & Maintenance
							</h3>
						</div>
						<button
							on:click={() => runAnalysis('quality')}
							class="group flex flex-col items-start rounded-xl border border-gray-700 bg-gray-900/50 p-4 text-left transition-colors hover:bg-gray-700"
						>
							<div
								class="mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/20 text-blue-500 transition-colors group-hover:bg-blue-500 group-hover:text-white"
							>
								<svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z"
									/>
								</svg>
							</div>
							<span class="font-bold text-gray-100">Quality Analysis</span>
							<span class="mt-1 text-xs text-gray-400"
								>Review code readability, complexity and bugs</span
							>
						</button>

						<button
							on:click={() => runAnalysis('duplication')}
							class="group flex flex-col items-start rounded-xl border border-gray-700 bg-gray-900/50 p-4 text-left transition-colors hover:bg-gray-700"
						>
							<div
								class="mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/20 text-purple-500 transition-colors group-hover:bg-purple-500 group-hover:text-white"
							>
								<svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M8 7v8a2 2 0 002 2h6M8 7V5a2 2 0 012-2h4.586a1 1 0 01.707.293l4.414 4.414a1 1 0 01.293.707V15a2 2 0 01-2 2h-2M8 7H6a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2v-2"
									/>
								</svg>
							</div>
							<span class="font-bold text-gray-100">Detect Duplication</span>
							<span class="mt-1 text-xs text-gray-400"
								>Find similar or identical blocks of code</span
							>
						</button>

						<button
							on:click={() => runAnalysis('dead-code')}
							class="group flex flex-col items-start rounded-xl border border-gray-700 bg-gray-900/50 p-4 text-left transition-colors hover:bg-gray-700"
						>
							<div
								class="mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-gray-500/20 text-gray-400 transition-colors group-hover:bg-gray-500 group-hover:text-white"
							>
								<svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
									/>
								</svg>
							</div>
							<span class="font-bold text-gray-100">Dead Code</span>
							<span class="mt-1 text-xs text-gray-400"
								>Identify unused imports, variables and functions</span
							>
						</button>

						<!-- DevOps Section -->
						<div
							class="col-span-1 mt-4 mb-2 border-b border-gray-700 pb-2 md:col-span-2 lg:col-span-3"
						>
							<h3 class="text-sm font-bold tracking-wider text-gray-400 uppercase">
								DevOps & Tooling
							</h3>
						</div>
						<button
							on:click={() => runAnalysis('dependencies')}
							class="group flex flex-col items-start rounded-xl border border-gray-700 bg-gray-900/50 p-4 text-left transition-colors hover:bg-gray-700"
						>
							<div
								class="mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/20 text-green-500 transition-colors group-hover:bg-green-500 group-hover:text-white"
							>
								<svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
									/>
								</svg>
							</div>
							<span class="font-bold text-gray-100">Dependencies</span>
							<span class="mt-1 text-xs text-gray-400"
								>Check for outdated or vulnerable dependencies</span
							>
						</button>

						<button
							on:click={() => runAnalysis('docs')}
							class="group flex flex-col items-start rounded-xl border border-gray-700 bg-gray-900/50 p-4 text-left transition-colors hover:bg-gray-700"
						>
							<div
								class="mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-500/20 text-emerald-500 transition-colors group-hover:bg-emerald-500 group-hover:text-white"
							>
								<svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
									/>
								</svg>
							</div>
							<span class="font-bold text-gray-100">Documentation</span>
							<span class="mt-1 text-xs text-gray-400"
								>Scan for undocumented code and generate docs</span
							>
						</button>

						<button
							on:click={() => runAnalysis('tests')}
							class="group flex flex-col items-start rounded-xl border border-gray-700 bg-gray-900/50 p-4 text-left transition-colors hover:bg-gray-700"
						>
							<div
								class="mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-500/20 text-indigo-500 transition-colors group-hover:bg-indigo-500 group-hover:text-white"
							>
								<svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"
									/>
								</svg>
							</div>
							<span class="font-bold text-gray-100">Test Generation</span>
							<span class="mt-1 text-xs text-gray-400"
								>Identify testable code and generate unit tests</span
							>
						</button>
					</div>
				{:else if analysisLoading}
					<div class="flex flex-col items-center justify-center py-20">
						<div
							class="mb-4 h-16 w-16 animate-spin rounded-full border-4 border-indigo-500 border-t-transparent"
						></div>
						<h3 class="text-xl font-bold text-gray-100">Running {analysisType} scan...</h3>
						<p class="mt-2 text-gray-400 italic">
							This may take a few minutes for large repositories
						</p>
					</div>
				{:else if analysisError}
					<div class="p-8 text-center">
						<div
							class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-red-900/30 text-red-500"
						>
							<svg class="h-10 w-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
								/>
							</svg>
						</div>
						<h3 class="mb-2 text-xl font-bold text-gray-100">Analysis Failed</h3>
						<p class="mb-6 text-red-400">{analysisError}</p>
						<button
							on:click={() => (analysisResult = null)}
							class="rounded-lg bg-gray-700 px-6 py-2 text-white transition-colors hover:bg-gray-600"
						>
							Back to Scanners
						</button>
					</div>
				{:else if analysisResult}
					<div class="animate-in fade-in slide-in-from-bottom-4 space-y-6 duration-500">
						<div class="flex items-center justify-between">
							<h3 class="text-xl font-bold text-gray-100 capitalize">{analysisType} Results</h3>
							<button
								on:click={() => (analysisResult = null)}
								class="text-sm font-medium text-indigo-400 hover:text-indigo-300"
							>
								← Back to all scanners
							</button>
						</div>

						<!-- Results summary -->
						<div class="grid grid-cols-1 gap-4 md:grid-cols-3">
							{#if analysisType === 'secrets'}
								<div class="rounded-xl border border-gray-700 bg-gray-900 p-4">
									<span class="text-xs font-bold text-gray-500 uppercase">Total Secrets</span>
									<div class="mt-1 text-3xl font-bold text-orange-500">
										{(analysisResult.secrets || []).length}
									</div>
								</div>
								<div class="rounded-xl border border-gray-700 bg-gray-900 p-4">
									<span class="text-xs font-bold text-gray-500 uppercase">Files Scanned</span>
									<div class="mt-1 text-3xl font-bold text-blue-400">
										{analysisResult.files_scanned || 0}
									</div>
								</div>
								<div class="rounded-xl border border-gray-700 bg-gray-900 p-4">
									<span class="text-xs font-bold text-gray-500 uppercase">Severity</span>
									<div class="mt-1 text-3xl font-bold text-red-500">High</div>
								</div>
							{:else if analysisType === 'quality'}
								<div class="rounded-xl border border-gray-700 bg-gray-900 p-4">
									<span class="text-xs font-bold text-gray-500 uppercase">Overall Score</span>
									<div class="mt-1 text-3xl font-bold text-green-400">
										{analysisResult.score || 0}/100
									</div>
								</div>
								<div class="rounded-xl border border-gray-700 bg-gray-900 p-4">
									<span class="text-xs font-bold text-gray-500 uppercase">Issues Found</span>
									<div class="mt-1 text-3xl font-bold text-yellow-500">
										{(analysisResult.issues || []).length}
									</div>
								</div>
								<div class="rounded-xl border border-gray-700 bg-gray-900 p-4">
									<span class="text-xs font-bold text-gray-500 uppercase">Files Analyzed</span>
									<div class="mt-1 text-3xl font-bold text-blue-400">
										{analysisResult.files_count || 0}
									</div>
								</div>
							{:else}
								<div class="col-span-3 rounded-xl border border-gray-700 bg-gray-900 p-4">
									<span class="text-xs font-bold text-gray-500 uppercase">Status</span>
									<div class="mt-1 text-xl font-bold text-green-400">
										Scan completed successfully
									</div>
								</div>
							{/if}
						</div>

						<!-- Results list placeholder -->
						<div class="overflow-hidden rounded-xl border border-gray-700 bg-gray-900/30">
							<div class="border-b border-gray-700 bg-gray-900/50 p-4">
								<h4 class="font-bold text-gray-200">Detailed Findings</h4>
							</div>
							<div class="p-6">
								<pre class="font-mono text-xs whitespace-pre-wrap text-gray-400">
									{JSON.stringify(analysisResult, null, 2)}
								</pre>
							</div>
						</div>

						<div class="flex justify-end gap-3 border-t border-gray-700 pt-4">
							<button
								on:click={() => (showAnalysisModal = false)}
								class="rounded-lg bg-gray-700 px-6 py-2 text-white transition-colors hover:bg-gray-600"
							>
								Dismiss
							</button>
							<button
								class="rounded-lg bg-indigo-600 px-6 py-2 text-white transition-colors hover:bg-indigo-700"
								on:click={() => alert('Feature coming soon: Create issues from results')}
							>
								Create GitLab Issues
							</button>
						</div>
					</div>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	:global(body) {
		background-color: #111827;
	}
</style>
