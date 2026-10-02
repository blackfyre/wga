import { afterAll, expect, test } from "bun:test";
import {
	applyPreferenceConsent,
	resetPreferenceConsentForTests,
} from "./preference-consent";
import {
	addStudyBoardID,
	initialiseStudyBoard,
	moveStudyBoardID,
	normaliseStudyBoardIDs,
	removeStudyBoardID,
	STUDY_BOARD_CAPACITY,
	STUDY_BOARD_STORAGE_KEY,
	studyBoardControlState,
	studyBoardPath,
} from "./study-board";
import {
	installPreferenceBrowser,
	restorePreferenceBrowser,
} from "./testing/preference-browser";

afterAll(restorePreferenceBrowser);

// A Study Board page whose root carries the given board IDs, recording every
// restore navigation.
const installBoardPage = (ids: string, urlState: boolean) => {
	const root = {
		dataset: {
			studyBoard: "",
			studyBoardIds: ids,
			...(urlState ? { studyBoardUrlState: "true" } : {}),
		} as Record<string, string>,
		querySelector: () => null,
	};
	const replacements: string[] = [];
	const browser = installPreferenceBrowser({
		window: { location: { replace: (url: string) => replacements.push(url) } },
		document: {
			querySelector: (selector: string) =>
				selector === "[data-study-board]" ? root : null,
		},
	});
	return { ...browser, replacements };
};

test("normalises duplicate and invalid board identifiers at capacity", () => {
	const valid = Array.from(
		{ length: STUDY_BOARD_CAPACITY + 2 },
		(_, i) => `work${String(i).padStart(11, "0")}`,
	);
	const result = normaliseStudyBoardIDs(
		[valid[0], "invalid id", valid[0], ...valid.slice(1)].join(","),
	);

	expect(result).toEqual(valid.slice(0, STUDY_BOARD_CAPACITY));
});

test("adds once and refuses overflow", () => {
	expect(addStudyBoardID(["one"], "two")).toEqual(["one", "two"]);
	expect(addStudyBoardID(["one", "two"], "one")).toEqual(["one", "two"]);
	const full = Array.from(
		{ length: STUDY_BOARD_CAPACITY },
		(_, i) => `work${i}`,
	);
	expect(addStudyBoardID(full, "overflow")).toEqual(full);
});

test("moves and removes while retaining common order", () => {
	expect(moveStudyBoardID(["one", "two", "three"], "two", "earlier")).toEqual([
		"two",
		"one",
		"three",
	]);
	expect(moveStudyBoardID(["one", "two", "three"], "two", "later")).toEqual([
		"one",
		"three",
		"two",
	]);
	expect(moveStudyBoardID(["one", "two"], "one", "earlier")).toEqual([
		"one",
		"two",
	]);
	expect(removeStudyBoardID(["one", "two", "three"], "two")).toEqual([
		"one",
		"three",
	]);
});

test("reports available, present, and full control states", () => {
	expect(studyBoardControlState(["one"], "two")).toBe("available");
	expect(studyBoardControlState(["one"], "one")).toBe("present");
	const full = Array.from(
		{ length: STUDY_BOARD_CAPACITY },
		(_, i) => `work${i}`,
	);
	expect(studyBoardControlState(full, "outside")).toBe("full");
});

test("builds the stable canonical board path", () => {
	expect(studyBoardPath([])).toBe("/study-board");
	expect(studyBoardPath(["work00000000000", "work00000000001"])).toBe(
		"/study-board?board=work00000000000,work00000000001",
	);
});

// The board module binds its consent store once per page load, so these tests
// share it and run in order.
test("a URL board shapes the page without replacing the remembered board", () => {
	resetPreferenceConsentForTests();
	const page = installBoardPage("work00000000001,work00000000002", true);
	page.grantConsent();
	page.storage.set(STUDY_BOARD_STORAGE_KEY, "work00000000009");
	initialiseStudyBoard();
	expect(page.replacements).toEqual([]);
	expect(page.storage.get(STUDY_BOARD_STORAGE_KEY)).toBe("work00000000009");

	// Withdrawing deletes the copy; a later first grant stores the URL board
	// the page holds.
	page.withdrawConsent();
	applyPreferenceConsent(false);
	expect(page.storage.has(STUDY_BOARD_STORAGE_KEY)).toBe(false);
	page.grantConsent();
	applyPreferenceConsent(true);
	expect(page.storage.get(STUDY_BOARD_STORAGE_KEY)).toBe(
		"work00000000001,work00000000002",
	);
});

// Keep this test last: the restore guard deliberately lives for the module's
// lifetime (one page load), so it stays set once this test has run.
test("requests a remembered-board restore once per page load", () => {
	const remembered = "work00000000000";
	const page = installBoardPage("", false);
	page.grantConsent();
	page.storage.set(STUDY_BOARD_STORAGE_KEY, remembered);

	// Bootstrap calls the initialiser directly and again from the initial
	// htmx:load; a second replace would abort the first navigation.
	initialiseStudyBoard();
	initialiseStudyBoard();

	expect(page.replacements).toEqual([studyBoardPath([remembered])]);
});
