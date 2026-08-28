import { expect, test } from "bun:test";
import {
	registerItineraryHelpers,
	registerItineraryKeyboard,
} from "./itinerary";

type KeyEvent = {
	key: string;
	target: unknown;
	preventDefault(): void;
};

type Listener = (event: KeyEvent) => void;

function makeFakeDom() {
	const appendedLinks: Array<{ rel: string; href: string }> = [];
	const listeners: Record<string, Listener> = {};
	const navigations: string[] = [];

	// prev/next carry hx-get in the real markup, so keyboard navigation
	// dispatches a real click for htmx's delegated handler to intercept
	// (see bindViewerKeyboard) rather than assigning window.location.
	const navPrev = {
		getAttribute: () => "/itineraries/t?stop=0",
		click: () => {
			navigations.push("/itineraries/t?stop=0");
		},
	};
	const navNext = {
		getAttribute: () => "/itineraries/t?stop=2",
		click: () => {
			navigations.push("/itineraries/t?stop=2");
		},
	};
	const exit = { getAttribute: () => "/itineraries" };

	const viewer = {
		isConnected: true,
		dataset: {} as Record<string, string>,
		querySelector: (selector: string) => {
			if (selector === '[data-itinerary-nav="prev"][href]') return navPrev;
			if (selector === '[data-itinerary-nav="next"][href]') return navNext;
			if (selector === "[data-itinerary-exit][href]") return exit;
			return null;
		},
		querySelectorAll: (selector: string) => {
			if (selector === "[data-itinerary-prefetch][href]") {
				return [
					{ getAttribute: () => "/itineraries/t?stop=0" },
					{ getAttribute: () => "/itineraries/t?stop=2" },
					{ getAttribute: () => "/itineraries/t?stop=3" },
				];
			}
			return [];
		},
	};

	const document = {
		querySelectorAll: (selector: string) => {
			if (selector === "[data-itinerary-viewer]:not([data-itinerary-bound])") {
				return viewer.dataset.itineraryBound ? [] : [viewer];
			}
			if (
				selector === "[data-itinerary-viewer]:not([data-itinerary-keyboard])"
			) {
				return viewer.dataset.itineraryKeyboard ? [] : [viewer];
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
		createElement: (tag: string) => ({
			tagName: tag.toUpperCase(),
			rel: "",
			href: "",
		}),
		head: {
			appendChild: (element: { rel: string; href: string }) => {
				appendedLinks.push({ rel: element.rel, href: element.href });
			},
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

	return { appendedLinks, body, listeners, navigations, viewer };
}

test("switches builder stops without replacing unpublished fields", () => {
	const hidden = new Map<string, boolean>();
	const makeElement = (index: string) => ({
		dataset: {
			itineraryEditor: index,
			itineraryTab: index,
			itineraryTabItem: index,
			itineraryTabLabel: index,
		},
		classList: {
			toggle: (name: string, active: boolean) =>
				hidden.set(`${index}:${name}`, active),
		},
		setAttribute: (name: string, value: string) =>
			hidden.set(`${index}:${name}:${value}`, true),
		removeAttribute: (name: string) => hidden.delete(`${index}:${name}:page`),
	});
	const first = makeElement("0");
	const second = makeElement("1");
	let tabClick: ((event: Event) => void) | undefined;
	const tabs = [first, second].map((tab) => ({
		...tab,
		addEventListener: (_name: string, listener: (event: Event) => void) => {
			if (tab.dataset.itineraryTab === "1") {
				tabClick = listener;
			}
		},
	}));

	globalThis.document = {
		querySelectorAll: (selector: string) => {
			if (selector === "[data-itinerary-editor]") return [first, second];
			if (selector === "[data-itinerary-tab]:not([data-itinerary-tab-bound])")
				return tabs;
			if (selector === "[data-itinerary-tab]") return tabs;
			if (selector === "[data-itinerary-tab-item]") return [first, second];
			if (selector === "[data-itinerary-tab-label]") return [first, second];
			return [];
		},
	} as unknown as Document;

	registerItineraryHelpers();
	let prevented = false;
	tabClick?.({
		preventDefault: () => {
			prevented = true;
		},
	} as unknown as Event);

	expect(prevented).toBe(true);
	expect(hidden.get("0:hidden")).toBe(true);
	expect(hidden.get("1:hidden")).toBe(false);
	expect(hidden.get("1:aria-current:page")).toBe(true);
});

test("updates a filmstrip narration status while typing", () => {
	const classes = new Map<string, boolean>();
	let input: (() => void) | undefined;
	const narration = {
		dataset: {},
		value: "A note",
		addEventListener: (_name: string, listener: () => void) => {
			input = listener;
		},
	};
	const status = {
		dataset: { itineraryNarrationStatus: "stop-1" },
		textContent: "NO NARRATION YET",
		classList: {
			toggle: (name: string, active: boolean) => classes.set(name, active),
		},
	};
	(narration.dataset as Record<string, string>).itineraryNarration = "stop-1";

	globalThis.document = {
		querySelectorAll: (selector: string) => {
			if (
				selector ===
				"[data-itinerary-narration]:not([data-itinerary-narration-bound])"
			)
				return [narration];
			if (selector === "[data-itinerary-narration-status]") return [status];
			return [];
		},
	} as unknown as Document;

	registerItineraryHelpers();
	input?.();

	expect(status.textContent).toBe("6 CHARS");
	expect(classes.get("text-warning")).toBe(false);
	expect(classes.get("text-base-content/60")).toBe(true);
});

test("highlights the currently selected visibility option", () => {
	const classes = new Map<string, boolean>();
	const makeLabel = (name: string) => ({
		classList: {
			toggle: (className: string, active: boolean) =>
				classes.set(`${name}:${className}`, active),
		},
	});
	const listedLabel = makeLabel("listed");
	const linkOnlyLabel = makeLabel("link-only");
	let change: (() => void) | undefined;
	const listed = {
		dataset: {},
		checked: false,
		closest: () => listedLabel,
		addEventListener: () => {},
	};
	const linkOnly = {
		dataset: {},
		checked: true,
		closest: () => linkOnlyLabel,
		addEventListener: (_name: string, listener: () => void) => {
			change = listener;
		},
	};

	globalThis.document = {
		querySelectorAll: (selector: string) => {
			if (
				selector ===
				"[data-itinerary-visibility] input[name='listed']:not([data-itinerary-visibility-bound])"
			)
				return [listed, linkOnly];
			if (selector === "[data-itinerary-visibility] input[name='listed']")
				return [listed, linkOnly];
			return [];
		},
	} as unknown as Document;

	registerItineraryHelpers();
	change?.();

	expect(classes.get("listed:bg-primary")).toBe(false);
	expect(classes.get("link-only:bg-primary")).toBe(true);
	expect(classes.get("link-only:text-primary-content")).toBe(true);
});

test("prefetches at most two neighbouring stops", () => {
	const { appendedLinks } = makeFakeDom();

	registerItineraryHelpers();

	expect(appendedLinks.length).toBe(2);
	expect(appendedLinks.map((link) => link.rel)).toEqual([
		"prefetch",
		"prefetch",
	]);
});

test("listens for keydown at document scope", () => {
	const { listeners } = makeFakeDom();

	registerItineraryHelpers();

	expect(listeners.keydown).toBeDefined();
});

test("navigates to the next stop on ArrowRight with focus on the body", () => {
	const { body, listeners, navigations } = makeFakeDom();

	registerItineraryHelpers();

	listeners.keydown({
		key: "ArrowRight",
		target: body,
		preventDefault: () => {},
	});

	expect(navigations).toContain("/itineraries/t?stop=2");
});

test("navigates to the previous stop on ArrowLeft with focus on the body", () => {
	const { body, listeners, navigations } = makeFakeDom();

	registerItineraryHelpers();

	listeners.keydown({
		key: "ArrowLeft",
		target: body,
		preventDefault: () => {},
	});

	expect(navigations).toContain("/itineraries/t?stop=0");
});

test("exits the slideshow on Escape with focus on the body", () => {
	const { body, listeners, navigations } = makeFakeDom();

	registerItineraryHelpers();

	listeners.keydown({
		key: "Escape",
		target: body,
		preventDefault: () => {},
	});

	expect(navigations).toContain("/itineraries");
});

test("ignores keys when focus is inside an editable control", () => {
	const { listeners, navigations } = makeFakeDom();

	registerItineraryHelpers();

	listeners.keydown({
		key: "ArrowRight",
		target: { closest: () => ({}) },
		preventDefault: () => {},
	});

	expect(navigations.length).toBe(0);
});

test("removes its document listener once the viewer disconnects", () => {
	const { listeners, viewer } = makeFakeDom();

	registerItineraryHelpers();
	expect(listeners.keydown).toBeDefined();

	viewer.isConnected = false;
	listeners.keydown({
		key: "ArrowRight",
		target: { closest: () => null },
		preventDefault: () => {},
	});

	expect(listeners.keydown).toBeUndefined();
});

test("registers idempotently without doubling prefetches or listeners", () => {
	const { appendedLinks, listeners, navigations } = makeFakeDom();

	registerItineraryHelpers();
	registerItineraryHelpers();

	expect(appendedLinks.length).toBe(2);
	listeners.keydown({
		key: "ArrowRight",
		target: { closest: () => null },
		preventDefault: () => {},
	});
	expect(navigations).toHaveLength(1);
});

test("registerItineraryKeyboard binds navigation without prefetching", () => {
	const { appendedLinks, listeners, navigations } = makeFakeDom();

	registerItineraryKeyboard();

	expect(listeners.keydown).toBeDefined();
	expect(appendedLinks.length).toBe(0);

	listeners.keydown({
		key: "ArrowRight",
		target: { closest: () => null },
		preventDefault: () => {},
	});
	expect(navigations).toContain("/itineraries/t?stop=2");
});

test("registerItineraryKeyboard is idempotent", () => {
	const { listeners, navigations } = makeFakeDom();

	registerItineraryKeyboard();
	registerItineraryKeyboard();

	listeners.keydown({
		key: "ArrowRight",
		target: { closest: () => null },
		preventDefault: () => {},
	});
	expect(navigations).toHaveLength(1);
});

test("registerItineraryKeyboard ignores keys inside an editable control", () => {
	const { listeners, navigations } = makeFakeDom();

	registerItineraryKeyboard();

	listeners.keydown({
		key: "ArrowRight",
		target: { closest: () => ({}) },
		preventDefault: () => {},
	});
	expect(navigations.length).toBe(0);
});

test("registerItineraryKeyboard removes its listener when the viewer disconnects", () => {
	const { listeners, viewer } = makeFakeDom();

	registerItineraryKeyboard();
	expect(listeners.keydown).toBeDefined();

	viewer.isConnected = false;
	listeners.keydown({
		key: "ArrowRight",
		target: { closest: () => null },
		preventDefault: () => {},
	});
	expect(listeners.keydown).toBeUndefined();
});

test("registerItineraryHelpers after the synchronous binder does not double-bind keyboard", () => {
	const { appendedLinks, listeners, navigations } = makeFakeDom();

	registerItineraryKeyboard();
	registerItineraryHelpers();

	expect(appendedLinks.length).toBe(2);
	listeners.keydown({
		key: "ArrowRight",
		target: { closest: () => null },
		preventDefault: () => {},
	});
	expect(navigations).toHaveLength(1);
});

test("copies the published itinerary link and restores the label", async () => {
	const written: string[] = [];
	let clickHandler: (() => void) | undefined;

	const button = {
		dataset: { copyTarget: "#itinerary-url" } as Record<string, string>,
		textContent: "COPY LINK",
		addEventListener: (name: string, fn: () => void) => {
			if (name === "click") {
				clickHandler = fn;
			}
		},
	};

	const target = { innerText: "https://wga.example/itineraries/token" };

	globalThis.document = {
		querySelectorAll: (selector: string) => {
			if (selector === "[data-copy-itinerary]:not([data-copy-bound])") {
				return [button];
			}
			return [];
		},
		querySelector: (selector: string) =>
			selector === "#itinerary-url" ? target : null,
		createElement: (tag: string) => ({ tagName: tag.toUpperCase() }),
		body: { appendChild: () => {} },
	} as unknown as Document;

	globalThis.navigator = {
		clipboard: {
			writeText: async (text: string) => {
				written.push(text);
			},
		},
	} as unknown as Navigator;

	registerItineraryHelpers();

	expect(button.dataset.copyBound).toBe("true");
	expect(clickHandler).toBeDefined();

	clickHandler?.();
	await Promise.resolve();

	expect(written).toEqual(["https://wga.example/itineraries/token"]);
	expect(button.textContent).toBe("COPIED");
});
