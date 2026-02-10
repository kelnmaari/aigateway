// GitLab Integration API Client

import { api } from './client';

const API_BASE = '/api/admin/gitlab';

// Types
export interface GitLabIntegration {
  id: string;
  name: string;
  base_url: string;
  access_token: string; // Masked
  webhook_secret: string;
  status: 'active' | 'disabled' | 'error';
  last_sync_at?: string;
  last_error?: string;
  settings: GitLabIntegrationSettings;
  created_at: string;
  updated_at: string;
  project_count?: number;
}

export interface GitLabIntegrationSettings {
  api_version?: string;
  request_timeout?: number;
  max_retries?: number;
  rate_limit_per_min?: number;
  webhooks_paused?: boolean;
}

export interface GitLabProject {
  id: string;
  integration_id: string;
  gitlab_project_id: number;
  name: string;
  path_with_namespace: string;
  default_branch?: string;
  webhook_id?: number;
  status: 'active' | 'disabled' | 'error';
  auto_review: boolean;
  analysis_model_id: string;
  embedding_model_id: string;
  review_prompt?: string;
  settings: GitLabProjectSettings;
  index_status?: 'pending' | 'in_progress' | 'completed' | 'failed';
  last_indexed_at?: string;
  created_at: string;
  updated_at: string;
  integration_name?: string;
  review_count?: number;
  // Index statistics (computed from Qdrant)
  index_chunks?: number;
  index_vectors?: number;
}

export interface GitLabIndexStatus {
  project_id: string;
  branch: string;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
  files_indexed: number;
  chunks_total: number;
  last_indexed?: string;
  error?: string;
  started_at?: string;
  completed_at?: string;
}

export interface GitLabProjectSettings {
  include_patterns?: string[];
  exclude_patterns?: string[];
  max_files_per_mr?: number;
  max_lines_per_file?: number;
  skip_draft_mrs?: boolean;
  skip_bots?: boolean;
  max_review_tokens?: number; // Max tokens for LLM review response
  per_file_review?: boolean;  // Review each file separately with tool calling
  review_language?: string;   // Language for review output: "en", "ru"
  chunk_size?: number;
  chunk_overlap?: number;
  target_branches?: string[];
  ignore_branches?: string[];
  collection_name?: string; // Qdrant collection name
}

export interface GitLabReview {
  id: string;
  project_id: string;
  integration_id: string;
  mr_iid: number;
  mr_title: string;
  mr_author: string;
  mr_author_id: number;
  source_branch: string;
  target_branch: string;
  mr_url: string;
  status: 'pending' | 'queued' | 'analyzing' | 'completed' | 'failed' | 'cancelled' | 'skipped';
  priority: 'low' | 'normal' | 'high' | 'urgent';
  files_analyzed: number;
  lines_changed: number;
  issues_found: number;
  review_result?: GitLabReviewResult;
  note_id?: number;
  discussion_id?: string;
  processing_time_ms: number;
  tokens_used: number;
  model_used: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  error?: string;
  retry_count: number;
  project_name?: string;
}

export interface GitLabReviewResult {
  summary: string;
  overall_score: number;
  categories: GitLabReviewCategory[];
  file_reviews?: GitLabFileReview[];
  suggestions?: GitLabSuggestion[];
}

export interface GitLabReviewCategory {
  name: string;
  score: number;
  issue_count: number;
  description?: string;
}

export interface GitLabFileReview {
  file_path: string;
  language?: string;
  lines_added: number;
  lines_removed: number;
  issues?: GitLabCodeIssue[];
  approved: boolean;
}

export interface GitLabCodeIssue {
  line: number;
  end_line?: number;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
  category: string;
  message: string;
  suggestion?: string;
  code_snippet?: string;
}

export interface GitLabSuggestion {
  type: string;
  title: string;
  description: string;
  priority: string;
}

export interface GitLabQueueStats {
  pending: number;
  processing: number;
  completed: number;
  failed: number;
  cancelled: number;
  active_workers: number;
  idle_workers: number;
  total_workers: number;
  avg_processing_time_ms: number;
  jobs_last_hour: number;
  jobs_last_24_hours: number;
  completed_today: number;
}

export interface GitLabJob {
  id: string;
  review_id: string;
  project_id: string;
  integration_id: string;
  mr_iid: number;
  mr_title: string;
  status: 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled' | 'retrying';
  priority: string;
  worker_id?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  retry_count: number;
  max_retries: number;
  last_error?: string;
  next_retry_at?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  pagination: {
    limit: number;
    offset: number;
    total: number;
  };
}

export interface TestConnectionResult {
  success: boolean;
  error?: string;
  user?: {
    id: number;
    username: string;
    name: string;
    email: string;
    is_admin: boolean;
  };
}

async function apiRequest<T>(
  path: string,
  options: { method?: string; body?: unknown; query?: Record<string, unknown> } = {}
): Promise<T> {
  let url = `${API_BASE}${path}`;

  // Add query parameters if provided
  if (options.query) {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(options.query)) {
      if (value !== undefined && value !== null) {
        params.append(key, String(value));
      }
    }
    const queryString = params.toString();
    if (queryString) {
      url += `?${queryString}`;
    }
  }

  return api.request<T>(url, {
    method: (options.method || 'GET') as 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE',
    body: options.body,
  });
}

// Integration API
export const gitlabApi = {
  // Integrations
  async listIntegrations(params?: {
    limit?: number;
    offset?: number;
    status?: string;
    search?: string;
  }): Promise<PaginatedResponse<GitLabIntegration>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());
    if (params?.status) searchParams.set('status', params.status);
    if (params?.search) searchParams.set('search', params.search);

    return apiRequest(`/integrations?${searchParams}`);
  },

  async getIntegration(id: string): Promise<GitLabIntegration> {
    return apiRequest(`/integrations/${id}`);
  },

  async createIntegration(data: {
    name: string;
    base_url: string;
    access_token: string;
    webhook_secret?: string;
  }): Promise<GitLabIntegration> {
    return apiRequest('/integrations', {
      method: 'POST',
      body: data,
    });
  },

  async updateIntegration(id: string, data: {
    name?: string;
    base_url?: string;
    access_token?: string;
    status?: string;
    settings?: GitLabIntegrationSettings;
  }): Promise<GitLabIntegration> {
    return apiRequest(`/integrations/${id}`, {
      method: 'PUT',
      body: data,
    });
  },

  async deleteIntegration(id: string): Promise<void> {
    await apiRequest(`/integrations/${id}`, { method: 'DELETE' });
  },

  async testIntegration(id: string): Promise<TestConnectionResult> {
    return apiRequest(`/integrations/${id}/test`, { method: 'POST' });
  },

  // Projects
  async listProjects(integrationId: string, params?: {
    limit?: number;
    offset?: number;
    status?: string;
    search?: string;
  }): Promise<PaginatedResponse<GitLabProject>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());
    if (params?.status) searchParams.set('status', params.status);
    if (params?.search) searchParams.set('search', params.search);

    return apiRequest(`/integrations/${integrationId}/projects?${searchParams}`);
  },

  async getProject(projectId: string): Promise<GitLabProject> {
    return apiRequest(`/projects/${projectId}`);
  },

  async addProject(integrationId: string, data: {
    gitlab_project_id: number;
    analysis_model_id: string;
    embedding_model_id?: string;
    auto_review?: boolean;
    review_prompt?: string;
    settings?: GitLabProjectSettings;
  }): Promise<GitLabProject> {
    return apiRequest(`/integrations/${integrationId}/projects`, {
      method: 'POST',
      body: data,
    });
  },

  async bulkAddProjects(integrationId: string, data: {
    projects: Array<{
      gitlab_project_id: number;
      name: string;
      path_with_namespace?: string;
      default_branch?: string;
    }>;
    analysis_model_id: string;
    embedding_model_id?: string;
    auto_review?: boolean;
    review_prompt?: string;
    settings?: GitLabProjectSettings;
  }): Promise<{ data: GitLabProject[]; count: number }> {
    return apiRequest(`/integrations/${integrationId}/projects/bulk`, {
      method: 'POST',
      body: data,
    });
  },

  async discoverProjects(integrationId: string, params?: {
    search?: string;
    page?: number;
    per_page?: number;
  }): Promise<{ data: any[] }> {
    return apiRequest(`/integrations/${integrationId}/discover`, {
      query: params,
    });
  },

  async updateProject(projectId: string, data: {
    auto_review?: boolean;
    status?: string;
    default_branch?: string;
    analysis_model_id?: string;
    embedding_model_id?: string;
    review_prompt?: string;
    settings?: GitLabProjectSettings;
  }): Promise<GitLabProject> {
    return apiRequest(`/projects/${projectId}`, {
      method: 'PUT',
      body: data,
    });
  },

  async deleteProject(projectId: string): Promise<void> {
    await apiRequest(`/projects/${projectId}`, { method: 'DELETE' });
  },

  async setupWebhook(projectId: string, webhookUrl: string): Promise<{ webhook_id: number }> {
    return apiRequest(`/projects/${projectId}/webhook`, {
      method: 'POST',
      body: { webhook_url: webhookUrl },
    });
  },

  // Indexing
  async startIndexing(projectId: string, options?: { branch?: string; force?: boolean }): Promise<{
    message: string;
    project_id: string;
    branch: string;
    status: string
  }> {
    return apiRequest(`/projects/${projectId}/index`, {
      method: 'POST',
      body: options || {},
    });
  },

  async getIndexStatus(projectId: string, branch?: string): Promise<GitLabIndexStatus> {
    const params = branch ? `?branch=${encodeURIComponent(branch)}` : '';
    return apiRequest(`/projects/${projectId}/index/status${params}`);
  },

  async deleteIndex(projectId: string, branch?: string): Promise<{ message: string }> {
    const params = branch ? `?branch=${encodeURIComponent(branch)}` : '';
    return apiRequest(`/projects/${projectId}/index${params}`, { method: 'DELETE' });
  },

  // Reviews
  async listReviews(params?: {
    limit?: number;
    offset?: number;
    project_id?: string;
    status?: string;
    search?: string;
  }): Promise<PaginatedResponse<GitLabReview>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());
    if (params?.project_id) searchParams.set('project_id', params.project_id);
    if (params?.status) searchParams.set('status', params.status);
    if (params?.search) searchParams.set('search', params.search);

    return apiRequest(`/reviews?${searchParams}`);
  },

  async getReview(id: string): Promise<GitLabReview> {
    return apiRequest(`/reviews/${id}`);
  },

  async retryReview(id: string): Promise<{ job_id: string }> {
    return apiRequest(`/reviews/${id}/retry`, { method: 'POST' });
  },

  // Queue
  async getQueueStatus(): Promise<GitLabQueueStats> {
    return apiRequest('/queue/status');
  },

  async listJobs(params?: {
    limit?: number;
    offset?: number;
    status?: string;
  }): Promise<PaginatedResponse<GitLabJob>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());
    if (params?.status) searchParams.set('status', params.status);

    return apiRequest(`/queue/jobs?${searchParams}`);
  },

  async cancelJob(id: string): Promise<void> {
    await apiRequest(`/queue/jobs/${id}/cancel`, { method: 'POST' });
  },

  async retryJob(id: string): Promise<void> {
    await apiRequest(`/queue/jobs/${id}/retry`, { method: 'POST' });
  },

  // Model Selection
  async listActiveModels(capability?: string): Promise<{ models: ModelOption[]; total: number }> {
    const params = new URLSearchParams();
    if (capability) params.set('capability', capability);
    return apiRequest(`/models?${params}`);
  },

  async listAnalysisModels(): Promise<{ models: ModelOption[]; total: number }> {
    return apiRequest('/models/analysis');
  },

  async listEmbeddingModels(): Promise<{ models: ModelOption[]; total: number }> {
    return apiRequest('/models/embedding');
  },

  async listAvailableModels(type?: 'analysis' | 'embedding'): Promise<{ data: ModelOption[] }> {
    return apiRequest('/models/available', {
      query: type ? { type } : {},
    });
  },

  // Settings
  async getSettings(): Promise<GitLabSettings> {
    return apiRequest('/settings');
  },

  async updateSettings(settings: Partial<GitLabSettings>): Promise<{ message: string }> {
    return apiRequest('/settings', { method: 'PUT', body: settings });
  },

  // Analytics
  async getAnalytics(range?: string): Promise<GitLabAnalytics> {
    const params = new URLSearchParams();
    if (range) params.set('range', range);
    return apiRequest(`/analytics?${params}`);
  },

  // Feedback
  async listFeedback(params?: { limit?: number; offset?: number }): Promise<PaginatedResponse<GitLabFeedback>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());
    return apiRequest(`/feedback?${searchParams}`);
  },

  async submitFeedback(data: { review_id: string; rating: number; comment?: string }): Promise<{ message: string }> {
    return apiRequest('/feedback', { method: 'POST', body: data });
  },

  // Available Projects (from GitLab API)
  async listAvailableProjects(integrationId: string, params?: {
    search?: string;
    page?: number;
    per_page?: number;
  }): Promise<AvailableProjectsResponse> {
    const searchParams = new URLSearchParams();
    if (params?.search) searchParams.set('search', params.search);
    if (params?.page) searchParams.set('page', params.page.toString());
    if (params?.per_page) searchParams.set('per_page', params.per_page.toString());
    return apiRequest(`/integrations/${integrationId}/available-projects?${searchParams}`);
  },

  // ============================================================================
  // Secrets Scanning (v4.0+)
  // ============================================================================

  async scanSecrets(projectId: string, params?: {
    categories?: string[];
    min_severity?: SecretSeverity;
  }): Promise<SecretsScanResult> {
    return apiRequest(`/projects/${projectId}/scan-secrets`, {
      method: 'POST',
      body: params || {},
    });
  },

  async deepScanSecrets(projectId: string, params?: {
    model_id?: string;
    max_chunks?: number;
    language?: string;
  }): Promise<DeepScanResult> {
    return apiRequest(`/projects/${projectId}/deep-scan-secrets`, {
      method: 'POST',
      body: params || {},
    });
  },

  async getSecretsPatterns(): Promise<SecretsPatternsResponse> {
    return apiRequest('/secrets/patterns');
  },

  async sastScan(projectId: string, params?: {
    types?: SASTVulnerabilityType[];
    min_severity?: SecretSeverity;
    language?: string;
  }): Promise<SASTScanResult> {
    return apiRequest(`/projects/${projectId}/sast-scan`, {
      method: 'POST',
      body: params || {},
    });
  },

  // ============================================================================
  // Dependency Scanning (v4.1+)
  // ============================================================================

  async checkDependencies(projectId: string): Promise<MultiEcosystemDependencyScanResult> {
    return apiRequest(`/projects/${projectId}/check-dependencies`, {
      method: 'POST',
    });
  },

  async createDependencyIssue(projectId: string, params: {
    title: string;
    description?: string;
    labels?: string[];
    critical?: boolean;
  }): Promise<{ message: string; url: string }> {
    return apiRequest(`/projects/${projectId}/create-dependency-issue`, {
      method: 'POST',
      body: params,
    });
  },

  async createSecretsIssue(projectId: string, params: {
    title: string;
    description?: string;
    labels?: string[];
  }): Promise<{ message: string; url: string }> {
    return apiRequest(`/projects/${projectId}/secrets/create-issue`, {
      method: 'POST',
      body: params,
    });
  },

  async createQualityIssue(projectId: string, params: {
    title: string;
    description?: string;
    labels?: string[];
  }): Promise<{ message: string; url: string }> {
    return apiRequest(`/projects/${projectId}/quality/create-issue`, {
      method: 'POST',
      body: params,
    });
  },

  async createDeadCodeIssue(projectId: string, params: {
    title: string;
    description?: string;
    labels?: string[];
  }): Promise<{ message: string; url: string }> {
    return apiRequest(`/projects/${projectId}/dead-code-issue`, {
      method: 'POST',
      body: params,
    });
  },

  // ============================================================================
  // Changelog Analysis (v4.2+)
  // ============================================================================

  async analyzeChangelog(projectId: string, params: AnalyzeChangelogRequest): Promise<ChangelogAnalysis> {
    return apiRequest(`/projects/${projectId}/analyze-changelog`, {
      method: 'POST',
      body: params,
    });
  },

  async analyzeChangelogs(projectId: string, params: AnalyzeChangelogsRequest): Promise<AnalyzeChangelogsResponse> {
    return apiRequest(`/projects/${projectId}/analyze-changelogs`, {
      method: 'POST',
      body: params,
    });
  },

  // ============================================================================
  // Code Quality (v4.1+)
  // ============================================================================

  async analyzeQuality(projectId: string, params?: {
    max_files?: number;
    language?: string;
  }): Promise<QualityScore> {
    return apiRequest(`/projects/${projectId}/quality-score`, {
      method: 'POST',
      body: params || {},
    });
  },

  async detectDeadCode(projectId: string, params?: {
    max_chunks?: number;
    language?: string;
  }): Promise<DeadCodeResult> {
    return apiRequest(`/projects/${projectId}/dead-code`, {
      method: 'POST',
      body: params || {},
    });
  },

  // ============================================================================
  // Auto-Documentation (v4.1+)
  // ============================================================================

  async scanDocs(projectId: string, params?: {
    max_files?: number;
    language?: string;
    exported_only?: boolean;
  }): Promise<DocScanResult> {
    return apiRequest(`/projects/${projectId}/scan-docs`, {
      method: 'POST',
      body: params || {},
    });
  },

  async generateDocs(projectId: string, params?: {
    symbol_ids?: string[];
    max_symbols?: number;
    language?: string;
  }): Promise<DocGenerationResult> {
    return apiRequest(`/projects/${projectId}/generate-docs`, {
      method: 'POST',
      body: params || {},
    });
  },

  async bulkApplyDocs(projectId: string, params: BulkApplyDocsRequest): Promise<BulkApplyResult> {
    return apiRequest(`/projects/${projectId}/bulk-apply-docs`, {
      method: 'POST',
      body: params,
    });
  },

  async createDocsMR(projectId: string, params: CreateDocsMRRequest): Promise<CreateDocsMRResult> {
    return apiRequest(`/projects/${projectId}/create-docs-mr`, {
      method: 'POST',
      body: params,
    });
  },

  // ============================================================================
  // Test Generation (v4.1+)
  // ============================================================================

  async scanTests(projectId: string, params?: {
    max_files?: number;
    language?: string;
  }): Promise<TestScanResult> {
    return apiRequest(`/projects/${projectId}/scan-tests`, {
      method: 'POST',
      body: params || {},
    });
  },

  async generateTests(projectId: string, params?: {
    max_functions?: number;
    language?: string;
    framework?: string;
  }): Promise<TestGenerationResult> {
    return apiRequest(`/projects/${projectId}/generate-tests`, {
      method: 'POST',
      body: params || {},
    });
  },

  async downloadTests(projectId: string, params: DownloadTestsRequest): Promise<Blob> {
    const token = typeof window !== 'undefined' ? localStorage.getItem('auth_token') : null;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`/api/admin/gitlab/projects/${projectId}/download-tests`, {
      method: 'POST',
      headers,
      body: JSON.stringify(params),
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Download failed' }));
      throw new Error(error.error || 'Download failed');
    }

    return response.blob();
  },

  async createTestsMR(projectId: string, params: CreateTestsMRRequest): Promise<CreateTestsMRResult> {
    return apiRequest(`/projects/${projectId}/create-tests-mr`, {
      method: 'POST',
      body: params,
    });
  },

  // ============================================================================
  // Architecture Diagrams (v4.2+)
  // ============================================================================

  async scanArchitecture(projectId: string, params?: {
    max_files?: number;
    language?: string;
  }): Promise<ArchitectureScanResult> {
    return apiRequest(`/projects/${projectId}/scan-architecture`, {
      method: 'POST',
      body: params || {},
    });
  },

  async generateDiagram(projectId: string, params?: {
    type?: DiagramType;
    format?: DiagramFormat;
    scope?: string;
    max_depth?: number;
  }): Promise<DiagramGenerationResult> {
    return apiRequest(`/projects/${projectId}/generate-diagram`, {
      method: 'POST',
      body: params || {},
    });
  },

  async getArchitecture(projectId: string, type?: DiagramType): Promise<DiagramGenerationResult> {
    const query = type ? `?type=${type}` : '';
    return apiRequest(`/projects/${projectId}/architecture${query}`);
  },

  // ============================================================================
  // Scheduled Scans API (v4.1.0+)
  // ============================================================================

  async listSchedules(params?: {
    project_id?: string;
    integration_id?: string;
    scan_type?: ScanType;
    enabled?: boolean;
    limit?: number;
    offset?: number;
  }): Promise<{ schedules: ScheduledScan[]; total: number }> {
    const query = new URLSearchParams();
    if (params?.project_id) query.set('project_id', params.project_id);
    if (params?.integration_id) query.set('integration_id', params.integration_id);
    if (params?.scan_type) query.set('scan_type', params.scan_type);
    if (params?.enabled !== undefined) query.set('enabled', String(params.enabled));
    if (params?.limit) query.set('limit', String(params.limit));
    if (params?.offset) query.set('offset', String(params.offset));
    const queryStr = query.toString();
    return apiRequest(`/schedules${queryStr ? '?' + queryStr : ''}`);
  },

  async getSchedule(scheduleId: string): Promise<ScheduledScan> {
    return apiRequest(`/schedules/${scheduleId}`);
  },

  async createSchedule(request: CreateScheduledScanRequest): Promise<ScheduledScan> {
    return apiRequest('/schedules', {
      method: 'POST',
      body: request,
    });
  },

  async updateSchedule(scheduleId: string, request: UpdateScheduledScanRequest): Promise<ScheduledScan> {
    return apiRequest(`/schedules/${scheduleId}`, {
      method: 'PUT',
      body: request,
    });
  },

  async deleteSchedule(scheduleId: string): Promise<void> {
    return apiRequest(`/schedules/${scheduleId}`, {
      method: 'DELETE',
    });
  },

  async triggerSchedule(scheduleId: string): Promise<{ message: string; history: ScanHistory }> {
    return apiRequest(`/schedules/${scheduleId}/trigger`, {
      method: 'POST',
    });
  },

  async getScheduleHistory(scheduleId: string, params?: {
    limit?: number;
    offset?: number;
  }): Promise<{ history: ScanHistory[]; total: number }> {
    const query = new URLSearchParams();
    if (params?.limit) query.set('limit', String(params.limit));
    if (params?.offset) query.set('offset', String(params.offset));
    const queryStr = query.toString();
    return apiRequest(`/schedules/${scheduleId}/history${queryStr ? '?' + queryStr : ''}`);
  },

  async getSchedulerStatus(): Promise<SchedulerStatus> {
    return apiRequest('/schedules/status');
  },

  // ============================================================================
  // Analytics Dashboard API (v4.2+)
  // ============================================================================

  async getAnalyticsDashboard(params?: {
    start_date?: string;
    end_date?: string;
    integration_id?: string;
    project_id?: string;
  }): Promise<AnalyticsDashboard> {
    return apiRequest('/analytics/dashboard', { query: params });
  },

  async getModelComparison(params?: {
    start_date?: string;
    end_date?: string;
  }): Promise<ModelComparison> {
    return apiRequest('/analytics/models', { query: params });
  },

  async getSecurityOverview(): Promise<SecurityScoreStats> {
    return apiRequest('/analytics/security');
  },

  async getDependencyHealth(): Promise<DependencyHealthStats> {
    return apiRequest('/analytics/dependencies');
  },

  async getProjectAnalytics(projectId: string, params?: {
    start_date?: string;
    end_date?: string;
  }): Promise<ProjectAnalytics> {
    return apiRequest(`/projects/${projectId}/analytics`, { query: params });
  },

  async exportAnalyticsReport(params?: {
    format?: 'json' | 'csv';
    start_date?: string;
    end_date?: string;
  }): Promise<Blob> {
    const token = typeof window !== 'undefined' ? localStorage.getItem('auth_token') : null;
    const headers: Record<string, string> = {};
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const queryStr = params ? `?${new URLSearchParams(params as Record<string, string>).toString()}` : '';
    const response = await fetch(`/api/admin/gitlab/analytics/export${queryStr}`, { headers });

    if (!response.ok) {
      throw new Error('Export failed');
    }

    return response.blob();
  },

  // ============================================================================
  // Code Duplication Detection API (v4.1.0+)
  // ============================================================================

  async detectDuplication(projectId: string, params?: {
    max_files?: number;
    language?: string;
  }): Promise<DuplicationResult> {
    return apiRequest(`/projects/${projectId}/detect-duplication`, {
      method: 'POST',
      body: params || {},
    });
  },

  // ============================================================================
  // Scan History API (v4.1.2+)
  // ============================================================================

  async listScanHistory(params?: {
    project_id?: string;
    scan_type?: string;
    status?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ results: ScanHistoryItem[]; total: number; limit: number; offset: number }> {
    const query = new URLSearchParams();
    if (params?.project_id) query.set('project_id', params.project_id);
    if (params?.scan_type) query.set('scan_type', params.scan_type);
    if (params?.status) query.set('status', params.status);
    if (params?.limit) query.set('limit', String(params.limit));
    if (params?.offset) query.set('offset', String(params.offset));
    const queryStr = query.toString();
    return apiRequest(`/scan-history${queryStr ? '?' + queryStr : ''}`);
  },

  async getScanHistoryItem(id: string): Promise<ScanHistoryItem> {
    return apiRequest(`/scan-history/${id}`);
  },

  async getScanTypes(): Promise<{ types: ScanTypeInfo[] }> {
    return apiRequest('/scan-history/types');
  },
};

// Model types
export interface ModelOption {
  id: string;
  model_id: string;
  name: string;
  provider_id: string;
  capabilities?: string[];
  description?: string;
}

export interface ModelUsage {
  model_id: string;
  is_used_in_gitlab: boolean;
  gitlab_projects: GitLabProjectRef[];
  can_deactivate: boolean;
  blocking_reason?: string;
}

export interface GitLabProjectRef {
  integration_id: string;
  integration_name: string;
  project_id: string;
  project_name: string;
  usage_type: 'analysis' | 'embedding';
}

// Settings types
export interface GitLabSettings {
  auto_review_enabled: boolean;
  default_analysis_model: string;
  default_embedding_model: string;
  max_files_per_mr: number;
  max_lines_per_file: number;
  webhook_secret_rotation: boolean;
  notification_email: string;
}

// Analytics types
export interface GitLabAnalytics {
  range: string;
  days: number;
  total_reviews: number;
  completed_reviews: number;
  failed_reviews: number;
  avg_processing_ms: number;
  total_issues_found: number;
  reviews_by_day: Array<{ date: string; count: number }>;
  reviews_by_project: Array<{ project: string; count: number }>;
  issue_categories: Array<{ category: string; count: number }>;
}

// Feedback types
export interface GitLabFeedback {
  id: string;
  review_id: string;
  rating: number;
  comment?: string;
  created_at: string;
}

// Available projects from GitLab
export interface AvailableProject {
  id: number;
  name: string;
  path_with_namespace: string;
  description?: string;
  web_url: string;
  default_branch: string;
  visibility: string;
  already_added: boolean;
}

export interface AvailableProjectsResponse {
  projects: AvailableProject[];
  total: number;
  page: number;
  per_page: number;
}

// ============================================================================
// Secrets Scanning Types (v4.0+)
// ============================================================================

export type SecretSeverity = 'critical' | 'high' | 'medium' | 'low' | 'info';

export interface SecretFinding {
  id: string;
  pattern_id: string;
  pattern_name: string;
  category: string;
  severity: SecretSeverity;
  file_path: string;
  start_line: number;
  end_line: number;
  match: string; // Redacted
  context: string; // Redacted surrounding code
  suggestion: string;
}

export interface SecretsScanSummary {
  total_findings: number;
  by_severity: Record<string, number>;
  by_category: Record<string, number>;
  files_affected: number;
}

export interface SecretsScanResult {
  project_id: string;
  scan_id: string;
  started_at: string;
  completed_at: string;
  duration: string;
  chunks_scanned: number;
  findings: SecretFinding[];
  summary: SecretsScanSummary;
  status: 'running' | 'completed' | 'failed';
  error?: string;
}

export interface SecretsPattern {
  id: string;
  name: string;
  description: string;
  severity: SecretSeverity;
  category: string;
}

export interface SecretsPatternsResponse {
  patterns: SecretsPattern[];
  total: number;
}

// ============================================================================
// Dependency Scanning Types
// ============================================================================

export interface DependencyInfo {
  name: string;
  current_version: string;
  latest_version: string;
  indirect: boolean;
  update_type: 'major' | 'minor' | 'patch' | 'none' | 'unknown';
  has_update: boolean;
}

export interface DependencyVulnerability {
  id: string;
  cve_id?: string;
  title?: string;
  summary: string;
  details: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  fixed_in: string;
  references: string[];
  published_at: string;
}

export interface DependencyWithVulns {
  dependency: DependencyInfo;
  vulnerabilities: DependencyVulnerability[];
  is_vulnerable: boolean;
}

export interface DependencyScanSummary {
  total_dependencies: number;
  direct_dependencies: number;
  outdated_count: number;
  vulnerable_count: number;
  up_to_date_count: number;
  by_update_type: Record<string, number>;
  by_severity: Record<string, number>;
  critical_vulns: number;
}

export interface DependencyScanResult {
  project_id: string;
  scan_id: string;
  language: string;
  file_path: string;
  scanned_at: string;
  duration: string;
  dependencies: DependencyWithVulns[];
  summary: DependencyScanSummary;
  status: 'completed' | 'failed';
  error?: string;
}

// Multi-ecosystem scan result (monorepo support)
export interface MultiEcosystemDependencyScanResult {
  project_id: string;
  scan_id: string;
  scanned_at: string;
  duration: string;
  ecosystems: DependencyScanResult[]; // Results per ecosystem (Go, Node.js, Python)
  total_summary: DependencyScanSummary;
  status: 'completed' | 'failed';
  error?: string;
}

// ============================================================================
// Changelog Analysis Types (v4.2+)
// ============================================================================

export interface BreakingChange {
  description: string;
  affected_area: string; // e.g., "API", "Config", "Behavior"
  severity: 'high' | 'medium' | 'low';
  workaround?: string;
}

export interface ChangelogAnalysis {
  package_name: string;
  current_version: string;
  latest_version: string;
  language: string;
  summary: string;
  breaking_changes: BreakingChange[];
  new_features: string[];
  bug_fixes: string[];
  security_fixes: string[];
  deprecated_features: string[];
  migration_guide: string;
  risk_level: 'low' | 'medium' | 'high' | 'critical';
  confidence: 'high' | 'medium' | 'low';
  tokens_used: number;
  analyzed_at: string;
}

export interface AnalyzeChangelogRequest {
  package_name: string;
  current_version: string;
  latest_version: string;
  language: string; // "go", "nodejs", "python"
  model_id?: string;
}

export interface AnalyzeChangelogsRequest {
  dependencies: Array<{
    name: string;
    current_version: string;
    latest_version: string;
  }>;
  language: string;
  model_id?: string;
}

export interface AnalyzeChangelogsResponse {
  analyses: ChangelogAnalysis[];
  total: number;
}

// ============================================================================
// Code Quality Types
// ============================================================================

export type QualityCategory = 'complexity' | 'documentation' | 'testing' | 'security' | 'maintainability' | 'naming' | 'error_handling';

export interface QualityScore {
  project_id: string;
  scan_id: string;
  scanned_at: string;
  duration: string;
  overall_score: number;
  breakdown: Record<QualityCategory, number>;
  file_scores: FileScore[];
  recommendations: QualityRecommendation[];
  summary: QualitySummary;
  status: 'completed' | 'failed';
  error?: string;
  model_id: string;
  tokens_used: number;
}

export interface FileScore {
  file_path: string;
  language: string;
  lines_of_code: number;
  score: number;
  breakdown: Record<QualityCategory, number>;
  issues: QualityIssue[];
  functions_count: number;
}

export interface QualityIssue {
  category: QualityCategory;
  severity: 'high' | 'medium' | 'low';
  file_path: string;
  line: number;
  message: string;
  suggestion?: string;
  code_snippet?: string;
}

export interface QualityRecommendation {
  category: QualityCategory;
  priority: 'high' | 'medium' | 'low';
  title: string;
  description: string;
  file_paths?: string[];
  impact: string;
}

export interface QualitySummary {
  total_files: number;
  total_lines_of_code: number;
  total_functions: number;
  issues_count: number;
  high_severity_count: number;
  medium_severity_count: number;
  low_severity_count: number;
  top_issue_categories: CategoryStat[];
  best_scoring_files: string[];
  worst_scoring_files: string[];
}

export interface CategoryStat {
  category: QualityCategory;
  count: number;
  avg_score: number;
}

// ============================================================================
// Dead Code Types
// ============================================================================

export type SymbolType = 'function' | 'type' | 'variable' | 'constant' | 'interface' | 'class' | 'method';
export type Confidence = 'high' | 'medium' | 'low';

export interface DeadSymbol {
  name: string;
  type: SymbolType;
  file_path: string;
  start_line: number;
  end_line: number;
  confidence: Confidence;
  reason: string;
  lines_of_code: number;
  exportable: boolean;
}

export interface DeadCodeResult {
  project_id: string;
  scan_id: string;
  scanned_at: string;
  duration: string;
  status: 'completed' | 'failed';
  error?: string;
  dead_symbols: DeadSymbol[];
  summary: DeadCodeSummary;
  model_id: string;
  tokens_used: number;
  files_scanned: number;
  chunks_scanned: number;
}

export interface DeadCodeSummary {
  total_dead_symbols: number;
  by_type: Record<SymbolType, number>;
  by_confidence: Record<Confidence, number>;
  estimated_dead_lines: number;
  top_affected_files: FileStats[];
}

export interface FileStats {
  file_path: string;
  dead_symbols: number;
  dead_lines: number;
}

// ============================================================================
// Auto-Documentation Types
// ============================================================================

export interface UndocumentedSymbol {
  name: string;
  type: string;
  file_path: string;
  start_line: number;
  end_line: number;
  language: string;
  signature: string;
  is_exported: boolean;
  importance: 'high' | 'medium' | 'low';
}

export interface DocScanResult {
  project_id: string;
  scan_id: string;
  scanned_at: string;
  duration: string;
  status: 'completed' | 'failed';
  error?: string;
  symbols: UndocumentedSymbol[];
  summary: DocScanSummary;
  files_scanned: number;
}

export interface DocScanSummary {
  total_symbols: number;
  exported_count: number;
  by_type: Record<string, number>;
  by_language: Record<string, number>;
  by_importance: Record<string, number>;
  top_affected_files: DocFileStats[];
}

export interface DocFileStats {
  file_path: string;
  count: number;
  exported: number;
}

export interface GeneratedDoc {
  symbol: UndocumentedSymbol;
  documentation: string;
  format: string;
  language: string;
  preview: string;
}

export interface DocGenerationResult {
  project_id: string;
  generated_at: string;
  duration: string;
  status: 'completed' | 'failed';
  error?: string;
  docs: GeneratedDoc[];
  tokens_used: number;
  model_id: string;
}

// Bulk Apply + MR Creation Types
export interface BulkApplyDocsRequest {
  docs: GeneratedDoc[];
  create_mr?: boolean;
  mr_title?: string;
  mr_description?: string;
  target_branch?: string;
  labels?: string[];
}

export interface BulkApplyResult {
  project_id: string;
  applied_at: string;
  duration: string;
  status: 'completed' | 'failed';
  error?: string;
  applied_count: number;
  failed_count: number;
  applied_files: AppliedFile[];
  mr?: MergeRequestInfo;
}

export interface AppliedFile {
  file_path: string;
  symbols_added: number;
  lines_added: number;
  status: 'success' | 'failed';
  error?: string;
}

export interface MergeRequestInfo {
  id?: number;
  iid?: number;
  url?: string;
  title: string;
  description: string;
  source_branch: string;
  target_branch: string;
  labels: string[];
  commit_message?: string;
  file_changes?: FileChange[];
  status?: string;
  message?: string;
}

export interface FileChange {
  file_path: string;
  symbol_name: string;
  symbol_type: string;
  line_number: number;
  documentation: string;
  language: string;
}

export interface CreateDocsMRRequest {
  docs: GeneratedDoc[];
  title?: string;
  description?: string;
  target_branch?: string;
  source_branch?: string;
  labels?: string[];
  commit_message?: string;
}

export interface CreateDocsMRResult {
  status: string;
  message: string;
  mr: MergeRequestInfo;
  mr_url?: string;
  docs_count: number;
  files_affected: number;
}

export interface CreateTestsMRRequest {
  tests: GeneratedTest[];
  title?: string;
  description?: string;
  target_branch?: string;
  labels?: string[];
  commit_message?: string;
  test_dir?: string;
}

export interface CreateTestsMRResult {
  status: string;
  message: string;
  mr_id: number;
  mr_iid: number;
  mr_url: string;
  tests_count: number;
  files_created: number;
}

// ============================================================================
// Test Generation Types
// ============================================================================

export interface TestableFunction {
  name: string;
  file_path: string;
  start_line: number;
  end_line: number;
  language: string;
  signature: string;
  code: string;
  is_exported: boolean;
  has_tests: boolean;
  complexity: 'low' | 'medium' | 'high';
}

export interface TestScanResult {
  project_id: string;
  scan_id: string;
  scanned_at: string;
  duration: string;
  status: 'completed' | 'failed';
  error?: string;
  functions: TestableFunction[];
  summary: TestScanSummary;
  files_scanned: number;
}

export interface TestScanSummary {
  total_functions: number;
  without_tests: number;
  with_tests: number;
  by_language: Record<string, number>;
  by_complexity: Record<string, number>;
  top_files: TestFileStats[];
}

export interface TestFileStats {
  file_path: string;
  function_count: number;
  untested: number;
}

export interface GeneratedTest {
  function: TestableFunction;
  test_code: string;
  test_name: string;
  framework: string;
  language: string;
  description: string;
}

export interface TestGenerationResult {
  project_id: string;
  generated_at: string;
  duration: string;
  status: 'completed' | 'failed';
  error?: string;
  tests: GeneratedTest[];
  tokens_used: number;
  model_id: string;
}

export interface DownloadTestsRequest {
  tests: GeneratedTest[];
  format?: 'single' | 'zip';
  filename?: string;
}

// ============================================================================
// Architecture Diagram Types
// ============================================================================

export type DiagramType = 'module_dependency' | 'call_graph' | 'data_flow' | 'package_structure';
export type DiagramFormat = 'mermaid' | 'svg' | 'png' | 'd3_json';

export interface ArchitectureNode {
  id: string;
  name: string;
  type: string; // "package", "module", "function", "class", "file"
  file_path?: string;
  language?: string;
  description?: string;
  metadata?: Record<string, string>;
}

export interface ArchitectureEdge {
  source: string;
  target: string;
  label?: string;
  type: string; // "dependency", "call", "inheritance", "composition"
  weight?: number;
  metadata?: Record<string, string>;
}

export interface ArchitectureGraph {
  nodes: ArchitectureNode[];
  edges: ArchitectureEdge[];
}

export interface ArchitectureScanSummary {
  total_nodes: number;
  total_edges: number;
  by_node_type: Record<string, number>;
  by_edge_type: Record<string, number>;
  by_language: Record<string, number>;
  module_count: number;
  function_count: number;
  package_count: number;
  circular_deps?: string[][];
}

export interface ArchitectureScanResult {
  project_id: string;
  scan_id: string;
  scanned_at: string;
  duration: string;
  status: 'completed' | 'failed';
  error?: string;
  graph: ArchitectureGraph;
  summary: ArchitectureScanSummary;
  languages: string[];
}

export interface Diagram {
  id: string;
  project_id: string;
  type: DiagramType;
  format: DiagramFormat;
  title: string;
  description?: string;
  content: string; // Mermaid code or SVG/PNG data
  graph?: ArchitectureGraph;
  created_at: string;
  duration: string;
  tokens_used: number;
  model_id?: string;
}

export interface DiagramGenerationResult {
  project_id: string;
  generated_at: string;
  duration: string;
  status: 'completed' | 'failed';
  error?: string;
  diagrams: Diagram[];
  tokens_used: number;
  model_id?: string;
}

// Deep scan (LLM-based) types
export interface DeepFinding {
  id: string;
  type: string; // "hardcoded_secret", "sensitive_data", "security_issue"
  severity: SecretSeverity;
  file_path: string;
  start_line: number;
  end_line: number;
  description: string;
  code_snippet: string;
  suggestion: string;
  confidence: 'high' | 'medium' | 'low';
}

export interface DeepScanResult {
  project_id: string;
  scan_id: string;
  model_id: string;
  started_at: string;
  completed_at: string;
  duration: string;
  chunks_scanned: number;
  tokens_used: number;
  findings: DeepFinding[];
  summary: SecretsScanSummary;
  status: 'running' | 'completed' | 'failed' | 'cancelled';
  error?: string;
}

// ============================================================================
// SAST (Static Application Security Testing) Types
// ============================================================================

export type SASTVulnerabilityType =
  | 'sql_injection'
  | 'xss'
  | 'path_traversal'
  | 'command_injection'
  | 'hardcoded_ip'
  | 'hardcoded_url'
  | 'insecure_crypto'
  | 'insecure_random'
  | 'open_redirect'
  | 'ssrf'
  | 'xxe'
  | 'insecure_deserialization'
  | 'hardcoded_credentials';

export interface SASTFinding {
  id: string;
  pattern_id: string;
  pattern_name: string;
  type: SASTVulnerabilityType;
  severity: SecretSeverity;
  cwe?: string;
  owasp?: string;
  file_path: string;
  start_line: number;
  end_line: number;
  match: string;
  context: string;
  suggestion: string;
  language: string;
}

export interface SASTScanSummary {
  total_findings: number;
  by_severity: Record<string, number>;
  by_type: Record<string, number>;
  by_cwe: Record<string, number>;
  files_affected: number;
  critical_count: number;
  high_count: number;
  medium_count: number;
  low_count: number;
}

export interface SASTScanResult {
  project_id: string;
  scan_id: string;
  started_at: string;
  completed_at: string;
  duration: string;
  chunks_scanned: number;
  findings: SASTFinding[];
  summary: SASTScanSummary;
  status: 'completed' | 'failed';
  error?: string;
}

// ============================================================================
// Scheduled Scans Types (v4.1.0+)
// ============================================================================

export type ScheduleFrequency = 'daily' | 'weekly' | 'monthly' | 'custom';
export type ScanType = 'dependencies' | 'secrets' | 'quality' | 'dead_code';

export interface ScheduledScan {
  id: string;
  project_id: string;
  integration_id: string;
  scan_type: ScanType;
  frequency: ScheduleFrequency;
  cron_expr?: string;
  enabled: boolean;
  notify_email?: string;
  create_issue: boolean;
  only_breaking: boolean;
  last_run_at?: string;
  next_run_at?: string;
  last_run_status?: 'success' | 'failed' | 'running';
  last_run_error?: string;
  last_run_duration_ms?: number;
  total_runs: number;
  successful_runs: number;
  failed_runs: number;
  created_at: string;
  updated_at: string;
  project_name?: string;
  integration_name?: string;
}

export interface ScanHistory {
  id: string;
  schedule_id: string;
  project_id: string;
  scan_type: ScanType;
  status: 'pending' | 'running' | 'success' | 'failed';
  started_at: string;
  completed_at?: string;
  duration_ms?: number;
  error?: string;
  dependencies_checked?: number;
  outdated_dependencies?: number;
  vulnerabilities_found?: number;
  breaking_changes?: number;
  issue_created: boolean;
  issue_url?: string;
  results_json?: string;
}

// ============================================================================
// Scan History Types (v4.1.2+) - unified scan results storage
// ============================================================================

export type ScanHistoryScanType =
  | 'secrets'
  | 'secrets_deep'
  | 'dependencies'
  | 'quality'
  | 'deadcode'
  | 'autodocs'
  | 'testgen'
  | 'architecture';

export type ScanHistoryStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

export interface ScanHistoryItem {
  id: string;
  project_id: string;
  integration_id?: string;
  scan_type: ScanHistoryScanType;
  status: ScanHistoryStatus;
  model_id?: string;
  started_at: string;
  completed_at?: string;
  duration_ms: number;
  findings_count: number;
  files_affected: number;
  tokens_used: number;
  error?: string;
  results_json?: string;
  project_name?: string;
}

export interface ScanTypeInfo {
  value: string;
  label: string;
  description: string;
}

export interface CreateScheduledScanRequest {
  project_id: string;
  scan_type: ScanType;
  frequency: ScheduleFrequency;
  cron_expr?: string;
  enabled: boolean;
  notify_email?: string;
  create_issue?: boolean;
  only_breaking?: boolean;
}

export interface UpdateScheduledScanRequest {
  frequency?: ScheduleFrequency;
  cron_expr?: string;
  enabled?: boolean;
  notify_email?: string;
  create_issue?: boolean;
  only_breaking?: boolean;
}

export interface SchedulerStatus {
  running: boolean;
  schedule_count: number;
}

// ============================================================================
// Analytics Dashboard Types (v4.2+)
// ============================================================================

export interface AnalyticsDashboard {
  overview: ReviewStats;
  top_projects: ProjectReviewStats[];
  top_models: ModelReviewStats[];
  review_trend: TrendPoint[];
  issues_by_category: CategoryStats[];
  issues_by_severity: SeverityStats[];
  recent_activity: ActivityItem[];
  dependency_health?: DependencyHealthStats;
  security_score?: SecurityScoreStats;
  team_productivity?: TeamProductivityStats;
}

export interface ReviewStats {
  total_reviews: number;
  completed_reviews: number;
  failed_reviews: number;
  pending_reviews: number;
  total_files_reviewed: number;
  total_lines_changed: number;
  total_issues_found: number;
  total_tokens_used: number;
  avg_score: number;
  avg_processing_time_ms: number;
  success_rate: number;
}

export interface ProjectReviewStats {
  project_id: string;
  project_name: string;
  total_reviews: number;
  completed_reviews: number;
  total_issues_found: number;
  avg_score: number;
  avg_processing_time_ms: number;
}

export interface ModelReviewStats {
  model_id: string;
  model_name: string;
  total_reviews: number;
  total_tokens_used: number;
  avg_tokens_per_review: number;
  avg_processing_time_ms: number;
  avg_score: number;
}

export interface TrendPoint {
  timestamp: string;
  value: number;
  count: number;
}

export interface CategoryStats {
  category: string;
  count: number;
  percentage: number;
}

export interface SeverityStats {
  severity: string;
  count: number;
  percentage: number;
}

export interface ActivityItem {
  id: string;
  type: string;
  project_name: string;
  mr_title: string;
  mr_iid: number;
  timestamp: string;
  details?: string;
}

export interface DependencyHealthStats {
  total_dependencies: number;
  outdated_count: number;
  outdated_percentage: number;
  vulnerable_count: number;
  critical_vulns: number;
  high_vulns: number;
  medium_vulns: number;
  last_scan_at?: string;
}

export interface SecurityScoreStats {
  overall_score: number;
  secrets_scan_score: number;
  sast_score: number;
  dependency_score: number;
  total_findings: number;
  critical_findings: number;
  high_findings: number;
  medium_findings: number;
  low_findings: number;
  score_trend?: TrendPoint[];
}

export interface TeamProductivityStats {
  total_mrs_reviewed: number;
  avg_review_time_minutes: number;
  issues_found_per_mr: number;
  auto_fix_applied: number;
  times_saved_hours: number;
  top_reviewers?: UserReviewStats[];
}

export interface UserReviewStats {
  user_id: string;
  username: string;
  total_mrs: number;
  total_issues_found: number;
  avg_issues_per_mr: number;
  avg_score: number;
}

export interface ModelComparison {
  models: ModelReviewStats[];
  best_quality?: ModelReviewStats;
  fastest?: ModelReviewStats;
  most_efficient?: ModelReviewStats;
}

export interface ProjectAnalytics {
  stats: ReviewStats;
  processing_time: ProcessingTimeStats;
  token_usage: TokenUsageStats;
  user_breakdown: UserReviewStats[];
  issue_trend: TrendPoint[];
}

export interface ProcessingTimeStats {
  min_ms: number;
  max_ms: number;
  avg_ms: number;
  median_ms: number;
  p95_ms: number;
  p99_ms: number;
}

export interface TokenUsageStats {
  total_tokens: number;
  prompt_tokens: number;
  completion_tokens: number;
  avg_per_review: number;
  max_per_review: number;
}

// ============================================================================
// Code Duplication Types (v4.1.0+)
// ============================================================================

export interface DuplicateBlock {
  file_path: string;
  start_line: number;
  end_line: number;
  code_snippet: string;
}

export interface DuplicateGroup {
  id: string;
  occurrences: DuplicateBlock[];
  lines_count: number;
  similarity: number;
  description: string;
  suggestion: string;
  refactor_type: string;
}

export interface DuplicationSummary {
  total_files_analyzed: number;
  files_with_duplicates: number;
  total_duplicate_groups: number;
  total_duplicate_lines: number;
  duplication_percent: number;
  top_duplicated_files: string[];
}

export interface DuplicationResult {
  project_id: string;
  scan_id: string;
  scanned_at: string;
  duration: string;
  status: 'completed' | 'failed';
  error?: string;
  duplicates: DuplicateGroup[];
  summary: DuplicationSummary;
  model_id: string;
  tokens_used: number;
}

export default gitlabApi;

