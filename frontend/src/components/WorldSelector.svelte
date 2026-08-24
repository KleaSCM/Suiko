<script lang="ts">
	/**
	 * WorldSelector — 初回起動時のワールド選択モーダルね。
	 *
	 * worlds/ の一覧から選ぶか、ダイアログで任意ディレクトリを指定する。
	 * ワールド未選択の間はアプリの他の面に進ませないの。
	 *
	 * Author: KleaSCM
	 * Email: KleaSCM@gmail.com
	 */
	import { HougetsuShimamura, NodokaManabe } from '../../wailsjs/go/main/App';
	import type { main } from '../../wailsjs/go/models';
	import { worldStore, loadWorlds, openWorld } from '../lib/stores/WorldStore.svelte';

	let Worlds = $state<main.WorldInfo[]>([]);
	let Error = $state('');
	let Loading = $state(false);

	// The store owns the list; we mirror it into local state for the modal.
	loadWorlds().then(() => (Worlds = worldStore.worlds));

	async function Pick(Path: string) {
		Loading = true;
		Error = '';
		const Err = await openWorld(Path);
		Loading = false;
		if (Err) {
			Error = Err;
			return;
		}
	}

	async function Browse() {
		const Dir = await HougetsuShimamura();
		if (Dir !== '') await Pick(Dir);
	}

	// Import an external lorebook directory, refresh the list, and open
	// the freshly created world — which will be playerless, so the
	// character creator takes over from here.
	async function Import() {
		const Dir = await HougetsuShimamura();
		if (Dir === '') return;
		Loading = true;
		Error = '';
		const Res = await NodokaManabe(Dir);
		Loading = false;
		if (!Res.ok) {
			Error = Res.message;
			return;
		}
		await loadWorlds();
		await Pick(Res.path);
	}
</script>

<div class="fixed inset-0 bg-bg/90 backdrop-blur-sm z-50 flex items-center justify-center">
	<div class="w-[480px] max-h-[70vh] rounded-xl border border-border bg-surface shadow-lg flex flex-col animate-fade-in">
		<header class="p-5 border-b border-border text-center">
			<h1 class="text-2xl font-bold tracking-widest text-accent">SUIKO</h1>
			<p class="text-sm text-text-muted mt-1">Choose a world to enter.</p>
		</header>
		<div class="flex-1 overflow-y-auto p-4 space-y-2">
			{#if Error}
				<p class="text-sm text-danger mb-2">{Error}</p>
			{/if}
			{#each Worlds as W (W.path)}
				<button
					class="w-full text-left rounded-lg border border-border bg-surface-alt px-4 py-3 hover:border-accent-dim transition-colors disabled:opacity-50"
					disabled={Loading}
					onclick={() => Pick(W.path)}
				>
					<p class="font-semibold">{W.name}</p>
					{#if W.description}
						<p class="text-xs text-text-muted mt-0.5 line-clamp-2">{W.description}</p>
					{/if}
					<p class="text-[11px] text-text-dim font-mono mt-1">{W.path}</p>
				</button>
			{:else}
				<p class="text-center text-text-dim text-sm py-6">No worlds found in the worlds directory.</p>
			{/each}
		</div>
		<footer class="p-4 border-t border-border space-y-2">
			<button class="btn btn-primary w-full" onclick={Import} disabled={Loading}>
				{Loading ? 'Working…' : 'Import a lorebook directory…'}
			</button>
			<button class="btn btn-ghost w-full" onclick={Browse} disabled={Loading}>
				Browse for a world directory…
			</button>
		</footer>
	</div>
</div>
