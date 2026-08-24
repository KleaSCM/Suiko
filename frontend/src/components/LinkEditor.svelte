<script lang="ts">
	/**
	 * LinkEditor — リンク辺のオートコンプリート付き編集器ね。
	 *
	 * 入力候補は全エントリから絞り込む。既存のリンクはチップ表示で、
	 * クリックで外せる。存在しないidへの追加は拒む — 宙ぶらりん辺を
	 * この面から生やさないの。
	 *
	 * Author: KleaSCM
	 * Email: KleaSCM@gmail.com
	 */
	import { worldStore } from '../lib/stores/WorldStore.svelte';

	let { Links = $bindable() }: { Links: string[] } = $props();

	let Query = $state('');

	// Candidates: every entry except self and already-linked ids.
	let Candidates = $derived(
		worldStore.tree.filter((E) => {
			if (!worldStore.activeEntry || E.id === worldStore.activeEntry.id) return false;
			if (Links.includes(E.id)) return false;
			const Q = Query.toLowerCase();
			return (
				Q === '' ||
				E.name.toLowerCase().includes(Q) ||
				E.id.toLowerCase().includes(Q)
			);
		}).slice(0, 8)
	);

	function Add(Id: string) {
		if (Id === '' || Links.includes(Id)) return;
		Links = [...Links, Id];
		Query = '';
	}

	function Remove(Id: string) {
		Links = Links.filter((L) => L !== Id);
	}
</script>

<div>
	<span class="label">Links</span>
	{#if Links.length > 0}
		<div class="flex flex-wrap gap-1.5 mb-2">
			{#each Links as Link (Link)}
				<button
					class="chip chip-removable"
					title="Remove link"
					onclick={() => Remove(Link)}
				>
					{Link} ✕
				</button>
			{/each}
		</div>
	{/if}
	<input class="input w-full" placeholder="Type to search entries to link..." bind:value={Query} />
	{#if Query.length > 0}
		<ul class="mt-1 border border-border rounded-md bg-surface divide-y divide-border-dim max-h-40 overflow-y-auto">
			{#each Candidates as C (C.id)}
				<li>
					<button class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-alt" onclick={() => Add(C.id)}>
						<span class="text-text">{C.name}</span>
						<span class="text-text-dim text-xs font-mono ml-2">{C.id}</span>
					</button>
				</li>
			{/each}
			{#if Candidates.length === 0}
				<li class="px-3 py-1.5 text-sm text-text-dim">No matching entry.</li>
			{/if}
		</ul>
	{/if}
</div>
