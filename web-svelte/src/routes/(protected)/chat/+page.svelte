<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { MessageSquare, PanelLeftClose, PanelLeft, Loader2 } from 'lucide-svelte';
	import { chatStore } from '$lib/stores/chat.svelte';
	import { chatApi } from '$lib/api';
	import {
		ConversationList,
		MessageList,
		ChatInput,
		ModelPanel
	} from '$lib/components/chat';
	import { cn, generateUUID } from '$lib/utils';
	import * as m from '$lib/paraglide/messages';

	let sidebarOpen = $state(true);
	let abortController: AbortController | null = null;

	onMount(async () => {
		await Promise.all([loadConversations(), loadModels()]);

		// Check if conversation ID in URL
		const conversationId = $page.url.searchParams.get('id');
		if (conversationId) {
			await loadConversation(conversationId);
		}
	});

	async function loadConversations() {
		try {
			const response = await chatApi.getConversations();
			chatStore.setConversations(response.conversations || []);
		} catch (error) {
			console.error('Failed to load conversations:', error);
		}
	}

	async function loadModels() {
		try {
			const response = await chatApi.getModels();
			chatStore.setAvailableModels(response.data || []);
		} catch (error) {
			console.error('Failed to load models:', error);
		}
	}

	async function loadConversation(id: string) {
		chatStore.setLoading(true);
		try {
			const [conversation, messagesResponse] = await Promise.all([
				chatApi.getConversation(id),
				chatApi.getMessages(id)
			]);

			chatStore.setCurrentConversation(id);
			chatStore.setMessages(messagesResponse.messages || []);

			if (conversation.model) {
				chatStore.setSelectedModel(conversation.model);
			}
		} catch (error) {
			console.error('Failed to load conversation:', error);
		} finally {
			chatStore.setLoading(false);
		}
	}

	async function handleNewChat() {
		chatStore.clearMessages();
		// Update URL
		goto('/chat', { replaceState: true });
	}

	async function handleSelectConversation(id: string) {
		if (id === chatStore.currentConversationId) return;
		await loadConversation(id);
		goto(`/chat?id=${id}`, { replaceState: true });
	}

	async function handleDeleteConversation(id: string) {
		if (!confirm('Delete this conversation?')) return;

		try {
			await chatApi.deleteConversation(id);
			chatStore.removeConversation(id);

			if (chatStore.currentConversationId === id) {
				handleNewChat();
			}
		} catch (error) {
			console.error('Failed to delete conversation:', error);
		}
	}

	async function handleSendMessage(content: string) {
		if (!chatStore.selectedModel) {
			alert('Please select a model first');
			return;
		}

		// Create conversation if new
		let conversationId = chatStore.currentConversationId;
		if (!conversationId) {
			try {
				const conversation = await chatApi.createConversation({
					title: content.slice(0, 50),
					model: chatStore.selectedModel
				});
				conversationId = conversation.id;
				chatStore.setCurrentConversation(conversationId);
				chatStore.setConversations([conversation, ...chatStore.conversations]);
				goto(`/chat?id=${conversationId}`, { replaceState: true });
			} catch (error) {
				console.error('Failed to create conversation:', error);
				return;
			}
		}

		// Add user message
		const userMessage = {
			id: generateUUID(),
			role: 'user' as const,
			content,
			created_at: new Date().toISOString()
		};
		chatStore.addMessage(userMessage);

		// Save user message to backend
		try {
			await chatApi.createMessage(conversationId, 'user', content, chatStore.selectedModel);
		} catch (error) {
			console.error('Failed to save user message:', error);
		}

		// Start streaming response
		chatStore.setStreaming(true);
		abortController = new AbortController();

		try {
			const messages = chatStore.messages.map((msg) => ({
				role: msg.role,
				content: msg.content
			}));

			let fullResponse = '';

			for await (const chunk of chatApi.streamChatCompletion(
				{
					model: chatStore.selectedModel,
					messages,
					temperature: chatStore.params.temperature,
					top_p: chatStore.params.top_p,
					max_tokens: chatStore.params.max_tokens,
					num_ctx: chatStore.params.num_ctx
				},
				abortController.signal
			)) {
				fullResponse += chunk;
				chatStore.setStreamingContent(fullResponse);
			}

			// Add assistant message
			const assistantMessage = {
				id: generateUUID(),
				role: 'assistant' as const,
				content: fullResponse,
				created_at: new Date().toISOString(),
				model: chatStore.selectedModel
			};
			chatStore.addMessage(assistantMessage);

			// Save to backend
			await chatApi.createMessage(
				conversationId,
				'assistant',
				fullResponse,
				chatStore.selectedModel
			);

			// Refresh conversations to update timestamp
			await loadConversations();
		} catch (error) {
			if ((error as Error).name !== 'AbortError') {
				console.error('Streaming error:', error);
				// Add error message
				chatStore.addMessage({
					id: generateUUID(),
					role: 'assistant',
					content: `Error: ${(error as Error).message}`,
					created_at: new Date().toISOString()
				});
			}
		} finally {
			chatStore.setStreaming(false);
			abortController = null;
		}
	}

	function handleStopStreaming() {
		if (abortController) {
			abortController.abort();
		}
	}
</script>

<svelte:head>
	<title>{m.nav_chat()} | AI Gateway</title>
</svelte:head>

<div class="flex h-[calc(100vh-4rem)]">
	<!-- Sidebar -->
	<div
		class={cn(
			'flex h-full flex-col border-r border-border bg-card transition-all duration-300',
			sidebarOpen ? 'w-72' : 'w-0 overflow-hidden'
		)}
	>
		<ConversationList
			conversations={chatStore.conversations}
			currentId={chatStore.currentConversationId}
			onSelect={handleSelectConversation}
			onNew={handleNewChat}
			onDelete={handleDeleteConversation}
		/>
	</div>

	<!-- Main Chat Area -->
	<div class="flex flex-1 flex-col">
		<!-- Header with Sidebar Toggle -->
		<div class="flex items-center gap-2 border-b border-border px-4 py-2">
			<button
				onclick={() => (sidebarOpen = !sidebarOpen)}
				class="rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-foreground"
				title={sidebarOpen ? 'Hide sidebar' : 'Show sidebar'}
			>
				{#if sidebarOpen}
					<PanelLeftClose class="h-5 w-5" />
				{:else}
					<PanelLeft class="h-5 w-5" />
				{/if}
			</button>

			{#if chatStore.currentConversation}
				<h1 class="truncate text-sm font-medium">
					{chatStore.currentConversation.title || 'Untitled'}
				</h1>
			{:else}
				<h1 class="text-sm font-medium text-muted-foreground">{m.chat_newChat()}</h1>
			{/if}
		</div>

		<!-- Model Panel -->
		<ModelPanel
			models={chatStore.availableModels}
			selectedModel={chatStore.selectedModel}
			params={chatStore.params}
			onModelSelect={(model) => chatStore.setSelectedModel(model)}
			onParamsChange={(params) => chatStore.setParams(params)}
			onPreset={(preset) => chatStore.applyPreset(preset)}
		/>

		<!-- Messages or Welcome -->
		{#if chatStore.isLoading}
			<div class="flex flex-1 items-center justify-center">
				<Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
			</div>
		{:else if chatStore.hasMessages || chatStore.isStreaming}
			<MessageList
				messages={chatStore.messages}
				streamingContent={chatStore.streamingContent}
				isStreaming={chatStore.isStreaming}
			/>
		{:else}
			<!-- Welcome Screen -->
			<div class="flex flex-1 flex-col items-center justify-center p-8">
				<div class="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10 text-primary">
					<MessageSquare class="h-8 w-8" />
				</div>
				<h2 class="mb-2 text-xl font-semibold text-foreground">{m.chat_newChat()}</h2>
				<p class="mb-8 max-w-md text-center text-muted-foreground">
					Start a conversation with AI. Choose a model above and type your message below.
				</p>

				<!-- Quick Prompts -->
				<div class="grid max-w-2xl gap-3 sm:grid-cols-2">
					{#each ['Explain quantum computing', 'Write a Python function', 'Help me debug code', 'Summarize a topic'] as prompt}
						<button
							onclick={() => handleSendMessage(prompt)}
							disabled={!chatStore.selectedModel}
							class="rounded-lg border border-border bg-card px-4 py-3 text-left text-sm transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50"
						>
							{prompt}
						</button>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Chat Input -->
		<ChatInput
			onSend={handleSendMessage}
			onStop={handleStopStreaming}
			isStreaming={chatStore.isStreaming}
			disabled={!chatStore.selectedModel}
		/>
	</div>
</div>

