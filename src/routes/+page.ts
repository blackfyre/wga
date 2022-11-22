import PBClient from '$lib/pb-client';
import { error } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	console.log(PBClient);
	const records = await PBClient.collection('artists').getFullList(200, {
		sort: '+name'
	});

	return {
		records
	};
};
