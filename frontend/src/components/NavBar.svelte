<script lang="ts">
	import { navStore, navigate } from "../lib/stores/NavStore.svelte";
</script>

<aside class="w-64 flex flex-col bg-surface-alt border-r border-border h-full">
	<div class="p-4 border-b border-border">
		<h1 class="text-xl font-bold tracking-wider text-accent text-center">SUIKO</h1>
	</div>
	
	<nav class="flex-1 p-2 space-y-1">
		<button 
			class="btn w-full justify-start {navStore.currentView === 'play' ? 'btn-primary' : 'btn-ghost'}"
			onclick={() => navigate('play')}
		>
			Play
		</button>
		<button 
			class="btn w-full justify-start {navStore.currentView === 'world' ? 'btn-primary' : 'btn-ghost'}"
			onclick={() => navigate('world')}
		>
			World
		</button>
		<button 
			class="btn w-full justify-start {navStore.currentView === 'sessions' ? 'btn-primary' : 'btn-ghost'}"
			onclick={() => navigate('sessions')}
		>
			Sessions
		</button>
	</nav>
	
	<div class="p-2 border-t border-border">
		<button 
			class="btn w-full justify-start {navStore.currentView === 'settings' ? 'btn-primary' : 'btn-ghost'}"
			onclick={() => navigate('settings')}
		>
			Settings
		</button>
	</div>
</aside>
