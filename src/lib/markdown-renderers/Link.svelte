<script lang="ts">
	import { onMount } from 'svelte';
	import Launch from 'carbon-icons-svelte/lib/Launch.svelte';

	export let href = '#';
	export let title: undefined | string = undefined;
	export let target = '_self';

	let link: undefined | HTMLAnchorElement = undefined;

	onMount(() => {
		if (link) {
			if (link.host !== window.location.host) {
				target = link.target = '_blank';
				link.rel = 'noopener noreferrer';
			} else {
				target = link.target = '_self';
			}
		}
	});
</script>

<a bind:this={link} {href} {title} {target}>
	<slot />
	{#if target === '_blank'}
		&nbsp;<Launch />
	{/if}
</a>
