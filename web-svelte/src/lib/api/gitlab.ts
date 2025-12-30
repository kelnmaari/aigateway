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
  options: { method?: string; body?: unknown } = {}
): Promise<T> {
  return api.request<T>(`${API_BASE}${path}`, {
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

  async updateProject(projectId: string, data: {
    auto_review?: boolean;
    status?: string;
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

  // Model Usage Check
  async checkModelUsage(modelId: string): Promise<ModelUsage> {
    return apiRequest(`/models/${modelId}/gitlab-usage`);
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

export default gitlabApi;

