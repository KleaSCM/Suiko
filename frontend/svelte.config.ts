import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

export default {
	preprocess: vitePreprocess(),
	compilerOptions: {
		// Svelte 5 runes モード — 全コンポーネントでデフォルト有効にするの。
		runes: true,
	},
};
