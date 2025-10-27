// About System Page JavaScript

class AboutPage {
    constructor() {
        this.systemInfo = null;
        this.changelogs = [];
    }

    async init() {
        await Promise.all([
            this.loadSystemInfo(),
            this.loadChangelogs()
        ]);
    }

    async loadSystemInfo() {
        const loadingEl = document.getElementById('system-info-loading');
        const contentEl = document.getElementById('system-info-content');
        const errorEl = document.getElementById('system-info-error');

        try {
            const response = await api.request(`${api.baseURL}/api/system/info`);
            
            if (!response.ok) {
                throw new Error('Failed to load system info');
            }

            this.systemInfo = await response.json();
            
            // Display system info
            document.getElementById('version-number').textContent = this.systemInfo.version || 'N/A';
            document.getElementById('git-commit').textContent = this.systemInfo.git_commit || 'N/A';
            document.getElementById('build-date').textContent = this.systemInfo.build_date || 'N/A';
            document.getElementById('go-version').textContent = this.systemInfo.go_version || 'N/A';

            loadingEl.style.display = 'none';
            contentEl.style.display = 'block';
        } catch (error) {
            console.error('Error loading system info:', error);
            loadingEl.style.display = 'none';
            errorEl.style.display = 'block';
        }
    }

    async loadChangelogs() {
        const loadingEl = document.getElementById('changelog-loading');
        const contentEl = document.getElementById('changelog-content');
        const errorEl = document.getElementById('changelog-error');

        try {
            const response = await api.request(`${api.baseURL}/api/system/changelogs`);
            
            if (!response.ok) {
                throw new Error('Failed to load changelogs');
            }

            const data = await response.json();
            this.changelogs = data.changelogs || [];
            
            // Sort changelogs by version (semver)
            this.changelogs.sort((a, b) => {
                return this.compareVersions(b.version, a.version); // DESC order
            });
            
            this.renderChangelogs();

            loadingEl.style.display = 'none';
            contentEl.style.display = 'block';
        } catch (error) {
            console.error('Error loading changelogs:', error);
            loadingEl.style.display = 'none';
            errorEl.style.display = 'block';
        }
    }

    compareVersions(v1, v2) {
        // Compare semantic versions (1.4.10 vs 1.4.9)
        const parts1 = v1.split('.').map(Number);
        const parts2 = v2.split('.').map(Number);
        
        for (let i = 0; i < Math.max(parts1.length, parts2.length); i++) {
            const num1 = parts1[i] || 0;
            const num2 = parts2[i] || 0;
            
            if (num1 > num2) return 1;
            if (num1 < num2) return -1;
        }
        
        return 0;
    }

    renderChangelogs() {
        const accordion = document.getElementById('changelog-accordion');
        
        if (this.changelogs.length === 0) {
            accordion.innerHTML = '<p style="text-align: center; color: var(--text-secondary);">Нет доступных changelog записей</p>';
            return;
        }

        accordion.innerHTML = this.changelogs.map((changelog, index) => `
            <div class="changelog-item">
                <button class="changelog-header" onclick="aboutPage.toggleChangelog('${changelog.version}')">
                    <div class="changelog-title">
                        <span class="changelog-version">v${changelog.version}</span>
                        <span class="changelog-date">${this.formatDate(changelog.release_date)}</span>
                    </div>
                    <svg class="changelog-chevron" id="chevron-${changelog.version}" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="6 9 12 15 18 9"></polyline>
                    </svg>
                </button>
                <div class="changelog-content" id="content-${changelog.version}" style="display: ${index === 0 ? 'block' : 'none'};">
                    <div class="changelog-markdown" id="markdown-${changelog.version}">
                        ${this.renderMarkdown(changelog.content)}
                    </div>
                </div>
            </div>
        `).join('');

        // Add badge classes to h4 elements
        this.changelogs.forEach(changelog => {
            const markdown = document.getElementById(`markdown-${changelog.version}`);
            if (markdown) {
                this.addCategoryBadges(markdown);
            }
        });
    }

    addCategoryBadges(markdownElement) {
        const h4Elements = markdownElement.querySelectorAll('h4');
        h4Elements.forEach(h4 => {
            const text = h4.textContent.toLowerCase();
            if (text.includes('added')) h4.classList.add('badge-added');
            else if (text.includes('changed')) h4.classList.add('badge-changed');
            else if (text.includes('fixed')) h4.classList.add('badge-fixed');
            else if (text.includes('technical')) h4.classList.add('badge-technical');
            else if (text.includes('security')) h4.classList.add('badge-security');
            else if (text.includes('removed')) h4.classList.add('badge-removed');
        });
    }

    toggleChangelog(version) {
        const content = document.getElementById(`content-${version}`);
        const chevron = document.getElementById(`chevron-${version}`);
        
        const isOpen = content.style.display === 'block';
        
        // Close all other changelogs
        document.querySelectorAll('.changelog-content').forEach(el => {
            el.style.display = 'none';
        });
        document.querySelectorAll('.changelog-chevron').forEach(el => {
            el.style.transform = 'rotate(0deg)';
        });
        
        // Toggle current changelog
        if (!isOpen) {
            content.style.display = 'block';
            chevron.style.transform = 'rotate(180deg)';
        }
    }

    formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString('ru-RU', {
            year: 'numeric',
            month: 'long',
            day: 'numeric'
        });
    }

    renderMarkdown(markdown) {
        // Split into lines
        const lines = markdown.split('\n');
        const result = [];
        let inList = false;
        
        for (let i = 0; i < lines.length; i++) {
            let line = lines[i].trim();
            
            // Skip empty lines
            if (!line) {
                if (inList) {
                    result.push('</ul>');
                    inList = false;
                }
                continue;
            }
            
            // Headers
            if (line.startsWith('### ')) {
                if (inList) {
                    result.push('</ul>');
                    inList = false;
                }
                line = line.replace(/^### (.+)$/, '<h4>$1</h4>');
                result.push(line);
                continue;
            }
            
            if (line.startsWith('## ')) {
                if (inList) {
                    result.push('</ul>');
                    inList = false;
                }
                line = line.replace(/^## (.+)$/, '<h3>$1</h3>');
                result.push(line);
                continue;
            }
            
            // List items
            if (line.startsWith('- ')) {
                if (!inList) {
                    result.push('<ul>');
                    inList = true;
                }
                line = line.replace(/^- (.+)$/, '<li>$1</li>');
            } else if (line.startsWith('  -')) {
                // Nested list item (keep as regular list item)
                if (!inList) {
                    result.push('<ul>');
                    inList = true;
                }
                line = line.replace(/^\s+- (.+)$/, '<li>$1</li>');
            } else {
                if (inList) {
                    result.push('</ul>');
                    inList = false;
                }
            }
            
            // Bold
            line = line.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
            
            // Code
            line = line.replace(/`([^`]+)`/g, '<code>$1</code>');
            
            result.push(line);
        }
        
        // Close list if still open
        if (inList) {
            result.push('</ul>');
        }
        
        return result.join('\n');
    }
}

// Initialize page
const aboutPage = new AboutPage();

document.addEventListener('DOMContentLoaded', () => {
    aboutPage.init();
});



