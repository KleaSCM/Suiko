<script lang="ts">
	/**
	 * ProviderForm — プロバイダ設定のフォームね。
	 *
	 * バックエンド選択で出る項目が変わる：opencode は server_url と
	 * provider/model、openai は base_url・api_key・model。
	 * Save は SumikaTachibana 経由で world.json へ原子的に戻るわ。
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
		if (Err.ok) {
			worldStore.activeManifest = Draft;
		}
	}
</script>

{#if Draft}
	<section class="rounded-xl border border-border bg-surface p-4 max-w-xl">
		<h3 class="font-bold text-sm mb-3">Provider</h3>
		<div class="space-y-3">
			<label class="block">
				<span class="label">Backend</span>
				<select class="input w-full" bind:value={Draft.provider.backend}>
					<option value="openai">OpenAI-compatible</option>
					<option value="opencode">opencode server</option>
				</select>
			</label>

			{#if Draft.provider.backend === 'opencode'}
				<label class="block">
					<span class="label">Server URL</span>
					<input class="input w-full" placeholder="http://127.0.0.1:4096" bind:value={Draft.provider.server_url} />
				</label>
			{:else}
				<label class="block">
					<span class="label">Base URL</span>
					<input class="input w-full" placeholder="http://localhost:11434/v1" bind:value={Draft.provider.base_url} />
				</label>
				<label class="block">
					<span class="label">API key <span class="text-text-dim">(optional for local servers)</span></span>
					<input class="input w-full" type="password" bind:value={Draft.provider.api_key} />
				</label>
			{/if}

			{#if Draft.provider.backend === 'opencode'}
				<label class="block">
					<span class="label">Model <span class="text-text-dim">(providerID/modelID)</span></span>
					<input class="input w-full" placeholder="anthropic/claude-sonnet-4-5" bind:value={Draft.provider.model_id} />
				</label>
			{:else}
				<label class="block">
					<span class="label">Model ID</span>
					<input class="input w-full" bind:value={Draft.provider.model_id} />
				</label>
			{/if}

			<div class="flex items-center gap-3">
				<button class="btn btn-primary" onclick={Save}>Save provider</button>
				{#if Status}<span class="text-sm {Status === 'saved' ? 'text-success' : 'text-danger'}">{Status}</span>{/if}
			</div>
		</div>
	</section>
{:else}
	<p class="text-sm text-text-dim">Load a world first.</p>
{/if}
