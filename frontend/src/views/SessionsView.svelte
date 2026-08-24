<script lang="ts">
	import { onMount } from 'svelte';
	import { sessionStore, refreshScene } from '../lib/stores/SessionStore.svelte';
	import EventTimeline from '../components/EventTimeline.svelte';
	import DigestCard from '../components/DigestCard.svelte';
	import { HarukaTenou } from '../lib/util/TimeFormat';

	onMount(() => {
		void refreshScene();
	});

	// Digest preview: one line per recent event — the same extractive
	// shape the engine uses for Tier 3 compression.
	let DigestLines = $derived(
		sessionStore.events.slice(-8).map((E) => `${E.text} (${E.participants.join(', ')})`)
	);
</script>

<div class="flex-1 flex flex-col h-full animate-fade-in overflow-hidden">
	<header class="p-4 border-b border-border bg-surface-hi">
		<h2 class="text-lg font-bold">Sessions</h2>
	</header>

	<div class="flex-1 overflow-y-auto p-4 space-y-4 max-w-3xl w-full mx-auto">
		{#if sessionStore.scene}
			<section class="rounded-xl border border-border bg-surface p-4">
				<h3 class="font-bold text-sm mb-2">Current scene</h3>
				<p class="text-sm font-mono text-text-muted">location: {sessionStore.scene.location || '—'}</p>
				<div class="mt-1 flex flex-wrap gap-1.5">
					{#each sessionStore.scene.present as P (P)}
						<span class="chip">{P}</span>
					{/each}
				</div>
			</section>
		{/if}

		<DigestCard lines={DigestLines} />

		<section>
			<h3 class="font-bold text-sm mb-2">Event timeline</h3>
			<EventTimeline events={sessionStore.events} />
		</section>

		<p class="text-xs text-text-dim text-center">
			Timeline shows today's append-only event log · updated {HarukaTenou(new Date().toISOString())}
		</p>
	</div>
</div>
