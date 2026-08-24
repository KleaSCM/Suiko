<script lang="ts">
	let { text }: { text: string } = $props();
</script>

<div class="flex w-full justify-start mb-4 animate-fade-in">
	<div class="max-w-[80%] p-4 text-sm whitespace-pre-wrap leading-relaxed shadow-sm bubble-narrator mr-12 cursor-stream border-accent/30 shadow-accent">
		{text}
	</div>
</div>
