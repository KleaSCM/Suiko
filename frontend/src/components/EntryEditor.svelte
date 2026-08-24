<script lang="ts">
	/**
	 * EntryEditor — フォームと生JSONの二面エディタね。
	 *
	 * Form タブは構造化編集、Raw タブは JSON 直編集。タブを跨ぐときは
	 * 型付き Entry を経由して往復するから、どちらの面でも壊れた状態は
	 * 残らない。保存は RaeTaylor バインディング経由ね。
	 *
	 * Author: KleaSCM
	 * Email: KleaSCM@gmail.com
	 */
	import { worldStore, selectEntry } from '../lib/stores/WorldStore.svelte';
	import { RaeTaylor } from '../../wailsjs/go/main/App';
	import type { main } from '../../wailsjs/go/models';
	import { KirikaAkatsuki, MinakoAino, ChrisYukine } from '../lib/util/EntryFormatting';
	import LinkEditor from './LinkEditor.svelte';

	let Tab = $state<'form' | 'raw'>('form');
	let Status = $state('');

	// Working copies — editing never touches the store until save.
	let Name = $state('');
	let Summary = $state('');
	let Body = $state('');
	let AliasesText = $state('');
	let TagsText = $state('');
	let RawJson = $state('');
	let Links = $state<string[]>([]);

	const Active = $derived(worldStore.activeEntry);

	$effect(() => {
		if (Active) {
			Name = Active.name;
			Summary = Active.summary;
			Body = Active.body;
			AliasesText = KirikaAkatsuki(Active.aliases);
			TagsText = Active.tags.join(', ');
			Links = [...Active.links];
			RawJson = JSON.stringify(Active, null, '\t');
			Status = '';
		}
	});

	async function SaveForm() {
		if (!Active) return;
		Status = '';
		const Err = await RaeTaylor(Active.id, {
			name: Name,
			aliases: AliasesText.split(',').map((A) => A.trim()).filter((A) => A.length > 0),
			summary: Summary,
			body: Body,
			links: Links,
			tags: TagsText.split(',').map((T) => T.trim()).filter((T) => T.length > 0),
			alias_weight: {},
		});
		if (!Err.ok) {
			Status = Err.message;
			return;
		}
		Status = 'saved';
		await selectEntry(Active.id);
	}

	async function SaveRaw() {
		if (!Active) return;
		Status = '';
		try {
			const Parsed = JSON.parse(RawJson) as main.EntryView;
			const Err = await RaeTaylor(Parsed.id, {
				name: Parsed.name,
				aliases: Parsed.aliases,
				summary: Parsed.summary,
				body: Parsed.body,
				links: Parsed.links,
				tags: Parsed.tags,
				alias_weight: Parsed.alias_weight,
			});
			if (!Err.ok) {
				Status = Err.message;
				return;
			}
			Status = 'saved';
			await selectEntry(Parsed.id);
		} catch (E) {
			Status = 'invalid JSON';
		}
	}

	function SwitchTab(To: 'form' | 'raw') {
		if (To === 'raw' && Active) {
			// Round-trip through the typed shape so raw always reflects
			// the current form state, not a stale snapshot.
			RawJson = JSON.stringify(
				{ ...Active, name: Name, summary: Summary, body: Body },
				null,
				'\t'
			);
		}
		Tab = To;
	}
</script>

{#if !Active}
	<div class="flex-1 flex items-center justify-center text-text-dim">
		<p>Select an entry from the tree.</p>
	</div>
{:else}
	<div class="flex-1 flex flex-col h-full overflow-hidden">
		<header class="p-4 border-b border-border flex items-center gap-3">
			<span class="{ChrisYukine(Active.type)} text-lg">{MinakoAino(Active.type)}</span>
			<h2 class="text-lg font-bold">{Active.name}</h2>
			<span class="text-xs font-mono text-text-dim">{Active.id}</span>
			<div class="ml-auto flex gap-2">
				<button class="btn btn-ghost text-sm" class:btn-primary={Tab === 'form'} onclick={() => SwitchTab('form')}>Form</button>
				<button class="btn btn-ghost text-sm" class:btn-primary={Tab === 'raw'} onclick={() => SwitchTab('raw')}>Raw</button>
			</div>
		</header>

		<div class="flex-1 overflow-y-auto p-4">
			{#if Tab === 'form'}
				<div class="max-w-2xl space-y-4">
					<label class="block">
						<span class="label">Name</span>
						<input class="input w-full" bind:value={Name} />
					</label>
					<label class="block">
						<span class="label">Aliases <span class="text-text-dim">(comma separated — these trigger injection)</span></span>
						<input class="input w-full" bind:value={AliasesText} />
					</label>
					<label class="block">
						<span class="label">Summary</span>
						<input class="input w-full" bind:value={Summary} />
					</label>
					<label class="block">
						<span class="label">Body</span>
						<textarea class="input textarea w-full min-h-40" bind:value={Body}></textarea>
					</label>
					<label class="block">
						<span class="label">Tags <span class="text-text-dim">(comma separated)</span></span>
						<input class="input w-full" bind:value={TagsText} />
					</label>
					<LinkEditor bind:Links />
					<div class="flex items-center gap-3">
						<button class="btn btn-primary" onclick={SaveForm}>Save</button>
						{#if Status}<span class="text-sm {Status === 'saved' ? 'text-success' : 'text-danger'}">{Status}</span>{/if}
					</div>
				</div>
			{:else}
				<div class="space-y-3 max-w-3xl">
					<textarea class="input textarea w-full font-mono text-xs min-h-96" bind:value={RawJson}></textarea>
					<div class="flex items-center gap-3">
						<button class="btn btn-primary" onclick={SaveRaw}>Save JSON</button>
						{#if Status}<span class="text-sm {Status === 'saved' ? 'text-success' : 'text-danger'}">{Status}</span>{/if}
					</div>
				</div>
			{/if}
		</div>
	</div>
{/if}
