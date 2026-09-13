import logger from "./logger";

const STUDY_BOARD_STORAGE_KEY = "wga-study-board";
export const STUDY_BOARD_CAPACITY = 12;

const recordIDPattern = /^[A-Za-z0-9_-]{1,255}$/;
let currentBoardIDs: string[] = [];
let eventsBound = false;

export function normaliseStudyBoardIDs(raw: string): string[] {
	const ids: string[] = [];
	const seen = new Set<string>();
	for (const value of raw.split(",")) {
		const id = value.trim();
		if (!recordIDPattern.test(id) || seen.has(id)) {
			continue;
		}
		seen.add(id);
		ids.push(id);
		if (ids.length === STUDY_BOARD_CAPACITY) {
			break;
		}
	}
	return ids;
}

export function studyBoardPath(ids: readonly string[]): string {
	return ids.length === 0
		? "/study-board"
		: `/study-board?board=${ids.join(",")}`;
}

export function addStudyBoardID(ids: readonly string[], id: string): string[] {
	const current = normaliseStudyBoardIDs(ids.join(","));
	if (
		!recordIDPattern.test(id) ||
		current.includes(id) ||
		current.length >= STUDY_BOARD_CAPACITY
	) {
		return current;
	}
	return [...current, id];
}

export function removeStudyBoardID(
	ids: readonly string[],
	id: string,
): string[] {
	return normaliseStudyBoardIDs(ids.join(",")).filter(
		(candidate) => candidate !== id,
	);
}

export function moveStudyBoardID(
	ids: readonly string[],
	id: string,
	direction: "earlier" | "later",
): string[] {
	const next = normaliseStudyBoardIDs(ids.join(","));
	const from = next.indexOf(id);
	const to = direction === "earlier" ? from - 1 : from + 1;
	if (from < 0 || to < 0 || to >= next.length) {
		return next;
	}
	[next[from], next[to]] = [next[to], next[from]];
	return next;
}

function readRememberedBoard(): string[] {
	try {
		return normaliseStudyBoardIDs(
			window.localStorage.getItem(STUDY_BOARD_STORAGE_KEY) ?? "",
		);
	} catch {
		return [];
	}
}

function rememberBoard(ids: readonly string[]): void {
	try {
		if (ids.length === 0) {
			window.localStorage.removeItem(STUDY_BOARD_STORAGE_KEY);
			return;
		}
		window.localStorage.setItem(STUDY_BOARD_STORAGE_KEY, ids.join(","));
	} catch {
		// Private browsing may make localStorage unavailable. The URL remains the
		// complete board state, so the workspace still functions honestly.
	}
}

function sameIDs(first: readonly string[], second: readonly string[]): boolean {
	return (
		first.length === second.length && first.every((id, i) => id === second[i])
	);
}

export function studyBoardControlState(
	ids: readonly string[],
	id: string,
): "available" | "present" | "full" {
	if (ids.includes(id)) {
		return "present";
	}
	if (ids.length >= STUDY_BOARD_CAPACITY) {
		return "full";
	}
	return "available";
}

function syncAddControls(): void {
	for (const button of document.querySelectorAll<HTMLButtonElement>(
		"[data-study-board-add]",
	)) {
		const state = studyBoardControlState(
			currentBoardIDs,
			button.dataset.studyBoardAdd ?? "",
		);
		const spent = state !== "available";
		button.disabled = spent;
		button.setAttribute("aria-disabled", String(spent));
		button.textContent =
			state === "present"
				? "ON STUDY BOARD ✓"
				: state === "full"
					? "STUDY BOARD FULL"
					: "ADD TO STUDY BOARD +";
		button.classList.toggle("cursor-not-allowed", spent);
		button.classList.toggle("hover:border-wga-accent", !spent);
		button.classList.toggle("hover:bg-wga-accent-tint", !spent);
	}
}

// Synchronises controls inserted by non-HTMX fragments without reloading the
// fixed shelf.
export function refreshStudyBoardControls(): void {
	syncAddControls();
}

async function refreshShelf(): Promise<void> {
	const mount = document.querySelector<HTMLElement>("#study-board-shelf");
	if (!mount) {
		return;
	}
	if (document.querySelector("[data-study-board]")) {
		mount.replaceChildren();
		return;
	}
	if (currentBoardIDs.length === 0) {
		mount.replaceChildren();
		return;
	}

	const requested = [...currentBoardIDs];
	try {
		const response = await fetch(
			`/study-board/shelf?board=${encodeURIComponent(requested.join(","))}`,
			{ headers: { Accept: "text/html" } },
		);
		if (!response.ok || !sameIDs(requested, currentBoardIDs)) {
			return;
		}
		const template = document.createElement("template");
		template.innerHTML = await response.text();
		mount.replaceChildren(template.content);
		const resolved = normaliseStudyBoardIDs(
			mount.querySelector<HTMLElement>("[data-study-board-shelf-content]")
				?.dataset.studyBoardIds ?? "",
		);
		if (!sameIDs(resolved, currentBoardIDs)) {
			currentBoardIDs = resolved;
			rememberBoard(resolved);
			syncAddControls();
		}
	} catch (error) {
		logger.warn("Unable to refresh Study Board shelf", error);
	}
}

function replaceBoard(ids: readonly string[]): void {
	const next = normaliseStudyBoardIDs(ids.join(","));
	currentBoardIDs = next;
	rememberBoard(next);
	if (document.querySelector("[data-study-board]")) {
		window.location.replace(studyBoardPath(next));
		return;
	}
	syncAddControls();
	void refreshShelf();
}

function setBoardView(view: "matrix" | "board"): void {
	for (const button of document.querySelectorAll<HTMLButtonElement>(
		"[data-study-board-view]",
	)) {
		const selected = button.dataset.studyBoardView === view;
		button.setAttribute("aria-pressed", String(selected));
		button.classList.toggle("border-wga-accent", selected);
		button.classList.toggle("bg-wga-accent-bg", selected);
		button.classList.toggle("text-wga-inv-fg", selected);
		button.classList.toggle("border-control", !selected);
		button.classList.toggle("text-muted", !selected);
	}
	for (const panel of document.querySelectorAll<HTMLElement>(
		"[data-study-board-panel]",
	)) {
		panel.classList.toggle("hidden", panel.dataset.studyBoardPanel !== view);
	}
}

type StudyBoardClickHandler = (target: Element, event: MouseEvent) => boolean;

const handleAddClick: StudyBoardClickHandler = (target, event) => {
	const button = target.closest<HTMLButtonElement>("[data-study-board-add]");
	if (!button) {
		return false;
	}
	event.preventDefault();
	event.stopPropagation();
	replaceBoard(
		addStudyBoardID(currentBoardIDs, button.dataset.studyBoardAdd ?? ""),
	);
	return true;
};

const handleRemoveClick: StudyBoardClickHandler = (target, event) => {
	const button = target.closest<HTMLButtonElement>("[data-study-board-remove]");
	if (!button) {
		return false;
	}
	event.preventDefault();
	replaceBoard(
		removeStudyBoardID(currentBoardIDs, button.dataset.studyBoardRemove ?? ""),
	);
	return true;
};

const handleMoveClick: StudyBoardClickHandler = (target, event) => {
	const button = target.closest<HTMLButtonElement>("[data-study-board-move]");
	if (!button) {
		return false;
	}
	event.preventDefault();
	const direction = button.dataset.studyBoardMove;
	if (direction === "earlier" || direction === "later") {
		replaceBoard(
			moveStudyBoardID(
				currentBoardIDs,
				button.dataset.studyBoardWorkId ?? "",
				direction,
			),
		);
	}
	return true;
};

const handleViewClick: StudyBoardClickHandler = (target) => {
	const button = target.closest<HTMLButtonElement>("[data-study-board-view]");
	const view = button?.dataset.studyBoardView;
	if (view !== "matrix" && view !== "board") {
		return false;
	}
	setBoardView(view);
	return true;
};

const handleShelfClearClick: StudyBoardClickHandler = (target, event) => {
	if (!target.closest("[data-clear-study-board-shelf]")) {
		return false;
	}
	event.preventDefault();
	if (window.confirm("Clear all works from this Study Board?")) {
		replaceBoard([]);
	}
	return true;
};

const studyBoardClickHandlers: StudyBoardClickHandler[] = [
	handleAddClick,
	handleRemoveClick,
	handleMoveClick,
	handleViewClick,
	handleShelfClearClick,
];

function handleStudyBoardPaletteKey(event: KeyboardEvent): void {
	const target = event.target as Element | null;
	const action = target?.closest<HTMLButtonElement>("[data-study-board-add]");
	if (action && event.key === "Enter") {
		event.preventDefault();
		event.stopPropagation();
		action.click();
		return;
	}
	if (action && event.key === "Tab") {
		event.stopPropagation();
		return;
	}
	if (
		event.key !== "Tab" ||
		event.shiftKey ||
		!target?.closest("#keyboard-palette")
	) {
		return;
	}
	const pickedAction = document.querySelector<HTMLButtonElement>(
		"#keyboard-palette [data-kbd-pick] [data-study-board-add]",
	);
	if (pickedAction) {
		event.preventDefault();
		event.stopPropagation();
		pickedAction.focus();
	}
}

function bindStudyBoardEvents(): void {
	if (eventsBound) {
		return;
	}
	eventsBound = true;
	document.addEventListener("keydown", handleStudyBoardPaletteKey, true);
	document.addEventListener("click", (event) => {
		const target = event.target as Element | null;
		if (!target) {
			return;
		}
		studyBoardClickHandlers.some((handler) => handler(target, event));
	});
}

async function copyCurrentLink(button: HTMLButtonElement): Promise<void> {
	let copied = false;
	try {
		await navigator.clipboard.writeText(window.location.href);
		copied = true;
	} catch {
		const field = document.createElement("textarea");
		field.value = window.location.href;
		document.body.appendChild(field);
		field.select();
		copied = document.execCommand("copy");
		field.remove();
	}

	if (!copied) {
		logger.warn("Unable to copy Study Board link");
		return;
	}
	button.textContent = "COPIED";
	window.setTimeout(() => {
		button.textContent = "COPY LINK";
	}, 2000);
}

/** Restores or remembers canonical state and binds the board's transient actions. */
export function initialiseStudyBoard(): void {
	const root = document.querySelector<HTMLElement>("[data-study-board]");
	bindStudyBoardEvents();
	if (root) {
		root.dataset.studyBoardBound = "true";
		const ids = normaliseStudyBoardIDs(root.dataset.studyBoardIds ?? "");
		if (root.dataset.studyBoardUrlState === "true") {
			currentBoardIDs = ids;
			rememberBoard(ids);
		} else {
			const remembered = readRememberedBoard();
			if (remembered.length > 0) {
				window.location.replace(studyBoardPath(remembered));
				return;
			}
			currentBoardIDs = [];
		}
	} else {
		currentBoardIDs = readRememberedBoard();
	}
	syncAddControls();
	void refreshShelf();

	const copyButton = root?.querySelector<HTMLButtonElement>(
		"[data-copy-study-board]",
	);
	if (copyButton && copyButton.dataset.studyBoardActionBound !== "true") {
		copyButton.dataset.studyBoardActionBound = "true";
		copyButton.addEventListener("click", (event) => {
			void copyCurrentLink(event.currentTarget as HTMLButtonElement);
		});
	}
	const clearLink = root?.querySelector<HTMLAnchorElement>(
		"[data-clear-study-board]",
	);
	if (clearLink && clearLink.dataset.studyBoardActionBound !== "true") {
		clearLink.dataset.studyBoardActionBound = "true";
		clearLink.addEventListener("click", (event) => {
			event.preventDefault();
			if (window.confirm("Clear all works from this Study Board?")) {
				replaceBoard([]);
			}
		});
	}
}
