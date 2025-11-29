import { browser } from '$app/environment';
import en from '../../../messages/en.json';
import ru from '../../../messages/ru.json';

type Locale = 'en' | 'ru';
type Messages = typeof en;

const messages: Record<Locale, Messages> = { en, ru };

const STORAGE_KEY = 'aigateway_locale';
const DEFAULT_LOCALE: Locale = 'en';

class I18n {
	private _locale = $state<Locale>(DEFAULT_LOCALE);
	private _initialized = false;

	init() {
		if (!browser || this._initialized) return;
		this._initialized = true;

		const saved = localStorage.getItem(STORAGE_KEY) as Locale | null;
		if (saved && (saved === 'en' || saved === 'ru')) {
			this._locale = saved;
		} else {
			const browserLang = navigator.language.split('-')[0];
			if (browserLang === 'ru') {
				this._locale = 'ru';
			}
		}

		document.documentElement.lang = this._locale;
	}

	get locale() {
		return this._locale;
	}

	set(locale: Locale) {
		this._locale = locale;
		if (browser) {
			localStorage.setItem(STORAGE_KEY, locale);
			document.documentElement.lang = locale;
		}
	}

	t(key: string): string {
		const keys = key.split('_');
		let value: unknown = messages[this._locale];

		for (const k of keys) {
			if (value && typeof value === 'object' && k in value) {
				value = (value as Record<string, unknown>)[k];
			} else {
				// Fallback to English
				value = messages.en;
				for (const fallbackKey of keys) {
					if (value && typeof value === 'object' && fallbackKey in value) {
						value = (value as Record<string, unknown>)[fallbackKey];
					} else {
						return key; // Return key if not found
					}
				}
				break;
			}
		}

		return typeof value === 'string' ? value : key;
	}
}

export const i18n = new I18n();

// Helper function for templates
export function t(key: string): string {
	return i18n.t(key);
}

