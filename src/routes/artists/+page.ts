import PBClient from '$lib/pb-client.js';
import type { PageLoad } from './$types';

type Artist = {
	id: string;
	name: string;
	url?: string;
};

export const load: PageLoad = async () => {
	const records = (await PBClient.collection('artists').getFullList(200, {
		sort: '+name'
	})) as Artist[];

	records.map((record: Artist): void => {
		record.url = `/artists/${record.id}`;
	});

	return {
		records
	};
};
