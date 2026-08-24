<script lang="ts">
	import { worldStore } from '../lib/stores/WorldStore.svelte';
	import ProviderForm from '../components/ProviderForm.svelte';
	import BudgetForm from '../components/BudgetForm.svelte';
</script>

<div class="flex-1 flex flex-col h-full animate-fade-in overflow-hidden">
	<header class="p-4 border-b border-border bg-surface-hi">
		<h2 class="text-lg font-bold">Settings</h2>
	</header>

	<div class="flex-1 overflow-y-auto p-4 space-y-4">
		{#if !worldStore.activeManifest}
			<p class="text-sm text-text-dim">Load a world to edit its settings.</p>
		{/if}
		<ProviderForm />
		<BudgetForm />

		<section class="rounded-xl border border-border bg-surface p-4 max-w-xl">
			<h3 class="font-bold text-sm mb-2">MCP server</h3>
			<p class="text-sm text-text-muted leading-relaxed">
				Run <code class="font-mono text-accent">suiko serve &lt;world-dir&gt;</code> to expose this
				world over stdio MCP — opencode, Zed, or any MCP host can then drive the same lore.
			</p>
		</section>
	</div>
</div>
