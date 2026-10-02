// Remembers the artwork search presentation choices (sort key, direction,
// view, and whether per-result itinerary/Study Board actions are shown). They
// are presentation state, never part of the query, so filter reset leaves them
// alone.
//
// The same JSON object lives in localStorage["wga-aw-prefs"] (the client's
// source of truth) and, URL-encoded, in the wga_aw_prefs cookie, which the
// artwork search handler reads to render a bare /artworks visit in the
// remembered state and to set <html data-aw-actions> without a flash.
//
// Sort and view controls stay ordinary links: a capture-phase click listener
// records the state a [data-wga-aw-pref] link addresses before HTMX (or the
// browser) follows it. The [data-wga-aw-actions] toggle only flips the <html>
// attribute, which CSS uses to show or hide every .wga-work-actions block.
// Listeners are document-level and bound once, so they survive every swap.
//
// Remembering is optional storage, gated on the cookie-consent "preferences"
// category through ./preference-consent. Without that consent the choices
// apply to the current page only: nothing is read from or written to the
// cookie or localStorage, any stale copy is deleted, and the server renders
// its defaults.

import {
	cookieAttributes,
	expireCookie,
	preferenceStorageAllowed,
	readCookie,
	readPreference,
	registerPreferenceStore,
	removePreference,
	writePreference,
} from "./preference-consent";

export type SortKey = "title" | "artist" | "date";
export type SortDir = "asc" | "desc";
export type ResultView = "grid" | "list";

export type SearchPrefs = {
	sort: SortKey;
	dir: SortDir;
	view: ResultView;
	actions: boolean;
};

export const STORAGE_KEY = "wga-aw-prefs";
export const COOKIE_NAME = "wga_aw_prefs";
const COOKIE_MAX_AGE = 60 * 60 * 24 * 365;

export const DEFAULT_PREFS: SearchPrefs = {
	sort: "title",
	dir: "asc",
	view: "grid",
	actions: false,
};

const SORT_KEYS: readonly string[] = ["title", "artist", "date"];

const isRecord = (value: unknown): value is Record<string, unknown> =>
	typeof value === "object" && value !== null && !Array.isArray(value);

// parsePrefs validates each field independently, falling back to the default
// for any missing or invalid value. It returns null when nothing usable was
// stored at all.
export const parsePrefs = (
	raw: string | null | undefined,
): SearchPrefs | null => {
	if (!raw) {
		return null;
	}
	let value: unknown;
	try {
		value = JSON.parse(raw);
	} catch {
		return null;
	}
	if (!isRecord(value)) {
		return null;
	}
	return {
		sort:
			typeof value.sort === "string" && SORT_KEYS.includes(value.sort)
				? (value.sort as SortKey)
				: DEFAULT_PREFS.sort,
		dir:
			value.dir === "asc" || value.dir === "desc"
				? value.dir
				: DEFAULT_PREFS.dir,
		view:
			value.view === "grid" || value.view === "list"
				? value.view
				: DEFAULT_PREFS.view,
		actions:
			typeof value.actions === "boolean"
				? value.actions
				: DEFAULT_PREFS.actions,
	};
};

// prefsFromHref reads the sort, direction, and view a search link addresses.
// Canonical URLs omit default values, so an absent parameter is the default.
export const prefsFromHref = (
	href: string,
	current: SearchPrefs,
	base: string,
): SearchPrefs => {
	const params = new URL(href, base).searchParams;
	const parsed = parsePrefs(
		JSON.stringify({
			sort: params.get("sort") ?? DEFAULT_PREFS.sort,
			dir: params.get("dir") ?? DEFAULT_PREFS.dir,
			view: params.get("view") ?? DEFAULT_PREFS.view,
		}),
	);
	return { ...current, ...parsed, actions: current.actions };
};

export const cookieValue = (prefs: SearchPrefs): string =>
	encodeURIComponent(JSON.stringify(prefs));

// renderedPrefs reads the state the page currently presents: the sort,
// direction, and view the server rendered onto the search results, and the
// actions setting on <html>. Any field the page does not present keeps its
// current in-memory value.
export const renderedPrefs = (
	results: DOMStringMap | null | undefined,
	actions: string | undefined,
	current: SearchPrefs,
): SearchPrefs => {
	const parsed = parsePrefs(
		JSON.stringify({
			sort: results?.wgaAwSort ?? current.sort,
			dir: results?.wgaAwDir ?? current.dir,
			view: results?.wgaAwView ?? current.view,
			actions: actions === undefined ? current.actions : actions === "on",
		}),
	);
	return parsed ?? current;
};

export const actionsLabel = (shown: boolean): string =>
	shown ? "ACTIONS ✓" : "ACTIONS +";

// Every read checks the live consent record, so a grant or a withdrawal made
// in another tab applies here at once.
const readStoredPrefs = (): SearchPrefs | null =>
	preferenceStorageAllowed()
		? (parsePrefs(readPreference(STORAGE_KEY)) ??
			parsePrefs(readCookie(document.cookie, COOKIE_NAME)))
		: null;

const writePrefs = (prefs: SearchPrefs): void => {
	if (!preferenceStorageAllowed()) {
		return;
	}
	writePreference(STORAGE_KEY, JSON.stringify(prefs));
	// biome-ignore lint/suspicious/noDocumentCookie: the Cookie Store API is not available in every supported browser.
	document.cookie = `${COOKIE_NAME}=${cookieValue(prefs)}; ${cookieAttributes(COOKIE_MAX_AGE)}`;
};

const clearStoredPrefs = (): void => {
	removePreference(STORAGE_KEY);
	expireCookie(COOKIE_NAME);
};

const applyActions = (shown: boolean): void => {
	document.documentElement.dataset.awActions = shown ? "on" : "off";
	for (const toggle of document.querySelectorAll<HTMLElement>(
		"[data-wga-aw-actions]",
	)) {
		toggle.setAttribute("aria-pressed", String(shown));
		toggle.textContent = actionsLabel(shown);
	}
};

const presentedPrefs = (): SearchPrefs =>
	renderedPrefs(
		document.querySelector<HTMLElement>("#artwork-search-results")?.dataset,
		document.documentElement.dataset.awActions,
		prefs,
	);

let initialised = false;
let prefs: SearchPrefs = DEFAULT_PREFS;

export const registerSearchPrefs = (): void => {
	if (initialised) {
		return;
	}
	initialised = true;

	// Registering without consent deletes any stale copy. A first grant stores
	// the choices the page presents, which an explicit search URL may have set.
	registerPreferenceStore({
		remember: () => {
			prefs = presentedPrefs();
			writePrefs(prefs);
		},
		forget: clearStoredPrefs,
	});
	const stored = readStoredPrefs();
	prefs = stored ?? DEFAULT_PREFS;
	if (stored) {
		// Refresh the cookie from the client's source of truth so the server
		// keeps rendering the remembered state.
		writePrefs(prefs);
	}
	applyActions(prefs.actions);
	// Reveal the client-only actions toggle. The server may render
	// data-aw-actions from the cookie, so only this marker proves the handler
	// below is bound.
	document.documentElement.dataset.wgaSearchPrefs = "ready";

	document.addEventListener(
		"click",
		(event) => {
			if (!(event.target instanceof Element)) {
				return;
			}
			const toggle = event.target.closest("[data-wga-aw-actions]");
			const link = event.target.closest<HTMLAnchorElement>(
				"a[data-wga-aw-pref][href]",
			);
			if (!toggle && !link) {
				return;
			}
			// Another tab may have changed the choices since this page loaded.
			prefs = readStoredPrefs() ?? prefs;
			if (toggle) {
				// Invert what the page shows, not the stored value: another tab
				// may have changed it without this page's label following.
				prefs = {
					...prefs,
					actions: document.documentElement.dataset.awActions !== "on",
				};
				applyActions(prefs.actions);
				writePrefs(prefs);
				return;
			}
			// A modified click opens another tab or window rather than changing
			// this page's presentation.
			const modified =
				event instanceof MouseEvent &&
				(event.button !== 0 ||
					event.metaKey ||
					event.ctrlKey ||
					event.shiftKey ||
					event.altKey);
			if (link && !modified) {
				prefs = prefsFromHref(link.href, prefs, window.location.href);
				writePrefs(prefs);
			}
		},
		true,
	);

	// Swapped search markup is rendered from the cookie; re-sync in case only
	// localStorage holds the choice (for example, when cookies are blocked).
	const sync = (event: Event) => {
		// Another tab may have changed the remembered setting since this page
		// last read it.
		const stored = readStoredPrefs();
		if (stored) {
			prefs = stored;
		} else if (
			event.target instanceof Element &&
			event.target.id === "mc-area"
		) {
			// An enhanced navigation replaced the whole page. Unremembered
			// choices belonged to the previous page, so start from the defaults.
			prefs = { ...DEFAULT_PREFS };
		}
		applyActions(prefs.actions);
	};
	document.addEventListener("htmx:load", sync);
	document.addEventListener("htmx:historyRestore", sync);
};
