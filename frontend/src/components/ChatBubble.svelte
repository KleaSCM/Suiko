<script lang="ts">
	/**
	 * ChatBubble — ユーザーかナレーターの発言ひとつぶんね。
	 *
	 * ユーザーは右寄せのアクセント色、ナレーターは左寄せのサーフェス色。
	 * ナレーター発言には発火ロアカードを折り畳みで添える — プレイの
	 * 透明性をここに置くの。
	 *
	 * Author: KleaSCM
	 * Email: KleaSCM@gmail.com
	 */
	import type { Message } from '../lib/stores/SessionStore.svelte';
	import LoreCard from './LoreCard.svelte';

	let { message }: { message: Message } = $props();

	let LoreOpen = $state(false);
</script>

{#if message.role === 'user'}
	<div class="flex justify-end animate-fade-in">
		<div class="max-w-[75%] rounded-xl rounded-br-sm bg-accent-dim/40 border border-accent-dim px-4 py-2.5">
			<p class="whitespace-pre-wrap leading-relaxed">{message.text}</p>
		</div>
	</div>
{:else}
	<div class="flex flex-col gap-1.5 animate-fade-in">
		<div class="max-w-[85%] rounded-xl rounded-bl-sm bg-surface-alt border border-border px-4 py-2.5">
			<p class="whitespace-pre-wrap leading-relaxed">{message.text}</p>
		</div>
		{#if message.firedEntries && message.firedEntries.length > 0}
			<button
				class="self-start text-xs text-fired-text/70 hover:text-fired-text transition-colors"
				onclick={() => (LoreOpen = !LoreOpen)}
			>
				{LoreOpen ? '▾' : '▸'} lore fired this turn ({message.firedEntries.length})
			</button>
			{#if LoreOpen}
				<div class="flex flex-col gap-1.5 pl-3 border-l border-fired-dim ml-2">
					{#each message.firedEntries as Entry (Entry.id)}
						<LoreCard entry={Entry} />
					{/each}
				</div>
			{/if}
		{/if}
	</div>
{/if}
