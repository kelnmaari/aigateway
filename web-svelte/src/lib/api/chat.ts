import { api } from './client';
import type { Conversation, Message, ChatParams } from '$lib/stores/chat.svelte';

export interface ConversationsResponse {
	conversations: Conversation[];
	total: number;
}

export interface MessagesResponse {
	messages: Message[];
}

export interface CreateConversationRequest {
	title?: string;
	model: string;
}

export interface ChatCompletionRequest {
	model: string;
	messages: Array<{
		role: 'user' | 'assistant' | 'system';
		content: string | Array<{ type: string; text?: string; image_url?: { url: string } }>;
	}>;
	stream?: boolean;
	temperature?: number;
	top_p?: number;
	max_tokens?: number;
	num_ctx?: number;
	use_tools?: boolean; // Enable web search and other tools
}

// Tool event types for displaying tool usage in UI (like Cursor's "Searched", "Read", etc.)
export interface ToolEvent {
	type: 'tool_start' | 'tool_end' | 'thinking' | 'error';
	tool: string; // Tool name (e.g., "web_search")
	query: string; // Query/action being performed
	result?: string; // Result summary (for tool_end)
	elapsed?: number; // Duration in milliseconds
}

// Streaming event that can be either content or tool event
export type StreamEvent =
	| { type: 'content'; content: string }
	| { type: 'tool'; event: ToolEvent }
	| { type: 'done' };

export interface Model {
	id: string;
	name?: string;
	object?: string;
	created?: number;
	owned_by?: string;
}

export interface ModelsResponse {
	data: Model[];
	object: string;
}

export const chatApi = {
	// Conversations
	getConversations: (limit = 50, offset = 0) =>
		api.get<ConversationsResponse>(`/api/conversations?limit=${limit}&offset=${offset}`),

	getConversation: (id: string) => api.get<Conversation>(`/api/conversations/${id}`),

	createConversation: (data: CreateConversationRequest) =>
		api.post<Conversation>('/api/conversations', data),

	updateConversation: (id: string, data: Partial<Conversation>) =>
		api.put<Conversation>(`/api/conversations/${id}`, data),

	deleteConversation: (id: string) => api.delete(`/api/conversations/${id}`),

	// Messages
	getMessages: (conversationId: string) =>
		api.get<MessagesResponse>(`/api/conversations/${conversationId}/messages`),

	createMessage: (
		conversationId: string,
		role: 'user' | 'assistant',
		content: string,
		model?: string
	) =>
		api.post<Message>(`/api/conversations/${conversationId}/messages`, {
			role,
			content,
			model
		}),

	// Models
	getModels: () => api.get<ModelsResponse>('/v1/models'),

	// Chat Completion (streaming)
	streamChatCompletion: async function* (
		request: ChatCompletionRequest,
		signal?: AbortSignal
	): AsyncGenerator<string, void, unknown> {
		const token = localStorage.getItem('access_token');

		const response = await fetch('/v1/chat/completions', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				...(token ? { Authorization: `Bearer ${token}` } : {})
			},
			body: JSON.stringify({ ...request, stream: true }),
			signal
		});

		if (!response.ok) {
			const error = await response.json().catch(() => ({ error: { message: 'Unknown error' } }));
			throw new Error(error.error?.message || `HTTP ${response.status}`);
		}

		const reader = response.body?.getReader();
		if (!reader) throw new Error('No response body');

		const decoder = new TextDecoder();
		let buffer = '';

		try {
			while (true) {
				const { done, value } = await reader.read();
				if (done) break;

				buffer += decoder.decode(value, { stream: true });
				const lines = buffer.split('\n');
				buffer = lines.pop() || '';

				for (const line of lines) {
					if (line.startsWith('data: ')) {
						const data = line.slice(6);
						if (data === '[DONE]') return;

						try {
							const parsed = JSON.parse(data);
							const content = parsed.choices?.[0]?.delta?.content;
							if (content) {
								yield content;
							}
						} catch {
							// Ignore parse errors for incomplete chunks
						}
					}
				}
			}
		} finally {
			reader.releaseLock();
		}
	},

	// Non-streaming chat completion
	chatCompletion: (request: ChatCompletionRequest) =>
		api.post<{
			id: string;
			choices: Array<{ message: { role: string; content: string } }>;
			usage?: { prompt_tokens: number; completion_tokens: number; total_tokens: number };
		}>('/v1/chat/completions', { ...request, stream: false }),

	// Chat Completion with tools support (streaming)
	// Uses /api/chat/completions which supports web search and other tools
	streamChatWithTools: async function* (
		request: ChatCompletionRequest,
		signal?: AbortSignal
	): AsyncGenerator<StreamEvent, void, unknown> {
		const token = localStorage.getItem('access_token');

		const response = await fetch('/api/chat/completions', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				...(token ? { Authorization: `Bearer ${token}` } : {})
			},
			body: JSON.stringify({ ...request, stream: true, use_tools: true }),
			signal
		});

		if (!response.ok) {
			const error = await response.json().catch(() => ({ error: { message: 'Unknown error' } }));
			throw new Error(error.error?.message || `HTTP ${response.status}`);
		}

		const reader = response.body?.getReader();
		if (!reader) throw new Error('No response body');

		const decoder = new TextDecoder();
		let buffer = '';

		try {
			while (true) {
				const { done, value } = await reader.read();
				if (done) break;

				buffer += decoder.decode(value, { stream: true });
				const lines = buffer.split('\n');
				buffer = lines.pop() || '';

				for (const line of lines) {
					const trimmed = line.trim();
					if (!trimmed) continue;

					// Handle tool events (SSE with "event: tool")
					if (trimmed.startsWith('event: tool')) {
						continue; // Next line will be the data
					}

					if (trimmed.startsWith('data: ')) {
						const data = trimmed.slice(6);
						if (data === '[DONE]') {
							yield { type: 'done' };
							return;
						}

						try {
							const parsed = JSON.parse(data);

							// Check if it's a tool event
							if (parsed.type && parsed.tool) {
								yield { type: 'tool', event: parsed as ToolEvent };
								continue;
							}

							// Regular chat completion chunk
							const content = parsed.choices?.[0]?.delta?.content;
							if (content) {
								yield { type: 'content', content };
							}
						} catch {
							// Ignore parse errors for incomplete chunks
						}
					}
				}
			}
		} finally {
			reader.releaseLock();
		}
	}
};

