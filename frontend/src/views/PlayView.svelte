<script lang="ts">
	import { sessionStore, startSession, abortTurn } from "../lib/stores/SessionStore.svelte";
	import { worldStore } from "../lib/stores/WorldStore.svelte";
	import ChatBubble from "../components/ChatBubble.svelte";
	import TokenStream from "../components/TokenStream.svelte";

	let promptText = $state("");

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!promptText.trim() || sessionStore.isStreaming) return;
		
		const txt = promptText;
		promptText = "";
		await startSession(txt);
	}
</script>

<div class="flex-1 flex h-full animate-fade-in overflow-hidden">
	<!-- Session metadata panel -->
	<aside class="w-56 shrink-0 border-r border-border bg-surface p-4 space-y-3 hidden md:block">
		{#if worldStore.activeManifest}
			<div>
				<h3 class="text-xs uppercase tracking-wider text-text-dim mb-1">World</h3>
				<p class="font-semibold">{worldStore.activeManifest.name}</p>
			</div>
		{/if}
		<div>
			<h3 class="text-xs uppercase tracking-wider text-text-dim mb-1">Turn</h3>
			<p class="font-semibold font-mono">{sessionStore.messages.filter((M) => M.role === 'user').length}</p>
		</div>
		{#if sessionStore.scene}
			<div>
				<h3 class="text-xs uppercase tracking-wider text-text-dim mb-1">Scene</h3>
				<p class="text-sm text-text-muted break-all">{sessionStore.scene.location || '—'}</p>
			</div>
			{#if sessionStore.scene.open_threads.length > 0}
				<div>
					<h3 class="text-xs uppercase tracking-wider text-text-dim mb-1">Open threads</h3>
					<ul class="text-sm text-text-muted space-y-1">
						{#each sessionStore.scene.open_threads as T (T)}
							<li class="leading-snug">· {T}</li>
						{/each}
					</ul>
				</div>
			{/if}
		{/if}
	</aside>

	<div class="flex-1 flex flex-col overflow-hidden">
	<header class="p-4 border-b border-border bg-surface-hi flex items-center justify-between">
		<h2 class="text-lg font-bold">Play Session</h2>

		<!-- Optional Scene Data from sessionStore.scene could go here -->
		{#if sessionStore.scene}
			<span class="text-sm text-text-muted">{sessionStore.scene.location}</span>
		{/if}
	</header>
	
	<!-- Chat Feed -->
	<div class="flex-1 p-6 scroll-y flex flex-col">
		{#if sessionStore.messages.length === 0}
			<div class="m-auto text-center text-text-dim">
				<p>The world awaits. Type a message to begin.</p>
			</div>
		{/if}
		
		{#each sessionStore.messages as msg}
			<ChatBubble message={msg} />
		{/each}
		
		{#if sessionStore.isStreaming}
			<TokenStream text={sessionStore.streamingText} />
		{/if}
	</div>
	
	<!-- Input Area -->
	<div class="p-4 border-t border-border bg-surface-alt">
		<form onsubmit={handleSubmit} class="flex items-end gap-3 max-w-4xl mx-auto">
			<textarea 
				class="input flex-1 textarea resize-none overflow-y-auto"
				placeholder="Write your action..."
				bind:value={promptText}
				onkeydown={(e) => {
					if (e.key === 'Enter' && !e.shiftKey) {
						e.preventDefault();
						handleSubmit(e);
					}
				}}
			></textarea>
			
			<div class="flex flex-col gap-2">
				{#if sessionStore.isStreaming}
					<button type="button" class="btn btn-danger" onclick={abortTurn}>
						Abort
					</button>
				{:else}
					<button type="submit" class="btn btn-primary" disabled={!promptText.trim()}>
						Send
					</button>
				{/if}
			</div>
		</form>
	</div>
	</div>
</div>
