<script lang="ts">
	import { Book, Package, Shield, Brain, GitBranch, FileCode, Search, Bot, FileEdit, TestTube } from 'lucide-svelte';
	import * as m from '$lib/paraglide/messages';

	// Documentation sections
	const sections = [
		{
			id: 'dependency-scanner',
			icon: Package,
			title: 'Dependency Scanner',
			description: 'Автоматическая проверка зависимостей на обновления и уязвимости'
		},
		{
			id: 'secrets-scanner',
			icon: Shield,
			title: 'Secrets Scanner',
			description: 'Поиск хардкод секретов и конфиденциальных данных в коде'
		},
		{
			id: 'quality-score',
			icon: Brain,
			title: 'Quality Score',
			description: 'LLM-based оценка качества кода с метриками'
		},
		{
			id: 'dead-code',
			icon: FileCode,
			title: 'Dead Code',
			description: 'Обнаружение неиспользуемого кода'
		},
		{
			id: 'mr-review',
			icon: GitBranch,
			title: 'MR Review',
			description: 'Автоматический code review merge requests с помощью LLM'
		},
		{
			id: 'autodoc',
			icon: FileEdit,
			title: 'Auto-Documentation',
			description: 'Автоматическая генерация документации для кода'
		},
		{
			id: 'testgen',
			icon: TestTube,
			title: 'Test Generation',
			description: 'Генерация unit-тестов с помощью LLM'
		},
		{
			id: 'rag',
			icon: Search,
			title: 'RAG & Indexing',
			description: 'Индексация кодовой базы для семантического поиска'
		}
	];

	let activeSection = $state('dependency-scanner');
</script>

<svelte:head>
	<title>{m.nav_docs?.() || 'Documentation'} | AIGateway</title>
</svelte:head>

<div class="container mx-auto max-w-6xl p-6">
	<div class="mb-8">
		<div class="flex items-center gap-3 mb-2">
			<Book class="h-8 w-8 text-primary" />
			<h1 class="text-3xl font-bold">{m.nav_docs?.() || 'Documentation'}</h1>
		</div>
		<p class="text-muted-foreground">{m.docs_subtitle?.() || 'Руководство по использованию AIGateway'}</p>
	</div>

	<div class="grid grid-cols-1 md:grid-cols-4 gap-6">
		<!-- Sidebar -->
		<div class="space-y-2">
			{#each sections as section}
				<button
					onclick={() => activeSection = section.id}
					class="w-full flex items-center gap-3 p-3 rounded-lg text-left transition-colors
						{activeSection === section.id ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'}"
				>
					<section.icon class="h-5 w-5" />
					<span class="text-sm font-medium">{section.title}</span>
				</button>
			{/each}
		</div>

		<!-- Content -->
		<div class="md:col-span-3 bg-card rounded-lg border p-6">
			{#if activeSection === 'dependency-scanner'}
				<h2 class="text-2xl font-bold mb-4 flex items-center gap-3">
					<Package class="h-6 w-6 text-blue-500" />
					Dependency Scanner
				</h2>
				
				<div class="prose prose-sm dark:prose-invert max-w-none">
					<p>Автоматическая проверка зависимостей проекта на обновления и уязвимости.</p>
					
					<h3>Поддерживаемые языки</h3>
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b">
								<th class="text-left py-2">Язык</th>
								<th class="text-left py-2">Файл</th>
								<th class="text-left py-2">Реестр</th>
							</tr>
						</thead>
						<tbody>
							<tr class="border-b"><td class="py-2">Go</td><td>go.mod</td><td>proxy.golang.org</td></tr>
							<tr class="border-b"><td class="py-2">Node.js</td><td>package.json</td><td>npmjs.org</td></tr>
							<tr class="border-b"><td class="py-2">Python</td><td>requirements.txt</td><td>pypi.org</td></tr>
						</tbody>
					</table>

					<h3>Как использовать</h3>
					<ol>
						<li><strong>Проиндексируйте репозиторий</strong> — Admin → GitLab → [Проект] → 🔄 Index</li>
						<li><strong>Запустите проверку</strong> — нажмите 📦 Check Dependencies</li>
						<li><strong>Просмотрите результаты</strong> — обновления, уязвимости, рекомендации</li>
					</ol>

					<h3>Типы обновлений</h3>
					<ul>
						<li><span class="text-red-500 font-bold">major</span> — мажорное (1.x → 2.x), возможны breaking changes</li>
						<li><span class="text-yellow-500 font-bold">minor</span> — минорное (1.1 → 1.2), новые функции</li>
						<li><span class="text-green-500 font-bold">patch</span> — патч (1.1.1 → 1.1.2), исправления</li>
					</ul>

					<h3>Проверка уязвимостей</h3>
					<p>Каждая зависимость проверяется через <a href="https://osv.dev" target="_blank" class="text-primary hover:underline">OSV.dev</a> — 
					открытую базу данных уязвимостей (CVE, GHSA).</p>
				</div>

			{:else if activeSection === 'secrets-scanner'}
				<h2 class="text-2xl font-bold mb-4 flex items-center gap-3">
					<Shield class="h-6 w-6 text-green-500" />
					Secrets Scanner
				</h2>
				
				<div class="prose prose-sm dark:prose-invert max-w-none">
					<p>Поиск хардкод секретов и конфиденциальных данных в проиндексированном коде.</p>
					
					<h3>Два режима сканирования</h3>
					
					<h4>🛡️ Regex Scan (быстрый)</h4>
					<ul>
						<li>~30 предустановленных паттернов</li>
						<li>AWS ключи, GitHub токены, пароли</li>
						<li>Мгновенный результат</li>
					</ul>

					<h4>🧠 Deep Scan (LLM)</h4>
					<ul>
						<li>Семантический анализ с помощью LLM</li>
						<li>Понимает контекст (placeholder vs реальный секрет)</li>
						<li>Выдает confidence (high/medium/low)</li>
						<li>Расходует токены, требует analysis model</li>
					</ul>

					<h3>Как использовать</h3>
					<ol>
						<li>Проиндексируйте репозиторий</li>
						<li>Нажмите 🛡️ (Regex) или 🧠 (LLM) рядом с проектом</li>
						<li>Просмотрите найденные секреты</li>
					</ol>
				</div>

			{:else if activeSection === 'quality-score'}
				<h2 class="text-2xl font-bold mb-4 flex items-center gap-3">
					<Brain class="h-6 w-6 text-indigo-500" />
					Code Quality Score
				</h2>
				
				<div class="prose prose-sm dark:prose-invert max-w-none">
					<p>LLM-based оценка качества кода с детальным разбором по категориям.</p>
					
					<h3>Категории оценки</h3>
					<ul>
						<li><strong>complexity</strong> — сложность кода (цикломатическая)</li>
						<li><strong>documentation</strong> — качество комментариев и документации</li>
						<li><strong>naming</strong> — качество именования переменных и функций</li>
						<li><strong>error_handling</strong> — полнота обработки ошибок</li>
						<li><strong>maintainability</strong> — общая поддерживаемость</li>
					</ul>

					<h3>Как использовать</h3>
					<ol>
						<li>Проиндексируйте репозиторий</li>
						<li>Настройте Analysis Model в проекте</li>
						<li>Нажмите 📊 рядом с проектом</li>
						<li>Просмотрите результат: общая оценка, breakdown, рекомендации</li>
					</ol>

					<h3>Интерпретация результатов</h3>
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b">
								<th class="text-left py-2">Оценка</th>
								<th class="text-left py-2">Статус</th>
							</tr>
						</thead>
						<tbody>
							<tr class="border-b"><td class="py-2 text-green-600 font-bold">80-100</td><td>Отличное качество</td></tr>
							<tr class="border-b"><td class="py-2 text-yellow-600 font-bold">60-79</td><td>Хорошо, но есть что улучшить</td></tr>
							<tr class="border-b"><td class="py-2 text-orange-600 font-bold">40-59</td><td>Требуется внимание</td></tr>
							<tr class="border-b"><td class="py-2 text-red-600 font-bold">0-39</td><td>Критическое состояние</td></tr>
						</tbody>
					</table>
				</div>

			{:else if activeSection === 'dead-code'}
				<h2 class="text-2xl font-bold mb-4 flex items-center gap-3">
					<FileCode class="h-6 w-6 text-orange-500" />
					Dead Code Detection
				</h2>
				
				<div class="prose prose-sm dark:prose-invert max-w-none">
					<p>Обнаружение неиспользуемого кода через RAG + LLM анализ.</p>
					
					<h3>Что обнаруживает</h3>
					<ul>
						<li>Неиспользуемые функции</li>
						<li>Неиспользуемые типы и структуры</li>
						<li>Неиспользуемые переменные и константы</li>
						<li>Классы без использования</li>
					</ul>

					<h3>Уровни уверенности</h3>
					<ul>
						<li><span class="text-red-500 font-bold">high</span> — высокая вероятность, что код не используется</li>
						<li><span class="text-yellow-500 font-bold">medium</span> — возможно не используется</li>
						<li><span class="text-blue-500 font-bold">low</span> — требует ручной проверки</li>
					</ul>

					<h3>Как использовать</h3>
					<ol>
						<li>Проиндексируйте репозиторий</li>
						<li>Настройте Analysis Model</li>
						<li>Нажмите 🗑️ (Dead Code) рядом с проектом</li>
						<li>Просмотрите найденные символы</li>
					</ol>

					<h3>Ограничения</h3>
					<ul>
						<li>Не обнаруживает использование через reflection</li>
						<li>Не видит внешнее использование экспортируемых API</li>
						<li>Рекомендуется ручная проверка перед удалением</li>
					</ul>
				</div>

			{:else if activeSection === 'mr-review'}
				<h2 class="text-2xl font-bold mb-4 flex items-center gap-3">
					<GitBranch class="h-6 w-6 text-purple-500" />
					MR Review
				</h2>
				
				<div class="prose prose-sm dark:prose-invert max-w-none">
					<p>Автоматический code review merge requests с использованием LLM.</p>
					
					<h3>Возможности</h3>
					<ul>
						<li>Анализ diff файлов</li>
						<li>Поиск проблем безопасности, производительности, стиля</li>
						<li>Контекст из RAG индекса (связанные функции)</li>
						<li>Комментарии в GitLab MR</li>
						<li>Поддержка русского и английского языков</li>
					</ul>

					<h3>Настройка</h3>
					<ol>
						<li><strong>Добавьте GitLab интеграцию</strong> — Admin → GitLab → Add Integration</li>
						<li><strong>Добавьте проект</strong> — укажите Analysis Model</li>
						<li><strong>Настройте webhook</strong> — для автоматического запуска</li>
						<li><strong>Проиндексируйте код</strong> — для контекста</li>
					</ol>

					<h3>Per-File Review</h3>
					<p>Для больших MR используется режим per-file review:</p>
					<ul>
						<li>Каждый файл анализируется отдельно</li>
						<li>LLM использует tools для поиска контекста в RAG</li>
						<li>Результаты агрегируются в один комментарий</li>
					</ul>
				</div>

			{:else if activeSection === 'autodoc'}
				<h2 class="text-2xl font-bold mb-4 flex items-center gap-3">
					<FileEdit class="h-6 w-6 text-purple-500" />
					Auto-Documentation
				</h2>
				
				<div class="prose prose-sm dark:prose-invert max-w-none">
					<p>Автоматическая генерация документации для недокументированного кода.</p>
					
					<h3>Поддерживаемые языки</h3>
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b">
								<th class="text-left py-2">Язык</th>
								<th class="text-left py-2">Формат</th>
							</tr>
						</thead>
						<tbody>
							<tr class="border-b"><td class="py-2">Go</td><td>GoDoc comments</td></tr>
							<tr class="border-b"><td class="py-2">JavaScript/TypeScript</td><td>JSDoc</td></tr>
							<tr class="border-b"><td class="py-2">Python</td><td>docstrings (Google style)</td></tr>
						</tbody>
					</table>

					<h3>Как использовать</h3>
					<ol>
						<li>Проиндексируйте репозиторий</li>
						<li>Настройте Analysis Model</li>
						<li>Нажмите ✏️ (Auto Doc) рядом с проектом</li>
						<li>Система найдет недокументированные функции и сгенерирует для них документацию</li>
						<li>Скопируйте документацию кнопкой "Copy"</li>
					</ol>

					<h3>Что обнаруживается</h3>
					<ul>
						<li>Экспортируемые функции без комментариев</li>
						<li>Типы и структуры без документации</li>
						<li>Классы без docstrings</li>
					</ul>
				</div>

			{:else if activeSection === 'testgen'}
				<h2 class="text-2xl font-bold mb-4 flex items-center gap-3">
					<TestTube class="h-6 w-6 text-green-500" />
					Test Generation
				</h2>
				
				<div class="prose prose-sm dark:prose-invert max-w-none">
					<p>LLM-генерация unit-тестов для функций без тестов.</p>
					
					<h3>Поддерживаемые фреймворки</h3>
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b">
								<th class="text-left py-2">Язык</th>
								<th class="text-left py-2">Фреймворк</th>
							</tr>
						</thead>
						<tbody>
							<tr class="border-b"><td class="py-2">Go</td><td>testing + testify</td></tr>
							<tr class="border-b"><td class="py-2">TypeScript/JS</td><td>Jest / Vitest</td></tr>
							<tr class="border-b"><td class="py-2">Python</td><td>pytest</td></tr>
						</tbody>
					</table>

					<h3>Как использовать</h3>
					<ol>
						<li>Проиндексируйте репозиторий</li>
						<li>Настройте Analysis Model</li>
						<li>Нажмите 🧪 (Generate Tests) рядом с проектом</li>
						<li>Система найдет функции без тестов и сгенерирует тесты</li>
						<li>Скопируйте тесты кнопкой "Copy"</li>
					</ol>

					<h3>Генерируемые тесты включают</h3>
					<ul>
						<li>Happy path (основной сценарий)</li>
						<li>Edge cases (граничные случаи)</li>
						<li>Error cases (обработка ошибок)</li>
						<li>Table-driven tests (для Go)</li>
						<li>Parametrized tests (для pytest)</li>
					</ul>
				</div>

			{:else if activeSection === 'rag'}
				<h2 class="text-2xl font-bold mb-4 flex items-center gap-3">
					<Search class="h-6 w-6 text-orange-500" />
					RAG & Indexing
				</h2>
				
				<div class="prose prose-sm dark:prose-invert max-w-none">
					<p>Индексация кодовой базы для семантического поиска и контекста.</p>
					
					<h3>Как это работает</h3>
					<ol>
						<li><strong>Клонирование</strong> — репозиторий клонируется локально</li>
						<li><strong>Chunking</strong> — код разбивается на семантические чанки</li>
						<li><strong>Embedding</strong> — каждый чанк преобразуется в вектор</li>
						<li><strong>Qdrant</strong> — векторы сохраняются в Qdrant DB</li>
					</ol>

					<h3>Использование индекса</h3>
					<ul>
						<li><strong>MR Review</strong> — поиск связанных функций</li>
						<li><strong>Secrets Scanner</strong> — проход по всем чанкам</li>
						<li><strong>Dependency Scanner</strong> — поиск файлов зависимостей</li>
						<li><strong>Chat</strong> — контекст для ответов</li>
					</ul>

					<h3>Требования</h3>
					<ul>
						<li>Qdrant сервер (Docker или standalone)</li>
						<li>Embedding модель (nomic-embed-text или другая)</li>
						<li>Достаточно места для индекса (~1MB на 1000 строк кода)</li>
					</ul>
				</div>
			{/if}
		</div>
	</div>
</div>

