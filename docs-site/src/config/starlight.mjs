import { head } from './head.mjs';
import { fallbackLocale, locales } from './locales.mjs';
import { sidebar } from './sidebar.mjs';
import { site } from './site.mjs';
import { customCss } from './styles.mjs';

/** @type {import('@astrojs/starlight/types').StarlightUserConfig} */
export const starlightOptions = {
	title: site.title,
	logo: {
		light: './src/assets/fairyc-logo-horizontal-v1.png',
		dark: './src/assets/fairyc-logo-horizontal-dark-v2.png',
		alt: site.title,
		replacesTitle: true,
	},
	defaultLocale: fallbackLocale,
	disable404Route: true,
	head,
	customCss,
	locales,
	sidebar,
	components: {
		LanguageSelect: './src/components/LanguageSelect.astro',
	},
};
