import { describe, expect, test } from "bun:test";
import {
	applyPreferenceConsent,
	readCookie,
	resetPreferenceConsentForTests,
} from "./preference-consent";
import {
	actionsLabel,
	COOKIE_NAME,
	cookieValue,
	DEFAULT_PREFS,
	parsePrefs,
	prefsFromHref,
	prefsWithView,
	registerSearchPrefs,
	renderedPrefs,
	STORAGE_KEY,
} from "./search-prefs";

test("parses stored preferences field by field", () => {
	expect(parsePrefs(null)).toBeNull();
	expect(parsePrefs("not json")).toBeNull();
	expect(parsePrefs("[]")).toBeNull();
	expect(
		parsePrefs('{"sort":"date","dir":"desc","view":"list","actions":true}'),
	).toEqual({ sort: "date", dir: "desc", view: "list", actions: true });
	expect(
		parsePrefs(
			'{"sort":"catalogue","dir":"down","view":"list","actions":"yes"}',
		),
	).toEqual({ ...DEFAULT_PREFS, view: "list" });
});

test("reads the addressed sort and view from a search link", () => {
	const current = { ...DEFAULT_PREFS, actions: true };
	const base = "https://wga.example/artworks";
	expect(prefsFromHref("/artworks?sort=date&dir=desc", current, base)).toEqual({
		sort: "date",
		dir: "desc",
		view: "grid",
		actions: true,
	});
	expect(
		prefsFromHref(
			"/artworks?sort=artist&view=list",
			{ ...current, sort: "date", dir: "desc" },
			base,
		),
	).toEqual({ sort: "artist", dir: "asc", view: "list", actions: true });
	expect(
		prefsFromHref("/artworks", { ...current, view: "list" }, base),
	).toEqual(current);
});

test("records a listing toggle's view and keeps the other fields", () => {
	const current = {
		sort: "date",
		dir: "desc",
		view: "grid",
		actions: true,
	} as const;
	expect(prefsWithView(current, "list")).toEqual({ ...current, view: "list" });
	expect(prefsWithView(current, "table")).toEqual(current);
	expect(prefsWithView(current, undefined)).toEqual(current);
});

test("round-trips the cookie value", () => {
	const prefs = {
		sort: "date",
		dir: "desc",
		view: "list",
		actions: true,
	} as const;
	const value = cookieValue(prefs);
	expect(value).not.toContain('"');
	expect(value).not.toContain(",");
	expect(
		parsePrefs(readCookie(`other=1; wga_aw_prefs=${value}`, "wga_aw_prefs")),
	).toEqual(prefs);
	expect(readCookie("other=1", "wga_aw_prefs")).toBeNull();
});

test("labels the actions toggle", () => {
	expect(actionsLabel(false)).toBe("ACTIONS +");
	expect(actionsLabel(true)).toBe("ACTIONS ✓");
});

test("reads the presented state from the rendered page", () => {
	const current = { ...DEFAULT_PREFS, actions: true };
	expect(
		renderedPrefs(
			{ wgaAwSort: "date", wgaAwDir: "desc", wgaAwView: "list" },
			"off",
			current,
		),
	).toEqual({ sort: "date", dir: "desc", view: "list", actions: false });
	expect(renderedPrefs(null, undefined, current)).toEqual(current);
	expect(
		renderedPrefs({ wgaAwSort: "catalogue" }, "on", DEFAULT_PREFS),
	).toEqual({ ...DEFAULT_PREFS, actions: true });
});

// A minimal page: the search results, the actions toggle, a cookie jar, and
// localStorage, enough to drive the module's consent and click handling.
class FakeElement {
	dataset: Record<string, string> = {};
	textContent = "";
	attributes: Record<string, string> = {};
	constructor(private readonly selectors: string[] = []) {}
	setAttribute(name: string, value: string) {
		this.attributes[name] = value;
	}
	closest(selector: string) {
		return this.selectors.includes(selector) ? this : null;
	}
}

class FakeMouseEvent {
	target: unknown = null;
	button = 0;
	metaKey = false;
	ctrlKey = false;
	shiftKey = false;
	altKey = false;
}

const fakePage = () => {
	const jar = new Map<string, string>();
	const storage = new Map<string, string>();
	const listeners: Record<string, ((event: unknown) => void)[]> = {};
	const results = new FakeElement();
	const toggle = new FakeElement(["[data-wga-aw-actions]"]);
	const documentElement = new FakeElement();
	Object.assign(globalThis, {
		Element: FakeElement,
		MouseEvent: FakeMouseEvent,
		window: {
			location: { protocol: "http:", href: "http://wga.test/artworks" },
			localStorage: {
				getItem: (key: string) => storage.get(key) ?? null,
				setItem: (key: string, value: string) => storage.set(key, value),
				removeItem: (key: string) => storage.delete(key),
			},
		},
		document: {
			get cookie() {
				return [...jar].map(([key, value]) => `${key}=${value}`).join("; ");
			},
			set cookie(value: string) {
				const [pair, ...attributes] = value.split("; ");
				const [key, ...rest] = pair.split("=");
				if (attributes.includes("Max-Age=0")) {
					jar.delete(key);
				} else {
					jar.set(key, rest.join("="));
				}
			},
			documentElement,
			querySelector: (selector: string) =>
				selector === "#artwork-search-results" ? results : null,
			querySelectorAll: (selector: string) =>
				selector === "[data-wga-aw-actions]" ? [toggle] : [],
			addEventListener: (type: string, listener: (event: unknown) => void) => {
				listeners[type] = [...(listeners[type] ?? []), listener];
			},
		},
	});
	const grantConsent = () =>
		jar.set(
			"cc_cookie",
			encodeURIComponent(JSON.stringify({ categories: ["preferences"] })),
		);
	const revokeConsent = () => {
		jar.clear();
		storage.clear();
	};
	const click = (target: FakeElement, init: Partial<FakeMouseEvent> = {}) => {
		const event = Object.assign(new FakeMouseEvent(), init, { target });
		for (const listener of listeners.click ?? []) {
			listener(event);
		}
	};
	const stored = () => ({
		storage: parsePrefs(storage.get(STORAGE_KEY)),
		cookie: parsePrefs(readCookie(document.cookie, COOKIE_NAME)),
	});
	return {
		results,
		toggle,
		documentElement,
		storage,
		grantConsent,
		revokeConsent,
		click,
		stored,
	};
};

describe("on the rendered search page", () => {
	const page = fakePage();
	// The visitor opened /artworks?sort=date&view=list without consent.
	page.results.dataset = {
		wgaAwSort: "date",
		wgaAwDir: "asc",
		wgaAwView: "list",
	};
	resetPreferenceConsentForTests();
	registerSearchPrefs();

	test("granting consent stores the rendered choices", () => {
		const shown = { sort: "date", dir: "asc", view: "list", actions: false };
		page.grantConsent();
		applyPreferenceConsent(true);
		expect(page.stored()).toEqual({ storage: shown, cookie: shown });

		// The consent library confirms existing consent on every page load; an
		// explicit URL then must not replace the remembered choices.
		page.results.dataset = { ...page.results.dataset, wgaAwSort: "artist" };
		applyPreferenceConsent(true);
		expect(page.stored()).toEqual({ storage: shown, cookie: shown });
	});

	test("the actions toggle inverts the visible state, not the stored one", () => {
		// Another tab turned actions on; this tab still shows them hidden.
		page.storage.set(
			STORAGE_KEY,
			JSON.stringify({ ...DEFAULT_PREFS, actions: true }),
		);
		expect(page.documentElement.dataset.awActions).toBe("off");
		page.click(page.toggle);
		expect(page.documentElement.dataset.awActions).toBe("on");
		expect(page.toggle.textContent).toBe("ACTIONS ✓");
		expect(page.stored().storage?.actions).toBe(true);
		expect(page.stored().cookie?.actions).toBe(true);
	});

	test("a listing view toggle records its view and keeps the other fields", () => {
		const remembered = {
			sort: "date",
			dir: "desc",
			view: "grid",
			actions: true,
		};
		page.grantConsent();
		page.storage.set(STORAGE_KEY, JSON.stringify(remembered));
		const listToggle = new FakeElement(["a[data-wga-view-pref][href]"]);
		listToggle.dataset = { wgaViewPref: "list" };
		page.click(listToggle);
		const expected = { ...remembered, view: "list" };
		expect(page.stored()).toEqual({ storage: expected, cookie: expected });

		// A modified click opens another tab and records nothing.
		const gridToggle = new FakeElement(["a[data-wga-view-pref][href]"]);
		gridToggle.dataset = { wgaViewPref: "grid" };
		page.click(gridToggle, { ctrlKey: true });
		expect(page.stored()).toEqual({ storage: expected, cookie: expected });
	});

	test("a listing view toggle records nothing without consent", () => {
		page.revokeConsent();
		const listToggle = new FakeElement(["a[data-wga-view-pref][href]"]);
		listToggle.dataset = { wgaViewPref: "list" };
		page.click(listToggle);
		expect(page.stored()).toEqual({ storage: null, cookie: null });
	});
});
