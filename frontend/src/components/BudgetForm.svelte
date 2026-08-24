<script lang="ts">
	/**
	 * BudgetForm — 注入予算の調整フォームね。
	 *
	 * §5/§7 の調整値をここでいじる。値は保存時に Go 側でクランプされる
	 * から、ここでは正の整数だけを保証するわ。
	 *
	 * Author: KleaSCM
	 * Email: KleaSCM@gmail.com
	 */
	import { SumikaTachibana } from '../../wailsjs/go/main/App';
	import type { world } from '../../wailsjs/go/models';
	import { worldStore } from '../lib/stores/WorldStore.svelte';

	let Status = $state('');
	let Draft = $state<world.WorldManifest | null>(null);

	$effect(() => {
		if (worldStore.activeManifest) {
			Draft = JSON.parse(JSON.stringify(worldStore.activeManifest)) as world.WorldManifest;
			Status = '';
		}
	});

	async function Save() {
		if (!Draft) return;
		const Err = await SumikaTachibana(Draft);
		Status = Err.ok ? 'saved' : Err.message;
		if (Err.ok) worldStore.activeManifest = Draft;
	}

	function Num(E: Event): number {
		return parseInt((E.target as HTMLInputElement).value, 10) || 0;
	}
</script>

{#if Draft}
	<section class="rounded-xl border border-border bg-surface p-4 max-w-xl">
		<h3 class="font-bold text-sm mb-3">Context budget</h3>
		<div class="grid grid-cols-2 gap-3">
			<label class="block">
				<span class="label">Inject max tokens</span>
				<input class="input w-full" type="number" min="1" value={Draft.budget.inject_max_tokens}
					onchange={(E) => (Draft!.budget.inject_max_tokens = Num(E))} />
			</label>
			<label class="block">
				<span class="label">Top-K entries</span>
				<input class="input w-full" type="number" min="1" value={Draft.budget.top_k_entries}
					onchange={(E) => (Draft!.budget.top_k_entries = Num(E))} />
			</label>
			<label class="block">
				<span class="label">Recency boost window</span>
				<input class="input w-full" type="number" min="1" value={Draft.budget.recency_boost_turns}
					onchange={(E) => (Draft!.budget.recency_boost_turns = Num(E))} />
			</label>
			<label class="block">
				<span class="label">Dedup window (turns)</span>
				<input class="input w-full" type="number" min="1" value={Draft.budget.dedup_window_turns}
					onchange={(E) => (Draft!.budget.dedup_window_turns = Num(E))} />
			</label>
		</div>
		<div class="flex items-center gap-3 mt-4">
			<label class="flex items-center gap-2 text-sm">
				<input type="checkbox" bind:checked={Draft.auto_accept_writes} />
				Auto-accept model writes
			</label>
		</div>
		<div class="flex items-center gap-3 mt-4">
			<button class="btn btn-primary" onclick={Save}>Save budget</button>
			{#if Status}<span class="text-sm {Status === 'saved' ? 'text-success' : 'text-danger'}">{Status}</span>{/if}
		</div>
	</section>
{:else}
	<p class="text-sm text-text-dim">Load a world first.</p>
{/if}
