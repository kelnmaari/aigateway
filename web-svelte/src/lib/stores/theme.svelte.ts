import { browser } from '$app/environment';

type Theme = 'light' | 'dark' | 'system';

const STORAGE_KEY = 'aigateway_theme';

class ThemeStore {
	private _theme = $state<Theme>('system');
	private _resolved = $state<'light' | 'dark'>('dark');
	private _initialized = false;

	init() {
		if (!browser || this._initialized) return;
		this._initialized = true;

		// Load saved preference
		const saved = localStorage.getItem(STORAGE_KEY) as Theme | null;
		if (saved && ['light', 'dark', 'system'].includes(saved)) {
			this._theme = saved;
		}

		// Resolve actual theme
		this.resolve();

		// Listen for system preference changes
		window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
			if (this._theme === 'system') {
				this.resolve();
			}
		});
	}

	private resolve() {
		if (!browser) return;

		let resolved: 'light' | 'dark';

		if (this._theme === 'system') {
			resolved = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
		} else {
			resolved = this._theme;
		}

		this._resolved = resolved;
		this.applyToDOM(resolved);
	}

	private applyToDOM(theme: 'light' | 'dark') {
		const root = document.documentElement;

		if (theme === 'dark') {
			root.classList.add('dark');
		} else {
			root.classList.remove('dark');
		}

		// Update meta theme-color for mobile browsers
		const metaThemeColor = document.querySelector('meta[name="theme-color"]');
		if (metaThemeColor) {
			metaThemeColor.setAttribute('content', theme === 'dark' ? '#0f172a' : '#ffffff');
		}
	}

	get theme() {
		return this._theme;
	}

	get resolved() {
		return this._resolved;
	}

	get isDark() {
		return this._resolved === 'dark';
	}

	set(theme: Theme) {
		this._theme = theme;

		if (browser) {
			localStorage.setItem(STORAGE_KEY, theme);
			this.resolve();
		}
	}

	toggle() {
		this.set(this._resolved === 'dark' ? 'light' : 'dark');
	}
}

export const themeStore = new ThemeStore();

