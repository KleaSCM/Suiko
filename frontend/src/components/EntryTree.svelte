<script lang="ts">
	/**
	 * EntryTree — 型ごとにグルーピングしたワールドツリーね。
	 *
	 * ファジーフィルタは名前・要約・エイリアスの包含一致。選択は
	 * WorldStore の activeEntry へ流れる — エディタはそれを見るだけ。
	 *
	 * Author: KleaSCM
	 * Email: KleaSCM@gmail.com
	 */
	import type { main } from '../../wailsjs/go/models';
	import { worldStore } from '../lib/stores/WorldStore.svelte';
	import { MinakoAino, ReiHino, ChrisYukine } from '../lib/util/EntryFormatting';

	let Filter = $state('');

	const TypeOrder = ['player', 'character', 'location', 'item', 'faction', 'lore'];

	let Filtered = $derived(
		worldStore.tree.filter((E) => {
			if (Filter === '') return true;
			const Q = Filter.toLowerCase();
			return (
				E.name.toLowerCase().includes(Q) ||
				E.summary.toLowerCase().includes(Q) ||
				E.id.toLowerCase().includes(Q) ||
				E.aliases.some((A) => A.toLowerCase().includes(Q))
			);
		})
	);

	function ByType(Type: string): main.EntryView[] {
		return Filtered.filter((E) => E.type === Type);
	}
</script>

<div class="flex flex-col h-full">
	<input class="input m-3" placeholder="Filter lore..." bind:value={Filter} />
	<nav class="flex-1 overflow-y-auto px-2 pb-3 space-y-3">
		{#each TypeOrder as Type (Type)}
			{@const Group = ByType(Type)}
			{#if Group.length > 0}
				<section>
					<h3 class="px-2 py-1 text-xs uppercase tracking-wider text-text-dim">
						{ReiHino(Type)} ({Group.length})
					</h3>
					{#each Group as E (E.id)}
						<button
							class="w-full text-left px-2 py-1.5 rounded-md flex items-center gap-2 text-sm transition-colors
								{worldStore.activeEntry?.id === E.id ? 'bg-accent-dim/40 text-text' : 'hover:bg-surface-alt text-text-muted'}"
							onclick={() => (worldStore.activeEntry = E)}
						>
							<span class="{ChrisYukine(E.type)} shrink-0">{MinakoAino(E.type)}</span>
							<span class="truncate">{E.name}</span>
							{#if E.sovereign}
								<span class="ml-auto text-[10px] text-type-player">PC</span>
							{/if}
						</button>
					{/each}
				</section>
			{/if}
		{/each}
		{#if Filtered.length === 0}
			<p class="p-4 text-sm text-text-dim text-center">No entries match “{Filter}”.</p>
		{/if}
	</nav>
</div>
