/**
 * Chat Store - manages conversations and messages with Svelte 5 runes
 */
import { browser } from '$app/environment';

export interface Message {
	id: string;
	role: 'user' | 'assistant' | 'system';
	content: string;
	created_at: string;
	model?: string;
	tokens?: number;
}

export interface Conversation {
	id: string;
	title: string;
	model: string;
	created_at: string;
	updated_at: string;
	message_count?: number;
}

export interface ChatParams {
	temperature: number;
	top_p: number;
	max_tokens: number;
	num_ctx: number;
	use_tools: boolean; // Enable web search and other tools
}

// Tool event for displaying tool usage in UI (like Cursor's "Searched", "Read", etc.)
export interface ToolEvent {
	type: 'tool_start' | 'tool_end' | 'thinking' | 'error';
	tool: string;
	query: string;
	result?: string;
	elapsed?: number;
	timestamp?: number;
}

const DEFAULT_PARAMS: ChatParams = {
	temperature: 0.7,
	top_p: 0.9,
	max_tokens: 4096,
	num_ctx: 8192,
	use_tools: true // Enabled by default
};

class ChatStore {
	// State
	conversations = $state<Conversation[]>([]);
	currentConversationId = $state<string | null>(null);
	messages = $state<Message[]>([]);
	selectedModel = $state<string | null>(null);
	availableModels = $state<Array<{ id: string; name?: string }>>([]);
	params = $state<ChatParams>({ ...DEFAULT_PARAMS });
	isStreaming = $state(false);
	streamingContent = $state('');
	isLoading = $state(false);
	
	// Tool events (for displaying tool usage like Cursor)
	toolEvents = $state<ToolEvent[]>([]);

	// Derived
	currentConversation = $derived(
		this.conversations.find((c) => c.id === this.currentConversationId) || null
	);

	hasMessages = $derived(this.messages.length > 0);

	// Presets
	readonly presets = {
		creative: { temperature: 0.9, top_p: 0.95, max_tokens: 4096, num_ctx: 8192, use_tools: true },
		balanced: { temperature: 0.7, top_p: 0.9, max_tokens: 4096, num_ctx: 8192, use_tools: true },
		precise: { temperature: 0.3, top_p: 0.8, max_tokens: 4096, num_ctx: 8192, use_tools: true },
		coding: { temperature: 0.2, top_p: 0.85, max_tokens: 8192, num_ctx: 16384, use_tools: true }
	} as const;

	// Actions
	setConversations(conversations: Conversation[]) {
		this.conversations = conversations;
	}

	setCurrentConversation(id: string | null) {
		this.currentConversationId = id;
	}

	setMessages(messages: Message[]) {
		this.messages = messages;
	}

	addMessage(message: Message) {
		this.messages = [...this.messages, message];
	}

	updateLastMessage(content: string) {
		if (this.messages.length > 0) {
			const lastIndex = this.messages.length - 1;
			this.messages[lastIndex] = { ...this.messages[lastIndex], content };
		}
	}

	setSelectedModel(model: string | null) {
		this.selectedModel = model;
		if (browser && model) {
			localStorage.setItem('chat_selected_model', model);
		}
	}

	setAvailableModels(models: Array<{ id: string; name?: string }>) {
		this.availableModels = models;
		// Auto-select first model if none selected
		if (!this.selectedModel && models.length > 0) {
			const saved = browser ? localStorage.getItem('chat_selected_model') : null;
			const savedModel = saved && models.find((m) => m.id === saved);
			this.setSelectedModel(savedModel ? saved : models[0].id);
		}
	}

	setParams(params: Partial<ChatParams>) {
		this.params = { ...this.params, ...params };
	}

	applyPreset(preset: keyof typeof this.presets) {
		this.params = { ...this.presets[preset] };
	}

	setStreaming(streaming: boolean) {
		this.isStreaming = streaming;
		if (!streaming) {
			this.streamingContent = '';
		}
	}

	setStreamingContent(content: string) {
		this.streamingContent = content;
	}
	
	// Tool events management
	addToolEvent(event: ToolEvent) {
		this.toolEvents = [...this.toolEvents, { ...event, timestamp: Date.now() }];
	}
	
	clearToolEvents() {
		this.toolEvents = [];
	}
	
	setUseTools(enabled: boolean) {
		this.params = { ...this.params, use_tools: enabled };
	}

	appendStreamingContent(chunk: string) {
		this.streamingContent += chunk;
	}

	setLoading(loading: boolean) {
		this.isLoading = loading;
	}

	clearMessages() {
		this.messages = [];
		this.currentConversationId = null;
	}

	// Delete conversation from list
	removeConversation(id: string) {
		this.conversations = this.conversations.filter((c) => c.id !== id);
		if (this.currentConversationId === id) {
			this.clearMessages();
		}
	}

	// Update conversation in list
	updateConversation(id: string, updates: Partial<Conversation>) {
		this.conversations = this.conversations.map((c) =>
			c.id === id ? { ...c, ...updates } : c
		);
	}
}

export const chatStore = new ChatStore();

