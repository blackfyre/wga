import logger from "./logger";

const STUDY_BOARD_STORAGE_KEY = "wga-study-board";
export const STUDY_BOARD_CAPACITY = 12;

const recordIDPattern = /^[A-Za-z0-9_-]{1,255}$/;

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
	if (!root || root.dataset.studyBoardBound === "true") {
		return;
	}
	root.dataset.studyBoardBound = "true";

	const ids = normaliseStudyBoardIDs(root.dataset.studyBoardIds ?? "");
	if (root.dataset.studyBoardUrlState === "true") {
		rememberBoard(ids);
	} else {
		const remembered = readRememberedBoard();
		if (remembered.length > 0) {
			window.location.replace(studyBoardPath(remembered));
			return;
		}
	}

	root
		.querySelector<HTMLButtonElement>("[data-copy-study-board]")
		?.addEventListener("click", (event) => {
			void copyCurrentLink(event.currentTarget as HTMLButtonElement);
		});
	root
		.querySelector<HTMLAnchorElement>("[data-clear-study-board]")
		?.addEventListener("click", (event) => {
			event.preventDefault();
			rememberBoard([]);
			window.location.replace("/study-board");
		});
}
