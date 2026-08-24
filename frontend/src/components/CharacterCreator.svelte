<script lang="ts">
	/**
	 * CharacterCreator — プレイヤーキャラクターの創造面ね。
	 *
	 * 名前さえ決まれば世界に入れる。既存のキャラシートを丸ごと貼っても
	 * いい — エンジンのパーサーが節（· Identity: … / [APPEARANCE] …）を
	 * 認識して要約と本文を組み立てるの。空欄は全部容認されるわ。
	 *
	 * Author: KleaSCM
	 * Email: KleaSCM@gmail.com
	 */
	import { UiHirasawa } from '../../wailsjs/go/main/App';
	import { worldStore, selectEntry } from '../lib/stores/WorldStore.svelte';
	import { navigate } from '../lib/stores/NavStore.svelte';
	import { slugifyId } from '../lib/util/EntryFormatting';

	let Name = $state('');
	let AliasesText = $state('');
	let Card = $state('');
	let Error = $state('');

	async function Create() {
		Error = '';
		const Err = await UiHirasawa(
			Name,
			AliasesText.split(',').map((A) => A.trim()).filter((A) => A.length > 0),
			'', // summary is derived from the card by the engine
			Card
		);
		if (!Err.ok) {
			Error = Err.message;
			return;
		}
		await selectEntry('player/' + slugifyId(Name));
		worldStore.needsPlayer = false;
		navigate('play');
	}
</script>

<div class="fixed inset-0 bg-bg/90 backdrop-blur-sm z-50 flex items-center justify-center">
	<div class="w-[640px] max-h-[88vh] rounded-xl border border-accent-dim bg-surface shadow-lg flex flex-col animate-fade-in">
		<header class="p-5 border-b border-border text-center">
			<h1 class="text-xl font-bold text-accent">Create your character</h1>
			<p class="text-sm text-text-muted mt-1">
				{worldStore.activeManifest?.name} awaits its protagonist. Only you control her.
			</p>
		</header>
		<div class="flex-1 overflow-y-auto p-5 space-y-4">
			<label class="block">
				<span class="label">Name <span class="text-danger">*</span></span>
				<input class="input w-full" placeholder="e.g. Yuriko" bind:value={Name} />
			</label>
			<label class="block">
				<span class="label">Aliases <span class="text-text-dim">(optional, comma separated)</span></span>
				<input class="input w-full" placeholder="Riko, huntress…" bind:value={AliasesText} />
			</label>
			<label class="block">
				<span class="label">Character sheet <span class="text-text-dim">(paste anything — sections are recognized automatically)</span></span>
				<textarea
					class="input textarea w-full min-h-64 font-mono text-xs"
					placeholder={'· Identity: 22-year-old Kyūbi huntress…\n· Appearance: petite frame, black hair…\n· Clothing: crimson Qipao…\n· Persona: playful, cheeky…\n· Magic: blinks through space…\n· Drive: prove herself as a huntress…'}
					bind:value={Card}
				></textarea>
			</label>
			<p class="text-xs text-text-dim leading-relaxed">
				Recognized sections: Identity · Appearance · Physical · Clothing · Persona · Magic · Drive.
				Anything left blank simply won't exist yet — you and the story fill it in as you play.
			</p>
			{#if Error}
				<p class="text-sm text-danger">{Error}</p>
			{/if}
		</div>
		<footer class="p-4 border-t border-border flex justify-end gap-2">
			<button class="btn btn-primary" disabled={!Name.trim()} onclick={Create}>
				Enter the world
			</button>
		</footer>
	</div>
</div>
