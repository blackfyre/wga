<script lang="ts">
	export const ssr = false;
	export const trailingSlash = 'never';
	export const prerender = true;
	export const csr = false;
	import 'carbon-components-svelte/css/g80.css';
	// let theme = 'white'; // "white" | "g10" | "g80" | "g90" | "g100"
	// $: document.documentElement.setAttribute('theme', theme);
	import {
		Header,
		HeaderNav,
		HeaderNavItem,
		HeaderNavMenu,
		SkipToContent,
		Content
		// Grid,
		// Row,
		// Column
	} from 'carbon-components-svelte';

	let isSideNavOpen = false;
</script>

<Header platformName="WGA" bind:isSideNavOpen>
	<svelte:fragment slot="skip-to-content">
		<SkipToContent />
	</svelte:fragment>
	<HeaderNav>
		<HeaderNavItem href="/" text="Home" />
		<HeaderNavItem href="/artists" text="Artist Index" />
		<HeaderNavItem href="/about-us" text="About Us" />
		<HeaderNavItem href="/blog" text="Blog" />
		<HeaderNavMenu text="Menu">
			<HeaderNavItem href="/" text="Link 1" />
			<HeaderNavItem href="/" text="Link 2" />
			<HeaderNavItem href="/" text="Link 3" />
		</HeaderNavMenu>
	</HeaderNav>
</Header>

<Content>
	<slot />
</Content>
