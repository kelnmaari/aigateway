<script lang="ts">
	import { onMount } from 'svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { api } from '$lib/api/client';
	import GitLabNav from '$lib/components/gitlab-nav.svelte';
	import * as m from '$lib/paraglide/messages';

	// Telegram Settings
	let telegramEnabled = false;
	let telegramBotToken = '';
	let telegramDefaultChat = '';
	let telegramTestStatus = '';

	// Priority Queue Settings
	interface BranchRule {
		pattern: string;
		branches: string[];
		priority: number;
	}

	let priorityRules: BranchRule[] = [
		{ pattern: '', branches: ['main', 'master', 'production'], priority: 3 },
		{ pattern: 'release/*', branches: ['staging', 'develop'], priority: 2 },
		{ pattern: 'hotfix/*', branches: [], priority: 2 },
		{ pattern: 'feature/*', branches: [], priority: 1 },
		{ pattern: 'wip/*', branches: [], priority: 0 }
	];

	// Language Prompts
	interface LanguagePrompt {
		language: string;
		instructions: string;
		enabled: boolean;
	}

	let languagePrompts: LanguagePrompt[] = [
		{ language: 'go', instructions: '', enabled: true },
		{ language: 'typescript', instructions: '', enabled: true },
		{ language: 'python', instructions: '', enabled: true },
		{ language: 'rust', instructions: '', enabled: true },
		{ language: 'java', instructions: '', enabled: true }
	];

	let selectedLanguage = 'go';
	let customPrompt = '';
	let activeTab = 'telegram';

	let loading = true;
	let saving = false;
	let error = '';
	let success = '';

	const priorityLabels: Record<number, string> = {
		0: 'Low',
		1: 'Normal',
		2: 'High',
		3: 'Critical'
	};

	const priorityColors: Record<number, string> = {
		0: 'bg-gray-500',
		1: 'bg-blue-500',
		2: 'bg-yellow-500',
		3: 'bg-red-500'
	};

	onMount(() => {
		loadSettings();
	});

	async function loadSettings() {
		loading = true;
		try {
			const data = await api.get<{
				telegram?: { enabled: boolean; bot_token: string; default_chat: string };
				priority_rules?: BranchRule[];
				language_prompts?: LanguagePrompt[];
			}>('/api/admin/gitlab/settings');
			telegramEnabled = data.telegram?.enabled || false;
			telegramBotToken = data.telegram?.bot_token || '';
			telegramDefaultChat = data.telegram?.default_chat || '';
			// Only override defaults if API returns non-empty arrays
			if (Array.isArray(data.priority_rules) && data.priority_rules.length > 0) {
				priorityRules = data.priority_rules;
			}
			if (Array.isArray(data.language_prompts) && data.language_prompts.length > 0) {
				languagePrompts = data.language_prompts;
			}
		} catch (e) {
			// Settings not configured yet - use defaults
			console.log('Using default settings');
		} finally {
			loading = false;
		}
	}

	async function saveSettings() {
		saving = true;
		error = '';
		success = '';
		try {
			await api.put('/api/admin/gitlab/settings', {
				telegram: {
					enabled: telegramEnabled,
					bot_token: telegramBotToken,
					default_chat: telegramDefaultChat
				},
				priority_rules: priorityRules,
				language_prompts: languagePrompts
			});
			success = 'Settings saved successfully';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Unknown error';
		} finally {
			saving = false;
		}
	}

	async function testTelegram() {
		telegramTestStatus = 'Testing...';
		try {
			await api.post('/api/admin/gitlab/settings/telegram/test', {
				bot_token: telegramBotToken,
				chat_id: telegramDefaultChat
			});
			telegramTestStatus = '✓ Connection successful!';
		} catch {
			telegramTestStatus = '✗ Connection failed';
		}
	}

	function addPriorityRule() {
		priorityRules = [...priorityRules, { pattern: '', branches: [], priority: 1 }];
	}

	function removePriorityRule(index: number) {
		priorityRules = priorityRules.filter((_, i) => i !== index);
	}

	function updatePrompt(lang: string) {
		if (!Array.isArray(languagePrompts)) return;
		const prompt = languagePrompts.find(p => p.language === lang);
		if (prompt) {
			customPrompt = prompt.instructions;
		}
	}

	function savePrompt() {
		languagePrompts = languagePrompts.map(p => 
			p.language === selectedLanguage ? { ...p, instructions: customPrompt } : p
		);
	}

	$: if (selectedLanguage) updatePrompt(selectedLanguage);
</script>

<div class="space-y-6">
	<GitLabNav />

	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">GitLab Integration Settings</h1>
			<p class="text-muted-foreground">Configure notifications, priorities, and prompts</p>
		</div>
		<Button onclick={saveSettings} disabled={saving}>
			{saving ? 'Saving...' : 'Save All Settings'}
		</Button>
	</div>

	{#if error}
		<div class="bg-destructive/10 text-destructive p-4 rounded-lg">{error}</div>
	{/if}
	{#if success}
		<div class="bg-green-500/10 text-green-500 p-4 rounded-lg">{success}</div>
	{/if}

	<!-- Tabs -->
	<div class="space-y-4">
		<div class="flex gap-2 border-b">
			<button
				class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'telegram' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
				onclick={() => activeTab = 'telegram'}
			>
				Telegram
			</button>
			<button
				class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'priority' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
				onclick={() => activeTab = 'priority'}
			>
				Priority Queue
			</button>
			<button
				class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'prompts' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
				onclick={() => activeTab = 'prompts'}
			>
				Language Prompts
			</button>
		</div>

		<!-- Telegram Settings -->
		{#if activeTab === 'telegram'}
			<Card>
				<CardHeader>
					<CardTitle>Telegram Notifications</CardTitle>
					<CardDescription>Configure Telegram bot for review notifications</CardDescription>
				</CardHeader>
				<CardContent class="space-y-4">
					<div class="flex items-center justify-between">
						<div>
							<label for="telegram-enabled" class="font-medium">Enable Telegram Notifications</label>
							<p class="text-sm text-muted-foreground">Send review updates to Telegram</p>
						</div>
						<input id="telegram-enabled" type="checkbox" bind:checked={telegramEnabled} class="h-4 w-4" />
					</div>

					{#if telegramEnabled}
						<div class="space-y-4 pt-4 border-t">
							<div class="space-y-2">
								<label for="bot-token" class="text-sm font-medium">Bot Token</label>
								<input
									id="bot-token"
									type="password"
									bind:value={telegramBotToken}
									placeholder="123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
									class="w-full h-10 rounded-md border bg-background px-3 text-sm"
								/>
								<p class="text-xs text-muted-foreground">Get this from @BotFather on Telegram</p>
							</div>

							<div class="space-y-2">
								<label for="default-chat" class="text-sm font-medium">Default Chat ID</label>
								<input
									id="default-chat"
									bind:value={telegramDefaultChat}
									placeholder="-1001234567890"
									class="w-full h-10 rounded-md border bg-background px-3 text-sm"
								/>
								<p class="text-xs text-muted-foreground">Group or channel ID for notifications</p>
							</div>

							<div class="flex items-center gap-4">
								<Button variant="outline" onclick={testTelegram}>
									Test Connection
								</Button>
								{#if telegramTestStatus}
									<span class={telegramTestStatus.includes('✓') ? 'text-green-500' : 'text-red-500'}>
										{telegramTestStatus}
									</span>
								{/if}
							</div>
						</div>
					{/if}
				</CardContent>
			</Card>
		{:else if activeTab === 'priority'}
			<!-- Priority Queue Settings -->
			<Card>
				<CardHeader>
					<CardTitle>Priority Queue Rules</CardTitle>
					<CardDescription>Configure branch-based priority for review queue</CardDescription>
				</CardHeader>
				<CardContent class="space-y-4">
					{#each priorityRules as rule, index}
						<div class="flex items-center gap-4 p-3 bg-muted/50 rounded-lg">
							<div class="flex-1 space-y-2">
								<div class="flex gap-2">
									<input
										placeholder={m.placeholder_pattern()}
										bind:value={rule.pattern}
										class="flex-1 h-10 rounded-md border bg-background px-3 text-sm"
									/>
									<input
										placeholder={m.placeholder_branches()}
										value={rule.branches.join(', ')}
										onchange={(e) => rule.branches = (e.target as HTMLInputElement).value.split(',').map((b: string) => b.trim()).filter(Boolean)}
										class="flex-1 h-10 rounded-md border bg-background px-3 text-sm"
									/>
								</div>
							</div>
							<select
								bind:value={rule.priority}
								class="px-3 py-2 rounded-md border bg-background"
							>
								{#each Object.entries(priorityLabels) as [value, label]}
									<option value={parseInt(value)}>{label}</option>
								{/each}
							</select>
							<span class="px-2 py-1 text-xs font-medium text-white rounded {priorityColors[rule.priority]}">
								{priorityLabels[rule.priority]}
							</span>
							<button class="text-muted-foreground hover:text-foreground" onclick={() => removePriorityRule(index)}>×</button>
						</div>
					{/each}

					<Button variant="outline" onclick={addPriorityRule}>
						+ Add Rule
					</Button>

					<div class="text-sm text-muted-foreground pt-4 border-t">
						<p><strong>Priority Order:</strong> Critical → High → Normal → Low</p>
						<p>Jobs targeting main/master branches get highest priority by default.</p>
					</div>
				</CardContent>
			</Card>
		{:else}
			<!-- Language Prompts Settings -->
			<Card>
				<CardHeader>
					<CardTitle>Language-Specific Prompts</CardTitle>
					<CardDescription>Customize review instructions for each programming language</CardDescription>
				</CardHeader>
				<CardContent class="space-y-4">
					<div class="flex gap-2 flex-wrap">
						{#each languagePrompts as prompt}
							<button
								class="px-3 py-1.5 text-sm rounded-md border {selectedLanguage === prompt.language ? 'bg-primary text-primary-foreground' : 'bg-background hover:bg-muted'}"
								onclick={() => selectedLanguage = prompt.language}
							>
								{prompt.language}
								{#if !prompt.enabled}
									<span class="ml-1 text-xs opacity-60">(off)</span>
								{/if}
							</button>
						{/each}
					</div>

					<div class="space-y-2">
						<div class="flex items-center justify-between">
							<label for="custom-instructions" class="text-sm font-medium">Custom Instructions for {selectedLanguage}</label>
							<div class="flex items-center gap-2">
								<label for="lang-enabled" class="text-sm">Enabled</label>
								<input
									id="lang-enabled"
									type="checkbox"
									checked={languagePrompts.find(p => p.language === selectedLanguage)?.enabled}
									onchange={(e) => {
										languagePrompts = languagePrompts.map(p =>
											p.language === selectedLanguage ? { ...p, enabled: (e.target as HTMLInputElement).checked } : p
										);
									}}
									class="h-4 w-4"
								/>
							</div>
						</div>
						<textarea
							bind:value={customPrompt}
							placeholder={m.placeholder_review_instructions()}
							rows="10"
							class="w-full rounded-md border bg-background px-3 py-2 text-sm"
						></textarea>
						<Button variant="outline" onclick={savePrompt}>
							Update {selectedLanguage} Prompt
						</Button>
					</div>

					<div class="text-sm text-muted-foreground pt-4 border-t">
						<p>Leave empty to use default prompts. Custom prompts are appended to the base review instructions.</p>
					</div>
				</CardContent>
			</Card>
		{/if}
	</div>
</div>
