import { api } from './client';

export interface FileItem {
	id: string;
	name: string;
	path: string;
	size: number;
	mime_type: string;
	is_directory: boolean;
	created_at: string;
	modified_at: string;
	owner_id?: string;
	owner_name?: string;
	metadata?: Record<string, unknown>;
}

export interface FilesResponse {
	files: FileItem[];
	total: number;
	path: string;
}

export interface UploadResponse {
	file: FileItem;
}

export const filesApi = {
	// List files
	list: (path = '/', page = 1, perPage = 50) => {
		const params = new URLSearchParams({
			path,
			page: String(page),
			per_page: String(perPage)
		});
		return api.get<FilesResponse>(`/api/files?${params}`);
	},

	// Get file info
	get: (id: string) => api.get<FileItem>(`/api/files/${id}`),

	// Upload file
	upload: async (file: File, path = '/') => {
		const formData = new FormData();
		formData.append('file', file);
		formData.append('path', path);

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

	// Create directory
	createDirectory: (name: string, path = '/') =>
		api.post<FileItem>('/api/files/directory', { name, path }),

	// Delete file/directory
	delete: (id: string) => api.delete(`/api/files/${id}`),

	// Download file
	download: (id: string) => {
		window.open(`/api/files/${id}/download`, '_blank');
	},

	// Rename
	rename: (id: string, newName: string) => api.put<FileItem>(`/api/files/${id}`, { name: newName })
};

