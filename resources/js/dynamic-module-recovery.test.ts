import { expect, test } from "bun:test";

import { clearDynamicModuleRecovery, recoverDynamicModule } from "./dynamic-module-recovery";

test("recovers once for a dynamic module failure", () => {
	const storage = new Map<string, string>();
	const session = {
		getItem: (key: string) => storage.get(key) ?? null,
		setItem: (key: string, value: string) => storage.set(key, value),
		removeItem: (key: string) => storage.delete(key),
	} as Storage;
	let reloads = 0;

	expect(recoverDynamicModule(session, "https://example.test/artworks", () => reloads++)).toBe(true);
	expect(recoverDynamicModule(session, "https://example.test/artworks", () => reloads++)).toBe(false);
	expect(reloads).toBe(1);

	clearDynamicModuleRecovery(session);
	expect(recoverDynamicModule(session, "https://example.test/artworks", () => reloads++)).toBe(true);
});
