// navigationPath resolves the public record aliases to the top-level
// destination that owns them. This mirrors the server-side shell helper.
export const navigationPath = (pathname: string): string => {
	const segments = pathname.split("/").filter(Boolean);
	if (segments[0] !== "artist" && segments[0] !== "artists") {
		return pathname;
	}

	if (segments.length === 3 && segments[2] !== "selections") {
		return "/artworks";
	}
	if (segments.length >= 2) {
		return "/artists";
	}

	return pathname;
};

export const activeNavigationPath = (
	pathname: string,
	destinations: string[],
): string | null => {
	const currentPath = navigationPath(pathname);
	let active: string | null = null;
	for (const path of destinations) {
		const matches =
			path === currentPath ||
			(path !== "/" && currentPath.startsWith(`${path}/`));
		if (matches && (active === null || path.length > active.length)) {
			active = path;
		}
	}
	return active;
};

const syncLinks = (
	selector: string,
	activeClasses: string[],
	currentPath: string,
) => {
	const links = Array.from(
		document.querySelectorAll<HTMLAnchorElement>(selector),
	).filter((link) => link.getAttribute("href")?.startsWith("/"));
	const active = activeNavigationPath(
		currentPath,
		links.map((link) => link.pathname),
	);
	for (const link of links) {
		const isActive = link.pathname === active;
		for (const activeClass of activeClasses) {
			link.classList.toggle(activeClass, isActive);
		}
		if (isActive) {
			link.setAttribute("aria-current", "page");
		} else {
			link.removeAttribute("aria-current");
		}
	}
};

// syncNavigation reconciles the persistent shell after an HTMX swap. The
// server marks a complete document correctly; this repeats that state for the
// desktop primary and MORE destinations, mobile navigation, footer, and both
// home identities when only #mc-area was replaced.
export const syncNavigation = () => {
	const currentPath = window.location.pathname;
	syncLinks(
		"header > nav[aria-label='Primary navigation'] > ul a",
		["border-primary", "text-primary"],
		currentPath,
	);
	syncLinks(
		"header > nav[aria-label='Primary navigation'] > details a",
		["bg-primary/10", "text-primary"],
		currentPath,
	);
	syncLinks(
		"[data-mobile-navigation] a",
		["bg-primary", "text-primary-content", "pl-3"],
		currentPath,
	);
	syncLinks("footer a", ["text-primary"], currentPath);
	syncLinks("header a[href='/']", [], currentPath);

	const more = document.querySelector<HTMLDetailsElement>(
		"header > nav[aria-label='Primary navigation'] > details",
	);
	if (more !== null) {
		const active = more.querySelector("a[aria-current='page']") !== null;
		const summary = more.querySelector("summary");
		summary?.classList.toggle("border-primary", active);
		summary?.classList.toggle("text-primary", active);
	}
};

// closeMobileNavigation is delegated from document so links introduced by an
// HTMX replacement retain the disclosure-close behaviour without inline code.
export const closeMobileNavigation = (event: Event) => {
	if (!(event.target instanceof Element)) {
		return;
	}

	const link = event.target.closest<HTMLAnchorElement>(
		"[data-mobile-navigation] a",
	);
	if (link === null) {
		return;
	}

	const disclosure = link.closest<HTMLDetailsElement>(
		"details[data-kbd-mobile-navigation]",
	);
	if (disclosure !== null) {
		disclosure.removeAttribute("open");
	}
};
