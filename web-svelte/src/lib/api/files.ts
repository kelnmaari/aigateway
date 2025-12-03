import { api } from './client';

export interface FileItem {
	id: string;
	name: string;
	filename: string;
	original_name: string;
	size: number;
	mime_type: string;
	extraction_status: string;
	created_at: string;
	updated_at: string;
	user_id?: string;
	metadata?: Record<string, unknown>;
}

export interface FilesResponse {
	files: FileItem[];
	total: number;
}

export interface UploadResponse {
	file: FileItem;
}

export interface ListFilesOptions {
	limit?: number;
	offset?: number;
	sort?: string;
	order?: 'asc' | 'desc';
	mimeType?: string;
	extractionStatus?: string;
}

export const filesApi = {
	// List files
	list: (options: ListFilesOptions = {}) => {
		const params = new URLSearchParams();
		params.set('limit', String(options.limit ?? 50));
		params.set('offset', String(options.offset ?? 0));
		params.set('sort', options.sort ?? 'created_at');
		params.set('order', options.order ?? 'desc');
		if (options.mimeType) params.set('mime_type', options.mimeType);
		if (options.extractionStatus) params.set('extraction_status', options.extractionStatus);
		return api.get<FilesResponse>(`/api/files?${params}`);
	},

	// Get file info
	get: (id: string) => api.get<FileItem>(`/api/files/${id}`),

	// Upload file
	upload: async (file: File) => {
		const formData = new FormData();
		formData.append('file', file);

		const response = await fetch('/api/files/upload', {
			method: 'POST',
			headers: {
				Authorization: `Bearer ${localStorage.getItem('access_token')}`
			},
			body: formData
		});

		if (!response.ok) {
			const error = await response.json();
			throw new Error(error.message || 'Upload failed');
		}

		return response.json() as Promise<UploadResponse>;
	},

	// Delete file
	delete: (id: string) => api.delete(`/api/files/${id}`),

	// Download file
	download: (id: string) => {
		window.open(`/api/files/${id}/download`, '_blank');
	},

	// Get file text content
	getText: (id: string) => api.get<{ text: string }>(`/api/files/${id}/text`),

	// Search files
	search: (query: string) => api.post<FilesResponse>('/api/files/search', { query })
};

