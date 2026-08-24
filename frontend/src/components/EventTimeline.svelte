<script lang="ts">
	/**
	 * EventTimeline — セッション履歴の年表ビューね。
	 *
	 * イベントを時系列カードで並べる。種別バッジ、参加者、舞台を
	 * 一目で拾える形に。履歴は荷重構造 — 見た目にもそれ相応に。
	 *
	 * Author: KleaSCM
	 * Email: KleaSCM@gmail.com
	 */
	import type { main } from '../../wailsjs/go/models';
	import { MichiruKaioh } from '../lib/util/TimeFormat';

	let { events }: { events: main.EventView[] } = $props();

	const KindColors: Record<string, string> = {
		scene: 'bg-info/20 text-info',
		move: 'bg-type-location/20 text-type-location',
		thread: 'bg-fired/20 text-fired-text',
		resolution: 'bg-success/20 text-success',
		offscreen: 'bg-surface-hi text-text-muted',
		note: 'bg-surface-hi text-text-muted',
	};

	function KindClass(Kind: string): string {
		return KindColors[Kind] ?? 'bg-surface-hi text-text-muted';
	}
</script>

{#if events.length === 0}
	<p class="p-6 text-center text-text-dim text-sm">No events recorded yet.</p>
{:else}
	<ol class="space-y-2">
		{#each events as Ev, I (I)}
			<li class="rounded-lg border border-border bg-surface px-4 py-2.5 animate-fade-in">
				<div class="flex items-center gap-2 text-xs mb-1">
					<span class="px-2 py-0.5 rounded-full {KindClass(Ev.kind)}">{Ev.kind}</span>
					<span class="text-text-dim font-mono">turn {Ev.turn}</span>
					<span class="ml-auto text-text-dim">{MichiruKaioh(Ev.timestamp)}</span>
				</div>
				<p class="text-sm leading-snug">{Ev.text}</p>
				<div class="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-[11px] text-text-muted">
					{#if Ev.location}
						<span>at <span class="font-mono">{Ev.location}</span></span>
					{/if}
					{#each Ev.participants as P (P)}
						<span class="font-mono">{P}</span>
					{/each}
				</div>
			</li>
		{/each}
	</ol>
{/if}
