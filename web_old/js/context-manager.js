// Context Manager (v1.9.1)
// Tracks token usage, manages context window, and triggers auto-summarization

class ContextManager {
    constructor() {
        this.contextIndicator = document.getElementById('context-indicator');
        this.contextText = document.getElementById('context-text');
        
        // Token estimation constants
        this.CHARS_PER_TOKEN = 4; // Approximate: 1 token ≈ 4 characters
        this.SUMMARIZATION_THRESHOLD = 0.75; // Summarize at 75% usage
        this.WARNING_THRESHOLD = 0.60; // Show warning at 60%
        
        // Current context state
        this.currentTokens = 0;
        this.maxContextTokens = 4096; // Default, will be updated from model params
        this.messages = [];
        
        this.init();
    }

    init() {
        console.log('ContextManager initialized');
    }

    // Estimate tokens in text
    estimateTokens(text) {
        if (typeof text !== 'string') return 0;
        return Math.ceil(text.length / this.CHARS_PER_TOKEN);
    }

    // Update context window size from model params
    updateContextWindow(numCtx) {
        this.maxContextTokens = numCtx || 4096;
        this.updateUI();
        console.log(`Context window updated: ${this.maxContextTokens} tokens`);
    }

    // Add message to context
    addMessage(role, content) {
        const tokens = this.estimateTokens(content);
        
        this.messages.push({
            role,
            content,
            tokens
        });

        this.currentTokens += tokens;
        this.updateUI();

        // Check if summarization needed
        const usage = this.currentTokens / this.maxContextTokens;
        if (usage >= this.SUMMARIZATION_THRESHOLD) {
            console.warn(`Context at ${Math.round(usage * 100)}% - summarization recommended`);
            this.triggerSummarization();
        }
    }

    // Update UI indicator
    updateUI() {
        if (!this.contextText || !this.contextIndicator) return;

        const usage = this.currentTokens / this.maxContextTokens;
        const percentage = Math.round(usage * 100);

        this.contextText.textContent = `${this.currentTokens}/${this.maxContextTokens}`;

        // Update indicator class based on usage
        this.contextIndicator.classList.remove('warning', 'danger');
        
        if (usage >= this.SUMMARIZATION_THRESHOLD) {
            this.contextIndicator.classList.add('danger');
            this.contextIndicator.title = `Context ${percentage}% full - summarization recommended`;
        } else if (usage >= this.WARNING_THRESHOLD) {
            this.contextIndicator.classList.add('warning');
            this.contextIndicator.title = `Context ${percentage}% full`;
        } else {
            this.contextIndicator.title = `Context usage: ${percentage}%`;
        }
    }

    // Trigger automatic summarization
    async triggerSummarization() {
        console.log('🔄 Auto-summarization triggered');

        try {
            // Find messages to summarize (exclude last 2-3 exchanges)
            const keepLast = 6; // Keep last 3 exchanges (user + assistant pairs)
            const messagesToSummarize = this.messages.slice(0, -keepLast);
            const messagesToKeep = this.messages.slice(-keepLast);

            if (messagesToSummarize.length < 4) {
                console.log('Not enough messages to summarize');
                return;
            }

            // Build conversation text for summarization
            const conversationText = messagesToSummarize
                .map(msg => `${msg.role}: ${msg.content}`)
                .join('\n\n');

            // Request summarization from model
            const summary = await this.requestSummarization(conversationText);

            if (summary) {
                // Replace old messages with summary
                const summaryTokens = this.estimateTokens(summary);
                const oldTokens = messagesToSummarize.reduce((sum, msg) => sum + msg.tokens, 0);

                this.messages = [
                    {
                        role: 'system',
                        content: `Previous conversation summary:\n${summary}`,
                        tokens: summaryTokens
                    },
                    ...messagesToKeep
                ];

                this.currentTokens = this.currentTokens - oldTokens + summaryTokens;
                this.updateUI();

                console.log(`✅ Summarization complete: ${oldTokens} → ${summaryTokens} tokens (saved ${oldTokens - summaryTokens})`);
                
                // Show notification to user
                this.showSummarizationNotification(oldTokens, summaryTokens);
            }
        } catch (error) {
            console.error('Summarization failed:', error);
        }
    }

    // Request summarization from AI model
    async requestSummarization(conversationText) {
        try {
            const token = localStorage.getItem('access_token');
            if (!token) throw new Error('No auth token');

            const model = window.modelPanel?.getSelectedModel() || 'qwen2.5-coder-tuned:7b';

            const response = await fetch('/v1/chat/completions', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    model: model,
                    messages: [
                        {
                            role: 'system',
                            content: 'You are a helpful assistant that summarizes conversations concisely while preserving key information, context, and important details. Create a clear, factual summary in 2-3 paragraphs.'
                        },
                        {
                            role: 'user',
                            content: `Please summarize the following conversation, preserving important context and key points:\n\n${conversationText}`
                        }
                    ],
                    temperature: 0.3,
                    max_tokens: 500
                })
            });

            if (!response.ok) {
                throw new Error(`Summarization request failed: ${response.status}`);
            }

            const data = await response.json();
            return data.choices?.[0]?.message?.content || null;
        } catch (error) {
            console.error('Summarization API error:', error);
            return null;
        }
    }

    // Show notification about summarization
    showSummarizationNotification(oldTokens, newTokens) {
        // Check if notifications utility exists
        if (typeof window.showNotification === 'function') {
            window.showNotification(
                `Context summarized: ${oldTokens} → ${newTokens} tokens saved`,
                'info',
                5000
            );
        } else {
            console.log(`📝 Context summarized: ${oldTokens} → ${newTokens} tokens`);
        }
    }

    // Get current messages for API request
    getMessages() {
        return this.messages.map(msg => ({
            role: msg.role,
            content: msg.content
        }));
    }

    // Clear context
    clear() {
        this.messages = [];
        this.currentTokens = 0;
        this.updateUI();
        console.log('Context cleared');
    }

    // Get context stats
    getStats() {
        return {
            totalMessages: this.messages.length,
            currentTokens: this.currentTokens,
            maxTokens: this.maxContextTokens,
            usage: (this.currentTokens / this.maxContextTokens * 100).toFixed(1) + '%'
        };
    }
}

// Initialize global instance
window.contextManager = new ContextManager();

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ContextManager;
}

