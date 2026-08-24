<script lang="ts">
	import { onMount } from "svelte";
	import { navStore } from "./lib/stores/NavStore.svelte";
	import { loadWorlds, worldStore } from "./lib/stores/WorldStore.svelte";
	import { TsubasaKazanari } from "./lib/bindings";
	import NavBar from "./components/NavBar.svelte";
	import PlayView from "./views/PlayView.svelte";
	import WorldView from "./views/WorldView.svelte";
	import SessionsView from "./views/SessionsView.svelte";
	import SettingsView from "./views/SettingsView.svelte";
	import WorldSelector from "./components/WorldSelector.svelte";
	import CharacterCreator from "./components/CharacterCreator.svelte";

	let WorldsLoaded = $state(false);

	onMount(() => {
		// Event subscriptions live for the whole app session.
		TsubasaKazanari();
		loadWorlds().then(() => (WorldsLoaded = true));
	});

	// No world open → the selector gates everything else.
	let NeedsSelector = $derived(WorldsLoaded && !worldStore.activeManifest);
</script>

<div class="flex h-screen w-full bg-bg text-text font-ui overflow-hidden">
	<!-- Sidebar -->
	<NavBar />

	<!-- Main Content Area -->
	<main class="flex-1 flex flex-col relative overflow-hidden bg-bg">
		{#if navStore.currentView === 'play'}
			<PlayView />
		{:else if navStore.currentView === 'world'}
			<WorldView />
		{:else if navStore.currentView === 'sessions'}
			<SessionsView />
		{:else if navStore.currentView === 'settings'}
			<SettingsView />
		{/if}
	</main>

	{#if NeedsSelector}
		<WorldSelector />
	{:else if worldStore.needsPlayer}
		<CharacterCreator />
	{/if}
</div>
