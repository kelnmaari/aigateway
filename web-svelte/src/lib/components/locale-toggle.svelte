<script lang="ts">
	import { FontAwesomeIcon } from '@fortawesome/svelte-fontawesome';
	import { faLanguage } from '@fortawesome/free-solid-svg-icons';
	import { Button } from '$lib/components/ui/button';
	import { locales, getLocale, setLocale } from '$lib/paraglide/runtime';
	import { m } from '$lib/paraglide/messages';

	const STORAGE_KEY = 'PARAGLIDE_LOCALE';

	function toggleLocale() {
		const current = getLocale();
		const currentIndex = locales.indexOf(current);
		const nextIndex = (currentIndex + 1) % locales.length;
		const newLocale = locales[nextIndex];

		setLocale(newLocale);
		localStorage.setItem(STORAGE_KEY, newLocale);

		// Force page reload to apply new locale
		window.location.reload();
	}

	function getLocaleLabel(locale: string): string {
		switch (locale) {
			case 'en': return 'English';
			case 'ru': return 'Русский';
			default: return locale.toUpperCase();
		}
	}
</script>

<Button variant="ghost" size="icon" onclick={toggleLocale} title="Change language">
	<FontAwesomeIcon icon={faLanguage} class="h-5 w-5" />
	<span class="sr-only">Toggle language</span>
</Button>
