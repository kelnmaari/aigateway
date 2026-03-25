<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import { faCodeBranch, faPlus, faArrowsRotate, faSpinner, faTrash, faGear, faXmark, faMagnifyingGlass, faArrowUpRightFromSquare, faCircleCheck, faCircleXmark, faCircleExclamation, faPlay, faEye, faArrowLeft, faTowerBroadcast, faRobot, faFileCode, faClock, faChartColumn, faShieldHalved, faBrain, faBox, faFilePen, faFlask } from '@fortawesome/free-solid-svg-icons';
	import { gitlabApi, type GitLabIntegration, type GitLabProject, type GitLabReview, type SecretsScanResult, type SecretFinding, type DeepScanResult, type DeepFinding, type DependencyScanResult, type MultiEcosystemDependencyScanResult, type DependencyWithVulns, type QualityScore, type QualityCategory, type DeadCodeResult, type DocScanResult, type DocGenerationResult, type TestScanResult, type TestGenerationResult, type ChangelogAnalysis, type BreakingChange } from '$lib/api/gitlab';
	import { cn, formatRelativeTime, debounce } from '$lib/utils';
	import { Button } from '$lib/components/ui/button';
	import { ActionButton } from '$lib/components/ui/action-button';
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
	let editDefaultBranch = $state('');
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
	let editReviewMode = $state('standard');
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
	
	// Dependency scan state
	let showDependenciesModal = $state(false);
	let dependenciesScanResult = $state<MultiEcosystemDependencyScanResult | null>(null);
	let isCheckingDependencies = $state(false);
	let checkingDependenciesProjectId = $state<string | null>(null);
	let isCreatingDependencyIssue = $state(false);
	let selectedEcosystemIndex = $state(0); // For multi-ecosystem tab selection
	
	// Create Issue state for all scanners
	let isCreatingSecretsIssue = $state(false);
	let isCreatingQualityIssue = $state(false);
	let isCreatingDeadCodeIssue = $state(false);
	
	// Changelog analysis state
	let showChangelogModal = $state(false);
	let changelogResult = $state<ChangelogAnalysis | null>(null);
	let isAnalyzingChangelog = $state(false);
	let analyzingPackage = $state<{ name: string; current: string; latest: string; language: string } | null>(null);
	
	// Quality analysis state
	let showQualityModal = $state(false);
	let qualityResult = $state<QualityScore | null>(null);
	let isAnalyzingQuality = $state(false);
	let analyzingQualityProjectId = $state<string | null>(null);
	
	// Dead code detection state
	let showDeadCodeModal = $state(false);
	let deadCodeResult = $state<DeadCodeResult | null>(null);
	let isDetectingDeadCode = $state(false);
	let detectingDeadCodeProjectId = $state<string | null>(null);
	
	// Auto-documentation state
	let showAutoDocModal = $state(false);
	let docScanResult = $state<DocScanResult | null>(null);
	let docGenResult = $state<DocGenerationResult | null>(null);
	let isGeneratingDocs = $state(false);
	let generatingDocsProjectId = $state<string | null>(null);
	let autoDocStep = $state<'scan' | 'generate' | 'results'>('scan');
	let isCreatingDocsMR = $state(false);
	
	// Test generation state
	let showTestGenModal = $state(false);
	let testScanResult = $state<TestScanResult | null>(null);
	let testGenResult = $state<TestGenerationResult | null>(null);
	let isGeneratingTests = $state(false);
	let generatingTestsProjectId = $state<string | null>(null);
	let testGenStep = $state<'scan' | 'generate' | 'results'>('scan');
	let isCreatingTestsMR = $state(false);

	// Webhooks pause toggle
	let isTogglingWebhooks = $state(false);

	async function toggleWebhooksPaused() {
		if (!integration || isTogglingWebhooks) return;
		isTogglingWebhooks = true;
		try {
			const newPaused = !integration.settings?.webhooks_paused;
			integration = await gitlabApi.updateIntegration(integrationId, {
				settings: { ...integration.settings, webhooks_paused: newPaused }
			});
		} catch (error) {
			console.error('Failed to toggle webhooks:', error);
		} finally {
			isTogglingWebhooks = false;
		}
	}

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

	const integrationId = $derived($page.params.id ?? '');

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
			
			// Start polling for any projects with in_progress status
			// Clear old polling for projects no longer in list
			const currentProjectIds = new Set(projects.map(p => p.id));
			for (const [projectId, interval] of pollingIntervals.entries()) {
				if (!currentProjectIds.has(projectId)) {
					clearInterval(interval);
					pollingIntervals.delete(projectId);
					indexingProjects = new Set([...indexingProjects].filter(id => id !== projectId));
				}
			}
			
			// Start/restart polling for in_progress projects
			for (const project of projects) {
				if (project.index_status === 'in_progress') {
					if (!indexingProjects.has(project.id)) {
						indexingProjects = new Set([...indexingProjects, project.id]);
					}
					pollIndexStatus(project.id);
				}
			}
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

	// Store active polling intervals
	let pollingIntervals = $state<Map<string, ReturnType<typeof setInterval>>>(new Map());

	async function pollIndexStatus(projectId: string) {
		// Clear existing interval for this project if any
		if (pollingIntervals.has(projectId)) {
			clearInterval(pollingIntervals.get(projectId));
			pollingIntervals.delete(projectId);
		}

		const maxAttempts = 60; // 5 minutes max
		let attempts = 0;

		const poll = async () => {
			try {
				const status = await gitlabApi.getIndexStatus(projectId);
				
				// Update project in list reactively
				const projectIndex = projects.findIndex(p => p.id === projectId);
				if (projectIndex !== -1) {
					projects[projectIndex] = { 
						...projects[projectIndex], 
						index_status: status.status as GitLabProject['index_status'],
						last_indexed_at: status.last_indexed,
						index_chunks: status.chunks_total
					};
					// Trigger reactivity
					projects = [...projects];
				}

				if (status.status !== 'in_progress' || attempts >= maxAttempts) {
					// Done or failed - stop polling
					if (pollingIntervals.has(projectId)) {
						clearInterval(pollingIntervals.get(projectId));
						pollingIntervals.delete(projectId);
					}
					indexingProjects = new Set([...indexingProjects].filter(id => id !== projectId));
				}
				attempts++;
			} catch {
				// Error - stop polling
				if (pollingIntervals.has(projectId)) {
					clearInterval(pollingIntervals.get(projectId));
					pollingIntervals.delete(projectId);
				}
				indexingProjects = new Set([...indexingProjects].filter(id => id !== projectId));
			}
		};

		// Initial poll
		await poll();
		
		// Set up interval for ongoing polling (every 3 seconds for better UX)
		if (indexingProjects.has(projectId)) {
			const interval = setInterval(poll, 3000);
			pollingIntervals.set(projectId, interval);
		}
	}

	// Cleanup polling intervals on unmount
	onDestroy(() => {
		for (const interval of pollingIntervals.values()) {
			clearInterval(interval);
		}
		pollingIntervals.clear();
	});

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

		selectedProject = project; // Required for Create Issue button
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

		selectedProject = project; // Required for Create Issue button
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
	
	async function handleCheckDependencies(project: GitLabProject) {
		if (project.index_status !== 'completed') {
			alert(m.alert_index_required_for_scan?.() || 'Please index the repository first.');
			return;
		}

		selectedProject = project; // Required for Create Issue button
		checkingDependenciesProjectId = project.id;
		isCheckingDependencies = true;
		dependenciesScanResult = null;
		selectedEcosystemIndex = 0; // Reset to first ecosystem
		showDependenciesModal = true;

		try {
			const result = await gitlabApi.checkDependencies(project.id);
			dependenciesScanResult = result;
		} catch (error) {
			console.error('Dependencies check failed:', error);
			dependenciesScanResult = {
				project_id: project.id,
				scan_id: '',
				scanned_at: new Date().toISOString(),
				duration: '0s',
				ecosystems: [],
				total_summary: { 
					total_dependencies: 0, 
					direct_dependencies: 0,
					outdated_count: 0,
					vulnerable_count: 0,
					up_to_date_count: 0,
					by_update_type: {}, 
					by_severity: {},
					critical_vulns: 0
				},
				status: 'failed',
				error: error instanceof Error ? error.message : 'Check failed'
			};
		} finally {
			isCheckingDependencies = false;
			checkingDependenciesProjectId = null;
		}
	}

	async function handleCreateDependencyIssue() {
		if (!dependenciesScanResult || !selectedProject) return;
		
		// Aggregate dependencies from all ecosystems
		const allDeps = dependenciesScanResult.ecosystems.flatMap(eco => 
			eco.dependencies.map(d => ({ ...d, ecosystem: eco.language }))
		);
		
		const vulnerableDeps = allDeps.filter(d => d.is_vulnerable);
		const outdatedDeps = allDeps.filter(d => d.dependency.has_update && !d.is_vulnerable);
		
		let description = `## Dependency Security & Update Report\n\n`;
		description += `**Project:** ${selectedProject.name}\n`;
		description += `**Scanned:** ${new Date(dependenciesScanResult.scanned_at).toLocaleString()}\n`;
		description += `**Ecosystems:** ${dependenciesScanResult.ecosystems.map(e => e.language).join(', ')}\n\n`;
		
		if (vulnerableDeps.length > 0) {
			description += `### 🚨 Vulnerable Dependencies (${vulnerableDeps.length})\n\n`;
			for (const dep of vulnerableDeps) {
				description += `- **${dep.dependency.name}** (${dep.ecosystem}) ${dep.dependency.current_version} → ${dep.dependency.latest_version}\n`;
				for (const vuln of dep.vulnerabilities) {
					description += `  - ${vuln.severity.toUpperCase()}: ${vuln.title} (${vuln.cve_id || 'N/A'})\n`;
				}
			}
			description += `\n`;
		}
		
		if (outdatedDeps.length > 0) {
			description += `### ⚠️ Outdated Dependencies (${outdatedDeps.length})\n\n`;
			for (const dep of outdatedDeps.slice(0, 20)) { // Limit to 20 to avoid huge issues
				description += `- **${dep.dependency.name}** (${dep.ecosystem}) ${dep.dependency.current_version} → ${dep.dependency.latest_version}\n`;
			}
			if (outdatedDeps.length > 20) {
				description += `\n_...and ${outdatedDeps.length - 20} more outdated dependencies_\n`;
			}
		}
		
		description += `\n---\n_Generated by AI Gateway dependency scanner_`;
		
		isCreatingDependencyIssue = true;
		try {
			const result = await gitlabApi.createDependencyIssue(dependenciesScanResult.project_id, {
				title: `[Security] ${vulnerableDeps.length} vulnerable, ${outdatedDeps.length} outdated dependencies`,
				description,
				labels: ['dependencies', 'security', 'automated'],
				critical: vulnerableDeps.length > 0
			});
			
			alert(`Issue created successfully!\n\n${result.url}`);
			window.open(result.url, '_blank');
		} catch (error) {
			console.error('Failed to create issue:', error);
			alert('Failed to create issue: ' + (error instanceof Error ? error.message : 'Unknown error'));
		} finally {
			isCreatingDependencyIssue = false;
		}
	}

	async function handleCreateSecretsIssue() {
		if (!secretsScanResult || !selectedProject) return;
		
		let description = `## Secrets Scan Report\n\n`;
		description += `**Project:** ${selectedProject.name}\n`;
		description += `**Scanned:** ${secretsScanResult.completed_at || new Date().toLocaleString()}\n\n`;
		
		description += `### Findings: ${secretsScanResult.summary.total_findings}\n\n`;
		
		if (secretsScanResult.findings && secretsScanResult.findings.length > 0) {
			for (const finding of secretsScanResult.findings.slice(0, 20)) {
				description += `- **${finding.pattern_name}** in \`${finding.file_path}\` (line ${finding.start_line})\n`;
				description += `  - Severity: ${finding.severity}\n`;
			}
			if (secretsScanResult.findings.length > 20) {
				description += `\n_...and ${secretsScanResult.findings.length - 20} more findings_\n`;
			}
		}
		
		description += `\n---\n_Generated by AI Gateway secrets scanner_`;
		
		isCreatingSecretsIssue = true;
		try {
			const result = await gitlabApi.createSecretsIssue(selectedProject.id, {
				title: `[Security] ${secretsScanResult.summary.total_findings} secrets found in codebase`,
				description,
				labels: ['security', 'secrets', 'urgent']
			});
			
			alert(`Issue created successfully!\n\n${result.url}`);
			window.open(result.url, '_blank');
		} catch (error) {
			console.error('Failed to create issue:', error);
			alert('Failed to create issue: ' + (error instanceof Error ? error.message : 'Unknown error'));
		} finally {
			isCreatingSecretsIssue = false;
		}
	}

	async function handleCreateQualityIssue() {
		if (!qualityResult || !selectedProject) return;
		
		let description = `## Code Quality Report\n\n`;
		description += `**Project:** ${selectedProject.name}\n`;
		description += `**Analyzed:** ${qualityResult.scanned_at || new Date().toLocaleString()}\n\n`;
		
		description += `### Quality Score: ${qualityResult.overall_score}/100\n\n`;
		
		if (qualityResult.recommendations && qualityResult.recommendations.length > 0) {
			description += `### Recommendations (${qualityResult.recommendations.length})\n\n`;
			for (const rec of qualityResult.recommendations.slice(0, 20)) {
				description += `- **${rec.priority}**: ${rec.title}\n`;
				description += `  - ${rec.description}\n`;
			}
			if (qualityResult.recommendations.length > 20) {
				description += `\n_...and ${qualityResult.recommendations.length - 20} more recommendations_\n`;
			}
		}
		
		description += `\n---\n_Generated by AI Gateway quality analyzer_`;
		
		isCreatingQualityIssue = true;
		try {
			const result = await gitlabApi.createQualityIssue(selectedProject.id, {
				title: `[Quality] Code quality score: ${qualityResult.overall_score}/100`,
				description,
				labels: ['code-quality', 'improvement']
			});
			
			alert(`Issue created successfully!\n\n${result.url}`);
			window.open(result.url, '_blank');
		} catch (error) {
			console.error('Failed to create issue:', error);
			alert('Failed to create issue: ' + (error instanceof Error ? error.message : 'Unknown error'));
		} finally {
			isCreatingQualityIssue = false;
		}
	}

	async function handleCreateDeadCodeIssue() {
		if (!deadCodeResult || !selectedProject) return;
		
		const totalItems = deadCodeResult.summary?.total_dead_symbols || deadCodeResult.dead_symbols?.length || 0;
		
		let description = `## Dead Code Report\n\n`;
		description += `**Project:** ${selectedProject.name}\n`;
		description += `**Detected:** ${deadCodeResult.scanned_at || new Date().toLocaleString()}\n\n`;
		
		description += `### Summary\n\n`;
		description += `- Total dead symbols: ${totalItems}\n`;
		description += `- Estimated dead lines: ${deadCodeResult.summary?.estimated_dead_lines || 0}\n\n`;
		
		if (deadCodeResult.dead_symbols && deadCodeResult.dead_symbols.length > 0) {
			description += `### Dead Symbols\n\n`;
			for (const symbol of deadCodeResult.dead_symbols.slice(0, 15)) {
				description += `- **${symbol.type}** \`${symbol.name}\` in \`${symbol.file_path}\` (line ${symbol.start_line})\n`;
				description += `  - Confidence: ${symbol.confidence}, Reason: ${symbol.reason}\n`;
			}
			if (deadCodeResult.dead_symbols.length > 15) {
				description += `\n_...and ${deadCodeResult.dead_symbols.length - 15} more_\n`;
			}
			description += `\n`;
		}
		
		description += `### Recommendations\n\n`;
		description += `1. Review each item before deletion\n`;
		description += `2. Ensure no dynamic references exist\n`;
		description += `3. Run tests after cleanup\n`;
		
		description += `\n---\n_Generated by AI Gateway dead code detector_`;
		
		isCreatingDeadCodeIssue = true;
		try {
			const result = await gitlabApi.createDeadCodeIssue(selectedProject.id, {
				title: `[Cleanup] ${totalItems} dead code items detected`,
				description,
				labels: ['dead-code', 'cleanup', 'ai-generated']
			});
			
			alert(`Issue created successfully!\n\n${result.url}`);
			window.open(result.url, '_blank');
		} catch (error) {
			console.error('Failed to create issue:', error);
			alert('Failed to create issue: ' + (error instanceof Error ? error.message : 'Unknown error'));
		} finally {
			isCreatingDeadCodeIssue = false;
		}
	}
	
	async function handleAnalyzeChangelog(dep: DependencyWithVulns, projectId: string, language: string) {
		if (!dep.dependency.has_update) return;
		
		analyzingPackage = {
			name: dep.dependency.name,
			current: dep.dependency.current_version,
			latest: dep.dependency.latest_version,
			language: language
		};
		isAnalyzingChangelog = true;
		changelogResult = null;
		showChangelogModal = true;

		try {
			const result = await gitlabApi.analyzeChangelog(projectId, {
				package_name: dep.dependency.name,
				current_version: dep.dependency.current_version,
				latest_version: dep.dependency.latest_version,
				language: language
			});
			changelogResult = result;
		} catch (error) {
			console.error('Changelog analysis failed:', error);
			changelogResult = {
				package_name: dep.dependency.name,
				current_version: dep.dependency.current_version,
				latest_version: dep.dependency.latest_version,
				language: language,
				summary: 'Analysis failed: ' + (error instanceof Error ? error.message : 'Unknown error'),
				breaking_changes: [],
				new_features: [],
				bug_fixes: [],
				security_fixes: [],
				deprecated_features: [],
				migration_guide: '',
				risk_level: 'medium',
				confidence: 'low',
				tokens_used: 0,
				analyzed_at: new Date().toISOString()
			};
		} finally {
			isAnalyzingChangelog = false;
		}
	}
	
	function getRiskLevelColor(level: string): string {
		switch (level) {
			case 'low': return 'text-green-600 bg-green-100 dark:bg-green-900/30';
			case 'medium': return 'text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30';
			case 'high': return 'text-orange-600 bg-orange-100 dark:bg-orange-900/30';
			case 'critical': return 'text-red-600 bg-red-100 dark:bg-red-900/30';
			default: return 'text-gray-600 bg-gray-100 dark:bg-gray-900/30';
		}
	}
	
	function getUpdateTypeColor(updateType: string): string {
		switch (updateType) {
			case 'major': return 'text-red-600 bg-red-100 dark:bg-red-900/30';
			case 'minor': return 'text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30';
			case 'patch': return 'text-green-600 bg-green-100 dark:bg-green-900/30';
			default: return 'text-gray-600 bg-gray-100 dark:bg-gray-900/30';
		}
	}
	
	async function handleAnalyzeQuality(project: GitLabProject) {
		if (project.index_status !== 'completed' || !project.analysis_model_id) {
			alert(m.alert_index_and_model_required?.() || 'Please index the repository and configure an analysis model.');
			return;
		}

		selectedProject = project; // Required for Create Issue button
		analyzingQualityProjectId = project.id;
		isAnalyzingQuality = true;
		qualityResult = null;
		showQualityModal = true;

		try {
			const result = await gitlabApi.analyzeQuality(project.id, { max_files: 30 });
			qualityResult = result;
		} catch (error) {
			console.error('Quality analysis failed:', error);
			qualityResult = {
				project_id: project.id,
				scan_id: '',
				scanned_at: new Date().toISOString(),
				duration: '0s',
				overall_score: 0,
				breakdown: {} as Record<QualityCategory, number>,
				file_scores: [],
				recommendations: [],
				summary: { 
					total_files: 0, 
					total_lines_of_code: 0,
					total_functions: 0,
					issues_count: 0,
					high_severity_count: 0,
					medium_severity_count: 0,
					low_severity_count: 0,
					top_issue_categories: [],
					best_scoring_files: [],
					worst_scoring_files: []
				},
				status: 'failed',
				error: error instanceof Error ? error.message : 'Analysis failed',
				model_id: '',
				tokens_used: 0
			};
		} finally {
			isAnalyzingQuality = false;
			analyzingQualityProjectId = null;
		}
	}
	
	function getScoreColor(score: number): string {
		if (score >= 80) return 'text-green-600';
		if (score >= 60) return 'text-yellow-600';
		if (score >= 40) return 'text-orange-600';
		return 'text-red-600';
	}
	
	function getScoreBgColor(score: number): string {
		if (score >= 80) return 'bg-green-500';
		if (score >= 60) return 'bg-yellow-500';
		if (score >= 40) return 'bg-orange-500';
		return 'bg-red-500';
	}
	
	async function handleDetectDeadCode(project: GitLabProject) {
		if (project.index_status !== 'completed' || !project.analysis_model_id) {
			alert(m.alert_index_and_model_required?.() || 'Please index the repository and configure an analysis model.');
			return;
		}

		selectedProject = project; // Required for Create Issue button
		detectingDeadCodeProjectId = project.id;
		isDetectingDeadCode = true;
		deadCodeResult = null;
		showDeadCodeModal = true;

		try {
			const result = await gitlabApi.detectDeadCode(project.id, { max_chunks: 100 });
			deadCodeResult = result;
		} catch (error) {
			console.error('Dead code detection failed:', error);
			deadCodeResult = {
				project_id: project.id,
				scan_id: '',
				scanned_at: new Date().toISOString(),
				duration: '0s',
				status: 'failed',
				error: error instanceof Error ? error.message : 'Detection failed',
				dead_symbols: [],
				summary: {
					total_dead_symbols: 0,
					by_type: {} as Record<string, number>,
					by_confidence: {} as Record<string, number>,
					estimated_dead_lines: 0,
					top_affected_files: []
				},
				model_id: '',
				tokens_used: 0,
				files_scanned: 0,
				chunks_scanned: 0
			};
		} finally {
			isDetectingDeadCode = false;
			detectingDeadCodeProjectId = null;
		}
	}
	
	function getSymbolTypeIcon(type: string): string {
		switch (type) {
			case 'function': return '𝑓';
			case 'type': return 'T';
			case 'class': return 'C';
			case 'interface': return 'I';
			case 'variable': return 'v';
			case 'constant': return 'K';
			default: return '?';
		}
	}
	
	async function handleAutoDoc(project: GitLabProject) {
		if (project.index_status !== 'completed' || !project.analysis_model_id) {
			alert(m.alert_index_and_model_required?.() || 'Please index the repository and configure an analysis model.');
			return;
		}

		generatingDocsProjectId = project.id;
		isGeneratingDocs = true;
		docScanResult = null;
		docGenResult = null;
		autoDocStep = 'scan';
		showAutoDocModal = true;

		try {
			// Step 1: Scan for undocumented code
			autoDocStep = 'scan';
			const scanResult = await gitlabApi.scanDocs(project.id, { exported_only: true });
			docScanResult = scanResult;
			
			if (scanResult.symbols.length === 0) {
				autoDocStep = 'results';
				return;
			}

			// Step 2: Generate documentation
			autoDocStep = 'generate';
			const genResult = await gitlabApi.generateDocs(project.id, { max_symbols: 15 });
			docGenResult = genResult;
			autoDocStep = 'results';
		} catch (error) {
			console.error('Auto-documentation failed:', error);
			docScanResult = {
				project_id: project.id,
				scan_id: '',
				scanned_at: new Date().toISOString(),
				duration: '0s',
				status: 'failed',
				error: error instanceof Error ? error.message : 'Failed',
				symbols: [],
				summary: {
					total_symbols: 0,
					exported_count: 0,
					by_type: {},
					by_language: {},
					by_importance: {},
					top_affected_files: []
				},
				files_scanned: 0
			};
			autoDocStep = 'results';
		} finally {
			isGeneratingDocs = false;
			generatingDocsProjectId = null;
		}
	}

	async function handleCreateDocsMR() {
		if (!docGenResult || !selectedProject || docGenResult.docs.length === 0) return;
		
		isCreatingDocsMR = true;
		try {
			const result = await gitlabApi.createDocsMR(selectedProject.id, {
				docs: docGenResult.docs,
				title: `[Auto-Doc] Add documentation for ${docGenResult.docs.length} symbols`
			});
			
			if (result.mr_url) {
				alert(`Documentation MR created successfully!\n\n${result.mr_url}`);
				window.open(result.mr_url, '_blank');
			} else {
				alert(result.message);
			}
		} catch (error) {
			console.error('Failed to create docs MR:', error);
			alert('Failed to create MR: ' + (error instanceof Error ? error.message : 'Unknown error'));
		} finally {
			isCreatingDocsMR = false;
		}
	}
	
	async function handleTestGen(project: GitLabProject) {
		if (project.index_status !== 'completed' || !project.analysis_model_id) {
			alert(m.alert_index_and_model_required?.() || 'Please index the repository and configure an analysis model.');
			return;
		}

		generatingTestsProjectId = project.id;
		isGeneratingTests = true;
		testScanResult = null;
		testGenResult = null;
		testGenStep = 'scan';
		showTestGenModal = true;

		try {
			// Step 1: Scan for testable functions
			testGenStep = 'scan';
			const scanResult = await gitlabApi.scanTests(project.id);
			testScanResult = scanResult;
			
			if (scanResult.summary.without_tests === 0) {
				testGenStep = 'results';
				return;
			}

			// Step 2: Generate tests
			testGenStep = 'generate';
			const genResult = await gitlabApi.generateTests(project.id, { max_functions: 10 });
			testGenResult = genResult;
			testGenStep = 'results';
		} catch (error) {
			console.error('Test generation failed:', error);
			testScanResult = {
				project_id: project.id,
				scan_id: '',
				scanned_at: new Date().toISOString(),
				duration: '0s',
				status: 'failed',
				error: error instanceof Error ? error.message : 'Failed',
				functions: [],
				summary: {
					total_functions: 0,
					without_tests: 0,
					with_tests: 0,
					by_language: {},
					by_complexity: {},
					top_files: []
				},
				files_scanned: 0
			};
			testGenStep = 'results';
		} finally {
			isGeneratingTests = false;
			generatingTestsProjectId = null;
		}
	}

	async function handleCreateTestsMR() {
		if (!testGenResult || !selectedProject || testGenResult.tests.length === 0) return;
		
		isCreatingTestsMR = true;
		try {
			const result = await gitlabApi.createTestsMR(selectedProject.id, {
				tests: testGenResult.tests,
				title: `[Auto-Test] Add ${testGenResult.tests.length} unit tests`
			});
			
			if (result.mr_url) {
				alert(`Tests MR created successfully!\n\n${result.mr_url}`);
				window.open(result.mr_url, '_blank');
			} else {
				alert(result.message);
			}
		} catch (error) {
			console.error('Failed to create tests MR:', error);
			alert('Failed to create MR: ' + (error instanceof Error ? error.message : 'Unknown error'));
		} finally {
			isCreatingTestsMR = false;
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
				return faCircleXmark;
			case 'medium':
				return faCircleExclamation;
			default:
				return faCircleCheck;
		}
	}

	function viewProjectDetails(project: GitLabProject) {
		selectedProject = project;
		showProjectDetailModal = true;
	}

	function openEditProject(project: GitLabProject) {
		selectedProject = project;
		// Populate edit form with current values
		editDefaultBranch = project.default_branch || '';
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
		editReviewMode = project.settings?.review_mode || (project.settings?.per_file_review ? 'per_file' : 'standard');
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
				default_branch: editDefaultBranch || undefined,
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
					review_mode: editReviewMode,
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
				return faCircleCheck;
			case 'error':
			case 'failed':
				return faCircleXmark;
			case 'pending':
			case 'queued':
			case 'analyzing':
				return faClock;
			default:
				return faCircleExclamation;
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
			<FontAwesomeIcon icon={faArrowLeft} class="h-5 w-5" />
		</a>
		<div class="flex-1">
			<h2 class="text-2xl font-bold text-foreground">{integration?.name || 'Loading...'}</h2>
			{#if integration}
				<a href={integration.base_url} target="_blank" class="text-sm text-primary hover:underline flex items-center gap-1">
					{integration.base_url}
					<FontAwesomeIcon icon={faArrowUpRightFromSquare} class="h-3 w-3" />
				</a>
			{/if}
		</div>
		{#if integration}
			<div class="flex items-center gap-2">
				<span class="text-sm text-muted-foreground">Webhooks</span>
				<button
					onclick={toggleWebhooksPaused}
					disabled={isTogglingWebhooks}
					class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 {integration.settings?.webhooks_paused ? 'bg-yellow-500' : 'bg-green-500'}"
					title={integration.settings?.webhooks_paused ? 'Webhooks paused — click to resume' : 'Webhooks active — click to pause'}
				>
					<span
						class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform {integration.settings?.webhooks_paused ? 'translate-x-1' : 'translate-x-6'}"
					></span>
				</button>
				{#if integration.settings?.webhooks_paused}
					<span class="text-xs font-medium text-yellow-600 dark:text-yellow-400">Paused</span>
				{/if}
			</div>
		{/if}
		{#if activeTab === 'projects'}
			<Button onclick={openAddProjectModal}>
				<FontAwesomeIcon icon={faPlus} class="mr-2 h-4 w-4" />
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
			<FontAwesomeIcon icon={faFileCode} class="inline-block mr-2 h-4 w-4" />
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
			<FontAwesomeIcon icon={faChartColumn} class="inline-block mr-2 h-4 w-4" />
			{m.admin_project_reviews_tab()} ({totalReviews})
		</button>
	</div>

	<!-- Filters -->
	<div class="flex flex-wrap items-center gap-4">
		<div class="relative flex-1 min-w-[200px]">
			<FontAwesomeIcon icon={faMagnifyingGlass} class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
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
			<FontAwesomeIcon icon={faArrowsRotate} class="mr-2 h-4 w-4" />
			{m.common_reset()}
		</Button>
	</div>

	<!-- Content -->
	<div class="rounded-lg border bg-card">
		{#if isLoading}
			<div class="flex items-center justify-center py-12">
				<FontAwesomeIcon icon={faSpinner} class="h-8 w-8 animate-spin text-primary" />
			</div>
		{:else if activeTab === 'projects'}
			<!-- Projects Table -->
			{#if projects.length === 0}
				<div class="flex flex-col items-center justify-center py-12 text-center">
					<FontAwesomeIcon icon={faFileCode} class="h-12 w-12 text-muted-foreground/50" />
					<p class="mt-4 text-lg font-medium text-muted-foreground">No projects configured</p>
					<p class="text-sm text-muted-foreground">Add a GitLab project to start AI reviews</p>
					<Button class="mt-4" onclick={openAddProjectModal}>
						<FontAwesomeIcon icon={faPlus} class="mr-2 h-4 w-4" />
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
											<FontAwesomeIcon icon={StatusIcon} class={cn('h-4 w-4', getStatusColor(project.status))} />
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
											<FontAwesomeIcon icon={faTowerBroadcast} class="h-4 w-4" />
											#{project.webhook_id}
										</span>
									{:else}
										<button
											onclick={() => handleSetupWebhook(project)}
											class="text-sm text-primary hover:underline flex items-center gap-1"
										>
											<FontAwesomeIcon icon={faTowerBroadcast} class="h-4 w-4" />
											Setup
										</button>
									{/if}
								</td>
								<td class="px-4 py-3">
								{#if indexingProjects.has(project.id) || project.index_status === 'in_progress'}
									<span class="flex items-center gap-1 text-sm text-yellow-500">
										<FontAwesomeIcon icon={faSpinner} class="h-4 w-4 animate-spin" />
										Indexing...
									</span>
								{:else if project.index_status === 'completed'}
									<div class="flex flex-col">
										<span class="flex items-center gap-1 text-sm text-green-500">
											<FontAwesomeIcon icon={faCircleCheck} class="h-4 w-4" />
											{project.last_indexed_at ? formatRelativeTime(project.last_indexed_at) : 'Ready'}
										</span>
										{#if project.index_chunks && project.index_chunks > 0}
											<span class="text-xs text-muted-foreground">{project.index_chunks.toLocaleString()} chunks</span>
										{/if}
									</div>
								{:else if project.index_status === 'failed'}
									<span class="flex items-center gap-1 text-sm text-red-500">
										<FontAwesomeIcon icon={faCircleXmark} class="h-4 w-4" />
										Failed
									</span>
								{:else if project.index_status === 'pending'}
									<span class="flex items-center gap-1 text-sm text-blue-500">
										<FontAwesomeIcon icon={faClock} class="h-4 w-4" />
										Pending
									</span>
								{:else}
									<span class="text-sm text-muted-foreground">Not indexed</span>
								{/if}
								</td>
								<td class="px-4 py-3">
									<span class="text-sm">{project.review_count || 0}</span>
								</td>
								<td class="px-4 py-3">
									<div class="flex items-center gap-1">
										<ActionButton
											label={project.index_status === 'completed' ? 'Reindex' : 'Index'}
											onclick={() => handleStartIndexing(project)}
											disabled={indexingProjects.has(project.id)}
											loading={indexingProjects.has(project.id)}
										>
											{#snippet children()}
												<FontAwesomeIcon icon={faArrowsRotate} class="h-4 w-4" />
											{/snippet}
											{#snippet loadingIcon()}
												<FontAwesomeIcon icon={faArrowsRotate} class="h-4 w-4 animate-spin" />
											{/snippet}
										</ActionButton>

										<ActionButton
											label="Details"
											onclick={() => viewProjectDetails(project)}
										>
											{#snippet children()}
												<FontAwesomeIcon icon={faEye} class="h-4 w-4" />
											{/snippet}
										</ActionButton>

										<ActionButton
											label="Secrets"
											onclick={() => handleScanSecrets(project)}
											disabled={scanningProjectId === project.id || project.index_status !== 'completed'}
											loading={scanningProjectId === project.id}
										>
											{#snippet children()}
												<FontAwesomeIcon icon={faShieldHalved} class="h-4 w-4" />
											{/snippet}
											{#snippet loadingIcon()}
												<FontAwesomeIcon icon={faSpinner} class="h-4 w-4 animate-spin" />
											{/snippet}
										</ActionButton>

										<ActionButton
											label="Deep Scan"
											onclick={() => handleDeepScan(project)}
											disabled={deepScanningProjectId === project.id || project.index_status !== 'completed' || !project.analysis_model_id}
											loading={deepScanningProjectId === project.id}
										>
											{#snippet children()}
												<FontAwesomeIcon icon={faBrain} class="h-4 w-4 text-purple-500" />
											{/snippet}
											{#snippet loadingIcon()}
												<FontAwesomeIcon icon={faSpinner} class="h-4 w-4 animate-spin text-purple-500" />
											{/snippet}
										</ActionButton>

										<ActionButton
											label="Dependencies"
											onclick={() => handleCheckDependencies(project)}
											disabled={checkingDependenciesProjectId === project.id || project.index_status !== 'completed'}
											loading={checkingDependenciesProjectId === project.id}
										>
											{#snippet children()}
												<FontAwesomeIcon icon={faBox} class="h-4 w-4 text-blue-500" />
											{/snippet}
											{#snippet loadingIcon()}
												<FontAwesomeIcon icon={faSpinner} class="h-4 w-4 animate-spin text-blue-500" />
											{/snippet}
										</ActionButton>

										<ActionButton
											label="Quality"
											onclick={() => handleAnalyzeQuality(project)}
											disabled={analyzingQualityProjectId === project.id || project.index_status !== 'completed' || !project.analysis_model_id}
											loading={analyzingQualityProjectId === project.id}
										>
											{#snippet children()}
												<FontAwesomeIcon icon={faChartColumn} class="h-4 w-4 text-indigo-500" />
											{/snippet}
											{#snippet loadingIcon()}
												<FontAwesomeIcon icon={faSpinner} class="h-4 w-4 animate-spin text-indigo-500" />
											{/snippet}
										</ActionButton>

										<ActionButton
											label="Dead Code"
											onclick={() => handleDetectDeadCode(project)}
											disabled={detectingDeadCodeProjectId === project.id || project.index_status !== 'completed' || !project.analysis_model_id}
											loading={detectingDeadCodeProjectId === project.id}
										>
											{#snippet children()}
												<FontAwesomeIcon icon={faFileCode} class="h-4 w-4 text-orange-500" />
											{/snippet}
											{#snippet loadingIcon()}
												<FontAwesomeIcon icon={faSpinner} class="h-4 w-4 animate-spin text-orange-500" />
											{/snippet}
										</ActionButton>

										<ActionButton
											label="Auto Doc"
											onclick={() => handleAutoDoc(project)}
											disabled={generatingDocsProjectId === project.id || project.index_status !== 'completed' || !project.analysis_model_id}
											loading={generatingDocsProjectId === project.id}
										>
											{#snippet children()}
												<FontAwesomeIcon icon={faFilePen} class="h-4 w-4 text-purple-500" />
											{/snippet}
											{#snippet loadingIcon()}
												<FontAwesomeIcon icon={faSpinner} class="h-4 w-4 animate-spin text-purple-500" />
											{/snippet}
										</ActionButton>

										<ActionButton
											label="Tests"
											onclick={() => handleTestGen(project)}
											disabled={generatingTestsProjectId === project.id || project.index_status !== 'completed' || !project.analysis_model_id}
											loading={generatingTestsProjectId === project.id}
										>
											{#snippet children()}
												<FontAwesomeIcon icon={faFlask} class="h-4 w-4 text-green-500" />
											{/snippet}
											{#snippet loadingIcon()}
												<FontAwesomeIcon icon={faSpinner} class="h-4 w-4 animate-spin text-green-500" />
											{/snippet}
										</ActionButton>

										<ActionButton
											label="Settings"
											onclick={() => openEditProject(project)}
										>
											{#snippet children()}
												<FontAwesomeIcon icon={faGear} class="h-4 w-4" />
											{/snippet}
										</ActionButton>

										<ActionButton
											label="Delete"
											onclick={() => handleDeleteProject(project)}
											class="text-red-500 hover:bg-red-500/10"
										>
											{#snippet children()}
												<FontAwesomeIcon icon={faTrash} class="h-4 w-4" />
											{/snippet}
										</ActionButton>
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
					<FontAwesomeIcon icon={faChartColumn} class="h-12 w-12 text-muted-foreground/50" />
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
											<FontAwesomeIcon icon={ReviewStatusIcon} class={cn('h-4 w-4', getStatusColor(review.status))} />
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
											<FontAwesomeIcon icon={faEye} class="h-4 w-4" />
										</button>
										<a
											href={review.mr_url}
											target="_blank"
											class="rounded p-1.5 hover:bg-muted"
											title="Open in GitLab"
										>
											<FontAwesomeIcon icon={faArrowUpRightFromSquare} class="h-4 w-4" />
										</a>
										{#if review.status === 'failed'}
											<button
												onclick={() => handleRetryReview(review)}
												class="rounded p-1.5 text-primary hover:bg-primary/10"
												title="Retry"
											>
												<FontAwesomeIcon icon={faArrowsRotate} class="h-4 w-4" />
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
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
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
					<FontAwesomeIcon icon={faGear} class="h-4 w-4" />
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
						<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
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
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
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
				<Button onclick={() => { showProjectDetailModal = false; if (selectedProject) openEditProject(selectedProject); }}>
					<FontAwesomeIcon icon={faGear} class="mr-2 h-4 w-4" />
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
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
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

				<!-- Default Branch -->
				<div>
					<label for="edit-default-branch" class="text-sm font-medium">Default Branch</label>
					<input
						id="edit-default-branch"
						type="text"
						bind:value={editDefaultBranch}
						placeholder="e.g., main, master"
						class="mt-1 w-full rounded-md border bg-background px-3 py-2"
					/>
					<p class="text-xs text-muted-foreground mt-1">Target branch for MR creation and indexing</p>
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
							<label for="editReviewMode" class="text-sm">Review Mode:</label>
							<select
								id="editReviewMode"
								bind:value={editReviewMode}
								class="rounded border px-2 py-1 bg-background text-sm"
							>
								<option value="standard">Standard (batch JSON)</option>
								<option value="per_file">Per-file (with read tools)</option>
								<option value="tool_based">Tool-based (structured output)</option>
							</select>
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
						<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
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
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
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
						<FontAwesomeIcon icon={faArrowUpRightFromSquare} class="mr-2 h-4 w-4" />
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
					<FontAwesomeIcon icon={faShieldHalved} class="h-5 w-5 text-primary" />
					{m.gitlab_secrets_scan_title?.() || 'Secrets Scan Results'}
				</h3>
				<button onclick={() => (showSecretsModal = false)} class="rounded p-1 hover:bg-muted">
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			{#if isScanning}
				<div class="flex flex-col items-center justify-center py-12">
					<FontAwesomeIcon icon={faSpinner} class="h-12 w-12 animate-spin text-primary" />
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
							<FontAwesomeIcon icon={faCircleCheck} class="h-16 w-16 text-green-500" />
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

			<div class="mt-6 flex justify-end gap-3">
				{#if secretsScanResult && !isScanning && secretsScanResult.summary?.total_findings > 0}
					<Button 
						variant="default" 
						onclick={handleCreateSecretsIssue}
						disabled={isCreatingSecretsIssue}
					>
						{#if isCreatingSecretsIssue}
							<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
							{m.common_creating?.() || 'Creating...'}
						{:else}
							<FontAwesomeIcon icon={faCircleExclamation} class="mr-2 h-4 w-4" />
							{m.gitlab_create_issue?.() || 'Create Issue'}
						{/if}
					</Button>
				{/if}
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
					<FontAwesomeIcon icon={faBrain} class="h-6 w-6 text-purple-500" />
					<h3 class="text-lg font-semibold">
						{m.gitlab_deep_scan_title?.() || 'Deep Scan Results (LLM Analysis)'}
					</h3>
				</div>
				<button onclick={() => (showDeepScanModal = false)} class="rounded p-1 hover:bg-muted">
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			{#if isDeepScanning}
				<div class="flex flex-col items-center justify-center py-12">
					<FontAwesomeIcon icon={faSpinner} class="h-12 w-12 animate-spin text-purple-500" />
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
						<FontAwesomeIcon icon={faRobot} class="h-4 w-4" />
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
												<FontAwesomeIcon icon={faFileCode} class="h-3 w-3" />
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
							<FontAwesomeIcon icon={faCircleCheck} class="h-16 w-16 text-green-500" />
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

			<div class="mt-6 flex justify-end gap-3">
				{#if deepScanResult && !isDeepScanning && deepScanResult.findings && deepScanResult.findings.length > 0}
					<Button 
						variant="default" 
						onclick={handleCreateSecretsIssue}
						disabled={isCreatingSecretsIssue}
					>
						{#if isCreatingSecretsIssue}
							<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
							{m.common_creating?.() || 'Creating...'}
						{:else}
							<FontAwesomeIcon icon={faCircleExclamation} class="mr-2 h-4 w-4" />
							{m.gitlab_create_issue?.() || 'Create Issue'}
						{/if}
					</Button>
				{/if}
				<Button variant="outline" onclick={() => (showDeepScanModal = false)}>
					{m.common_close()}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Dependencies Check Modal -->
{#if showDependenciesModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		role="dialog"
		aria-modal="true"
		onclick={(e) => e.target === e.currentTarget && (showDependenciesModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showDependenciesModal = false)}
		tabindex="-1"
	>
		<div class="max-h-[90vh] w-full max-w-5xl overflow-auto rounded-lg bg-background p-6 shadow-lg">
			<div class="mb-4 flex items-center justify-between">
				<div class="flex items-center gap-3">
					<FontAwesomeIcon icon={faBox} class="h-6 w-6 text-blue-500" />
					<h3 class="text-lg font-semibold">
						{m.gitlab_dependencies_title?.() || 'Dependencies Check Results'}
					</h3>
				</div>
				<button onclick={() => (showDependenciesModal = false)} class="rounded p-1 hover:bg-muted">
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			{#if isCheckingDependencies}
				<div class="flex flex-col items-center justify-center py-12">
					<FontAwesomeIcon icon={faSpinner} class="h-12 w-12 animate-spin text-blue-500" />
					<p class="mt-4 text-muted-foreground">{m.gitlab_checking_dependencies?.() || 'Checking dependencies...'}</p>
					<p class="mt-2 text-sm text-muted-foreground">{m.gitlab_deps_patience?.() || 'Fetching version info from registries'}</p>
				</div>
			{:else if dependenciesScanResult}
				{#if dependenciesScanResult.status === 'failed'}
					<div class="rounded-lg bg-red-100 dark:bg-red-900/30 p-4 text-red-700 dark:text-red-400">
						<p class="font-medium">{m.gitlab_scan_failed?.() || 'Check Failed'}</p>
						<p class="text-sm">{dependenciesScanResult.error}</p>
					</div>
				{:else if dependenciesScanResult.ecosystems.length === 0}
					<div class="flex flex-col items-center justify-center py-12 text-center">
						<FontAwesomeIcon icon={faBox} class="h-16 w-16 text-muted-foreground/50" />
						<p class="mt-4 text-lg font-medium text-muted-foreground">
							{m.gitlab_no_dependencies?.() || 'No dependencies found'}
						</p>
						<p class="text-sm text-muted-foreground">
							{m.gitlab_deps_file_not_found?.() || 'Could not find dependency file in indexed code.'}
						</p>
					</div>
				{:else}
					<!-- Total Summary (all ecosystems) -->
					<div class="mb-4 text-sm text-muted-foreground flex items-center gap-2">
						<FontAwesomeIcon icon={faClock} class="h-4 w-4" />
						<span>{dependenciesScanResult.duration}</span>
						<span class="mx-2">•</span>
						<span>{dependenciesScanResult.ecosystems.length} ecosystem(s) scanned</span>
					</div>

					<!-- Total Summary stats -->
					<div class="mb-6 grid grid-cols-2 md:grid-cols-5 gap-4">
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold">{dependenciesScanResult.total_summary.total_dependencies}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_deps_total?.() || 'Total'}</div>
						</div>
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold">{dependenciesScanResult.total_summary.direct_dependencies}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_deps_direct?.() || 'Direct'}</div>
						</div>
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold text-yellow-600">{dependenciesScanResult.total_summary.outdated_count}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_deps_outdated?.() || 'Outdated'}</div>
						</div>
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold text-red-600">{dependenciesScanResult.total_summary.vulnerable_count}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_deps_vulnerable?.() || 'Vulnerable'}</div>
						</div>
						<div class="rounded-lg bg-muted p-4">
							<div class="text-2xl font-bold text-green-600">{dependenciesScanResult.total_summary.up_to_date_count}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_deps_uptodate?.() || 'Up to Date'}</div>
						</div>
					</div>

					<!-- Update type breakdown -->
					{#if Object.keys(dependenciesScanResult.total_summary.by_update_type).length > 0}
						<div class="mb-4 flex flex-wrap gap-2">
							{#each Object.entries(dependenciesScanResult.total_summary.by_update_type) as [updateType, count]}
								<span class={cn('px-3 py-1 rounded-full text-sm font-medium', getUpdateTypeColor(updateType))}>
									{updateType}: {count}
								</span>
							{/each}
						</div>
					{/if}

					<!-- Ecosystem tabs -->
					<div class="mb-4 border-b">
						<div class="flex gap-1">
							{#each dependenciesScanResult.ecosystems as ecosystem, idx}
								<button
									onclick={() => selectedEcosystemIndex = idx}
									class={cn(
										'px-4 py-2 text-sm font-medium border-b-2 transition-colors',
										selectedEcosystemIndex === idx
											? 'border-primary text-primary'
											: 'border-transparent text-muted-foreground hover:text-foreground'
									)}
								>
									{ecosystem.language === 'go' ? 'Go' : ecosystem.language === 'nodejs' ? 'Node.js' : ecosystem.language === 'python' ? 'Python' : ecosystem.language}
									<span class="ml-1 text-xs text-muted-foreground">({ecosystem.dependencies.length})</span>
								</button>
							{/each}
						</div>
					</div>

					<!-- Selected ecosystem content -->
					{@const selectedEcosystem = dependenciesScanResult.ecosystems[selectedEcosystemIndex]}
					{#if selectedEcosystem}
						<!-- File info -->
						<div class="mb-4 text-sm text-muted-foreground flex items-center gap-2">
							<FontAwesomeIcon icon={faFileCode} class="h-4 w-4" />
							<span>{selectedEcosystem.file_path}</span>
							<span class="mx-2">•</span>
							<FontAwesomeIcon icon={faClock} class="h-4 w-4" />
							<span>{selectedEcosystem.duration}</span>
						</div>

						<!-- Dependencies list -->
						{#if selectedEcosystem.dependencies.length > 0}
							<div class="space-y-2 max-h-[50vh] overflow-auto">
								<!-- Header -->
								<div class="grid grid-cols-12 gap-2 px-3 py-2 text-sm font-medium text-muted-foreground border-b">
									<div class="col-span-4">{m.gitlab_deps_package?.() || 'Package'}</div>
									<div class="col-span-2">{m.gitlab_deps_current?.() || 'Current'}</div>
									<div class="col-span-2">{m.gitlab_deps_latest?.() || 'Latest'}</div>
									<div class="col-span-1">{m.gitlab_deps_update?.() || 'Update'}</div>
									<div class="col-span-1">{m.gitlab_deps_status?.() || 'Status'}</div>
									<div class="col-span-2">{m.gitlab_deps_actions?.() || 'Actions'}</div>
								</div>
								
								{#each selectedEcosystem.dependencies as depWithVulns}
									<div class={cn(
										'grid grid-cols-12 gap-2 px-3 py-2 rounded-lg text-sm',
										depWithVulns.is_vulnerable ? 'bg-red-50 dark:bg-red-900/20' : 
										depWithVulns.dependency.has_update ? 'bg-yellow-50 dark:bg-yellow-900/20' : 'hover:bg-muted/50'
									)}>
										<div class="col-span-4 font-mono text-xs truncate" title={depWithVulns.dependency.name}>
											{depWithVulns.dependency.name}
											{#if depWithVulns.dependency.indirect}
												<span class="text-muted-foreground ml-1">(indirect)</span>
											{/if}
										</div>
										<div class="col-span-2 font-mono text-xs">{depWithVulns.dependency.current_version}</div>
										<div class="col-span-2 font-mono text-xs">{depWithVulns.dependency.latest_version}</div>
										<div class="col-span-1">
											{#if depWithVulns.dependency.update_type !== 'none'}
												<span class={cn('px-2 py-0.5 rounded text-xs font-medium', getUpdateTypeColor(depWithVulns.dependency.update_type))}>
													{depWithVulns.dependency.update_type}
												</span>
											{:else}
												<span class="text-green-600 text-xs">✓</span>
											{/if}
										</div>
										<div class="col-span-1">
											{#if depWithVulns.is_vulnerable}
												<span class="text-red-600" title={`${depWithVulns.vulnerabilities.length} vulnerabilities`}>
													<FontAwesomeIcon icon={faTriangleExclamation} class="mr-0.5 h-3 w-3" /> {depWithVulns.vulnerabilities.length}
												</span>
											{:else}
												<span class="text-green-600">✓</span>
											{/if}
										</div>
										<div class="col-span-2">
											{#if depWithVulns.dependency.has_update}
												<button
													onclick={() => handleAnalyzeChangelog(depWithVulns, dependenciesScanResult!.project_id, selectedEcosystem.language)}
													class="px-2 py-1 text-xs rounded bg-blue-100 hover:bg-blue-200 dark:bg-blue-900/30 dark:hover:bg-blue-900/50 text-blue-700 dark:text-blue-300"
													title={m.gitlab_analyze_changelog?.() || 'Analyze Changelog'}
												>
													{m.gitlab_analyze_changelog_short?.() || 'Analyze'}
												</button>
											{/if}
										</div>
									</div>
									
									<!-- Vulnerabilities expandable -->
									{#if depWithVulns.is_vulnerable}
										<div class="ml-4 mb-2 space-y-1">
											{#each depWithVulns.vulnerabilities as vuln}
												<div class="text-xs p-2 rounded bg-red-100 dark:bg-red-900/30 border-l-4 border-red-500">
													<div class="flex items-center gap-2">
														<span class={cn('px-1.5 py-0.5 rounded text-xs font-bold', getSeverityColor(vuln.severity))}>
															{vuln.severity.toUpperCase()}
														</span>
														<a href={vuln.references?.[0]} target="_blank" class="text-blue-600 hover:underline font-medium">
															{vuln.id}
														</a>
													</div>
													<p class="mt-1 text-muted-foreground">{vuln.summary}</p>
													{#if vuln.fixed_in}
														<p class="mt-1"><strong>Fix:</strong> Upgrade to {vuln.fixed_in}</p>
													{/if}
												</div>
											{/each}
										</div>
									{/if}
								{/each}
							</div>
						{:else}
							<div class="flex flex-col items-center justify-center py-8 text-center">
								<FontAwesomeIcon icon={faBox} class="h-12 w-12 text-muted-foreground/50" />
								<p class="mt-2 text-muted-foreground">No dependencies in this ecosystem</p>
							</div>
						{/if}
					{/if}
				{/if}
			{/if}

			<div class="mt-6 flex justify-end gap-3">
				{#if dependenciesScanResult && !isCheckingDependencies && (dependenciesScanResult.total_summary.vulnerable_count > 0 || dependenciesScanResult.total_summary.outdated_count > 0)}
					<Button 
						variant="default" 
						onclick={handleCreateDependencyIssue}
						disabled={isCreatingDependencyIssue}
					>
						{#if isCreatingDependencyIssue}
							<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
							{m.common_creating?.() || 'Creating...'}
						{:else}
							<FontAwesomeIcon icon={faCircleExclamation} class="mr-2 h-4 w-4" />
							{m.gitlab_create_issue?.() || 'Create Issue'}
						{/if}
					</Button>
				{/if}
				<Button variant="outline" onclick={() => (showDependenciesModal = false)}>
					{m.common_close()}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Changelog Analysis Modal -->
{#if showChangelogModal}
	<div
		class="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 p-4"
		role="dialog"
		aria-modal="true"
		onclick={(e) => e.target === e.currentTarget && (showChangelogModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showChangelogModal = false)}
		tabindex="-1"
	>
		<div class="max-h-[90vh] w-full max-w-3xl overflow-auto rounded-lg bg-background p-6 shadow-lg">
			<div class="mb-4 flex items-center justify-between">
				<div class="flex items-center gap-3">
					<FontAwesomeIcon icon={faFileCode} class="h-6 w-6 text-blue-500" />
					<div>
						<h3 class="text-lg font-semibold">
							{m.gitlab_changelog_title?.() || 'Changelog Analysis'}
						</h3>
						{#if analyzingPackage}
							<p class="text-sm text-muted-foreground font-mono">
								{analyzingPackage.name}: {analyzingPackage.current} → {analyzingPackage.latest}
							</p>
						{/if}
					</div>
				</div>
				<button onclick={() => (showChangelogModal = false)} class="rounded p-1 hover:bg-muted">
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			{#if isAnalyzingChangelog}
				<div class="flex flex-col items-center justify-center py-12">
					<FontAwesomeIcon icon={faSpinner} class="h-12 w-12 animate-spin text-blue-500" />
					<p class="mt-4 text-muted-foreground">{m.gitlab_analyzing_changelog?.() || 'Analyzing changelog...'}</p>
					<p class="mt-2 text-sm text-muted-foreground">{m.gitlab_changelog_patience?.() || 'Fetching changelog and analyzing with LLM'}</p>
				</div>
			{:else if changelogResult}
				<!-- Risk Level Badge -->
				<div class="mb-4 flex items-center gap-4">
					<span class={cn('px-3 py-1 rounded-full text-sm font-bold uppercase', getRiskLevelColor(changelogResult.risk_level))}>
						{changelogResult.risk_level} {m.gitlab_risk?.() || 'Risk'}
					</span>
					<span class={cn('px-3 py-1 rounded-full text-sm', getConfidenceColor(changelogResult.confidence))}>
						{changelogResult.confidence} {m.gitlab_confidence?.() || 'confidence'}
					</span>
				</div>

				<!-- Summary -->
				<div class="mb-6 p-4 rounded-lg bg-muted">
					<h4 class="font-semibold mb-2">{m.gitlab_summary?.() || 'Summary'}</h4>
					<p class="text-sm">{changelogResult.summary}</p>
				</div>

				<!-- Breaking Changes -->
				{#if changelogResult.breaking_changes?.length > 0}
					<div class="mb-6">
						<h4 class="font-semibold mb-2 text-red-600 flex items-center gap-2">
							<FontAwesomeIcon icon={faCircleExclamation} class="h-5 w-5" />
							{m.gitlab_breaking_changes?.() || 'Breaking Changes'} ({changelogResult.breaking_changes.length})
						</h4>
						<div class="space-y-2">
							{#each changelogResult.breaking_changes as bc}
								<div class="p-3 rounded border-l-4 border-red-500 bg-red-50 dark:bg-red-900/20">
									<div class="flex items-center gap-2 mb-1">
										<span class={cn('px-1.5 py-0.5 rounded text-xs font-bold', 
											bc.severity === 'high' ? 'bg-red-200 text-red-800' : 
											bc.severity === 'medium' ? 'bg-yellow-200 text-yellow-800' : 
											'bg-gray-200 text-gray-800'
										)}>
											{bc.severity.toUpperCase()}
										</span>
										<span class="text-xs text-muted-foreground">{bc.affected_area}</span>
									</div>
									<p class="text-sm font-medium">{bc.description}</p>
									{#if bc.workaround}
										<p class="text-xs text-muted-foreground mt-1">
											<strong>{m.gitlab_workaround?.() || 'Workaround'}:</strong> {bc.workaround}
										</p>
									{/if}
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- New Features -->
				{#if changelogResult.new_features?.length > 0}
					<div class="mb-4">
						<h4 class="font-semibold mb-2 text-green-600">{m.gitlab_new_features?.() || 'New Features'}</h4>
						<ul class="list-disc list-inside text-sm space-y-1">
							{#each changelogResult.new_features as feature}
								<li>{feature}</li>
							{/each}
						</ul>
					</div>
				{/if}

				<!-- Bug Fixes -->
				{#if changelogResult.bug_fixes?.length > 0}
					<div class="mb-4">
						<h4 class="font-semibold mb-2 text-blue-600">{m.gitlab_bug_fixes?.() || 'Bug Fixes'}</h4>
						<ul class="list-disc list-inside text-sm space-y-1">
							{#each changelogResult.bug_fixes as fix}
								<li>{fix}</li>
							{/each}
						</ul>
					</div>
				{/if}

				<!-- Security Fixes -->
				{#if changelogResult.security_fixes?.length > 0}
					<div class="mb-4">
						<h4 class="font-semibold mb-2 text-orange-600">{m.gitlab_security_fixes?.() || 'Security Fixes'}</h4>
						<ul class="list-disc list-inside text-sm space-y-1">
							{#each changelogResult.security_fixes as fix}
								<li>{fix}</li>
							{/each}
						</ul>
					</div>
				{/if}

				<!-- Deprecated Features -->
				{#if changelogResult.deprecated_features?.length > 0}
					<div class="mb-4">
						<h4 class="font-semibold mb-2 text-yellow-600">{m.gitlab_deprecated?.() || 'Deprecated Features'}</h4>
						<ul class="list-disc list-inside text-sm space-y-1">
							{#each changelogResult.deprecated_features as dep}
								<li>{dep}</li>
							{/each}
						</ul>
					</div>
				{/if}

				<!-- Migration Guide -->
				{#if changelogResult.migration_guide}
					<div class="mb-4">
						<h4 class="font-semibold mb-2">{m.gitlab_migration_guide?.() || 'Migration Guide'}</h4>
						<div class="p-3 rounded bg-muted text-sm whitespace-pre-wrap">{changelogResult.migration_guide}</div>
					</div>
				{/if}

				<!-- Tokens Used -->
				<div class="text-xs text-muted-foreground text-right">
					{m.gitlab_tokens_used?.() || 'Tokens used'}: {changelogResult.tokens_used}
				</div>
			{/if}

			<div class="mt-6 flex justify-end">
				<Button variant="outline" onclick={() => (showChangelogModal = false)}>
					{m.common_close()}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Code Quality Modal -->
{#if showQualityModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		role="dialog"
		aria-modal="true"
		onclick={(e) => e.target === e.currentTarget && (showQualityModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showQualityModal = false)}
		tabindex="-1"
	>
		<div class="max-h-[90vh] w-full max-w-5xl overflow-auto rounded-lg bg-background p-6 shadow-lg">
			<div class="mb-4 flex items-center justify-between">
				<div class="flex items-center gap-3">
					<FontAwesomeIcon icon={faChartColumn} class="h-6 w-6 text-indigo-500" />
					<h3 class="text-lg font-semibold">
						{m.gitlab_quality_title?.() || 'Code Quality Score'}
					</h3>
				</div>
				<button onclick={() => (showQualityModal = false)} class="rounded p-1 hover:bg-muted">
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			{#if isAnalyzingQuality}
				<div class="flex flex-col items-center justify-center py-12">
					<FontAwesomeIcon icon={faSpinner} class="h-12 w-12 animate-spin text-indigo-500" />
					<p class="mt-4 text-muted-foreground">{m.gitlab_analyzing_quality?.() || 'Analyzing code quality...'}</p>
					<p class="mt-2 text-sm text-muted-foreground">{m.gitlab_quality_patience?.() || 'This may take a few minutes for large projects'}</p>
				</div>
			{:else if qualityResult}
				{#if qualityResult.status === 'failed'}
					<div class="rounded-lg bg-red-100 dark:bg-red-900/30 p-4 text-red-700 dark:text-red-400">
						<p class="font-medium">{m.gitlab_scan_failed?.() || 'Analysis Failed'}</p>
						<p class="text-sm">{qualityResult.error}</p>
					</div>
				{:else}
					<!-- Overall Score Gauge -->
					<div class="mb-8 flex flex-col items-center">
						<div class="relative w-48 h-48">
							<!-- Background circle -->
							<svg class="w-full h-full transform -rotate-90" viewBox="0 0 100 100">
								<circle cx="50" cy="50" r="45" fill="none" stroke="currentColor" class="text-muted" stroke-width="10" />
								<circle 
									cx="50" cy="50" r="45" fill="none" 
									class={getScoreBgColor(qualityResult.overall_score)} 
									stroke-width="10"
									stroke-dasharray={`${qualityResult.overall_score * 2.83} 283`}
									stroke-linecap="round"
								/>
							</svg>
							<!-- Score text -->
							<div class="absolute inset-0 flex flex-col items-center justify-center">
								<span class={cn('text-5xl font-bold', getScoreColor(qualityResult.overall_score))}>
									{qualityResult.overall_score}
								</span>
								<span class="text-sm text-muted-foreground">/100</span>
							</div>
						</div>
						<p class="mt-2 text-muted-foreground">{m.gitlab_overall_score?.() || 'Overall Score'}</p>
					</div>

					<!-- Category Breakdown -->
					<div class="mb-6 grid grid-cols-2 md:grid-cols-5 gap-4">
						{#each Object.entries(qualityResult.breakdown) as [category, score]}
							<div class="rounded-lg bg-muted p-3 text-center">
								<div class={cn('text-2xl font-bold', getScoreColor(score))}>{score}</div>
								<div class="text-xs text-muted-foreground capitalize">{category.replace('_', ' ')}</div>
							</div>
						{/each}
					</div>

					<!-- Summary -->
					<div class="mb-6 grid grid-cols-2 md:grid-cols-4 gap-4">
						<div class="rounded-lg border p-3">
							<div class="text-2xl font-bold">{qualityResult.summary.total_files}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_quality_files?.() || 'Files Analyzed'}</div>
						</div>
						<div class="rounded-lg border p-3">
							<div class="text-2xl font-bold">{qualityResult.summary.total_lines_of_code.toLocaleString()}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_quality_loc?.() || 'Lines of Code'}</div>
						</div>
						<div class="rounded-lg border p-3">
							<div class="text-2xl font-bold">{qualityResult.summary.issues_count}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_quality_issues?.() || 'Issues Found'}</div>
						</div>
						<div class="rounded-lg border p-3">
							<div class="text-2xl font-bold">{qualityResult.tokens_used.toLocaleString()}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_tokens_used?.() || 'Tokens Used'}</div>
						</div>
					</div>

					<!-- Severity breakdown -->
					<div class="mb-6 flex gap-4">
						<span class="px-3 py-1 rounded-full text-sm font-medium bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400">
							{qualityResult.summary.high_severity_count} High
						</span>
						<span class="px-3 py-1 rounded-full text-sm font-medium bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400">
							{qualityResult.summary.medium_severity_count} Medium
						</span>
						<span class="px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
							{qualityResult.summary.low_severity_count} Low
						</span>
					</div>

					<!-- Recommendations -->
					{#if qualityResult.recommendations.length > 0}
						<div class="mb-6">
							<h4 class="text-sm font-semibold mb-3">{m.gitlab_recommendations?.() || 'Recommendations'}</h4>
							<div class="space-y-2">
								{#each qualityResult.recommendations.slice(0, 5) as rec}
									<div class={cn(
										'p-3 rounded-lg border-l-4',
										rec.priority === 'high' ? 'border-red-500 bg-red-50 dark:bg-red-900/10' :
										rec.priority === 'medium' ? 'border-yellow-500 bg-yellow-50 dark:bg-yellow-900/10' :
										'border-blue-500 bg-blue-50 dark:bg-blue-900/10'
									)}>
										<div class="font-medium">{rec.title}</div>
										<div class="text-sm text-muted-foreground">{rec.description}</div>
										{#if rec.impact}
											<div class="text-xs text-green-600 mt-1">{rec.impact}</div>
										{/if}
									</div>
								{/each}
							</div>
						</div>
					{/if}

					<!-- Best/Worst files -->
					<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
						{#if qualityResult.summary.best_scoring_files.length > 0}
							<div class="rounded-lg border p-3">
								<h4 class="text-sm font-semibold text-green-600 mb-2"><FontAwesomeIcon icon={faCircleCheck} class="mr-1 h-3.5 w-3.5" /> {m.gitlab_best_files?.() || 'Best Files'}</h4>
								<ul class="text-xs space-y-1">
									{#each qualityResult.summary.best_scoring_files as file}
										<li class="truncate font-mono">{file}</li>
									{/each}
								</ul>
							</div>
						{/if}
						{#if qualityResult.summary.worst_scoring_files.length > 0}
							<div class="rounded-lg border p-3">
								<h4 class="text-sm font-semibold text-red-600 mb-2"><FontAwesomeIcon icon={faTriangleExclamation} class="mr-1 h-3.5 w-3.5" /> {m.gitlab_worst_files?.() || 'Needs Improvement'}</h4>
								<ul class="text-xs space-y-1">
									{#each qualityResult.summary.worst_scoring_files as file}
										<li class="truncate font-mono">{file}</li>
									{/each}
								</ul>
							</div>
						{/if}
					</div>

					<!-- Model info -->
					<div class="mt-4 text-sm text-muted-foreground flex items-center gap-2">
						<FontAwesomeIcon icon={faRobot} class="h-4 w-4" />
						<span>{m.gitlab_analyzed_by?.() || 'Analyzed by'}: <strong>{qualityResult.model_id}</strong></span>
						<span class="mx-2">•</span>
						<FontAwesomeIcon icon={faClock} class="h-4 w-4" />
						<span>{qualityResult.duration}</span>
					</div>
				{/if}
			{/if}

			<div class="mt-6 flex justify-end gap-3">
				{#if qualityResult && !isAnalyzingQuality && qualityResult.recommendations && qualityResult.recommendations.length > 0}
					<Button 
						variant="default" 
						onclick={handleCreateQualityIssue}
						disabled={isCreatingQualityIssue}
					>
						{#if isCreatingQualityIssue}
							<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
							{m.common_creating?.() || 'Creating...'}
						{:else}
							<FontAwesomeIcon icon={faCircleExclamation} class="mr-2 h-4 w-4" />
							{m.gitlab_create_issue?.() || 'Create Issue'}
						{/if}
					</Button>
				{/if}
				<Button variant="outline" onclick={() => (showQualityModal = false)}>
					{m.common_close()}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Dead Code Modal -->
{#if showDeadCodeModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		role="dialog"
		aria-modal="true"
		onclick={(e) => e.target === e.currentTarget && (showDeadCodeModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showDeadCodeModal = false)}
		tabindex="-1"
	>
		<div class="max-h-[90vh] w-full max-w-5xl overflow-auto rounded-lg bg-background p-6 shadow-lg">
			<div class="mb-4 flex items-center justify-between">
				<div class="flex items-center gap-3">
					<FontAwesomeIcon icon={faFileCode} class="h-6 w-6 text-orange-500" />
					<h3 class="text-lg font-semibold">
						{m.gitlab_dead_code_title?.() || 'Dead Code Detection'}
					</h3>
				</div>
				<button onclick={() => (showDeadCodeModal = false)} class="rounded p-1 hover:bg-muted">
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			{#if isDetectingDeadCode}
				<div class="flex flex-col items-center justify-center py-12">
					<FontAwesomeIcon icon={faSpinner} class="h-12 w-12 animate-spin text-orange-500" />
					<p class="mt-4 text-muted-foreground">{m.gitlab_detecting_dead_code?.() || 'Detecting unused code...'}</p>
					<p class="mt-2 text-sm text-muted-foreground">{m.gitlab_dead_code_patience?.() || 'Analyzing symbols and references...'}</p>
				</div>
			{:else if deadCodeResult}
				{#if deadCodeResult.status === 'failed'}
					<div class="rounded-lg bg-red-100 dark:bg-red-900/30 p-4 text-red-700 dark:text-red-400">
						<p class="font-medium">{m.gitlab_scan_failed?.() || 'Detection Failed'}</p>
						<p class="text-sm">{deadCodeResult.error}</p>
					</div>
				{:else}
					<!-- Summary Stats -->
					<div class="mb-6 grid grid-cols-2 md:grid-cols-4 gap-4">
						<div class="rounded-lg border p-3 text-center">
							<div class="text-3xl font-bold text-orange-600">{deadCodeResult.summary.total_dead_symbols}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_dead_symbols?.() || 'Dead Symbols'}</div>
						</div>
						<div class="rounded-lg border p-3 text-center">
							<div class="text-3xl font-bold">{deadCodeResult.summary.estimated_dead_lines}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_dead_lines?.() || 'Lines'}</div>
						</div>
						<div class="rounded-lg border p-3 text-center">
							<div class="text-3xl font-bold">{deadCodeResult.files_scanned}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_files_scanned?.() || 'Files Scanned'}</div>
						</div>
						<div class="rounded-lg border p-3 text-center">
							<div class="text-3xl font-bold">{deadCodeResult.tokens_used.toLocaleString()}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_tokens_used?.() || 'Tokens'}</div>
						</div>
					</div>

					<!-- Confidence breakdown -->
					<div class="mb-6 flex gap-4">
						{#if deadCodeResult.summary.by_confidence['high']}
							<span class="px-3 py-1 rounded-full text-sm font-medium bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400">
								{deadCodeResult.summary.by_confidence['high']} High confidence
							</span>
						{/if}
						{#if deadCodeResult.summary.by_confidence['medium']}
							<span class="px-3 py-1 rounded-full text-sm font-medium bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400">
								{deadCodeResult.summary.by_confidence['medium']} Medium
							</span>
						{/if}
						{#if deadCodeResult.summary.by_confidence['low']}
							<span class="px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
								{deadCodeResult.summary.by_confidence['low']} Low
							</span>
						{/if}
					</div>

					<!-- Dead Symbols List -->
					{#if deadCodeResult.dead_symbols.length > 0}
						<div class="mb-6">
							<h4 class="text-sm font-semibold mb-3">{m.gitlab_dead_symbols_list?.() || 'Unused Symbols'}</h4>
							<div class="space-y-2 max-h-80 overflow-y-auto">
								{#each deadCodeResult.dead_symbols as symbol}
									<div class={cn(
										'p-3 rounded-lg border-l-4',
										symbol.confidence === 'high' ? 'border-red-500 bg-red-50 dark:bg-red-900/10' :
										symbol.confidence === 'medium' ? 'border-yellow-500 bg-yellow-50 dark:bg-yellow-900/10' :
										'border-blue-500 bg-blue-50 dark:bg-blue-900/10'
									)}>
										<div class="flex items-center gap-2">
											<span class="w-6 h-6 flex items-center justify-center rounded bg-muted font-mono text-xs">
												{getSymbolTypeIcon(symbol.type)}
											</span>
											<span class="font-mono font-medium">{symbol.name}</span>
											<span class={cn('text-xs px-2 py-0.5 rounded-full', getConfidenceColor(symbol.confidence))}>
												{symbol.confidence}
											</span>
											<span class="text-xs text-muted-foreground capitalize">{symbol.type}</span>
										</div>
										<div class="mt-1 text-sm text-muted-foreground">
											<span class="font-mono text-xs">{symbol.file_path}</span>
										</div>
										<div class="mt-1 text-sm">{symbol.reason}</div>
									</div>
								{/each}
							</div>
						</div>
					{:else}
						<div class="text-center py-8 text-muted-foreground">
							<FontAwesomeIcon icon={faCircleCheck} class="h-12 w-12 mx-auto text-green-500 mb-3" />
							<p class="font-medium">{m.gitlab_no_dead_code?.() || 'No dead code detected!'}</p>
							<p class="text-sm">{m.gitlab_code_clean?.() || 'Your codebase looks clean.'}</p>
						</div>
					{/if}

					<!-- Top affected files -->
					{#if deadCodeResult.summary.top_affected_files.length > 0}
						<div class="mb-4">
							<h4 class="text-sm font-semibold mb-2">{m.gitlab_top_affected?.() || 'Most Affected Files'}</h4>
							<div class="space-y-1">
								{#each deadCodeResult.summary.top_affected_files as file}
									<div class="flex items-center justify-between text-sm p-2 rounded bg-muted">
										<span class="font-mono truncate flex-1">{file.file_path}</span>
										<span class="text-muted-foreground">{file.dead_symbols} symbols, {file.dead_lines} lines</span>
									</div>
								{/each}
							</div>
						</div>
					{/if}

					<!-- Model info -->
					<div class="mt-4 text-sm text-muted-foreground flex items-center gap-2">
						<FontAwesomeIcon icon={faRobot} class="h-4 w-4" />
						<span>{m.gitlab_analyzed_by?.() || 'Analyzed by'}: <strong>{deadCodeResult.model_id}</strong></span>
						<span class="mx-2">•</span>
						<FontAwesomeIcon icon={faClock} class="h-4 w-4" />
						<span>{deadCodeResult.duration}</span>
					</div>
				{/if}
			{/if}

			<div class="mt-6 flex justify-end gap-3">
				{#if deadCodeResult && !isDetectingDeadCode && (deadCodeResult.summary?.total_dead_symbols || deadCodeResult.dead_symbols?.length || 0) > 0}
					<Button 
						variant="default" 
						onclick={handleCreateDeadCodeIssue}
						disabled={isCreatingDeadCodeIssue}
					>
						{#if isCreatingDeadCodeIssue}
							<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
							{m.common_creating?.() || 'Creating...'}
						{:else}
							<FontAwesomeIcon icon={faCircleExclamation} class="mr-2 h-4 w-4" />
							{m.gitlab_create_issue?.() || 'Create Issue'}
						{/if}
					</Button>
				{/if}
				<Button variant="outline" onclick={() => (showDeadCodeModal = false)}>
					{m.common_close()}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Auto-Documentation Modal -->
{#if showAutoDocModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		role="dialog"
		aria-modal="true"
		onclick={(e) => e.target === e.currentTarget && (showAutoDocModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showAutoDocModal = false)}
		tabindex="-1"
	>
		<div class="max-h-[90vh] w-full max-w-5xl overflow-auto rounded-lg bg-background p-6 shadow-lg">
			<div class="mb-4 flex items-center justify-between">
				<div class="flex items-center gap-3">
					<FontAwesomeIcon icon={faFilePen} class="h-6 w-6 text-purple-500" />
					<h3 class="text-lg font-semibold">
						{m.gitlab_auto_doc_title?.() || 'Auto-Documentation'}
					</h3>
				</div>
				<button onclick={() => (showAutoDocModal = false)} class="rounded p-1 hover:bg-muted">
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			{#if autoDocStep === 'scan'}
				<div class="flex flex-col items-center justify-center py-12">
					<FontAwesomeIcon icon={faSpinner} class="h-12 w-12 animate-spin text-purple-500" />
					<p class="mt-4 text-muted-foreground">{m.gitlab_scanning_docs?.() || 'Scanning for undocumented code...'}</p>
				</div>
			{:else if autoDocStep === 'generate'}
				<div class="flex flex-col items-center justify-center py-12">
					<FontAwesomeIcon icon={faSpinner} class="h-12 w-12 animate-spin text-purple-500" />
					<p class="mt-4 text-muted-foreground">{m.gitlab_generating_docs?.() || 'Generating documentation...'}</p>
					<p class="mt-2 text-sm text-muted-foreground">{docScanResult?.summary.total_symbols || 0} {m.gitlab_symbols_to_document?.() || 'symbols to document'}</p>
				</div>
			{:else if autoDocStep === 'results'}
				{#if docScanResult?.status === 'failed'}
					<div class="rounded-lg bg-red-100 dark:bg-red-900/30 p-4 text-red-700 dark:text-red-400">
						<p class="font-medium">{m.gitlab_scan_failed?.() || 'Scan Failed'}</p>
						<p class="text-sm">{docScanResult.error}</p>
					</div>
				{:else if docScanResult && docScanResult.symbols.length === 0}
					<div class="text-center py-8 text-muted-foreground">
						<FontAwesomeIcon icon={faCircleCheck} class="h-12 w-12 mx-auto text-green-500 mb-3" />
						<p class="font-medium">{m.gitlab_all_documented?.() || 'All code is documented!'}</p>
						<p class="text-sm">{m.gitlab_no_undocumented?.() || 'No undocumented exported symbols found.'}</p>
					</div>
				{:else if docGenResult}
					<!-- Summary -->
					<div class="mb-6 grid grid-cols-2 md:grid-cols-4 gap-4">
						<div class="rounded-lg border p-3 text-center">
							<div class="text-3xl font-bold text-purple-600">{docScanResult?.summary.total_symbols || 0}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_undocumented?.() || 'Undocumented'}</div>
						</div>
						<div class="rounded-lg border p-3 text-center">
							<div class="text-3xl font-bold text-green-600">{docGenResult.docs.length}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_docs_generated?.() || 'Docs Generated'}</div>
						</div>
						<div class="rounded-lg border p-3 text-center">
							<div class="text-3xl font-bold">{docScanResult?.files_scanned || 0}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_files_scanned?.() || 'Files'}</div>
						</div>
						<div class="rounded-lg border p-3 text-center">
							<div class="text-3xl font-bold">{docGenResult.tokens_used.toLocaleString()}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_tokens_used?.() || 'Tokens'}</div>
						</div>
					</div>

					<!-- Generated Docs -->
					<div class="mb-4">
						<h4 class="text-sm font-semibold mb-3">{m.gitlab_generated_docs?.() || 'Generated Documentation'}</h4>
						<div class="space-y-4 max-h-96 overflow-y-auto">
							{#each docGenResult.docs as doc}
								<div class="rounded-lg border p-4">
									<div class="flex items-center justify-between mb-2">
										<div class="flex items-center gap-2">
											<span class="w-6 h-6 flex items-center justify-center rounded bg-muted font-mono text-xs">
												{getSymbolTypeIcon(doc.symbol.type)}
											</span>
											<span class="font-mono font-medium">{doc.symbol.name}</span>
											<span class="text-xs px-2 py-0.5 rounded bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400">
												{doc.symbol.language}
											</span>
										</div>
										<button 
											onclick={() => navigator.clipboard.writeText(doc.documentation)}
											class="text-xs px-2 py-1 rounded bg-muted hover:bg-muted/80"
										>
											📋 Copy
										</button>
									</div>
									<p class="text-xs text-muted-foreground mb-2 font-mono">{doc.symbol.file_path}:{doc.symbol.start_line}</p>
									<pre class="text-sm bg-muted p-3 rounded overflow-x-auto"><code>{doc.preview}</code></pre>
								</div>
							{/each}
						</div>
					</div>

					<!-- Model info -->
					<div class="mt-4 text-sm text-muted-foreground flex items-center gap-2">
						<FontAwesomeIcon icon={faRobot} class="h-4 w-4" />
						<span>{m.gitlab_analyzed_by?.() || 'Generated by'}: <strong>{docGenResult.model_id}</strong></span>
						<span class="mx-2">•</span>
						<FontAwesomeIcon icon={faClock} class="h-4 w-4" />
						<span>{docGenResult.duration}</span>
					</div>
				{/if}
			{/if}

			<div class="mt-6 flex justify-end gap-3">
				{#if docGenResult && docGenResult.docs.length > 0 && !isGeneratingDocs}
					<Button 
						variant="default" 
						onclick={handleCreateDocsMR}
						disabled={isCreatingDocsMR}
					>
						{#if isCreatingDocsMR}
							<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
							{m.common_creating?.() || 'Creating...'}
						{:else}
							<FontAwesomeIcon icon={faCodeBranch} class="mr-2 h-4 w-4" />
							{m.gitlab_create_mr?.() || 'Create MR'}
						{/if}
					</Button>
				{/if}
				<Button variant="outline" onclick={() => (showAutoDocModal = false)}>
					{m.common_close()}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Test Generation Modal -->
{#if showTestGenModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		role="dialog"
		aria-modal="true"
		onclick={(e) => e.target === e.currentTarget && (showTestGenModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showTestGenModal = false)}
		tabindex="-1"
	>
		<div class="max-h-[90vh] w-full max-w-5xl overflow-auto rounded-lg bg-background p-6 shadow-lg">
			<div class="mb-4 flex items-center justify-between">
				<div class="flex items-center gap-3">
					<FontAwesomeIcon icon={faFlask} class="h-6 w-6 text-green-500" />
					<h3 class="text-lg font-semibold">
						{m.gitlab_test_gen_title?.() || 'Test Generation'}
					</h3>
				</div>
				<button onclick={() => (showTestGenModal = false)} class="rounded p-1 hover:bg-muted">
					<FontAwesomeIcon icon={faXmark} class="h-5 w-5" />
				</button>
			</div>

			{#if testGenStep === 'scan'}
				<div class="flex flex-col items-center justify-center py-12">
					<FontAwesomeIcon icon={faSpinner} class="h-12 w-12 animate-spin text-green-500" />
					<p class="mt-4 text-muted-foreground">{m.gitlab_scanning_tests?.() || 'Scanning for testable functions...'}</p>
				</div>
			{:else if testGenStep === 'generate'}
				<div class="flex flex-col items-center justify-center py-12">
					<FontAwesomeIcon icon={faSpinner} class="h-12 w-12 animate-spin text-green-500" />
					<p class="mt-4 text-muted-foreground">{m.gitlab_generating_tests?.() || 'Generating tests...'}</p>
					<p class="mt-2 text-sm text-muted-foreground">{testScanResult?.summary.without_tests || 0} {m.gitlab_functions_to_test?.() || 'functions to test'}</p>
				</div>
			{:else if testGenStep === 'results'}
				{#if testScanResult?.status === 'failed'}
					<div class="rounded-lg bg-red-100 dark:bg-red-900/30 p-4 text-red-700 dark:text-red-400">
						<p class="font-medium">{m.gitlab_scan_failed?.() || 'Scan Failed'}</p>
						<p class="text-sm">{testScanResult.error}</p>
					</div>
				{:else if testScanResult && testScanResult.summary.without_tests === 0}
					<div class="text-center py-8 text-muted-foreground">
						<FontAwesomeIcon icon={faCircleCheck} class="h-12 w-12 mx-auto text-green-500 mb-3" />
						<p class="font-medium">{m.gitlab_all_tested?.() || 'All functions have tests!'}</p>
						<p class="text-sm">{m.gitlab_good_coverage?.() || 'Great test coverage.'}</p>
					</div>
				{:else if testGenResult}
					<!-- Summary -->
					<div class="mb-6 grid grid-cols-2 md:grid-cols-4 gap-4">
						<div class="rounded-lg border p-3 text-center">
							<div class="text-3xl font-bold">{testScanResult?.summary.total_functions || 0}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_total_functions?.() || 'Total Functions'}</div>
						</div>
						<div class="rounded-lg border p-3 text-center">
							<div class="text-3xl font-bold text-red-600">{testScanResult?.summary.without_tests || 0}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_without_tests?.() || 'Without Tests'}</div>
						</div>
						<div class="rounded-lg border p-3 text-center">
							<div class="text-3xl font-bold text-green-600">{testGenResult.tests.length}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_tests_generated?.() || 'Tests Generated'}</div>
						</div>
						<div class="rounded-lg border p-3 text-center">
							<div class="text-3xl font-bold">{testGenResult.tokens_used.toLocaleString()}</div>
							<div class="text-sm text-muted-foreground">{m.gitlab_tokens_used?.() || 'Tokens'}</div>
						</div>
					</div>

					<!-- Generated Tests -->
					<div class="mb-4">
						<h4 class="text-sm font-semibold mb-3">{m.gitlab_generated_tests?.() || 'Generated Tests'}</h4>
						<div class="space-y-4 max-h-96 overflow-y-auto">
							{#each testGenResult.tests as test}
								<div class="rounded-lg border p-4">
									<div class="flex items-center justify-between mb-2">
										<div class="flex items-center gap-2">
											<span class="font-mono font-medium">{test.test_name}</span>
											<span class="text-xs px-2 py-0.5 rounded bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
												{test.framework}
											</span>
											<span class="text-xs text-muted-foreground">→ {test.function.name}</span>
										</div>
										<button 
											onclick={() => navigator.clipboard.writeText(test.test_code)}
											class="text-xs px-2 py-1 rounded bg-muted hover:bg-muted/80"
										>
											📋 Copy
										</button>
									</div>
									<p class="text-xs text-muted-foreground mb-2 font-mono">{test.function.file_path}:{test.function.start_line}</p>
									<pre class="text-sm bg-muted p-3 rounded overflow-x-auto max-h-48"><code>{test.test_code}</code></pre>
								</div>
							{/each}
						</div>
					</div>

					<!-- Model info -->
					<div class="mt-4 text-sm text-muted-foreground flex items-center gap-2">
						<FontAwesomeIcon icon={faRobot} class="h-4 w-4" />
						<span>{m.gitlab_analyzed_by?.() || 'Generated by'}: <strong>{testGenResult.model_id}</strong></span>
						<span class="mx-2">•</span>
						<FontAwesomeIcon icon={faClock} class="h-4 w-4" />
						<span>{testGenResult.duration}</span>
					</div>
				{/if}
			{/if}

			<div class="mt-6 flex justify-end gap-3">
				{#if testGenResult && testGenResult.tests.length > 0 && !isGeneratingTests}
					<Button 
						variant="default" 
						onclick={handleCreateTestsMR}
						disabled={isCreatingTestsMR}
					>
						{#if isCreatingTestsMR}
							<FontAwesomeIcon icon={faSpinner} class="mr-2 h-4 w-4 animate-spin" />
							{m.common_creating?.() || 'Creating...'}
						{:else}
							<FontAwesomeIcon icon={faCodeBranch} class="mr-2 h-4 w-4" />
							{m.gitlab_create_mr?.() || 'Create MR'}
						{/if}
					</Button>
				{/if}
				<Button variant="outline" onclick={() => (showTestGenModal = false)}>
					{m.common_close()}
				</Button>
			</div>
		</div>
	</div>
{/if}

