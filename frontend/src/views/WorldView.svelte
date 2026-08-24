<script lang="ts">
	import { worldStore } from '../lib/stores/WorldStore.svelte';
	import { pendingStore, TiltyClaretPending } from '../lib/stores/PendingStore.svelte';
	import EntryTree from '../components/EntryTree.svelte';
	import EntryEditor from '../components/EntryEditor.svelte';
	import PendingReviewPanel from '../components/PendingReviewPanel.svelte';
	import { onMount } from 'svelte';

	onMount(() => {
		// The Go queue is authoritative — reconcile on every visit.
		void TiltyClaretPending();
	});
</script>

<div class="flex-1 flex flex-col h-full animate-fade-in overflow-hidden">
	<header class="p-4 border-b border-border bg-surface-hi flex items-center justify-between">
		<h2 class="text-lg font-bold">World Browser</h2>
		{#if worldStore.activeManifest}
			<span class="text-sm text-text-muted">{worldStore.activeManifest.name}</span>
		{/if}
	</header>

	<div class="flex-1 flex overflow-hidden">
		<!-- Tree panel -->
		<aside class="w-72 border-r border-border bg-surface flex flex-col overflow-hidden">
			<EntryTree />
		</aside>

		<!-- Editor + review queue -->
		<div class="flex-1 flex flex-col overflow-y-auto p-4 gap-4">
			{#if pendingStore.items.length > 0}
				<PendingReviewPanel />
			{/if}
			<div class="flex-1 flex rounded-xl border border-border bg-surface overflow-hidden min-h-[60vh]">
				<EntryEditor />
			</div>
		</div>
	</div>
</div>
