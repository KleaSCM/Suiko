<script lang="ts">
	/**
	 * PendingReviewPanel — モデル書き込みの審査パネルね。
	 *
	 * 待ち行列の各項目を Accept / Reject できる。既定は人間の拒否権 —
	 * ここを素通りさせないのが設計書 §8 の約束なの。
	 *
	 * Author: KleaSCM
	 * Email: KleaSCM@gmail.com
	 */
	import { pendingStore, acceptPending, rejectPending } from '../lib/stores/PendingStore.svelte';
	import { MinakoAino, ReiHino } from '../lib/util/EntryFormatting';
	import { TiltyClaret } from '../../wailsjs/go/main/App';
	import { worldStore } from '../lib/stores/WorldStore.svelte';

	let BusyId = $state('');
	let Error = $state('');

	async function Accept(Id: string) {
		BusyId = Id;
		Error = '';
		const Err = await acceptPending(Id);
		BusyId = '';
		if (Err) { Error = Err; return; }
		// Accepted entries must appear in the tree immediately.
		worldStore.tree = await TiltyClaret();
	}

	async function Reject(Id: string) {
		BusyId = Id;
		Error = await rejectPending(Id) ?? '';
		BusyId = '';
	}
</script>

<section class="rounded-xl border border-warn/40 bg-surface p-4">
	<h3 class="font-bold text-sm mb-3 flex items-center gap-2">
		Pending model writes
		<span class="px-1.5 py-0.5 rounded-full bg-warn/20 text-warn text-xs">{pendingStore.items.length}</span>
	</h3>
	{#if Error}
		<p class="mb-2 text-sm text-danger">{Error}</p>
	{/if}
	{#if pendingStore.items.length === 0}
		<p class="text-sm text-text-dim">Nothing waiting for review.</p>
	{:else}
		<ul class="space-y-2">
			{#each pendingStore.items as Item (Item.id)}
				<li class="flex items-center gap-3 rounded-lg bg-surface-alt border border-border px-3 py-2">
					<span>{MinakoAino(Item.type)}</span>
					<div class="min-w-0 flex-1">
						<p class="text-sm font-semibold truncate">{Item.name}</p>
						<p class="text-xs text-text-muted truncate">{Item.summary}</p>
						<p class="text-[11px] text-text-dim font-mono">{Item.id} · {ReiHino(Item.type)}</p>
					</div>
					<button class="btn btn-primary text-xs" disabled={BusyId === Item.id} onclick={() => Accept(Item.id)}>
						Accept
					</button>
					<button class="btn btn-danger text-xs" disabled={BusyId === Item.id} onclick={() => Reject(Item.id)}>
						Reject
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</section>
