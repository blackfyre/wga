<script lang="ts">
	import type { PageData } from './$types';
	import {
		DataTable,
		Toolbar,
		ToolbarContent,
		ToolbarSearch,
		Pagination,
		Link
	} from 'carbon-components-svelte';
	import { Launch } from 'carbon-icons-svelte';
	export let data: PageData;
	export let columns = [
		{
			key: 'name',
			value: 'Name'
		},
		{
			key: 'bio',
			value: 'Bio'
		},
		{
			key: 'url',
			value: ''
		}
	];

	let pageSize = 30;
	let page = 1;
	let filteredRowIds: string[] = [];
</script>

<svelte:head>
	<title>Artist Index - WGA</title>
</svelte:head>

<DataTable
	title="Artist Index"
	stickyHeader
	{pageSize}
	{page}
	headers={columns}
	rows={data.records}
>
	<Toolbar>
		<ToolbarContent>
			<ToolbarSearch persistent value="" shouldFilterRows bind:filteredRowIds />
		</ToolbarContent>
	</Toolbar>
	<svelte:fragment slot="cell" let:row let:cell>
		{#if cell.key === 'url'}
			<Link icon={Launch} href={cell.value} target="_self">View</Link>
		{:else}
			{cell.value}
		{/if}
	</svelte:fragment>
</DataTable>
<Pagination bind:pageSize bind:page totalItems={filteredRowIds.length} pageSizeInputDisabled />
