type Screen = {
	key: string;
	num: string;
	label: string;
	href: string;
};

const MAX_ITEMS = 9;
const NUMBER_TIMEOUT_MS = 1_000;
const SUGGEST_DEBOUNCE_MS = 140;
const SUGGEST_MINIMUM = 2;

let screens: Screen[] = [];
let caret = -1;
let pick = 0;
let numberBuffer = "";
let numberTimer = 0;
let suggestTimer = 0;
let request: AbortController | null = null;
let paletteInvoker: HTMLElement | null = null;
let helpInvoker: HTMLElement | null = null;

const editableTarget = (target: EventTarget | null) => {
	if (!(target instanceof HTMLElement)) {
		return false;
	}
	return (
		target.isContentEditable ||
		target.matches("input, textarea, select, [contenteditable='true']")
	);
};

const palette = () =>
	document.querySelector<HTMLDialogElement>("#keyboard-palette");
const help = () => document.querySelector<HTMLDialogElement>("#keyboard-help");
const mobileNavigation = () =>
	document.querySelector<HTMLDetailsElement>("[data-kbd-mobile-navigation]");

const nonKeyboardDialogOpen = () => {
	for (const dialog of document.querySelectorAll<HTMLDialogElement>("dialog")) {
		if (
			dialog.open &&
			dialog.id !== "keyboard-palette" &&
			dialog.id !== "keyboard-help"
		) {
			return true;
		}
	}
	return false;
};

const visible = (element: HTMLElement) => element.getClientRects().length > 0;

const restoreFocus = (invoker: HTMLElement | null) => {
	if (invoker?.isConnected) {
		invoker.focus();
	}
};

const clearNumberBuffer = () => {
	numberBuffer = "";
	window.clearTimeout(numberTimer);
	const readout = document.querySelector<HTMLElement>("#kbd-num");
	if (readout) {
		readout.textContent = "";
	}
};

const applyKeycaps = () => {
	const isMac = navigator.platform.includes("Mac");
	for (const keycap of document.querySelectorAll<HTMLElement>(
		"[data-kbd-modifier]",
	)) {
		if (isMac) {
			keycap.textContent = "⌘K";
		} else {
			keycap.textContent = "CTRL K";
		}
	}
};

const readScreens = () => {
	const holder = document.querySelector<HTMLElement>("#kbd-screens");
	if (!holder) {
		return [];
	}
	try {
		return JSON.parse(holder.dataset.json ?? "[]") as Screen[];
	} catch {
		return [];
	}
};

const markUsed = () => {
	document.documentElement.dataset.kbdOn = "true";
};

const list = () => {
	for (const candidate of document.querySelectorAll<HTMLElement>(
		"[data-kbd-list]",
	)) {
		if (!visible(candidate)) {
			continue;
		}
		const candidateRows =
			candidate.querySelectorAll<HTMLElement>("[data-kbd-idx]");
		if ([...candidateRows].some((row) => visible(row))) {
			return candidate;
		}
	}
	return null;
};

const rows = () => {
	const currentList = list();
	if (!currentList) {
		return [];
	}
	return [
		...currentList.querySelectorAll<HTMLElement>("[data-kbd-idx]"),
	].filter((row) => visible(row));
};

const columns = () => {
	const currentList = list();
	if (!currentList) {
		return 1;
	}
	let value = currentList.dataset.kbdCols;
	if (window.matchMedia("(min-width: 768px)").matches) {
		value = currentList.dataset.kbdColsMd ?? currentList.dataset.kbdCols;
	}
	const parsed = Number(value ?? "1");
	if (!Number.isFinite(parsed) || parsed < 1) {
		return 1;
	}
	return Math.floor(parsed);
};

const paintCaret = () => {
	for (const row of document.querySelectorAll<HTMLElement>(
		"[data-kbd-caret]",
	)) {
		row.removeAttribute("data-kbd-caret");
	}
	for (const [index, row] of rows().entries()) {
		row.toggleAttribute("data-kbd-caret", index === caret);
	}
};

const clearCaret = () => {
	caret = -1;
	paintCaret();
};

const moveCaret = (delta: number) => {
	const currentRows = rows();
	if (currentRows.length === 0) {
		return false;
	}
	if (caret < 0) {
		caret = 0;
	} else {
		caret = Math.max(0, Math.min(currentRows.length - 1, caret + delta));
	}
	markUsed();
	paintCaret();
	let behavior: ScrollBehavior = "smooth";
	if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
		behavior = "auto";
	}
	currentRows[caret].scrollIntoView({
		block: "nearest",
		behavior,
	});
	return true;
};

const openMarked = () => {
	const row = rows()[caret];
	if (!row) {
		return false;
	}
	let href = row.dataset.kbdHref;
	if (!href && row instanceof HTMLAnchorElement) {
		href = row.href;
	}
	if (!href) {
		href = row.querySelector<HTMLAnchorElement>("a[href]")?.href;
	}
	if (href) {
		navigate(href);
		return true;
	}
	return false;
};

const navigate = (href: string) => {
	closePalette();
	closeHelp();
	markUsed();
	window.location.assign(href);
};

const paletteItems = () => {
	const container = document.querySelector<HTMLElement>("#kbd-palette-results");
	if (!container) {
		return [];
	}
	return [...container.querySelectorAll<HTMLElement>("[data-kbd-item]")].filter(
		(item) => !item.hidden,
	);
};

const paintPick = () => {
	const items = paletteItems();
	if (items.length === 0) {
		pick = 0;
		return;
	}
	pick = ((pick % items.length) + items.length) % items.length;
	for (const [index, item] of items.entries()) {
		item.toggleAttribute("data-kbd-pick", index === pick);
	}
	items[pick].scrollIntoView({ block: "nearest" });
};

const closePalette = () => {
	window.clearTimeout(suggestTimer);
	request?.abort();
	request = null;
	palette()?.close();
};

const closeHelp = () => {
	help()?.close();
};

const openPalette = () => {
	const dialog = palette();
	if (!dialog) {
		return;
	}
	closeHelp();
	markUsed();
	pick = 0;
	if (!dialog.open) {
		if (document.activeElement instanceof HTMLElement) {
			paletteInvoker = document.activeElement;
		}
		dialog.showModal();
	}
	const field = dialog.querySelector<HTMLInputElement>("[data-kbd-query]");
	if (field) {
		field.value = "";
		field.focus();
	}
	filterPalette("");
};

const openHelp = () => {
	const dialog = help();
	if (!dialog) {
		return;
	}
	closePalette();
	markUsed();
	if (!dialog.open) {
		if (document.activeElement instanceof HTMLElement) {
			helpInvoker = document.activeElement;
		}
		dialog.showModal();
	}
	dialog.querySelector<HTMLElement>("[data-keyboard-close]")?.focus();
};

const loadSuggestions = async (query: string, limit: number) => {
	const target = document.querySelector<HTMLElement>("#kbd-palette-records");
	if (!target || limit < 1) {
		return;
	}
	request?.abort();
	const controller = new AbortController();
	request = controller;
	const path = target.dataset.kbdSuggest ?? "/keyboard/suggestions";
	try {
		const response = await fetch(
			`${path}?q=${encodeURIComponent(query)}&limit=${limit}`,
			{ headers: { "HX-Request": "true" }, signal: controller.signal },
		);
		if (!response.ok || request !== controller) {
			return;
		}
		target.innerHTML = await response.text();
		if (request !== controller) {
			return;
		}
		pick = 0;
		paintPick();
	} catch (error) {
		if (error instanceof DOMException && error.name === "AbortError") {
			return;
		}
		if (request === controller) {
			target.replaceChildren();
		}
	}
};

const filterPalette = (query: string) => {
	const normalized = query.trim().toLowerCase();
	let visibleSections = 0;
	for (const item of document.querySelectorAll<HTMLElement>(
		'[data-kbd-item="section"]',
	)) {
		const label = item.dataset.kbdLabel?.toLowerCase() ?? "";
		const key = item.dataset.kbdKey?.toLowerCase() ?? "";
		const number = item.dataset.kbdNum ?? "";
		const matches =
			normalized === "" ||
			label.includes(normalized) ||
			key === normalized ||
			number.startsWith(normalized);
		item.hidden = !matches || visibleSections >= MAX_ITEMS;
		if (!item.hidden) {
			visibleSections += 1;
		}
	}

	const records = document.querySelector<HTMLElement>("#kbd-palette-records");
	if (!records) {
		paintPick();
		return;
	}
	window.clearTimeout(suggestTimer);
	records.replaceChildren();
	pick = 0;
	if (normalized.length < SUGGEST_MINIMUM || visibleSections >= MAX_ITEMS) {
		paintPick();
		return;
	}
	suggestTimer = window.setTimeout(() => {
		void loadSuggestions(normalized, MAX_ITEMS - visibleSections);
	}, SUGGEST_DEBOUNCE_MS);
	paintPick();
};

const paletteKey = (event: KeyboardEvent) => {
	if (event.key === "Escape") {
		event.preventDefault();
		closePalette();
		return;
	}
	if (event.key === "ArrowDown" || (event.key === "Tab" && !event.shiftKey)) {
		event.preventDefault();
		pick += 1;
		paintPick();
		return;
	}
	if (event.key === "ArrowUp" || (event.key === "Tab" && event.shiftKey)) {
		event.preventDefault();
		pick -= 1;
		paintPick();
		return;
	}
	if (event.key === "Enter") {
		event.preventDefault();
		const item = paletteItems()[pick];
		const href =
			item?.dataset.kbdHref ??
			item?.querySelector<HTMLAnchorElement>("a")?.href;
		if (href) {
			navigate(href);
		}
	}
};

const numberKey = (event: KeyboardEvent) => {
	screens = readScreens();
	numberBuffer = (numberBuffer + event.key).slice(-2);
	const readout = document.querySelector<HTMLElement>("#kbd-num");
	if (readout) {
		readout.textContent = numberBuffer;
	}
	window.clearTimeout(numberTimer);
	numberTimer = window.setTimeout(() => {
		numberBuffer = "";
		if (readout) {
			readout.textContent = "";
		}
	}, NUMBER_TIMEOUT_MS);
	const screen = screens.find((candidate) => candidate.num === numberBuffer);
	if (screen) {
		window.location.assign(screen.href);
	}
};

const closeTransientState = () => {
	closePalette();
	closeHelp();
	mobileNavigation()?.removeAttribute("open");
	clearCaret();
	clearNumberBuffer();
};

const focusSearch = () => {
	const navigation = mobileNavigation();
	if (navigation && window.matchMedia("(max-width: 44.999rem)").matches) {
		navigation.setAttribute("open", "");
		const field =
			navigation.querySelector<HTMLInputElement>("[data-kbd-search]");
		field?.focus();
		field?.select();
		return;
	}
	for (const field of document.querySelectorAll<HTMLInputElement>(
		"[data-kbd-search]",
	)) {
		if (visible(field)) {
			field.focus();
			field.select();
			return;
		}
	}
};

const closeMobileNavigationForDesktop = () => {
	const navigation = mobileNavigation();
	if (!navigation || !navigation.hasAttribute("open")) {
		return;
	}
	const active = document.activeElement;
	const focusWasInside =
		active === null ||
		active === document.body ||
		active === document.documentElement ||
		(active instanceof HTMLElement &&
			(navigation.contains(active) || !visible(active)));
	navigation.removeAttribute("open");
	if (!focusWasInside) {
		return;
	}
	for (const field of document.querySelectorAll<HTMLInputElement>(
		"[data-kbd-search]",
	)) {
		if (visible(field) && !navigation.contains(field)) {
			field.focus();
			return;
		}
	}
};

export const initKeyboardNavigation = () => {
	if (document.documentElement.dataset.keyboardNavigationReady === "true") {
		return;
	}
	document.documentElement.dataset.keyboardNavigationReady = "true";
	screens = readScreens();
	applyKeycaps();

	window
		.matchMedia("(min-width: 45rem)")
		.addEventListener("change", (event) => {
			if (event.matches) {
				closeMobileNavigationForDesktop();
			}
		});

	document.addEventListener("click", (event) => {
		const target = event.target instanceof Element ? event.target : null;
		const navigation = mobileNavigation();
		if (navigation?.open && target && !navigation.contains(target)) {
			navigation.removeAttribute("open");
		}
		if (target?.closest("[data-keyboard-open]")) {
			event.preventDefault();
			openPalette();
			return;
		}
		if (target?.closest("[data-keyboard-help]")) {
			event.preventDefault();
			openHelp();
			return;
		}
		if (target?.closest("[data-keyboard-close]")) {
			closeTransientState();
		}
	});

	document.addEventListener("input", (event) => {
		const target = event.target;
		if (
			target instanceof HTMLInputElement &&
			target.matches("[data-kbd-query]")
		) {
			filterPalette(target.value);
		}
	});

	document.addEventListener(
		"close",
		(event) => {
			if (!(event.target instanceof HTMLDialogElement)) {
				return;
			}
			if (event.target.id === "keyboard-palette") {
				window.clearTimeout(suggestTimer);
				request?.abort();
				request = null;
				pick = 0;
				const field =
					event.target.querySelector<HTMLInputElement>("[data-kbd-query]");
				if (field) {
					field.value = "";
				}
				filterPalette("");
				restoreFocus(paletteInvoker);
				paletteInvoker = null;
			}
			if (event.target.id === "keyboard-help") {
				restoreFocus(helpInvoker);
				helpInvoker = null;
			}
		},
		true,
	);

	document.addEventListener("keydown", (event) => {
		const modified = event.metaKey || event.ctrlKey;
		if (modified && event.key.toLowerCase() === "k") {
			event.preventDefault();
			openPalette();
			return;
		}
		if (palette()?.open) {
			paletteKey(event);
			return;
		}
		if (nonKeyboardDialogOpen()) {
			return;
		}
		if (event.key === "Escape") {
			event.preventDefault();
			if (editableTarget(event.target)) {
				(event.target as HTMLElement).blur();
			}
			const navigation = mobileNavigation();
			const navigationWasOpen = navigation?.hasAttribute("open") ?? false;
			closeTransientState();
			if (navigationWasOpen) {
				navigation?.querySelector<HTMLElement>("summary")?.focus();
			}
			return;
		}
		if (editableTarget(event.target)) {
			return;
		}
		if (modified || event.altKey) {
			return;
		}
		if (event.key === "?") {
			event.preventDefault();
			if (help()?.open) {
				closeHelp();
			} else {
				openHelp();
			}
			return;
		}
		if (event.key === "/") {
			event.preventDefault();
			focusSearch();
			markUsed();
			return;
		}
		let moved = false;
		if (event.key === "ArrowDown" || event.key.toLowerCase() === "j") {
			moved = moveCaret(columns());
		}
		if (event.key === "ArrowUp" || event.key.toLowerCase() === "k") {
			moved = moveCaret(-columns());
		}
		if (event.key === "ArrowRight" || event.key.toLowerCase() === "l") {
			moved = moveCaret(1);
		}
		if (event.key === "ArrowLeft" || event.key.toLowerCase() === "h") {
			moved = moveCaret(-1);
		}
		if (moved) {
			event.preventDefault();
			return;
		}
		if (
			event.key === "Enter" &&
			!(
				event.target instanceof HTMLElement &&
				event.target.matches("a, button, summary")
			)
		) {
			if (openMarked()) {
				event.preventDefault();
				return;
			}
		}
		if (/^[0-9]$/.test(event.key)) {
			event.preventDefault();
			numberKey(event);
			return;
		}
		if (event.key.length !== 1) {
			return;
		}
		screens = readScreens();
		const screen = screens.find(
			(candidate) => candidate.key.toLowerCase() === event.key.toLowerCase(),
		);
		if (screen) {
			window.location.assign(screen.href);
		}
	});

	document.addEventListener("htmx:afterSettle", () => {
		screens = readScreens();
		clearCaret();
		clearNumberBuffer();
		applyKeycaps();
	});
};
