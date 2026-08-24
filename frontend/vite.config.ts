import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";

// NOTE(KleaSCM): Wails の dev サーバは wailsjs/ を自動注入するからここでは
// エイリアスだけ張る。本番ビルドでは frontend/dist に静的ファイルを吐くの。
export default defineConfig({
	plugins: [
		tailwindcss(),
		svelte(),
	],
	resolve: {
		alias: {
			// Wails runtime の自動生成バインディングへのエイリアス。
			// `wails dev` が wailsjs/ を作るまでは型スタブが居るわ。
			"$lib": "/src/lib",
			"$views": "/src/views",
			"$components": "/src/components",
		},
	},
	build: {
		outDir: "dist",
		// Wails が embed.FS で丸ごと取り込むから、ソースマップは不要。
		sourcemap: false,
		target: "esnext",
	},
	server: {
		port: 5173,
		strictPort: true,
	},
});
