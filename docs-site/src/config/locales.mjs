export const fallbackLocale = 'en';

export const languagePreference = {
	storageKey: 'fairyc-docs-language',
	defaultLocale: fallbackLocale,
};

export const languageSelectorOptions = [
	{
		code: 'en',
		label: 'English',
		lang: 'en',
		starlightLocale: 'en',
		hasContent: true,
	},
	{
		code: 'ko',
		label: '한국어',
		lang: 'ko',
		starlightLocale: 'ko',
		hasContent: true,
	},
];

export const routedLanguageCodes = languageSelectorOptions
	.filter(({ hasContent }) => hasContent)
	.map(({ code }) => code);

export const unsupportedLanguageCodes = languageSelectorOptions
	.filter(({ hasContent }) => !hasContent)
	.map(({ code }) => code);

export const locales = {
	en: {
		label: 'English',
		lang: 'en',
	},
	ko: {
		label: '한국어',
		lang: 'ko',
	},
};
