import { expect, test } from "bun:test";
import { initPeriodMusic, validPlayerURL } from "./music";

function stubWindowLocation() {
	globalThis.window = {
		location: {
			href: "https://wga.test/record",
			origin: "https://wga.test",
		},
	} as unknown as Window & typeof globalThis;
}

test("accepts only a same-origin /player song URL", () => {
	stubWindowLocation();

	const song = "aaaaaaaaaaaaaaa";
	expect(validPlayerURL(`/player?song=${song}`)).toBe(true);
	expect(validPlayerURL(`https://wga.test/player?song=${song}`)).toBe(true);
});

test("rejects foreign origins, routes, and extra parameters", () => {
	stubWindowLocation();

	const song = "aaaaaaaaaaaaaaa";
	const invalid = [
		`https://evil.test/player?song=${song}`,
		`https://wga.test/other?song=${song}`,
		`/player?song=${song}&piece=extra`,
		"/player",
		"/player?song=short",
		`/player?song=${song.toUpperCase()}`,
		`/player?song=${song}#fragment`,
		`/player?song=${song}@evil.test`,
		"javascript:alert(1)",
	];
	for (const href of invalid) {
		expect(validPlayerURL(href), href).toBe(false);
	}
});

test("initialises idempotently without doubling listeners", () => {
	const listeners: string[] = [];
	globalThis.document = {
		addEventListener: (name: string) => {
			listeners.push(name);
		},
		querySelector: () => null,
		body: {
			addEventListener: (name: string) => {
				listeners.push(name);
			},
		},
	} as unknown as Document;
	globalThis.window = {
		addEventListener: (name: string) => {
			listeners.push(name);
		},
		location: { href: "https://wga.test/record", origin: "https://wga.test" },
	} as unknown as Window & typeof globalThis;

	initPeriodMusic();
	initPeriodMusic();

	const clickListeners = listeners.filter((name) => name === "click");
	expect(clickListeners).toHaveLength(2);
});
