const editableTarget = (target: EventTarget | null) => {
	if (!(target instanceof HTMLElement)) {
		return false;
	}
	return (
		target.isContentEditable ||
		target.matches("input, textarea, select, [contenteditable='true']")
	);
};

const routes: Record<string, string> = {
	h: "/",
	a: "/artists",
	w: "/artworks",
	d: "/dual-mode",
	i: "/inspire",
	g: "/guestbook",
	s: "/statistics",
	l: "/glossary",
	c: "/contributors",
	p: "/postcard",
	b: "/pages/about",
};

let selectedIndex = -1;
let request: AbortController | null = null;

const palette = () =>
	document.querySelector<HTMLDialogElement>("#keyboard-palette");
const help = () => document.querySelector<HTMLDialogElement>("#keyboard-help");

const resetSelection = () => {
	selectedIndex = -1;
	for (const item of document.querySelectorAll<HTMLElement>(
		"[data-keyboard-item]",
	)) {
		item.removeAttribute("data-keyboard-current");
	}
};

const keyboardItems = () => {
	const dialog = palette();
	if (dialog?.open) {
		return [
			...dialog.querySelectorAll<HTMLAnchorElement>("[data-keyboard-item]"),
		];
	}
	return [
		...document.querySelectorAll<HTMLAnchorElement>(
			"[data-keyboard-list] [data-keyboard-item]",
		),
	];
};

const moveSelection = (direction: number) => {
	const items = keyboardItems();
	if (items.length === 0) {
		return;
	}
	selectedIndex = (selectedIndex + direction + items.length) % items.length;
	for (const [index, item] of items.entries()) {
		item.toggleAttribute("data-keyboard-current", index === selectedIndex);
	}
	items[selectedIndex].scrollIntoView({ block: "nearest" });
};

const openSelected = () => {
	const items = keyboardItems();
	if (selectedIndex >= 0 && items[selectedIndex]) {
		items[selectedIndex].click();
	}
};

const closeDialogs = () => {
	palette()?.close();
	help()?.close();
	resetSelection();
};

const openPalette = () => {
	const dialog = palette();
	if (!dialog) {
		return;
	}
	if (!dialog.open) {
		dialog.showModal();
	}
	dialog.querySelector<HTMLInputElement>("[data-keyboard-search]")?.focus();
};

const openHelp = () => {
	const dialog = help();
	if (!dialog) {
		return;
	}
	if (!dialog.open) {
		dialog.showModal();
	}
	dialog.querySelector<HTMLButtonElement>("[data-keyboard-close]")?.focus();
};

const loadSuggestions = async (query: string) => {
	const target = document.querySelector<HTMLElement>(
		"[data-keyboard-suggestions]",
	);
	if (!target) {
		return;
	}
	if (query.trim().length < 2) {
		target.replaceChildren();
		resetSelection();
		return;
	}
	request?.abort();
	request = new AbortController();
	try {
		const response = await fetch(
			`/keyboard/suggestions?q=${encodeURIComponent(query)}`,
			{ signal: request.signal },
		);
		if (!response.ok) {
			return;
		}
		target.innerHTML = await response.text();
		resetSelection();
	} catch (error) {
		if (!(error instanceof DOMException && error.name === "AbortError")) {
			throw error;
		}
	}
};

export const initKeyboardNavigation = () => {
	if (document.documentElement.dataset.keyboardNavigationReady === "true") {
		return;
	}
	document.documentElement.dataset.keyboardNavigationReady = "true";

	document.addEventListener("click", (event) => {
		const target = event.target instanceof Element ? event.target : null;
		if (target?.closest("[data-keyboard-open]")) {
			event.preventDefault();
			openPalette();
		}
		if (target?.closest("[data-keyboard-help]")) {
			event.preventDefault();
			openHelp();
		}
		if (target?.closest("[data-keyboard-close]")) {
			closeDialogs();
		}
	});
	document.addEventListener("input", (event) => {
		const target = event.target;
		if (
			target instanceof HTMLInputElement &&
			target.matches("[data-keyboard-search]")
		) {
			void loadSuggestions(target.value);
		}
	});
	document.addEventListener("keydown", (event) => {
		if (event.key === "Escape") {
			closeDialogs();
			return;
		}
		if (editableTarget(event.target)) {
			return;
		}
		if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
			event.preventDefault();
			openPalette();
			return;
		}
		if (event.key === "?" || (event.key === "/" && event.shiftKey)) {
			event.preventDefault();
			openHelp();
			return;
		}
		if (event.key === "/") {
			const search = document.querySelector<HTMLInputElement>(
				"[data-keyboard-page-search]",
			);
			if (search) {
				event.preventDefault();
				search.focus();
			}
			return;
		}
		if (event.key === "ArrowDown" || event.key.toLowerCase() === "j") {
			event.preventDefault();
			moveSelection(1);
			return;
		}
		if (event.key === "ArrowUp" || event.key.toLowerCase() === "k") {
			event.preventDefault();
			moveSelection(-1);
			return;
		}
		if (event.key === "Enter") {
			openSelected();
			return;
		}
		const route = routes[event.key.toLowerCase()];
		if (!event.metaKey && !event.ctrlKey && !event.altKey && route) {
			window.location.assign(route);
		}
	});
	document.addEventListener("htmx:afterSettle", resetSelection);
};
