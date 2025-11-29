/**
 * Locale store integrated with Paraglide JS
 */
import { locales, baseLocale, setLocale as paraglideSetLocale, getLocale as paraglideGetLocale } from '$lib/paraglide/runtime';

export type Locale = typeof locales[number];

const STORAGE_KEY = 'PARAGLIDE_LOCALE';

/**
 * Get locale from localStorage on client
 */
function getStoredLocale(): Locale {
	if (typeof window === 'undefined') return baseLocale as Locale;
	const stored = localStorage.getItem(STORAGE_KEY);
	if (stored && locales.includes(stored as Locale)) {
		return stored as Locale;
	}
	return baseLocale as Locale;
}

/**
 * Initialize locale from storage
 */
export function initLocale(): void {
	if (typeof window === 'undefined') return;
	const stored = getStoredLocale();
	paraglideSetLocale(stored);
}

/**
 * Get current locale
 */
export function getCurrentLocale(): Locale {
	return paraglideGetLocale() as Locale;
}

/**
 * Set locale and save to storage
 */
export function setCurrentLocale(locale: Locale): void {
	paraglideSetLocale(locale);
	if (typeof window !== 'undefined') {
		localStorage.setItem(STORAGE_KEY, locale);
	}
}

/**
 * Toggle between available locales
 */
export function toggleLocale(): void {
	const current = getCurrentLocale();
	const currentIndex = locales.indexOf(current);
	const nextIndex = (currentIndex + 1) % locales.length;
	setCurrentLocale(locales[nextIndex]);
}

// Export locales for use in components
export { locales, baseLocale };
