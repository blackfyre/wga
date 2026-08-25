import { expect, test } from "bun:test";
import { registerTourHelpers } from "./tours";

type KeyEvent = {
	key: string;
	target: unknown;
	preventDefault(): void;
};

type Listener = (event: KeyEvent) => void;

function makeFakeDom() {
	const listeners: Record<string, Listener> = {};
	const navigations: string[] = [];

	const navPrev = { getAttribute: () => "/tours/fixture" };
	const navNext = { getAttribute: () => "/tours/fixture/3" };

	const reading = {
		isConnected: true,
		dataset: {} as Record<string, string>,
		querySelector: (selector: string) => {
			if (selector === '[data-tour-nav="prev"][href]') return navPrev;
			if (selector === '[data-tour-nav="next"][href]') return navNext;
			return null;
		},
	};

	const document = {
		querySelectorAll: (selector: string) => {
			if (selector === "[data-tour-reading]:not([data-tour-bound])") {
				return reading.dataset.tourBound ? [] : [reading];
			}
			return [];
		},
		addEventListener: (name: string, fn: Listener) => {
			listeners[name] = fn;
		},
		removeEventListener: (name: string, fn: Listener) => {
			if (listeners[name] === fn) {
				delete listeners[name];
			}
		},
	};

	const window = {
		location: {
			set href(value: string) {
				navigations.push(value);
			},
			get href() {
				return navigations[navigations.length - 1] ?? "";
			},
		},
	};

	globalThis.document = document as unknown as Document;
	globalThis.window = window as unknown as Window & typeof globalThis;

	// The document body, as the target when page focus is outside the viewer.
	const body = { closest: () => null };

	return { body, listeners, navigations, reading };
}

test("listens for keydown at document scope", () => {
	const { listeners } = makeFakeDom();

	registerTourHelpers();

	expect(listeners.keydown).toBeDefined();
});

test("turns to the next page on ArrowRight with focus on the body", () => {
	const { body, listeners, navigations } = makeFakeDom();

	registerTourHelpers();

	listeners.keydown({
		key: "ArrowRight",
		target: body,
		preventDefault: () => {},
	});

	expect(navigations).toContain("/tours/fixture/3");
});

test("turns to the previous page on ArrowLeft with focus on the body", () => {
	const { body, listeners, navigations } = makeFakeDom();

	registerTourHelpers();

	listeners.keydown({
		key: "ArrowLeft",
		target: body,
		preventDefault: () => {},
	});

	expect(navigations).toContain("/tours/fixture");
});

test("ignores keys when focus is inside an editable control", () => {
	const { listeners, navigations } = makeFakeDom();

	registerTourHelpers();

	listeners.keydown({
		key: "ArrowRight",
		target: { closest: () => ({}) },
		preventDefault: () => {},
	});

	expect(navigations.length).toBe(0);
});

test("ignores keys while the artwork viewer surface owns the keyboard", () => {
	const { listeners, navigations } = makeFakeDom();

	registerTourHelpers();

	listeners.keydown({
		key: "ArrowLeft",
		target: { closest: () => ({ className: "wga-viewer-surface" }) },
		preventDefault: () => {},
	});

	expect(navigations.length).toBe(0);
});

test("removes its document listener once the reading disconnects", () => {
	const { listeners, reading } = makeFakeDom();

	registerTourHelpers();
	expect(listeners.keydown).toBeDefined();

	reading.isConnected = false;
	listeners.keydown({
		key: "ArrowRight",
		target: { closest: () => null },
		preventDefault: () => {},
	});

	expect(listeners.keydown).toBeUndefined();
});

test("registers idempotently without doubling listeners", () => {
	const { listeners, navigations } = makeFakeDom();

	registerTourHelpers();
	registerTourHelpers();

	listeners.keydown({
		key: "ArrowRight",
		target: { closest: () => null },
		preventDefault: () => {},
	});
	expect(navigations).toHaveLength(1);
});
