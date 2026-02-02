// GitLab Jobs API Client
// For managing background jobs and real-time progress tracking

import { api } from './client';

// Types
export type JobStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
export type JobType =
    | 'secrets_scan'
    | 'deep_secrets_scan'
    | 'sast_scan'
    | 'dependency_scan'
    | 'quality_scan'
    | 'deadcode_scan'
    | 'autodocs_scan'
    | 'testgen_scan'
    | 'project_index'
    | 'generate_docs'
    | 'generate_tests'
    | 'create_mr';

export interface UserJob {
    id: string;
    user_id: string;
    tenant_id?: string;
    project_id: string;
    integration_id: string;
    job_type: JobType;
    status: JobStatus;
    config?: string;
    progress: number;
    progress_msg?: string;
    created_at: string;
    started_at?: string;
    completed_at?: string;
    result_id?: string;
    result_type?: string;
    result_url?: string;
    error?: string;
    project_name?: string;
}

export interface JobTypeInfo {
    type: JobType;
    name: string;
    description: string;
}

export interface JobStatusInfo {
    status: JobStatus;
    name: string;
    description: string;
}

export interface SubmitJobRequest {
    model_id?: string;
    language?: string;
    max_chunks?: number;
    categories?: string[];
    min_severity?: string;
}

export interface SubmitJobResponse {
    job_id: string;
    status: string;
    message?: string;
    project_id: string;
}

export interface JobProgressEvent {
    job_id: string;
    status: JobStatus;
    progress: number;
    progress_msg?: string;
    result_id?: string;
    error?: string;
}

// ============================================================================
// Job Listing & Management
// ============================================================================

export async function listMyJobs(params?: {
    project_id?: string;
    integration_id?: string;
    job_type?: JobType;
    status?: JobStatus;
    limit?: number;
    offset?: number;
}): Promise<{ jobs: UserJob[]; total: number; limit: number; offset: number }> {
    const query = new URLSearchParams();
    if (params?.project_id) query.append('project_id', params.project_id);
    if (params?.integration_id) query.append('integration_id', params.integration_id);
    if (params?.job_type) query.append('job_type', params.job_type);
    if (params?.status) query.append('status', params.status);
    if (params?.limit) query.append('limit', params.limit.toString());
    if (params?.offset) query.append('offset', params.offset.toString());

    const queryStr = query.toString();
    return api.get(`/api/gitlab/jobs${queryStr ? '?' + queryStr : ''}`);
}

export async function getJob(jobId: string): Promise<UserJob> {
    return api.get(`/api/gitlab/jobs/${jobId}`);
}

export async function getActiveJobs(): Promise<{ jobs: UserJob[]; count: number }> {
    return api.get('/api/gitlab/jobs/active');
}

export async function cancelJob(jobId: string): Promise<{ message: string }> {
    return api.post(`/api/gitlab/jobs/${jobId}/cancel`, {});
}

export async function getJobTypes(): Promise<{ types: JobTypeInfo[] }> {
    return api.get('/api/gitlab/jobs/types');
}

export async function getJobStatuses(): Promise<{ statuses: JobStatusInfo[] }> {
    return api.get('/api/gitlab/jobs/statuses');
}

// ============================================================================
// Job Submission (Background Processing)
// ============================================================================

export async function submitDeepScanJob(projectId: string, request?: SubmitJobRequest): Promise<SubmitJobResponse> {
    return api.post(`/api/gitlab/projects/${projectId}/jobs/deep-scan`, request || {});
}

export async function submitSecretsScanJob(projectId: string, request?: SubmitJobRequest): Promise<SubmitJobResponse> {
    return api.post(`/api/gitlab/projects/${projectId}/jobs/secrets-scan`, request || {});
}

export async function submitSASTJob(projectId: string, request?: SubmitJobRequest): Promise<SubmitJobResponse> {
    return api.post(`/api/gitlab/projects/${projectId}/jobs/sast-scan`, request || {});
}

export async function submitQualityScanJob(projectId: string, request?: SubmitJobRequest): Promise<SubmitJobResponse> {
    return api.post(`/api/gitlab/projects/${projectId}/jobs/quality-scan`, request || {});
}

export async function submitDependencyScanJob(projectId: string, request?: SubmitJobRequest): Promise<SubmitJobResponse> {
    return api.post(`/api/gitlab/projects/${projectId}/jobs/dependency-scan`, request || {});
}

export async function submitDeadCodeScanJob(projectId: string, request?: SubmitJobRequest): Promise<SubmitJobResponse> {
    return api.post(`/api/gitlab/projects/${projectId}/jobs/deadcode-scan`, request || {});
}

// ============================================================================
// SSE Streaming for Real-time Progress
// ============================================================================

export interface JobStreamCallbacks {
    onInit?: (job: UserJob) => void;
    onProgress?: (event: JobProgressEvent) => void;
    onComplete?: (job: UserJob) => void;
    onError?: (error: string) => void;
}

/**
 * Subscribe to real-time job progress updates via SSE
 * @param jobId Job ID to monitor
 * @param callbacks Event handlers
 * @returns Cleanup function to close the connection
 */
export function streamJobProgress(jobId: string, callbacks: JobStreamCallbacks): () => void {
    // Get auth token from localStorage
    const authData = localStorage.getItem('auth');
    let token = '';
    if (authData) {
        try {
            const parsed = JSON.parse(authData);
            token = parsed.token || '';
        } catch {
            // Ignore parse errors
        }
    }

    // Create EventSource with auth
    // Note: EventSource doesn't support headers, so we pass token as query param
    const url = `/api/gitlab/jobs/${jobId}/stream?token=${encodeURIComponent(token)}`;
    const eventSource = new EventSource(url);

    eventSource.addEventListener('init', (event) => {
        try {
            const job = JSON.parse(event.data) as UserJob;
            callbacks.onInit?.(job);
        } catch (e) {
            console.error('Failed to parse init event:', e);
        }
    });

    eventSource.addEventListener('progress', (event) => {
        try {
            const progress = JSON.parse(event.data) as JobProgressEvent;
            callbacks.onProgress?.(progress);
        } catch (e) {
            console.error('Failed to parse progress event:', e);
        }
    });

    eventSource.addEventListener('complete', (event) => {
        try {
            const job = JSON.parse(event.data) as UserJob;
            callbacks.onComplete?.(job);
        } catch (e) {
            console.error('Failed to parse complete event:', e);
        }
    });

    eventSource.addEventListener('done', () => {
        eventSource.close();
    });

    eventSource.addEventListener('error', (event) => {
        try {
            const data = JSON.parse((event as MessageEvent).data);
            callbacks.onError?.(data.error || 'Unknown error');
        } catch {
            callbacks.onError?.('Connection error');
        }
        eventSource.close();
    });

    eventSource.onerror = () => {
        callbacks.onError?.('Connection lost');
        eventSource.close();
    };

    // Return cleanup function
    return () => {
        eventSource.close();
    };
}

// ============================================================================
// Helper Functions
// ============================================================================

export function getJobTypeName(type: JobType): string {
    const names: Record<JobType, string> = {
        secrets_scan: 'Secrets Scan',
        deep_secrets_scan: 'Deep Secrets Scan',
        sast_scan: 'SAST Scan',
        dependency_scan: 'Dependency Scan',
        quality_scan: 'Quality Scan',
        deadcode_scan: 'Dead Code Scan',
        autodocs_scan: 'Auto-Docs Scan',
        testgen_scan: 'Test Generation',
        project_index: 'Project Index',
        generate_docs: 'Generate Docs',
        generate_tests: 'Generate Tests',
        create_mr: 'Create MR',
    };
    return names[type] || type;
}

export function getJobStatusColor(status: JobStatus): string {
    const colors: Record<JobStatus, string> = {
        pending: 'gray',
        running: 'blue',
        completed: 'green',
        failed: 'red',
        cancelled: 'orange',
    };
    return colors[status] || 'gray';
}

export function isJobActive(status: JobStatus): boolean {
    return status === 'pending' || status === 'running';
}
