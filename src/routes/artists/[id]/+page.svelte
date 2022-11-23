<script lang="ts">
	import type { PageData } from './$types';
	import SvelteMarkdown from 'svelte-markdown';
	import Link from '$lib/markdown-renderers/Link.svelte';
	import { Grid, Row, Column, ImageLoader, CodeSnippet, Tile } from 'carbon-components-svelte';
	export let data: PageData;
	export const code = JSON.stringify(data.artist, null, 2);
</script>

<svelte:head>
	<title>{data.artist.name} - WGA</title>
	<meta name="robots" content="noindex nofollow" />
	<html lang="en" />
</svelte:head>

<Grid fullWidth>
	<Row>
		<Column>
			<h1>{data.artist.name}</h1>
		</Column>
	</Row>
	<Row>
		<Column sm={12} md={8} lg={8}>
			<SvelteMarkdown source={data.artist.bio} renderers={{ link: Link }} />
		</Column>
		<Column sm={12} md={4} lg={4}>
			<Grid fullWidth noGutter>
				<Row>
					{#each data.artist.expand.art as art, i}
						<Column sm={6} md={6} lg={6}>
							<Tile>
								<ImageLoader fadeIn src={art.expand.media[0].url} alt={art.expand.media[0].title} />
								<h3>{art.title}</h3>
							</Tile>
						</Column>
					{/each}
				</Row>
			</Grid>
		</Column>
	</Row>
	<Row>
		<Column>
			<CodeSnippet type="multi" {code} />
		</Column>
	</Row>
</Grid>
<div />
