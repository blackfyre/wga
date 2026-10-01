// Scroll position contract for public navigation.
//
// In-page anchors: activating a link to a fragment of the current document
// adds .wga-smooth-scroll to <html> for the duration of the jump. The browser
// then performs its native fragment navigation (URL hash, :target, focus
// starting point, history entry) with smooth scroll-behavior, in the document
// and in Dual Mode panes. Smooth behaviour is not left on permanently because
// it would also animate scripted, history-restoration and automation scrolls.
// Under prefers-reduced-motion the class is never added.
//
// Page navigations: a swap of the main area that pushes or replaces the URL is
// a page navigation, so the new page starts at its top. Fragment updates
// (search results, filters, Dual Mode panes, trays, dialogs) target other
// elements and keep their position, even when they push a URL. HTMX raises
// htmx:beforeHistoryUpdate only for request-driven history updates, after
// caching the outgoing page and its scroll offset and in the same task as the
// DOM swap. Back therefore still restores the old position, and the new page is
// never painted at the old offset. For /page#section, HTMX scrolls the section
// into view after settling.

export const MAIN_AREA_ID = "mc-area";
export const SMOOTH_SCROLL_CLASS = "wga-smooth-scroll";
// Upper bound for a smooth jump when no scrollend event arrives, for example
// when the destination is already in place.
const SMOOTH_SCROLL_TIMEOUT_MS = 1500;

type HistoryUpdateDetail = {
	target?: { id?: string } | null;
	history?: { type?: string | null } | null;
};

export const isMainAreaNavigation = (
	detail: HistoryUpdateDetail | null | undefined,
): boolean => {
	const type = detail?.history?.type;
	return (
		detail?.target?.id === MAIN_AREA_ID &&
		(type === "push" || type === "replace")
	);
};

// isInPageFragmentLink reports whether following `href` from `current` is a
// fragment navigation within the same document.
export const isInPageFragmentLink = (
	href: string,
	current: string,
): boolean => {
	let link: URL;
	let page: URL;
	try {
		link = new URL(href, current);
		page = new URL(current);
	} catch {
		return false;
	}
	return (
		link.hash !== "" &&
		link.origin === page.origin &&
		link.pathname === page.pathname &&
		link.search === page.search
	);
};

const prefersReducedMotion = (): boolean =>
	window.matchMedia("(prefers-reduced-motion: reduce)").matches;

let initialised = false;

export const initNavigationScroll = (): void => {
	if (initialised) {
		return;
	}
	initialised = true;

	const root = document.documentElement;
	let smoothTimer: number | undefined;
	const endSmoothScroll = () => {
		window.clearTimeout(smoothTimer);
		smoothTimer = undefined;
		root.classList.remove(SMOOTH_SCROLL_CLASS);
	};

	document.addEventListener("click", (event) => {
		if (
			event.defaultPrevented ||
			event.button !== 0 ||
			event.metaKey ||
			event.ctrlKey ||
			event.shiftKey ||
			event.altKey ||
			!(event.target instanceof Element)
		) {
			return;
		}
		const link = event.target.closest<HTMLAnchorElement>("a[href]");
		if (
			!link ||
			(link.target !== "" && link.target !== "_self") ||
			link.hasAttribute("download") ||
			!isInPageFragmentLink(link.href, window.location.href) ||
			prefersReducedMotion()
		) {
			return;
		}
		// The default action runs after dispatch, so the jump picks the class up.
		root.classList.add(SMOOTH_SCROLL_CLASS);
		window.clearTimeout(smoothTimer);
		smoothTimer = window.setTimeout(endSmoothScroll, SMOOTH_SCROLL_TIMEOUT_MS);
	});
	// scrollend does not bubble; capture sees it for the window and for panes.
	document.addEventListener(
		"scrollend",
		() => {
			if (smoothTimer !== undefined) {
				endSmoothScroll();
			}
		},
		true,
	);

	document.addEventListener("htmx:beforeHistoryUpdate", (event) => {
		if (isMainAreaNavigation((event as CustomEvent).detail)) {
			endSmoothScroll();
			window.scrollTo({ top: 0, left: 0, behavior: "instant" });
		}
	});
};
