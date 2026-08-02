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

const list = () => document.querySelector<HTMLElement>("[data-kbd-list]");

const rows = () => {
	const currentList = list();
	if (!currentList) {
		return [];
	}
	return [...currentList.querySelectorAll<HTMLElement>("[data-kbd-idx]")];
};

const columns = () => {
	const currentList = list();
	if (!currentList) {
		return 1;
	}
	const value = window.matchMedia("(min-width: 768px)").matches
		? (currentList.dataset.kbdColsMd ?? currentList.dataset.kbdCols)
		: currentList.dataset.kbdCols;
	const parsed = Number(value ?? "1");
	if (!Number.isFinite(parsed) || parsed < 1) {
		return 1;
	}
	return Math.floor(parsed);
};

const paintCaret = () => {
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
		return;
	}
	if (caret < 0) {
		caret = 0;
	} else {
		caret = Math.max(0, Math.min(currentRows.length - 1, caret + delta));
	}
	markUsed();
	paintCaret();
	currentRows[caret].scrollIntoView({
		block: "nearest",
		behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches
			? "auto"
			: "smooth",
	});
};

const openMarked = () => {
	const row = rows()[caret];
	if (!row) {
		return;
	}
	const href =
		row.dataset.kbdHref ??
		row.querySelector<HTMLAnchorElement>("a[href]")?.href;
	if (href) {
		window.location.assign(href);
	}
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
	pick = Math.max(0, Math.min(items.length - 1, pick));
	for (const [index, item] of items.entries()) {
		item.toggleAttribute("data-kbd-pick", index === pick);
	}
	items[pick].scrollIntoView({ block: "nearest" });
};

const closePalette = () => {
	palette()?.close();
	request?.abort();
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
		dialog.showModal();
	}
	dialog.querySelector<HTMLElement>("[data-keyboard-close]")?.focus();
};

const loadSuggestions = async (query: string, limit: number) => {
	const target = document.querySelector<HTMLElement>(
		"[data-keyboard-suggestions]",
	);
	if (!target || limit < 1) {
		return;
	}
	request?.abort();
	request = new AbortController();
	const path = target.dataset.kbdSuggest ?? "/keyboard/suggestions";
	try {
		const response = await fetch(
			`${path}?q=${encodeURIComponent(query)}&limit=${limit}`,
			{ signal: request.signal },
		);
		if (!response.ok) {
			return;
		}
		target.innerHTML = await response.text();
		pick = 0;
		paintPick();
	} catch (error) {
		if (error instanceof DOMException && error.name === "AbortError") {
			return;
		}
		target.replaceChildren();
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

	const records = document.querySelector<HTMLElement>(
		"[data-keyboard-suggestions]",
	);
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
			window.location.assign(href);
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
	document.querySelector<HTMLDialogElement>("#d")?.close();
	document.querySelector<HTMLDialogElement>("#artist_lookup")?.close();
	mobileNavigation()?.removeAttribute("open");
	clearCaret();
};

const focusSearch = () => {
	const navigation = mobileNavigation();
	if (navigation?.offsetParent !== null) {
		navigation.setAttribute("open", "");
		return;
	}
	document.querySelector<HTMLInputElement>("[data-kbd-search]")?.focus();
};

export const initKeyboardNavigation = () => {
	if (document.documentElement.dataset.keyboardNavigationReady === "true") {
		return;
	}
	document.documentElement.dataset.keyboardNavigationReady = "true";
	screens = readScreens();

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
		if (event.key === "Escape") {
			event.preventDefault();
			if (editableTarget(event.target)) {
				(event.target as HTMLElement).blur();
			}
			closeTransientState();
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
		if (event.key === "ArrowDown" || event.key.toLowerCase() === "j") {
			event.preventDefault();
			moveCaret(columns());
			return;
		}
		if (event.key === "ArrowUp" || event.key.toLowerCase() === "k") {
			event.preventDefault();
			moveCaret(-columns());
			return;
		}
		if (event.key === "ArrowRight") {
			event.preventDefault();
			moveCaret(1);
			return;
		}
		if (event.key === "ArrowLeft") {
			event.preventDefault();
			moveCaret(-1);
			return;
		}
		if (
			event.key === "Enter" &&
			!(event.target instanceof HTMLElement && event.target.matches("a, button, summary"))
		) {
			event.preventDefault();
			openMarked();
			return;
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
	});
};
