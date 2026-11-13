import {
	languagePreference,
	languageSelectorOptions,
} from './locales.mjs';
import { site } from './site.mjs';

const supportedLanguageCodes = languageSelectorOptions.map(({ code }) => code);
const basePath = site.base.replace(/\/$/, '');

const languagePreferenceScript = `
(() => {
	const storageKey = ${JSON.stringify(languagePreference.storageKey)};
	const defaultLocale = ${JSON.stringify(languagePreference.defaultLocale)};
	const supportedLocales = new Set(${JSON.stringify(supportedLanguageCodes)});

	const normalizeLocale = (value) => {
		if (!value || typeof value !== 'string') return undefined;
		const normalized = value.toLowerCase();
		const base = normalized.split('-')[0];
		return supportedLocales.has(base) ? base : undefined;
	};

	const preferredBrowserLocale = () => {
		const languages = Array.isArray(navigator.languages) && navigator.languages.length > 0
			? navigator.languages
			: [navigator.language];

		for (const language of languages) {
			const locale = normalizeLocale(language);
			if (locale) return locale;
		}

		return defaultLocale;
	};

	const storedLocale = () => {
		try {
			return normalizeLocale(localStorage.getItem(storageKey));
		} catch {
			return undefined;
		}
	};

	const storeLocale = (locale) => {
		try {
			localStorage.setItem(storageKey, locale);
		} catch {}
	};

	const localeFromOption = (option) => {
		return normalizeLocale(option.dataset.locale) || defaultLocale;
	};

	const optionForLocale = (select, locale) =>
		Array.from(select.options).find((option) => localeFromOption(option) === locale);

	const applyLocale = (select, locale, shouldRedirect) => {
		const option = optionForLocale(select, locale) || optionForLocale(select, defaultLocale);
		if (!option) return;

		select.selectedIndex = option.index;

		if (!shouldRedirect) return;

		const target = new URL(option.value, window.location.href);
		target.search = window.location.search;
		target.hash = window.location.hash;

		if (target.pathname !== window.location.pathname) {
			window.location.replace(target.href);
		}
	};

	const initLanguagePreference = () => {
		const select = document.querySelector('starlight-lang-select select');
		if (!(select instanceof HTMLSelectElement)) return;

		const stored = storedLocale();
		const preferred = stored || preferredBrowserLocale();
		if (!stored) storeLocale(preferred);

		applyLocale(select, preferred, true);
	};

	document.addEventListener(
		'change',
		(event) => {
			const target = event.target;
			if (!(target instanceof HTMLSelectElement)) return;
			if (!target.matches('starlight-lang-select select')) return;

			const option = target.selectedOptions[0];
			if (!option) return;

			storeLocale(localeFromOption(option));
		},
		true
	);

	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', initLanguagePreference, { once: true });
	} else {
		initLanguagePreference();
	}
})();
`.trim();

/** @type {import('@astrojs/starlight/types').StarlightUserConfig['head']} */
export const head = [
	{
		tag: 'script',
		content: languagePreferenceScript,
	},
	{
		tag: 'script',
		attrs: {
			type: 'module',
			src: `${basePath}/keyboard-navigation.js`,
		},
	},
];
