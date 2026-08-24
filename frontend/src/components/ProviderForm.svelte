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
	import { SumikaTachibana, NagisaKiryu } from '../../wailsjs/go/main/App';
	import { world, provider } from '../../wailsjs/go/models';
	import { worldStore } from '../lib/stores/WorldStore.svelte';

	let Status = $state('');
	let Draft = $state<world.WorldManifest | null>(null);

	// opencode バックエンド用のモデル一覧。サーバが応答しないときはテキスト入力へ
	// フォールバックするから、ここが空でも画面は壊れないわ。
	let ModelOptions = $state<provider.ModelOption[]>([]);
	let ModelStatus = $state('');
	let ModelLoading = $state(false);
	let LoadTimer: ReturnType<typeof setTimeout> | undefined;

	// opencode の既定 — "suiko" は Suiko 専有インスタンスの合図。URL 未入力、
	// あるいは "suiko" のままなら、Suiko が自分用の opencode を起動して繋ぐ。
	// Hanako の人格や MCP が混ざらないよう、既定はもう 4096(Hanako) ではないの。
	const DefaultOpenCodeUrl = 'suiko';

	$effect(() => {
		if (worldStore.activeManifest) {
			Draft = JSON.parse(JSON.stringify(worldStore.activeManifest)) as world.WorldManifest;
			Status = '';
			ModelOptions = [];
			ModelStatus = '';
		}
	});

	// backend が opencode なら一覧を引く。URL が空でも既定ポートへ飛ばすから
	// 空欄のまま自動ロードする。キー入力ごとに叩かないよう 400ms デバウンス。
	$effect(() => {
		const Back = Draft?.provider.backend ?? '';
		const RawUrl = (Draft?.provider.server_url ?? '').trim();
		const Url = Back === 'opencode' && (RawUrl === '' || RawUrl === 'suiko') ? DefaultOpenCodeUrl : RawUrl;
		if (Back === 'opencode' && Url.trim() !== '') {
			clearTimeout(LoadTimer);
			LoadTimer = setTimeout(() => void LoadModels(), 400);
		} else {
			ModelOptions = [];
			ModelStatus = '';
		}
	});

	// opencode を選んだ瞬間、URL が空なら既定ポートを入れて一覧を即ロードするの。
	function OnBackendChange() {
		if (!Draft) return;
		if (Draft.provider.backend === 'opencode' && (Draft.provider.server_url ?? '').trim() === '') {
			Draft.provider.server_url = DefaultOpenCodeUrl;
		}
	}

	async function LoadModels() {
		if (!Draft) return;
		const Url = (Draft.provider.server_url ?? '').trim() || DefaultOpenCodeUrl;
		ModelLoading = true;
		ModelStatus = '';
		try {
			const Res = await NagisaKiryu(Draft.provider.backend, Url);
			if (!Res.ok) {
				ModelStatus = Res.message || 'could not load models';
				ModelOptions = [];
			} else {
				ModelOptions = Res.models ?? [];
			}
		} catch (Err) {
			ModelStatus = String(Err);
			ModelOptions = [];
		} finally {
			ModelLoading = false;
		}
	}

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
				<select class="input w-full" bind:value={Draft.provider.backend} onchange={OnBackendChange}>
					<option value="openai">OpenAI-compatible</option>
					<option value="opencode">opencode server</option>
				</select>
			</label>

			{#if Draft && Draft.provider.backend === 'opencode'}
				<label class="block">
					<span class="label">Server URL</span>
					<input class="input w-full" placeholder="suiko（Suiko 専有インスタンス）または http://host:port" bind:value={Draft.provider.server_url} />
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

			{#if Draft && Draft.provider.backend === 'opencode'}
				<label class="block">
					<span class="label">Model</span>
					{#if Draft}
						{@const CurModel = Draft.provider.model_id}
						<div class="flex gap-2">
							<select class="input w-full" bind:value={Draft.provider.model_id} disabled={ModelLoading}>
								{#if ModelOptions.length === 0}
									<option value="">{ModelLoading ? 'Loading models…' : (ModelStatus ? 'load failed — type below' : 'no models yet')}</option>
								{/if}
								{#if CurModel && !ModelOptions.some((o) => o.value === CurModel)}
									<option value={CurModel}>{CurModel} (saved)</option>
								{/if}
								{#each ModelOptions as Opt (Opt.value)}
									<option value={Opt.value}>{Opt.label}</option>
								{/each}
							</select>
							<button class="btn" type="button" onclick={() => void LoadModels()} disabled={ModelLoading} title="Refresh models">
								{ModelLoading ? '…' : '↻'}
							</button>
						</div>
						<input class="input w-full mt-2" placeholder="anthropic/claude-sonnet-4-5" bind:value={Draft.provider.model_id} />
						{#if ModelStatus}<p class="text-sm text-danger mt-1">{ModelStatus}</p>{/if}
						{#if ModelOptions.length > 0}<p class="text-sm text-text-dim mt-1">{ModelOptions.length} models loaded — pick one or keep typing.</p>{/if}
						{#if ModelOptions.length === 0 && !ModelStatus && !ModelLoading}
							<p class="text-sm text-text-dim mt-1">Set the Server URL above; models load automatically (or click ↻).</p>
						{/if}
					{/if}
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
