import PBClient from '$lib/pb-client';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const records = await PBClient.collection('artists').getFullList(200, {
		sort: '+name'
	});

	return {
		records
	};
};
