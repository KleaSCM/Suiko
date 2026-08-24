<script lang="ts">
	/**
	 * DigestCard — 圧縮済み履歴の「これまでのあらすじ」カードね。
	 *
	 * Tier 3 の要約はイベントから決定的に抽出される。ここではそれを
	 * 読みやすい形で見せるだけ — 編集はしない。履歴は正史じゃないから、
	 * 表示も謙虚にするの。
	 *
	 * Author: KleaSCM
	 * Email: KleaSCM@gmail.com
	 */
	let { lines }: { lines: string[] } = $props();

	let Open = $state(false);
</script>

{#if lines.length > 0}
	<section class="rounded-xl border border-border-dim bg-surface-alt/60 p-3">
		<button class="text-xs uppercase tracking-wider text-text-muted hover:text-text transition-colors" onclick={() => (Open = !Open)}>
			{Open ? '▾' : '▸'} Story so far ({lines.length})
		</button>
		{#if Open}
			<ul class="mt-2 space-y-1 text-sm text-text-muted">
				{#each lines as Line, I (I)}
					<li class="leading-snug"><span class="text-text-dim mr-1">·</span>{Line}</li>
				{/each}
			</ul>
		{/if}
	</section>
{/if}
