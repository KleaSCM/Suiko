<script lang="ts">
	/**
	 * LoreCard — 発火したロアのインラインカードね。
	 *
	 * チャットストリームの中に、このターンで注入されたエントリを
	 * 透明な形で見せる。「Dig deeper」でサイドパネルに全文を開くわ。
	 * 注入は魔法じゃなくて見えるもの — それがこのカードの存在意義。
	 *
	 * Author: KleaSCM
	 * Email: KleaSCM@gmail.com
	 */
	import type { main } from '../../wailsjs/go/models';
	import { MinakoAino, ReiHino, KirikaAkatsuki, ChrisYukine } from '../lib/util/EntryFormatting';
	import { selectEntry } from '../lib/stores/WorldStore.svelte';
	import { navigate } from '../lib/stores/NavStore.svelte';

	let { entry }: { entry: main.EntryView } = $props();

	// Dig deeper: load the full entry and jump to the World view editor.
	async function Deeper() {
		await selectEntry(entry.id);
		navigate('world');
	}
</script>

<div class="rounded-lg border border-fired-dim bg-fired/5 px-3 py-2 text-sm">
	<button
		class="flex items-center gap-2 w-full text-left group"
		onclick={Deeper}
		title="Dig deeper — open {entry.id}"
	>
		<span class="{ChrisYukine(entry.type)}">{MinakoAino(entry.type)}</span>
		<span class="font-semibold text-fired-text group-hover:underline">{entry.name}</span>
		<span class="text-text-dim text-xs uppercase tracking-wide">{ReiHino(entry.type)}</span>
	</button>
	{#if entry.summary}
		<p class="mt-1 text-text-muted text-xs leading-snug">{entry.summary}</p>
	{/if}
	{#if entry.aliases.length > 0}
		<p class="mt-1 text-text-dim text-[11px]">aka {KirikaAkatsuki(entry.aliases)}</p>
	{/if}
</div>
