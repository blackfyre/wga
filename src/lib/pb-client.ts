import { PUBLIC_PB_SERVER_URL } from '$env/static/public';

import PocketBase from 'pocketbase';

const client = new PocketBase(PUBLIC_PB_SERVER_URL);
export default client;
