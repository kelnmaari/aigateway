import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { authStore } from '$lib/stores/auth.svelte';

interface ApiError {
	error: string;
	code?: string;
	details?: unknown;
}

interface RequestOptions {
	method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
	body?: unknown;
	headers?: Record<string, string>;
	skipAuth?: boolean;
}

class ApiClient {
	private baseUrl = '';

	async request<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
		const { method = 'GET', body, headers = {}, skipAuth = false } = options;

		// Build headers
		const requestHeaders: Record<string, string> = {
			'Content-Type': 'application/json',
			Accept: 'application/json',
			...headers
		};

		// Add auth token (fallback to localStorage if store not initialized)
		if (!skipAuth) {
			const token = authStore.accessToken || (browser ? localStorage.getItem('access_token') : null);
			if (token) {
				requestHeaders['Authorization'] = `Bearer ${token}`;
			}
		}

		// Build request
		const requestInit: RequestInit = {
			method,
			headers: requestHeaders
		};

		if (body && method !== 'GET') {
			requestInit.body = JSON.stringify(body);
		}

		try {
			const response = await fetch(`${this.baseUrl}${endpoint}`, requestInit);

			// Handle 401 - redirect to login
			if (response.status === 401 && !skipAuth) {
				authStore.logout();
				if (browser) {
					goto('/login');
				}
				throw new Error('Unauthorized');
			}

			// Handle non-OK responses
			if (!response.ok) {
				const errorData: ApiError = await response.json().catch(() => ({
					error: `HTTP ${response.status}: ${response.statusText}`
				}));
				throw new Error(errorData.error || 'Request failed');
			}

			// Handle empty responses (204 No Content)
			if (response.status === 204) {
				return {} as T;
			}

			// Handle non-JSON responses
			const contentType = response.headers.get('Content-Type');
			if (!contentType || !contentType.includes('application/json')) {
				return {} as T;
			}

			// Handle empty body
			const text = await response.text();
			if (!text) {
				return {} as T;
			}

			return JSON.parse(text);
		} catch (error) {
			if (error instanceof Error) {
				throw error;
			}
			throw new Error('Network error');
		}
	}

	get<T>(endpoint: string, options?: Omit<RequestOptions, 'method' | 'body'>) {
		return this.request<T>(endpoint, { ...options, method: 'GET' });
	}

	post<T>(endpoint: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>) {
		return this.request<T>(endpoint, { ...options, method: 'POST', body });
	}

	put<T>(endpoint: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>) {
		return this.request<T>(endpoint, { ...options, method: 'PUT', body });
	}

	patch<T>(endpoint: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>) {
		return this.request<T>(endpoint, { ...options, method: 'PATCH', body });
	}

	delete<T>(endpoint: string, options?: Omit<RequestOptions, 'method' | 'body'>) {
		return this.request<T>(endpoint, { ...options, method: 'DELETE' });
	}
}

export const api = new ApiClient();

