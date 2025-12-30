<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		GitBranch,
		Plus,
		RefreshCw,
		Loader2,
		Trash2,
		Settings,
		X,
		Search,
		ExternalLink,
		CheckCircle,
		XCircle,
		AlertCircle,
		Play,
		Eye,
		ArrowLeft,
		Webhook,
		Bot,
		FileCode,
		Clock,
		BarChart3
	} from 'lucide-svelte';
	import { gitlabApi, type GitLabIntegration, type GitLabProject, type GitLabReview, type SecretsScanResult, type SecretFinding, type DeepScanResult, type DeepFinding } from '$lib/api/gitlab';
	import { Shield, Brain } from 'lucide-svelte';
	import { cn, formatRelativeTime, debounce } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import * as m from '$lib/paraglide/messages';

	let integration = $state<GitLabIntegration | null>(null);
	let projects = $state<GitLabProject[]>([]);
	let reviews = $state<GitLabReview[]>([]);
	let totalProjects = $state(0);
	let totalReviews = $state(0);
	let isLoading = $state(true);
	let activeTab = $state<'projects' | 'reviews'>('projects');

	// Filters
	let searchQuery = $state('');
	let statusFilter = $state('');

	// Pagination
	let currentPage = $state(0);
	let pageSize = 10;

	// Modals
	let showAddProjectModal = $state(false);
	let showProjectDetailModal = $state(false);
	let showEditProjectModal = $state(false);
	let showReviewDetailModal = $state(false);
	let selectedProject = $state<GitLabProject | null>(null);
	let selectedReview = $state<GitLabReview | null>(null);

	// Edit Project form
	let editAnalysisModelId = $state('');
	let editEmbeddingModelId = $state('');
	let editAutoReview = $state(true);
	let editReviewPrompt = $state('');
	let editIncludePatterns = $state('');
	let editExcludePatterns = $state('');
	let editChunkSize = $state('4000');
	let editChunkOverlap = $state('200');
	let editMaxFilesPerMR = $state('50');
	let editMaxLinesPerFile = $state('1000');
	let editMaxReviewTokens = $state('8192');
	let editSkipDraftMRs = $state(true);
	let editSkipBots = $state(true);
	let editPerFileReview = $state(false);
	let editReviewLanguage = $state('en');
	let editCollectionName = $state('');
	let editStatus = $state<'active' | 'disabled'>('active');
	let isSaving = $state(false);
	let saveError = $state('');

	// Indexing state
	let indexingProjects = $state<Set<string>>(new Set());

	// Secrets scanning state
	let showSecretsModal = $state(false);
	let secretsScanResult = $state<SecretsScanResult | null>(null);
	let isScanning = $state(false);
	let scanningProjectId = $state<string | null>(null);
	
	// Deep scan (LLM) state
	let showDeepScanModal = $state(false);
	let deepScanResult = $state<DeepScanResult | null>(null);
	let isDeepScanning = $state(false);
	let deepScanningProjectId = $state<string | null>(null);

	// Add Project form
	let formGitLabProjectId = $state('');
	let formAnalysisModelId = $state('');
	let formEmbeddingModelId = $state('');
	let formAutoReview = $state(true);
	let formReviewPrompt = $state('');
	// Include/Exclude patterns (GITLAB-055c)
	let formIncludePatterns = $state('');
	let formExcludePatterns = $state('');
	// Advanced settings (GITLAB-055e)
	let formChunkSize = $state('4000');
	let formChunkOverlap = $state('200');
	let formMaxFilesPerMR = $state('50');
	let formMaxLinesPerFile = $state('1000');
	let formMaxReviewTokens = $state('8192');
	let formSkipDraftMRs = $state(true);
	let formSkipBots = $state(true);
	let showAdvancedSettings = $state(false);
	let isAdding = $state(false);
	let addError = $state('');

	const integrationId = $derived($page.params.id);

	onMount(async () => {
		await loadIntegration();
		await loadProjects();
	});

	const debouncedSearch = debounce(() => {
		currentPage = 0;
		if (activeTab === 'projects') {
			loadProjects();
		} else {
			loadReviews();
		}
	}, 300);

	async function loadIntegration() {
		try {
			integration = await gitlabApi.getIntegration(integrationId);
		} catch (error) {
			console.error('Failed to load integration:', error);
			goto('/admin/gitlab');
		}
	}

	async function loadProjects() {
		isLoading = true;
		try {
			const response = await gitlabApi.listProjects(integrationId, {
				limit: pageSize,
				offset: currentPage * pageSize,
				search: searchQuery || undefined,
				status: statusFilter || undefined
			});
			projects = response.data || [];
			totalProjects = response.total || projects.length;
		} catch (error) {
			console.error('Failed to load projects:', error);
			projects = [];
		} finally {
			isLoading = false;
		}
	}

	async function loadReviews() {
		isLoading = true;
		try {
			const response = await gitlabApi.listReviews({
				limit: pageSize,
				offset: currentPage * pageSize,
				status: statusFilter || undefined,
				search: searchQuery || undefined
			});
			reviews = response.data || [];
			totalReviews = response.total || reviews.length;
		} catch (error) {
			console.error('Failed to load reviews:', error);
			reviews = [];
		} finally {
			isLoading = false;
		}
	}

	function switchTab(tab: 'projects' | 'reviews') {
		activeTab = tab;
		currentPage = 0;
		searchQuery = '';
		statusFilter = '';
		if (tab === 'projects') {
			loadProjects();
		} else {
			loadReviews();
		}
	}

	function openAddProjectModal() {
		formGitLabProjectId = '';
		formAnalysisModelId = '';
		formEmbeddingModelId = '';
		formAutoReview = true;
		formReviewPrompt = '';
		// Reset patterns (GITLAB-055c)
		formIncludePatterns = '';
		formExcludePatterns = '';
		// Reset advanced settings (GITLAB-055e)
		formChunkSize = '4000';
		formChunkOverlap = '200';
		formMaxFilesPerMR = '50';
		formMaxLinesPerFile = '1000';
		formMaxReviewTokens = '8192';
		formSkipDraftMRs = true;
		formSkipBots = true;
		showAdvancedSettings = false;
		addError = '';
		showAddProjectModal = true;
	}

	async function handleAddProject() {
		const projectIdStr = String(formGitLabProjectId).trim();
		if (!projectIdStr) {
			addError = 'GitLab Project ID is required';
			return;
		}
		if (!formAnalysisModelId.trim()) {
			addError = 'Analysis Model is required';
			return;
		}

		isAdding = true;
		addError = '';

		// Build settings object
		const settings: Record<string, unknown> = {};
		
		// Include/Exclude patterns (GITLAB-055c)
		if (formIncludePatterns.trim()) {
			settings.include_patterns = formIncludePatterns.split('\n').map(p => p.trim()).filter(Boolean);
		}
		if (formExcludePatterns.trim()) {
			settings.exclude_patterns = formExcludePatterns.split('\n').map(p => p.trim()).filter(Boolean);
		}
		
		// Advanced settings (GITLAB-055e)
		const chunkSize = parseInt(formChunkSize, 10);
		const chunkOverlap = parseInt(formChunkOverlap, 10);
		const maxFiles = parseInt(formMaxFilesPerMR, 10);
		const maxLines = parseInt(formMaxLinesPerFile, 10);
		
		const maxReviewTokens = parseInt(formMaxReviewTokens, 10);
		
		if (!isNaN(chunkSize) && chunkSize !== 4000) settings.chunk_size = chunkSize;
		if (!isNaN(chunkOverlap) && chunkOverlap !== 200) settings.chunk_overlap = chunkOverlap;
		if (!isNaN(maxFiles) && maxFiles !== 50) settings.max_files_per_mr = maxFiles;
		if (!isNaN(maxLines) && maxLines !== 1000) settings.max_lines_per_file = maxLines;
		if (!isNaN(maxReviewTokens) && maxReviewTokens !== 8192) settings.max_review_tokens = maxReviewTokens;
		if (!formSkipDraftMRs) settings.skip_draft_mrs = false;
		if (!formSkipBots) settings.skip_bots = false;

		try {
			const newProject = await gitlabApi.addProject(integrationId, {
				gitlab_project_id: parseInt(projectIdStr, 10),
				analysis_model_id: formAnalysisModelId.trim(),
				embedding_model_id: formEmbeddingModelId.trim() || undefined,
				auto_review: formAutoReview,
				review_prompt: formReviewPrompt.trim() || undefined,
				settings: Object.keys(settings).length > 0 ? settings : undefined
			});

			projects = [newProject, ...projects];
			totalProjects++;
			showAddProjectModal = false;
		} catch (error) {
			addError = error instanceof Error ? error.message : 'Failed to add project';
		} finally {
			isAdding = false;
		}
	}

	async function handleSetupWebhook(project: GitLabProject) {
		const webhookUrl = `${window.location.origin}/api/gitlab/webhook/${integrationId}`;
		
		try {
			const result = await gitlabApi.setupWebhook(project.id, webhookUrl);
			// Refresh projects
			await loadProjects();
			alert(m.alert_webhook_created({ id: result.webhook_id }));
		} catch (error) {
			alert(m.alert_failed_webhook() + ': ' + (error instanceof Error ? error.message : 'Unknown error'));
		}
	}

	async function handleDeleteProject(project: GitLabProject) {
		if (!confirm(m.confirm_remove_project({ name: project.path_with_namespace }))) {
			return;
		}

		try {
			await gitlabApi.deleteProject(project.id);
			projects = projects.filter((p) => p.id !== project.id);
			totalProjects--;
		} catch (error) {
			alert(m.alert_failed_delete_project());
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
			await gitlabApi.startIndexing(project.id, { 
				branch: project.default_branch || 'main',
				force 
			});
			
			// Poll for status updates
			pollIndexStatus(project.id);
		} catch (error) {
			console.error('Failed to start indexing:', error);
			indexingProjects = new Set([...indexingProjects].filter(id => id !== project.id));
			alert(m.alert_failed_indexing());
		}
	}

	async function pollIndexStatus(projectId: string) {
		const maxAttempts = 60; // 5 minutes max
		let attempts = 0;

		const poll = async () => {
			try {
				const status = await gitlabApi.getIndexStatus(projectId);
				
				// Update project in list
				projects = projects.map(p => {
					if (p.id === projectId) {
						return { 
							...p, 
							index_status: status.status as GitLabProject['index_status'],
							last_indexed_at: status.last_indexed 
						};
					}
					return p;
				});

				if (status.status === 'in_progress' && attempts < maxAttempts) {
					attempts++;
					setTimeout(poll, 5000); // Poll every 5 seconds
				} else {
					// Done or failed
					indexingProjects = new Set([...indexingProjects].filter(id => id !== projectId));
				}
			} catch {
				indexingProjects = new Set([...indexingProjects].filter(id => id !== projectId));
			}
		};

		poll();
	}

	async function handleRetryReview(review: GitLabReview) {
		try {
			await gitlabApi.retryReview(review.id);
			// Refresh reviews
			await loadReviews();
		} catch (error) {
			alert(m.alert_failed_retry_review());
		}
	}

	async function handleScanSecrets(project: GitLabProject) {
		if (project.index_status !== 'completed') {
			alert(m.alert_index_required_for_scan?.() || 'Please index the repository first before scanning for secrets.');
			return;
		}

		scanningProjectId = project.id;
		isScanning = true;
		secretsScanResult = null;
		showSecretsModal = true;

		try {
			const result = await gitlabApi.scanSecrets(project.id);
			secretsScanResult = result;
		} catch (error) {
			console.error('Secrets scan failed:', error);
			secretsScanResult = {
				project_id: project.id,
				scan_id: '',
				started_at: new Date().toISOString(),
				completed_at: new Date().toISOString(),
				duration: '0s',
				chunks_scanned: 0,
				findings: [],
				summary: { total_findings: 0, by_severity: {}, by_category: {}, files_affected: 0 },
				status: 'failed',
				error: error instanceof Error ? error.message : 'Scan failed'
			};
		} finally {
			isScanning = false;
			scanningProjectId = null;
		}
	}
	
	async function handleDeepScan(project: GitLabProject) {
		if (project.index_status !== 'completed') {
			alert(m.alert_index_required_for_scan?.() || 'Please index the repository first before scanning for secrets.');
			return;
		}
		
		if (!project.analysis_model_id) {
			alert(m.alert_analysis_model_required?.() || 'Please configure an analysis model for this project first.');
			return;
		}

		deepScanningProjectId = project.id;
		isDeepScanning = true;
		deepScanResult = null;
		showDeepScanModal = true;

		try {
			const result = await gitlabApi.deepScanSecrets(project.id);
			deepScanResult = result;
		} catch (error) {
			console.error('Deep secrets scan failed:', error);
			deepScanResult = {
				project_id: project.id,
				scan_id: '',
				model_id: project.analysis_model_id || '',
				started_at: new Date().toISOString(),
				completed_at: new Date().toISOString(),
				duration: '0s',
				chunks_scanned: 0,
				tokens_used: 0,
				findings: [],
				summary: { total_findings: 0, by_severity: {}, by_category: {}, files_affected: 0 },
				status: 'failed',
				error: error instanceof Error ? error.message : 'Deep scan failed'
			};
		} finally {
			isDeepScanning = false;
			deepScanningProjectId = null;
		}
	}
	
	function getConfidenceColor(confidence: string): string {
		switch (confidence) {
			case 'high': return 'text-green-600 bg-green-100 dark:bg-green-900/30';
			case 'medium': return 'text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30';
			case 'low': return 'text-gray-600 bg-gray-100 dark:bg-gray-900/30';
			default: return 'text-gray-600 bg-gray-100 dark:bg-gray-900/30';
		}
	}

	function getSeverityColor(severity: string): string {
		switch (severity) {
			case 'critical': return 'text-red-600 bg-red-100 dark:bg-red-900/30';
			case 'high': return 'text-orange-600 bg-orange-100 dark:bg-orange-900/30';
			case 'medium': return 'text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30';
			case 'low': return 'text-blue-600 bg-blue-100 dark:bg-blue-900/30';
			default: return 'text-gray-600 bg-gray-100 dark:bg-gray-900/30';
		}
	}

	function getSeverityIcon(severity: string) {
		switch (severity) {
			case 'critical':
			case 'high':
				return XCircle;
			case 'medium':
				return AlertCircle;
			default:
				return CheckCircle;
		}
	}

	function viewProjectDetails(project: GitLabProject) {
		selectedProject = project;
		showProjectDetailModal = true;
	}

	function openEditProject(project: GitLabProject) {
		selectedProject = project;
		// Populate edit form with current values
		editAnalysisModelId = project.analysis_model_id || '';
		editEmbeddingModelId = project.embedding_model_id || '';
		editAutoReview = project.auto_review;
		editReviewPrompt = project.review_prompt || '';
		editStatus = project.status as 'active' | 'disabled';
		editIncludePatterns = project.settings?.include_patterns?.join(', ') || '';
		editExcludePatterns = project.settings?.exclude_patterns?.join(', ') || '';
		editChunkSize = String(project.settings?.chunk_size || 4000);
		editChunkOverlap = String(project.settings?.chunk_overlap || 200);
		editMaxFilesPerMR = String(project.settings?.max_files_per_mr || 50);
		editMaxLinesPerFile = String(project.settings?.max_lines_per_file || 1000);
		editMaxReviewTokens = String(project.settings?.max_review_tokens || 8192);
		editSkipDraftMRs = project.settings?.skip_draft_mrs ?? true;
		editSkipBots = project.settings?.skip_bots ?? true;
		editPerFileReview = project.settings?.per_file_review ?? false;
		editReviewLanguage = project.settings?.review_language || 'en';
		editCollectionName = project.settings?.collection_name || '';
		saveError = '';
		showEditProjectModal = true;
	}

	async function saveProjectChanges() {
		if (!selectedProject) return;

		isSaving = true;
		saveError = '';

		try {
			const includePatterns = editIncludePatterns
				.split(',')
				.map(p => p.trim())
				.filter(p => p);
			const excludePatterns = editExcludePatterns
				.split(',')
				.map(p => p.trim())
				.filter(p => p);

			await gitlabApi.updateProject(selectedProject.id, {
				auto_review: editAutoReview,
				status: editStatus,
				analysis_model_id: editAnalysisModelId || undefined,
				embedding_model_id: editEmbeddingModelId || undefined,
				review_prompt: editReviewPrompt || undefined,
				settings: {
					include_patterns: includePatterns.length > 0 ? includePatterns : undefined,
					exclude_patterns: excludePatterns.length > 0 ? excludePatterns : undefined,
					chunk_size: parseInt(editChunkSize) || 4000,
					chunk_overlap: parseInt(editChunkOverlap) || 200,
					max_files_per_mr: parseInt(editMaxFilesPerMR) || 50,
					max_lines_per_file: parseInt(editMaxLinesPerFile) || 1000,
					max_review_tokens: parseInt(editMaxReviewTokens) || 8192,
					skip_draft_mrs: editSkipDraftMRs,
					skip_bots: editSkipBots,
					per_file_review: editPerFileReview,
					review_language: editReviewLanguage,
					collection_name: editCollectionName || undefined
				}
			});

			showEditProjectModal = false;
			await loadProjects();
		} catch (error) {
			saveError = error instanceof Error ? error.message : 'Failed to save changes';
		} finally {
			isSaving = false;
		}
	}

	function viewReviewDetails(review: GitLabReview) {
		selectedReview = review;
		showReviewDetailModal = true;
	}

	function getStatusIcon(status: string) {
		switch (status) {
			case 'active':
			case 'completed':
				return CheckCircle;
			case 'error':
			case 'failed':
				return XCircle;
			case 'pending':
			case 'queued':
			case 'analyzing':
				return Clock;
			default:
				return AlertCircle;
		}
	}

	function getStatusColor(status: string) {
		switch (status) {
			case 'active':
			case 'completed':
				return 'text-green-500';
			case 'error':
			case 'failed':
				return 'text-red-500';
			case 'pending':
			case 'queued':
			case 'analyzing':
				return 'text-yellow-500';
			default:
				return 'text-muted-foreground';
		}
	}

	const totalPages = $derived(Math.ceil((activeTab === 'projects' ? totalProjects : totalReviews) / pageSize));
</script>

<div class="space-y-6">
	<!-- Header -->
	<div class="flex items-center gap-4">
		<a href="/admin/gitlab" class="rounded p-2 hover:bg-muted">
			<ArrowLeft class="h-5 w-5" />
		</a>
		<div class="flex-1">
			<h2 class="text-2xl font-bold text-foreground">{integration?.name || 'Loading...'}</h2>
			{#if integration}
				<a href={integration.base_url} target="_blank" class="text-sm text-primary hover:underline flex items-center gap-1">
					{integration.base_url}
					<ExternalLink class="h-3 w-3" />
				</a>
			{/if}
		</div>
		{#if activeTab === 'projects'}
			<Button onclick={openAddProjectModal}>
				<Plus class="mr-2 h-4 w-4" />
				{m.common_add()} {m.admin_project_projects_tab()}
			</Button>
		{/if}
	</div>

	<!-- Tabs -->
	<div class="flex gap-2 border-b">
		<button
			onclick={() => switchTab('projects')}
			class={cn(
				'px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors',
				activeTab === 'projects'
					? 'border-primary text-primary'
					: 'border-transparent text-muted-foreground hover:text-foreground'
			)}
		>
			<FileCode class="inline-block mr-2 h-4 w-4" />
			{m.admin_project_projects_tab()} ({totalProjects})
		</button>
		<button
			onclick={() => switchTab('reviews')}
			class={cn(
				'px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors',
				activeTab === 'reviews'
					? 'border-primary text-primary'
					: 'border-transparent text-muted-foreground hover:text-foreground'
			)}
		>
			<BarChart3 class="inline-block mr-2 h-4 w-4" />
			{m.admin_project_reviews_tab()} ({totalReviews})
		</button>
	</div>

	<!-- Filters -->
	<div class="flex flex-wrap items-center gap-4">
		<div class="relative flex-1 min-w-[200px]">
			<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
			<input
				type="text"
				placeholder={activeTab === 'projects' ? 'Search projects...' : 'Search reviews...'}
				bind:value={searchQuery}
				oninput={debouncedSearch}
				class="h-10 w-full rounded-md border bg-background pl-10 pr-4 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
			/>
		</div>
		<select
			bind:value={statusFilter}
			onchange={() => { currentPage = 0; activeTab === 'projects' ? loadProjects() : loadReviews(); }}
			class="h-10 rounded-md border bg-background px-3 text-sm"
		>
			<option value="">{m.admin_gitlab_all_statuses()}</option>
			{#if activeTab === 'projects'}
				<option value="active">{m.common_active()}</option>
				<option value="disabled">{m.common_disabled()}</option>
			{:else}
				<option value="pending">{m.admin_gitlab_pending()}</option>
				<option value="analyzing">{m.admin_gitlab_processing()}</option>
				<option value="completed">{m.admin_queue_completed()}</option>
				<option value="failed">{m.admin_gitlab_failed()}</option>
			{/if}
		</select>
		<Button variant="outline" onclick={() => { searchQuery = ''; statusFilter = ''; currentPage = 0; activeTab === 'projects' ? loadProjects() : loadReviews(); }}>
			<RefreshCw class="mr-2 h-4 w-4" />
			{m.common_reset()}
		</Button>
	</div>

	<!-- Content -->
	<div class="rounded-lg border bg-card">
		{#if isLoading}
			<div class="flex items-center justify-center py-12">
				<Loader2 class="h-8 w-8 animate-spin text-primary" />
			</div>
		{:else if activeTab === 'projects'}
			<!-- Projects Table -->
			{#if projects.length === 0}
				<div class="flex flex-col items-center justify-center py-12 text-center">
					<FileCode class="h-12 w-12 text-muted-foreground/50" />
					<p class="mt-4 text-lg font-medium text-muted-foreground">No projects configured</p>
					<p class="text-sm text-muted-foreground">Add a GitLab project to start AI reviews</p>
					<Button class="mt-4" onclick={openAddProjectModal}>
						<Plus class="mr-2 h-4 w-4" />
						Add Project
					</Button>
				</div>
			{:else}
				<table class="w-full">
					<thead>
						<tr class="border-b text-left text-sm text-muted-foreground">
							<th class="px-4 py-3 font-medium">Project</th>
							<th class="px-4 py-3 font-medium">Status</th>
							<th class="px-4 py-3 font-medium">Auto Review</th>
							<th class="px-4 py-3 font-medium">Webhook</th>
							<th class="px-4 py-3 font-medium">Index</th>
							<th class="px-4 py-3 font-medium">Reviews</th>
							<th class="px-4 py-3 font-medium">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each projects as project}
							<tr class="border-b last:border-0 hover:bg-muted/50">
								<td class="px-4 py-3">
									<div class="font-medium">{project.name}</div>
									<div class="text-sm text-muted-foreground">{project.path_with_namespace}</div>
								</td>
								<td class="px-4 py-3">
									{#if true}
										{@const StatusIcon = getStatusIcon(project.status)}
										<div class="flex items-center gap-2">
											<StatusIcon class={cn('h-4 w-4', getStatusColor(project.status))} />
											<span class="text-sm capitalize">{project.status}</span>
										</div>
									{/if}
								</td>
								<td class="px-4 py-3">
									<span class={cn('text-sm', project.auto_review ? 'text-green-500' : 'text-muted-foreground')}>
										{project.auto_review ? 'Enabled' : 'Disabled'}
									</span>
								</td>
								<td class="px-4 py-3">
									{#if project.webhook_id}
										<span class="flex items-center gap-1 text-sm text-green-500">
											<Webhook class="h-4 w-4" />
											#{project.webhook_id}
										</span>
									{:else}
										<button
											onclick={() => handleSetupWebhook(project)}
											class="text-sm text-primary hover:underline flex items-center gap-1"
										>
											<Webhook class="h-4 w-4" />
											Setup
										</button>
									{/if}
								</td>
								<td class="px-4 py-3">
								{#if indexingProjects.has(project.id)}
									<span class="flex items-center gap-1 text-sm text-yellow-500">
										<Loader2 class="h-4 w-4 animate-spin" />
										Indexing...
									</span>
								{:else if project.index_status === 'completed'}
									<div class="flex flex-col">
										<span class="flex items-center gap-1 text-sm text-green-500">
											<CheckCircle class="h-4 w-4" />
											{project.last_indexed_at ? formatRelativeTime(project.last_indexed_at) : 'Ready'}
										</span>
										{#if project.index_chunks && project.index_chunks > 0}
											<span class="text-xs text-muted-foreground">{project.index_chunks.toLocaleString()} chunks</span>
										{/if}
									</div>
								{:else if project.index_status === 'failed'}
									<span class="flex items-center gap-1 text-sm text-red-500">
										<XCircle class="h-4 w-4" />
										Failed
									</span>
								{:else}
									<span class="text-sm text-muted-foreground">Not indexed</span>
								{/if}
								</td>
								<td class="px-4 py-3">
									<span class="text-sm">{project.review_count || 0}</span>
								</td>
								<td class="px-4 py-3">
									<div class="flex items-center gap-2">
										<button
											onclick={() => handleStartIndexing(project)}
											class="rounded p-1.5 hover:bg-muted"
											title={project.index_status === 'completed' ? 'Reindex' : 'Start Indexing'}
											disabled={indexingProjects.has(project.id)}
										>
											<RefreshCw class={cn('h-4 w-4', indexingProjects.has(project.id) && 'animate-spin')} />
										</button>
										<button
											onclick={() => viewProjectDetails(project)}
											class="rounded p-1.5 hover:bg-muted"
											title="View Details"
										>
											<Eye class="h-4 w-4" />
										</button>
										<button
											onclick={() => handleScanSecrets(project)}
											class={cn(
												'rounded p-1.5 hover:bg-muted',
												project.index_status !== 'completed' && 'opacity-50 cursor-not-allowed'
											)}
											title={project.index_status === 'completed' ? m.gitlab_scan_secrets?.() || 'Scan Secrets (Regex)' : m.gitlab_index_first?.() || 'Index first'}
											disabled={scanningProjectId === project.id || project.index_status !== 'completed'}
										>
											{#if scanningProjectId === project.id}
												<Loader2 class="h-4 w-4 animate-spin" />
											{:else}
												<Shield class="h-4 w-4" />
											{/if}
										</button>
										<button
											onclick={() => handleDeepScan(project)}
											class={cn(
												'rounded p-1.5 hover:bg-muted',
												(project.index_status !== 'completed' || !project.analysis_model_id) && 'opacity-50 cursor-not-allowed'
											)}
											title={project.index_status === 'completed' && project.analysis_model_id ? m.gitlab_deep_scan?.() || 'Deep Scan (LLM)' : m.gitlab_index_and_model_required?.() || 'Requires index + analysis model'}
											disabled={deepScanningProjectId === project.id || project.index_status !== 'completed' || !project.analysis_model_id}
										>
											{#if deepScanningProjectId === project.id}
												<Loader2 class="h-4 w-4 animate-spin" />
											{:else}
												<Brain class="h-4 w-4 text-purple-500" />
											{/if}
										</button>
										<button
											onclick={() => openEditProject(project)}
											class="rounded p-1.5 hover:bg-muted"
											title="Edit Settings"
										>
											<Settings class="h-4 w-4" />
										</button>
										<button
											onclick={() => handleDeleteProject(project)}
											class="rounded p-1.5 text-red-500 hover:bg-red-500/10"
											title="Remove"
										>
											<Trash2 class="h-4 w-4" />
										</button>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		{:else}
			<!-- Reviews Table -->
			{#if reviews.length === 0}
				<div class="flex flex-col items-center justify-center py-12 text-center">
					<BarChart3 class="h-12 w-12 text-muted-foreground/50" />
					<p class="mt-4 text-lg font-medium text-muted-foreground">No reviews yet</p>
					<p class="text-sm text-muted-foreground">Reviews will appear here when MRs are analyzed</p>
				</div>
			{:else}
				<table class="w-full">
					<thead>
						<tr class="border-b text-left text-sm text-muted-foreground">
							<th class="px-4 py-3 font-medium">MR</th>
							<th class="px-4 py-3 font-medium">Status</th>
							<th class="px-4 py-3 font-medium">Files</th>
							<th class="px-4 py-3 font-medium">Issues</th>
							<th class="px-4 py-3 font-medium">Time</th>
							<th class="px-4 py-3 font-medium">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each reviews as review}
							<tr class="border-b last:border-0 hover:bg-muted/50">
								<td class="px-4 py-3">
									<div class="font-medium">!{review.mr_iid}: {review.mr_title}</div>
									<div class="text-sm text-muted-foreground">
										{review.source_branch} → {review.target_branch}
									</div>
								</td>
								<td class="px-4 py-3">
									{#if true}
										{@const ReviewStatusIcon = getStatusIcon(review.status)}
										<div class="flex items-center gap-2">
											<ReviewStatusIcon class={cn('h-4 w-4', getStatusColor(review.status))} />
											<span class="text-sm capitalize">{review.status}</span>
										</div>
									{/if}
									{#if review.error}
										<p class="mt-1 text-xs text-red-500 truncate max-w-[200px]" title={review.error}>
											{review.error}
										</p>
									{/if}
								</td>
								<td class="px-4 py-3">
									<span class="text-sm">{review.files_analyzed}</span>
								</td>
								<td class="px-4 py-3">
									<span class={cn('text-sm', review.issues_found > 0 ? 'text-yellow-500' : 'text-green-500')}>
										{review.issues_found}
									</span>
								</td>
								<td class="px-4 py-3">
									<span class="text-sm text-muted-foreground">
										{review.processing_time_ms ? `${(review.processing_time_ms / 1000).toFixed(1)}s` : '-'}
									</span>
								</td>
								<td class="px-4 py-3">
									<div class="flex items-center gap-2">
										<button
											onclick={() => viewReviewDetails(review)}
											class="rounded p-1.5 hover:bg-muted"
											title="View Details"
										>
											<Eye class="h-4 w-4" />
										</button>
										<a
											href={review.mr_url}
											target="_blank"
											class="rounded p-1.5 hover:bg-muted"
											title="Open in GitLab"
										>
											<ExternalLink class="h-4 w-4" />
										</a>
										{#if review.status === 'failed'}
											<button
												onclick={() => handleRetryReview(review)}
												class="rounded p-1.5 text-primary hover:bg-primary/10"
												title="Retry"
											>
												<RefreshCw class="h-4 w-4" />
											</button>
										{/if}
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		{/if}

		<!-- Pagination -->
		{#if totalPages > 1}
			<div class="flex items-center justify-between border-t px-4 py-3">
				<p class="text-sm text-muted-foreground">
					{m.pagination_page({ current: currentPage + 1, total: totalPages })}
				</p>
				<div class="flex items-center gap-2">
					<Button
						variant="outline"
						size="sm"
						disabled={currentPage === 0}
						onclick={() => { currentPage--; activeTab === 'projects' ? loadProjects() : loadReviews(); }}
					>
						{m.pagination_previous()}
					</Button>
					<Button
						variant="outline"
						size="sm"
						disabled={currentPage >= totalPages - 1}
						onclick={() => { currentPage++; activeTab === 'projects' ? loadProjects() : loadReviews(); }}
					>
						{m.pagination_next()}
					</Button>
				</div>
			</div>
		{/if}
	</div>
</div>

<!-- Add Project Modal -->
{#if showAddProjectModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showAddProjectModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showAddProjectModal = false)}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div class="w-full max-w-lg rounded-lg bg-card p-6 shadow-xl max-h-[90vh] overflow-y-auto">
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-semibold">{m.modal_add_project()}</h3>
				<button onclick={() => (showAddProjectModal = false)} class="rounded p-1 hover:bg-muted" title={m.common_close()}>
					<X class="h-5 w-5" />
				</button>
			</div>

			{#if addError}
				<div class="mb-4 rounded-md bg-red-500/10 p-3 text-sm text-red-500">
					{addError}
				</div>
			{/if}

			<div class="space-y-4">
				<div>
					<label for="add-project-id" class="mb-1.5 block text-sm font-medium">{m.modal_gitlab_project_id()} *</label>
					<input
						id="add-project-id"
						type="number"
						bind:value={formGitLabProjectId}
						placeholder="12345"
						class="h-10 w-full rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
					/>
					<p class="mt-1 text-xs text-muted-foreground">
						{m.modal_gitlab_project_id_hint()}
					</p>
				</div>

				<div>
					<label for="add-analysis-model" class="mb-1.5 block text-sm font-medium">{m.modal_analysis_model()} *</label>
					<input
						id="add-analysis-model"
						type="text"
						bind:value={formAnalysisModelId}
						placeholder="model-uuid-here"
						class="h-10 w-full rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
					/>
					<p class="mt-1 text-xs text-muted-foreground">
						{m.modal_analysis_model_hint()}
					</p>
				</div>

				<div>
					<label for="add-embedding-model" class="mb-1.5 block text-sm font-medium">{m.modal_embedding_model()} ({m.modal_optional()})</label>
					<input
						id="add-embedding-model"
						type="text"
						bind:value={formEmbeddingModelId}
						placeholder="model-uuid-here"
						class="h-10 w-full rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
					/>
					<p class="mt-1 text-xs text-muted-foreground">
						{m.modal_embedding_model_hint()}
					</p>
				</div>

				<div class="flex items-center gap-3">
					<input
						type="checkbox"
						id="autoReview"
						bind:checked={formAutoReview}
						class="h-4 w-4 rounded border"
					/>
					<label for="autoReview" class="text-sm font-medium">
						{m.modal_auto_review()}
					</label>
				</div>
				<p class="text-xs text-muted-foreground -mt-2">
					{m.modal_auto_review_hint()}
				</p>

				<div>
					<label for="add-review-prompt" class="mb-1.5 block text-sm font-medium">Custom Review Prompt (optional)</label>
					<textarea
						id="add-review-prompt"
						bind:value={formReviewPrompt}
						placeholder={m.placeholder_review_prompt()}
						rows="4"
						class="w-full rounded-md border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
					></textarea>
				</div>

				<!-- Include/Exclude Patterns (GITLAB-055c) -->
				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="add-include-patterns" class="mb-1.5 block text-sm font-medium">Include Patterns</label>
						<textarea
							id="add-include-patterns"
							bind:value={formIncludePatterns}
							placeholder="*.go&#10;*.ts&#10;src/**/*.js"
							rows="3"
							class="w-full rounded-md border bg-background px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary"
						></textarea>
						<p class="mt-1 text-xs text-muted-foreground">
							Glob patterns (one per line). If empty, all files included.
						</p>
					</div>
					<div>
						<label for="add-exclude-patterns" class="mb-1.5 block text-sm font-medium">Exclude Patterns</label>
						<textarea
							id="add-exclude-patterns"
							bind:value={formExcludePatterns}
							placeholder="*.min.js&#10;vendor/**&#10;*.lock"
							rows="3"
							class="w-full rounded-md border bg-background px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary"
						></textarea>
						<p class="mt-1 text-xs text-muted-foreground">
							Glob patterns to exclude (one per line)
						</p>
					</div>
				</div>

				<!-- Advanced Settings Toggle (GITLAB-055e) -->
				<button
					type="button"
					onclick={() => showAdvancedSettings = !showAdvancedSettings}
					class="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
				>
					<Settings class="h-4 w-4" />
					{showAdvancedSettings ? m.common_hide() : m.common_show()} {m.settings_title()}
				</button>

				{#if showAdvancedSettings}
					<div class="rounded-md border bg-muted/50 p-4 space-y-4">
						<div class="grid grid-cols-2 gap-4">
							<div>
								<label for="add-chunk-size" class="mb-1.5 block text-sm font-medium">Chunk Size (tokens)</label>
								<input
									id="add-chunk-size"
									type="number"
									bind:value={formChunkSize}
									min="500"
									max="32000"
									class="h-10 w-full rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
								/>
							</div>
							<div>
								<label for="add-chunk-overlap" class="mb-1.5 block text-sm font-medium">Chunk Overlap (tokens)</label>
								<input
									id="add-chunk-overlap"
									type="number"
									bind:value={formChunkOverlap}
									min="0"
									max="1000"
									class="h-10 w-full rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
								/>
							</div>
						</div>
						<div class="grid grid-cols-2 gap-4">
							<div>
								<label for="add-max-files" class="mb-1.5 block text-sm font-medium">Max Files per MR</label>
								<input
									id="add-max-files"
									type="number"
									bind:value={formMaxFilesPerMR}
									min="1"
									max="500"
									class="h-10 w-full rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
								/>
							</div>
							<div>
								<label for="add-max-lines" class="mb-1.5 block text-sm font-medium">Max Lines per File</label>
								<input
									id="add-max-lines"
									type="number"
									bind:value={formMaxLinesPerFile}
									min="100"
									max="10000"
									class="h-10 w-full rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
								/>
							</div>
							<div>
								<label for="add-max-review-tokens" class="mb-1.5 block text-sm font-medium">Max Review Tokens</label>
								<input
									id="add-max-review-tokens"
									type="number"
									bind:value={formMaxReviewTokens}
									min="1024"
									max="128000"
									step="1024"
									class="h-10 w-full rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
								/>
								<p class="mt-1 text-xs text-muted-foreground">Max tokens for LLM review response</p>
							</div>
						</div>
						<div class="flex items-center gap-6">
							<label class="flex items-center gap-2">
								<input
									type="checkbox"
									bind:checked={formSkipDraftMRs}
									class="h-4 w-4 rounded border"
								/>
								<span class="text-sm">Skip Draft MRs</span>
							</label>
							<label class="flex items-center gap-2">
								<input
									type="checkbox"
									bind:checked={formSkipBots}
									class="h-4 w-4 rounded border"
								/>
								<span class="text-sm">Skip Bot Authors</span>
							</label>
						</div>
					</div>
				{/if}
			</div>

			<div class="mt-6 flex justify-end gap-3">
				<Button variant="outline" onclick={() => (showAddProjectModal = false)}>
					Cancel
				</Button>
				<Button onclick={handleAddProject} disabled={isAdding}>
					{#if isAdding}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					Add Project
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Project Detail Modal -->
{#if showProjectDetailModal && selectedProject}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showProjectDetailModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showProjectDetailModal = false)}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div class="w-full max-w-2xl rounded-lg bg-card p-6 shadow-xl max-h-[90vh] overflow-y-auto">
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-semibold">{selectedProject.name}</h3>
				<button onclick={() => (showProjectDetailModal = false)} class="rounded p-1 hover:bg-muted">
					<X class="h-5 w-5" />
				</button>
			</div>

			<div class="space-y-4">
				<div class="grid grid-cols-2 gap-4">
					<div>
						<span class="text-sm text-muted-foreground">{m.modal_project_path()}:</span>
						<p class="font-mono text-sm">{selectedProject.path_with_namespace}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.modal_project_gitlab_id()}:</span>
						<p class="font-mono text-sm">{selectedProject.gitlab_project_id}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.common_status()}:</span>
						<p class={cn('text-sm capitalize', getStatusColor(selectedProject.status))}>
							{selectedProject.status}
						</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.modal_auto_review()}:</span>
						<p class="text-sm">{selectedProject.auto_review ? m.modal_enabled() : m.modal_disabled()}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.modal_analysis_model()}:</span>
						<p class="font-mono text-sm">{selectedProject.analysis_model_id || '-'}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.modal_embedding_model()}:</span>
						<p class="font-mono text-sm">{selectedProject.embedding_model_id || '-'}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.modal_project_webhook_id()}:</span>
						<p class="font-mono text-sm">{selectedProject.webhook_id || m.modal_project_webhook_not_configured()}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.modal_project_reviews_count()}:</span>
						<p class="text-sm">{selectedProject.review_count || 0}</p>
					</div>
				</div>

				{#if selectedProject.review_prompt}
					<div>
						<span class="text-sm text-muted-foreground">{m.modal_project_custom_prompt()}:</span>
						<pre class="mt-1 p-3 bg-muted rounded-md text-sm whitespace-pre-wrap">{selectedProject.review_prompt}</pre>
					</div>
				{/if}

				{#if selectedProject.settings}
					<div>
						<span class="text-sm text-muted-foreground">{m.modal_project_settings()}:</span>
						<pre class="mt-1 p-3 bg-muted rounded-md text-sm overflow-auto">{JSON.stringify(selectedProject.settings, null, 2)}</pre>
					</div>
				{/if}
			</div>

			<div class="mt-6 flex justify-end gap-2">
				<Button variant="outline" onclick={() => (showProjectDetailModal = false)}>
					{m.common_close()}
				</Button>
				<Button onclick={() => { showProjectDetailModal = false; openEditProject(selectedProject); }}>
					<Settings class="mr-2 h-4 w-4" />
					{m.common_edit()}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Edit Project Modal -->
{#if showEditProjectModal && selectedProject}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showEditProjectModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showEditProjectModal = false)}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div class="w-full max-w-2xl rounded-lg bg-card p-6 shadow-xl max-h-[90vh] overflow-y-auto">
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-semibold">{m.modal_edit_project()}: {selectedProject.name}</h3>
				<button onclick={() => (showEditProjectModal = false)} class="rounded p-1 hover:bg-muted" title={m.common_close()}>
					<X class="h-5 w-5" />
				</button>
			</div>

			{#if saveError}
				<div class="mb-4 rounded-md bg-destructive/15 p-3 text-sm text-destructive">
					{saveError}
				</div>
			{/if}

			<div class="space-y-4">
				<!-- Status -->
				<div>
					<label for="edit-status" class="text-sm font-medium">{m.common_status()}</label>
					<select
						id="edit-status"
						bind:value={editStatus}
						class="mt-1 w-full rounded-md border bg-background px-3 py-2"
					>
						<option value="active">{m.common_active()}</option>
						<option value="disabled">{m.common_disabled()}</option>
					</select>
				</div>

				<!-- Auto Review -->
				<div class="flex items-center gap-2">
					<input
						type="checkbox"
						id="editAutoReview"
						bind:checked={editAutoReview}
						class="rounded border-gray-300"
					/>
					<label for="editAutoReview" class="text-sm font-medium">{m.modal_auto_review()}</label>
				</div>

				<!-- Models -->
				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="edit-analysis-model" class="text-sm font-medium">{m.modal_analysis_model()}</label>
						<input
							id="edit-analysis-model"
							type="text"
							bind:value={editAnalysisModelId}
							placeholder="e.g., qwen-coder"
							class="mt-1 w-full rounded-md border bg-background px-3 py-2"
						/>
					</div>
					<div>
						<label for="edit-embedding-model" class="text-sm font-medium">{m.modal_embedding_model()}</label>
						<input
							id="edit-embedding-model"
							type="text"
							bind:value={editEmbeddingModelId}
							placeholder="e.g., bge-m3"
							class="mt-1 w-full rounded-md border bg-background px-3 py-2"
						/>
					</div>
				</div>

				<!-- Review Prompt -->
				<div>
					<label for="edit-review-prompt" class="text-sm font-medium">{m.admin_project_review_prompt()}</label>
					<textarea
						id="edit-review-prompt"
						bind:value={editReviewPrompt}
						placeholder={m.placeholder_review_prompt()}
						rows="3"
						class="mt-1 w-full rounded-md border bg-background px-3 py-2"
					></textarea>
				</div>

				<!-- Qdrant Collection -->
				<div>
					<label for="edit-collection" class="text-sm font-medium">{m.admin_project_collection()}</label>
					<input
						id="edit-collection"
						type="text"
						bind:value={editCollectionName}
						placeholder={m.placeholder_webhook_secret()}
						class="mt-1 w-full rounded-md border bg-background px-3 py-2"
					/>
				</div>

				<!-- File Patterns -->
				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="edit-include-patterns" class="text-sm font-medium">{m.admin_project_include_patterns()}</label>
						<input
							id="edit-include-patterns"
							type="text"
							bind:value={editIncludePatterns}
							placeholder="*.go, *.ts, *.py"
							class="mt-1 w-full rounded-md border bg-background px-3 py-2"
						/>
					</div>
					<div>
						<label for="edit-exclude-patterns" class="text-sm font-medium">{m.admin_project_exclude_patterns()}</label>
						<input
							id="edit-exclude-patterns"
							type="text"
							bind:value={editExcludePatterns}
							placeholder="vendor/*, node_modules/*"
							class="mt-1 w-full rounded-md border bg-background px-3 py-2"
						/>
					</div>
				</div>

				<!-- Advanced Settings -->
				<div class="border rounded-md p-4">
					<h4 class="font-medium mb-3">{m.settings_title()}</h4>
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label for="edit-chunk-size" class="text-sm font-medium">{m.admin_project_chunk_size()}</label>
							<input
								id="edit-chunk-size"
								type="number"
								bind:value={editChunkSize}
								class="mt-1 w-full rounded-md border bg-background px-3 py-2"
							/>
						</div>
						<div>
							<label for="edit-chunk-overlap" class="text-sm font-medium">{m.admin_project_chunk_overlap()}</label>
							<input
								id="edit-chunk-overlap"
								type="number"
								bind:value={editChunkOverlap}
								class="mt-1 w-full rounded-md border bg-background px-3 py-2"
							/>
						</div>
						<div>
							<label for="edit-max-files" class="text-sm font-medium">{m.admin_project_max_files()}</label>
							<input
								id="edit-max-files"
								type="number"
								bind:value={editMaxFilesPerMR}
								class="mt-1 w-full rounded-md border bg-background px-3 py-2"
							/>
						</div>
						<div>
							<label for="edit-max-lines" class="text-sm font-medium">{m.admin_project_max_lines()}</label>
							<input
								id="edit-max-lines"
								type="number"
								bind:value={editMaxLinesPerFile}
								class="mt-1 w-full rounded-md border bg-background px-3 py-2"
							/>
						</div>
						<div>
							<label for="edit-max-review-tokens" class="text-sm font-medium">{m.admin_project_max_review_tokens()}</label>
							<input
								id="edit-max-review-tokens"
								type="number"
								bind:value={editMaxReviewTokens}
								min="1024"
								max="128000"
								step="1024"
								class="mt-1 w-full rounded-md border bg-background px-3 py-2"
							/>
							<p class="mt-1 text-xs text-muted-foreground">{m.admin_project_max_review_tokens_hint()}</p>
						</div>
					</div>
					<div class="mt-3 flex flex-wrap gap-4">
						<div class="flex items-center gap-2">
							<input
								type="checkbox"
								id="editSkipDraftMRs"
								bind:checked={editSkipDraftMRs}
								class="rounded border-gray-300"
							/>
							<label for="editSkipDraftMRs" class="text-sm">{m.admin_project_skip_drafts()}</label>
						</div>
						<div class="flex items-center gap-2">
							<input
								type="checkbox"
								id="editSkipBots"
								bind:checked={editSkipBots}
								class="rounded border-gray-300"
							/>
							<label for="editSkipBots" class="text-sm">{m.admin_project_skip_bots()}</label>
						</div>
						<div class="flex items-center gap-2">
							<input
								type="checkbox"
								id="editPerFileReview"
								bind:checked={editPerFileReview}
								class="rounded border-gray-300"
							/>
							<label for="editPerFileReview" class="text-sm" title="Review each file separately with tool calling for better context">
								Per-file review (with tools)
							</label>
						</div>
						<div class="flex items-center gap-2">
							<label for="editReviewLanguage" class="text-sm">Review Language:</label>
							<select
								id="editReviewLanguage"
								bind:value={editReviewLanguage}
								class="rounded border px-2 py-1 bg-background text-sm"
							>
								<option value="en">English</option>
								<option value="ru">Русский</option>
							</select>
						</div>
					</div>
				</div>
			</div>

			<div class="mt-6 flex justify-end gap-2">
				<Button variant="outline" onclick={() => (showEditProjectModal = false)} disabled={isSaving}>
					{m.common_cancel()}
				</Button>
				<Button onclick={saveProjectChanges} disabled={isSaving}>
					{#if isSaving}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					{m.common_save()}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Review Detail Modal -->
{#if showReviewDetailModal && selectedReview}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		onclick={(e) => e.target === e.currentTarget && (showReviewDetailModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showReviewDetailModal = false)}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div class="w-full max-w-3xl rounded-lg bg-card p-6 shadow-xl max-h-[90vh] overflow-y-auto">
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-semibold">!{selectedReview.mr_iid}: {selectedReview.mr_title}</h3>
				<button onclick={() => (showReviewDetailModal = false)} class="rounded p-1 hover:bg-muted">
					<X class="h-5 w-5" />
				</button>
			</div>

			<div class="space-y-4">
				<div class="grid grid-cols-2 gap-4 sm:grid-cols-3">
					<div>
						<span class="text-sm text-muted-foreground">{m.common_status()}:</span>
						<p class={cn('text-sm capitalize', getStatusColor(selectedReview.status))}>
							{selectedReview.status}
						</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.review_author()}:</span>
						<p class="text-sm">{selectedReview.mr_author}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.review_branch()}:</span>
						<p class="text-sm font-mono">{selectedReview.source_branch} → {selectedReview.target_branch}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.review_files_analyzed()}:</span>
						<p class="text-sm">{selectedReview.files_analyzed}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.review_lines_changed()}:</span>
						<p class="text-sm">{selectedReview.lines_changed}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.review_issues_found()}:</span>
						<p class={cn('text-sm', selectedReview.issues_found > 0 ? 'text-yellow-500 font-medium' : 'text-green-500')}>
							{selectedReview.issues_found}
						</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.review_processing_time()}:</span>
						<p class="text-sm">{selectedReview.processing_time_ms ? `${(selectedReview.processing_time_ms / 1000).toFixed(2)}s` : '-'}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.review_tokens_used()}:</span>
						<p class="text-sm">{selectedReview.tokens_used || '-'}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.review_model()}:</span>
						<p class="text-sm font-mono">{selectedReview.model_used || '-'}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.review_retry_count()}:</span>
						<p class="text-sm">{selectedReview.retry_count}</p>
					</div>
					<div>
						<span class="text-sm text-muted-foreground">{m.review_created()}:</span>
						<p class="text-sm">{formatRelativeTime(selectedReview.created_at)}</p>
					</div>
					{#if selectedReview.completed_at}
						<div>
							<span class="text-sm text-muted-foreground">{m.review_completed()}:</span>
							<p class="text-sm">{formatRelativeTime(selectedReview.completed_at)}</p>
						</div>
					{/if}
				</div>

				{#if selectedReview.error}
					<div class="rounded-md bg-red-500/10 p-3">
						<span class="text-sm font-medium text-red-500">Error:</span>
						<p class="mt-1 text-sm text-red-500">{selectedReview.error}</p>
					</div>
				{/if}

				{#if selectedReview.review_result}
					<div>
						<span class="text-sm font-medium">{m.review_summary()}:</span>
						<div class="mt-2 rounded-md bg-muted p-4">
							<p class="text-sm">{selectedReview.review_result.summary}</p>
							{#if selectedReview.review_result.overall_score !== undefined}
								<div class="mt-2 flex items-center gap-2">
									<span class="text-sm text-muted-foreground">{m.review_score()}:</span>
									<span class={cn(
										'text-sm font-medium',
										selectedReview.review_result.overall_score >= 80 ? 'text-green-500' :
										selectedReview.review_result.overall_score >= 60 ? 'text-yellow-500' : 'text-red-500'
									)}>
										{selectedReview.review_result.overall_score}/100
									</span>
								</div>
							{/if}
						</div>
					</div>

					{#if selectedReview.review_result.categories?.length}
						<div>
							<span class="text-sm font-medium">{m.review_categories()}:</span>
							<div class="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-3">
								{#each selectedReview.review_result.categories as category}
									<div class="rounded-md border p-2">
										<p class="text-sm font-medium">{category.name}</p>
										<p class="text-xs text-muted-foreground">
											{m.review_score()}: {category.score}/100 · {category.issue_count} {m.review_issues()}
										</p>
									</div>
								{/each}
							</div>
						</div>
					{/if}
				{/if}
			</div>

			<div class="mt-6 flex justify-end gap-3">
				<Button variant="outline" onclick={() => (showReviewDetailModal = false)}>
					{m.common_close()}
				</Button>
				<a href={selectedReview.mr_url} target="_blank">
					<Button>
						<ExternalLink class="mr-2 h-4 w-4" />
						{m.table_open_gitlab()}
					</Button>
				</a>
			</div>
		</div>
	</div>
{/if}

<!-- Secrets Scan Modal -->
{#if showSecretsModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		role="dialog"
		aria-modal="true"
		onclick={(e) => e.target === e.currentTarget && (showSecretsModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showSecretsModal = false)}
		tabindex="-1"
	>
		<div class="max-h-[90vh] w-full max-w-5xl overflow-auto rounded-lg bg-background p-6 shadow-xl">
			<div class="mb-4 flex items-center justify-between">
				<h3 class="text-xl font-semibold flex items-center gap-2">
					<Shield class="h-5 w-5 text-primary" />
					{m.gitlab_secrets_scan_title?.() || 'Secrets Scan Results'}
				</h3>
				<button onclick={() => (showSecretsModal = false)} class="rounded p-1 hover:bg-muted">
					<X class="h-5 w-5" />
				</button>
			</div>

			{#if isScanning}
				<div class="flex flex-col items-center justify-center py-12">
					<Loader2 class="h-12 w-12 animate-spin text-primary" />
					<p class="mt-4 text-muted-foreground">{m.gitlab_scanning_secrets?.() || 'Scanning for secrets...'}</p>
				</div>
			{:else if secretsScanResult}
				{#if secretsScanResult.status === 'failed'}
					<div class="rounded-lg bg-red-100 dark:bg-red-900/30 p-4 text-red-700 dark:text-red-400">
						<p class="font-medium">{m.gitlab_scan_failed?.() || 'Scan Failed'}</p>
						<p class="text-sm">{secretsScanResult.error}</p>
					</div>
				{:else}
					<!-- Summary -->
					<div class="mb-6 grid grid-cols-2 md:grid-cols-4 gap-4">
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold">{secretsScanResult.summary.total_findings}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_secrets_total?.() || 'Total Findings'}</div>
						</div>
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold">{secretsScanResult.summary.files_affected}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_files_affected?.() || 'Files Affected'}</div>
						</div>
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold">{secretsScanResult.chunks_scanned}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_chunks_scanned?.() || 'Chunks Scanned'}</div>
						</div>
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold">{secretsScanResult.duration}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_scan_duration?.() || 'Duration'}</div>
						</div>
					</div>

					<!-- Severity breakdown -->
					{#if Object.keys(secretsScanResult.summary.by_severity).length > 0}
						<div class="mb-4 flex flex-wrap gap-2">
							{#each Object.entries(secretsScanResult.summary.by_severity) as [severity, count]}
								<span class={cn('px-3 py-1 rounded-full text-sm font-medium', getSeverityColor(severity))}>
									{severity}: {count}
								</span>
							{/each}
						</div>
					{/if}

					<!-- Findings list -->
					{#if secretsScanResult.findings.length > 0}
						<div class="space-y-3 max-h-[50vh] overflow-auto">
							{#each secretsScanResult.findings as finding}
								<div class="rounded-lg border p-4 hover:bg-muted/50">
									<div class="flex items-start justify-between gap-4">
										<div class="flex-1">
											<div class="flex items-center gap-2 mb-1">
												{#if true}
													{@const SevIcon = getSeverityIcon(finding.severity)}
													<span class={cn('px-2 py-0.5 rounded text-xs font-medium uppercase', getSeverityColor(finding.severity))}>
														{finding.severity}
													</span>
												{/if}
												<span class="font-medium">{finding.pattern_name}</span>
												<span class="text-xs text-muted-foreground px-2 py-0.5 bg-muted rounded">{finding.category}</span>
											</div>
											<div class="text-sm text-muted-foreground mb-2">
												<code class="px-1 bg-muted rounded">{finding.file_path}</code>
												{#if finding.start_line > 0}
													<span class="ml-1">:{finding.start_line}</span>
												{/if}
											</div>
											<div class="text-sm font-mono bg-muted p-2 rounded overflow-x-auto">
												{finding.match}
											</div>
											{#if finding.context}
												<details class="mt-2">
													<summary class="text-xs text-muted-foreground cursor-pointer hover:text-foreground">
														{m.gitlab_show_context?.() || 'Show context'}
													</summary>
													<pre class="mt-1 text-xs bg-muted p-2 rounded overflow-x-auto whitespace-pre-wrap">{finding.context}</pre>
												</details>
											{/if}
										</div>
									</div>
									{#if finding.suggestion}
										<div class="mt-2 text-sm text-muted-foreground border-t pt-2">
											<strong>{m.gitlab_suggestion?.() || 'Suggestion'}:</strong> {finding.suggestion}
										</div>
									{/if}
								</div>
							{/each}
						</div>
					{:else}
						<div class="flex flex-col items-center justify-center py-12 text-center">
							<CheckCircle class="h-16 w-16 text-green-500" />
							<p class="mt-4 text-lg font-medium text-green-600 dark:text-green-400">
								{m.gitlab_no_secrets_found?.() || 'No secrets found!'}
							</p>
							<p class="text-sm text-muted-foreground">
								{m.gitlab_code_looks_safe?.() || 'Your code looks safe from hardcoded secrets.'}
							</p>
						</div>
					{/if}
				{/if}
			{/if}

			<div class="mt-6 flex justify-end">
				<Button variant="outline" onclick={() => (showSecretsModal = false)}>
					{m.common_close()}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Deep Scan (LLM) Modal -->
{#if showDeepScanModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		role="dialog"
		aria-modal="true"
		onclick={(e) => e.target === e.currentTarget && (showDeepScanModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showDeepScanModal = false)}
		tabindex="-1"
	>
		<div class="max-h-[90vh] w-full max-w-4xl overflow-auto rounded-lg bg-background p-6 shadow-lg">
			<div class="mb-4 flex items-center justify-between">
				<div class="flex items-center gap-3">
					<Brain class="h-6 w-6 text-purple-500" />
					<h3 class="text-lg font-semibold">
						{m.gitlab_deep_scan_title?.() || 'Deep Scan Results (LLM Analysis)'}
					</h3>
				</div>
				<button onclick={() => (showDeepScanModal = false)} class="rounded p-1 hover:bg-muted">
					<X class="h-5 w-5" />
				</button>
			</div>

			{#if isDeepScanning}
				<div class="flex flex-col items-center justify-center py-12">
					<Loader2 class="h-12 w-12 animate-spin text-purple-500" />
					<p class="mt-4 text-muted-foreground">{m.gitlab_deep_scanning?.() || 'Analyzing code with LLM...'}</p>
					<p class="mt-2 text-sm text-muted-foreground">{m.gitlab_deep_scan_patience?.() || 'This may take several minutes for large repositories'}</p>
				</div>
			{:else if deepScanResult}
				{#if deepScanResult.status === 'failed'}
					<div class="rounded-lg bg-red-100 dark:bg-red-900/30 p-4 text-red-700 dark:text-red-400">
						<p class="font-medium">{m.gitlab_scan_failed?.() || 'Scan Failed'}</p>
						<p class="text-sm">{deepScanResult.error}</p>
					</div>
				{:else}
					<!-- Summary stats -->
					<div class="mb-6 grid grid-cols-2 md:grid-cols-5 gap-4">
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold">{deepScanResult.summary.total_findings}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_secrets_total?.() || 'Total Findings'}</div>
						</div>
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold">{deepScanResult.summary.files_affected}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_files_affected?.() || 'Files Affected'}</div>
						</div>
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold">{deepScanResult.chunks_scanned}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_chunks_scanned?.() || 'Chunks Scanned'}</div>
						</div>
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold">{deepScanResult.tokens_used.toLocaleString()}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_tokens_used?.() || 'Tokens Used'}</div>
						</div>
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold">{deepScanResult.duration}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_scan_duration?.() || 'Duration'}</div>
						</div>
					</div>
					
					<!-- Model info -->
					<div class="mb-4 text-sm text-muted-foreground flex items-center gap-2">
						<Bot class="h-4 w-4" />
						<span>{m.gitlab_analyzed_by?.() || 'Analyzed by'}: <strong>{deepScanResult.model_id}</strong></span>
					</div>

					<!-- Severity breakdown -->
					{#if Object.keys(deepScanResult.summary.by_severity).length > 0}
						<div class="mb-4 flex flex-wrap gap-2">
							{#each Object.entries(deepScanResult.summary.by_severity) as [severity, count]}
								<span class={cn('px-3 py-1 rounded-full text-sm font-medium', getSeverityColor(severity))}>
									{severity}: {count}
								</span>
							{/each}
						</div>
					{/if}

					<!-- Findings list -->
					{#if deepScanResult.findings.length > 0}
						<div class="space-y-3 max-h-[50vh] overflow-auto">
							{#each deepScanResult.findings as finding}
								<div class="rounded-lg border p-4 hover:bg-muted/50">
									<div class="flex items-start justify-between gap-4">
										<div class="flex-1 min-w-0">
											<div class="flex items-center gap-2 flex-wrap">
												<span class={cn('px-2 py-0.5 rounded text-xs font-medium', getSeverityColor(finding.severity))}>
													{finding.severity}
												</span>
												<span class="text-xs text-muted-foreground">{finding.type}</span>
												<span class={cn('px-2 py-0.5 rounded text-xs', getConfidenceColor(finding.confidence))}>
													{m.gitlab_confidence?.() || 'Confidence'}: {finding.confidence}
												</span>
											</div>
											<div class="mt-2 font-medium text-sm">
												{finding.description}
											</div>
											<div class="mt-1 text-xs text-muted-foreground flex items-center gap-2">
												<FileCode class="h-3 w-3" />
												{finding.file_path}:{finding.start_line}-{finding.end_line}
											</div>
											{#if finding.code_snippet}
												<pre class="mt-2 text-xs bg-muted p-2 rounded overflow-x-auto whitespace-pre-wrap font-mono">{finding.code_snippet}</pre>
											{/if}
										</div>
									</div>
									{#if finding.suggestion}
										<div class="mt-2 text-sm text-muted-foreground border-t pt-2">
											<strong>{m.gitlab_suggestion?.() || 'Suggestion'}:</strong> {finding.suggestion}
										</div>
									{/if}
								</div>
							{/each}
						</div>
					{:else}
						<div class="flex flex-col items-center justify-center py-12 text-center">
							<CheckCircle class="h-16 w-16 text-green-500" />
							<p class="mt-4 text-lg font-medium text-green-600 dark:text-green-400">
								{m.gitlab_no_secrets_found?.() || 'No secrets found!'}
							</p>
							<p class="text-sm text-muted-foreground">
								{m.gitlab_llm_analysis_clean?.() || 'LLM analysis did not detect any security issues.'}
							</p>
						</div>
					{/if}
				{/if}
			{/if}

			<div class="mt-6 flex justify-end">
				<Button variant="outline" onclick={() => (showDeepScanModal = false)}>
					{m.common_close()}
				</Button>
			</div>
		</div>
	</div>
{/if}

