import { expect, test } from "bun:test";

import {
	clearDynamicModuleRecovery,
	recoverDynamicModule,
	refreshEntryModule,
} from "./dynamic-module-recovery";

test("recovers once for a dynamic module failure", () => {
	const storage = new Map<string, string>();
	const session = {
		getItem: (key: string) => storage.get(key) ?? null,
		setItem: (key: string, value: string) => storage.set(key, value),
		removeItem: (key: string) => storage.delete(key),
	} as Storage;
	let reloads = 0;

	expect(
		recoverDynamicModule(
			session,
			"https://example.test/artworks",
			() => reloads++,
		),
	).toBe(true);
	expect(
		recoverDynamicModule(
			session,
			"https://example.test/artworks",
			() => reloads++,
		),
	).toBe(false);
	expect(reloads).toBe(1);

	clearDynamicModuleRecovery(session);
	expect(
		recoverDynamicModule(
			session,
			"https://example.test/artworks",
			() => reloads++,
		),
	).toBe(true);
});

const fakeDocument = (sources: string[]) =>
	({
		querySelectorAll: () => sources.map((src) => ({ src })),
	}) as unknown as Document;

test("refreshes the page's entry module past the browser cache", async () => {
	const requests: Array<{ url: string; init?: RequestInit }> = [];
	const fetcher = (async (url: string, init?: RequestInit) => {
		requests.push({ url, init });
		return new Response("entry");
	}) as unknown as typeof fetch;

	await refreshEntryModule(
		fakeDocument([
			"https://example.test/assets/js/vendor.js",
			"https://example.test/assets/js/app.js?v=0123456789ab",
		]),
		fetcher,
	);

	expect(requests).toEqual([
		{
			url: "https://example.test/assets/js/app.js?v=0123456789ab",
			init: { cache: "reload", credentials: "same-origin" },
		},
	]);
});

test("a failed or absent entry refresh still resolves", async () => {
	let calls = 0;
	const failing = (async () => {
		calls++;
		throw new TypeError("offline");
	}) as unknown as typeof fetch;

	await refreshEntryModule(
		fakeDocument(["https://example.test/assets/js/app.js?v=1"]),
		failing,
	);
	await refreshEntryModule(fakeDocument([]), failing);

	expect(calls).toBe(1);
});
