import PBClient from '$lib/pb-client.js';
import type { PageLoad } from './$types';
export const load: PageLoad = async ({ params }) => {
	const artist = await PBClient.collection('artists').getOne(params.id, {
		expand: 'art.media'
	});

	if (!artist) {
		return {
			status: 404,
			error: new Error('Not found')
		};
	}

	if (artist.art) {
		artist.art.media_art = artist.expand.art.map((art: { expand: { media: any[] } }) => {
			art.expand.media.map((media) => {
				media.url = PBClient.getFileUrl(media, media.image, { thumb: '100x100' });
			});
		});
	}

	return {
		artist
	};
};
