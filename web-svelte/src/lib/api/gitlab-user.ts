// GitLab User API Client
// For regular users to manage their own integrations

import { api } from './client';

// Types
export interface GitLabIntegration {
  id: string;
  owner_id: string;
  tenant_id?: string;
  name: string;
  base_url: string;
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
  max_review_tokens?: number;
  per_file_review?: boolean;
  review_language?: string;
  target_branches?: string[];
  ignore_branches?: string[];
  collection_name?: string;
}

export interface GitLabReview {
  id: string;
  project_id: string;
  integration_id: string;
  mr_iid: number;
  mr_title: string;
  mr_author: string;
  source_branch: string;
  target_branch: string;
  mr_url: string;
  status: 'pending' | 'queued' | 'processing' | 'completed' | 'failed' | 'cancelled';
  priority: 'low' | 'normal' | 'high' | 'urgent';
  created_at: string;
  updated_at?: string;
}

// ============================================================================
// Integration Management
// ============================================================================

export async function listMyIntegrations(): Promise<{ data: GitLabIntegration[]; total: number }> {
  return api.get('/api/gitlab/integrations');
}

export async function createMyIntegration(data: {
  name: string;
  base_url: string;
  access_token: string;
  webhook_secret?: string;
  settings?: GitLabIntegrationSettings;
}): Promise<GitLabIntegration> {
  return api.post('/api/gitlab/integrations', data);
}

export async function getMyIntegration(id: string): Promise<GitLabIntegration> {
  return api.get(`/api/gitlab/integrations/${id}`);
}

export async function updateMyIntegration(id: string, data: {
  name?: string;
  base_url?: string;
  access_token?: string;
  webhook_secret?: string;
  status?: 'active' | 'disabled';
  settings?: GitLabIntegrationSettings;
}): Promise<GitLabIntegration> {
  return api.put(`/api/gitlab/integrations/${id}`, data);
}

export async function deleteMyIntegration(id: string): Promise<void> {
  return api.delete(`/api/gitlab/integrations/${id}`);
}

// ============================================================================
// Project Management
// ============================================================================

export async function listMyProjects(integrationId: string): Promise<{ data: GitLabProject[]; total: number }> {
  return api.get(`/api/gitlab/integrations/${integrationId}/projects`);
}

export async function addMyProject(integrationId: string, data: {
  gitlab_project_id: number;
  name: string;
  path_with_namespace?: string;
  analysis_model_id: string;
  embedding_model_id?: string;
  settings?: GitLabProjectSettings;
}): Promise<GitLabProject> {
  return api.post(`/api/gitlab/integrations/${integrationId}/projects`, data);
}

export async function getMyProject(projectId: string): Promise<GitLabProject> {
  return api.get(`/api/gitlab/projects/${projectId}`);
}

export async function updateMyProject(projectId: string, data: {
  name?: string;
  analysis_model_id?: string;
  embedding_model_id?: string;
  auto_review?: boolean;
  status?: 'active' | 'disabled';
  settings?: GitLabProjectSettings;
}): Promise<GitLabProject> {
  return api.put(`/api/gitlab/projects/${projectId}`, data);
}

export async function deleteMyProject(projectId: string): Promise<void> {
  return api.delete(`/api/gitlab/projects/${projectId}`);
}

// ============================================================================
// Review History
// ============================================================================

export async function listMyReviews(): Promise<{ data: GitLabReview[]; total: number }> {
  return api.get('/api/gitlab/reviews');
}

export async function getMyReview(reviewId: string): Promise<GitLabReview> {
  return api.get(`/api/gitlab/reviews/${reviewId}`);
}

// ============================================================================
// Project Indexing (RAG)
// ============================================================================

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

export async function startMyProjectIndexing(
  projectId: string,
  options?: { branch?: string; force?: boolean }
): Promise<{ message: string; project_id: string; branch: string; status: string }> {
  return api.post(`/api/gitlab/projects/${projectId}/index`, options || {});
}

export async function getMyProjectIndexStatus(
  projectId: string,
  branch?: string
): Promise<GitLabIndexStatus> {
  const params = branch ? `?branch=${encodeURIComponent(branch)}` : '';
  return api.get(`/api/gitlab/projects/${projectId}/index/status${params}`);
}

export async function deleteMyProjectIndex(
  projectId: string,
  branch?: string
): Promise<{ message: string }> {
  const params = branch ? `?branch=${encodeURIComponent(branch)}` : '';
  return api.delete(`/api/gitlab/projects/${projectId}/index${params}`);
}

// ============================================================================
// Scanning & Analysis (v4.2+)
// ============================================================================

// Security Scans
export async function scanMySecrets(projectId: string): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/scan-secrets`, {});
}

export async function deepScanMySecrets(projectId: string): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/deep-scan-secrets`, {});
}

export async function sastScanMySecrets(projectId: string): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/sast-scan`, {});
}

export async function createMySecretsIssue(projectId: string, data: any): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/create-secrets-issue`, data);
}

// Quality Analysis
export async function analyzeMyQuality(projectId: string): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/quality-score`, {});
}

export async function detectMyDuplication(projectId: string): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/detect-duplication`, {});
}

export async function createMyQualityIssue(projectId: string, data: any): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/create-quality-issue`, data);
}

// Dependency Analysis
export async function checkMyDependencies(projectId: string): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/check-dependencies`, {});
}

export async function createMyDependencyIssue(projectId: string, data: any): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/create-dependency-issue`, data);
}

// Dead Code Analysis
export async function detectMyDeadCode(projectId: string): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/dead-code`, {});
}

export async function detectMyUnreachable(projectId: string): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/detect-unreachable`, {});
}

export async function detectMyCommentedCode(projectId: string): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/detect-commented-code`, {});
}

export async function createMyDeadCodeIssue(projectId: string, data: any): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/create-dead-code-issue`, data);
}

// Documentation
export async function scanMyUndocumented(projectId: string): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/scan-docs`, {});
}

export async function generateMyDocs(projectId: string, data: any): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/generate-docs`, data);
}

export async function createMyDocsMR(projectId: string, data: any): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/create-docs-mr`, data);
}

// Test Generation
export async function scanMyTestable(projectId: string): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/scan-tests`, {});
}

export async function generateMyTests(projectId: string, data: any): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/generate-tests`, data);
}

export async function createMyTestsMR(projectId: string, data: any): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/create-tests-mr`, data);
}

// Webhooks
export async function setupMyWebhook(projectId: string, url: string): Promise<{ message: string; webhook_id: number }> {
  return api.post(`/api/gitlab/projects/${projectId}/setup-webhook`, { webhook_url: url });
}

// Changelog Analysis
export async function analyzeMyChangelog(projectId: string, data: any): Promise<any> {
  return api.post(`/api/gitlab/projects/${projectId}/analyze-changelog`, data);
}
