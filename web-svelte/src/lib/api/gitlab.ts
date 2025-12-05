// GitLab Integration API Client

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
  webhook_id?: number;
  status: 'active' | 'disabled' | 'error';
  auto_review: boolean;
  analysis_model_id: string;
  embedding_model_id: string;
  review_prompt?: string;
  settings: GitLabProjectSettings;
  created_at: string;
  updated_at: string;
  integration_name?: string;
  review_count?: number;
}

export interface GitLabProjectSettings {
  include_patterns?: string[];
  exclude_patterns?: string[];
  max_files_per_mr?: number;
  max_lines_per_file?: number;
  skip_draft_mrs?: boolean;
  skip_bots?: boolean;
  chunk_size?: number;
  chunk_overlap?: number;
  target_branches?: string[];
  ignore_branches?: string[];
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
  options: RequestInit = {}
): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response.json();
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
      body: JSON.stringify(data),
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
      body: JSON.stringify(data),
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
      body: JSON.stringify(data),
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
      body: JSON.stringify(data),
    });
  },

  async deleteProject(projectId: string): Promise<void> {
    await apiRequest(`/projects/${projectId}`, { method: 'DELETE' });
  },

  async setupWebhook(projectId: string, webhookUrl: string): Promise<{ webhook_id: number }> {
    return apiRequest(`/projects/${projectId}/webhook`, {
      method: 'POST',
      body: JSON.stringify({ webhook_url: webhookUrl }),
    });
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

export default gitlabApi;

