/**
 * Single source of truth for Fairy C's verified support snapshot and Stage 0 status markers.
 * Validated across all locale status-and-limits pages by scripts/check-navigation.mjs.
 */
export const supportStatus = {
	baseline: {
		date: '2026-11-16',
		commit: 'e1d48cb7',
	},
	requiredTokens: [
		'Stage 0',
		'x86_64-windows-msvc',
		'x86_64-linux-gnu',
		'supported: no',
		'E2104',
		'E5104',
		'@repr(C)',
		'fairy.toml',
		'fairy.lock',
	],
};
