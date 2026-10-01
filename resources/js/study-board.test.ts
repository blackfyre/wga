import { expect, test } from "bun:test";
import {
	addStudyBoardID,
	initialiseStudyBoard,
	moveStudyBoardID,
	normaliseStudyBoardIDs,
	removeStudyBoardID,
	STUDY_BOARD_CAPACITY,
	studyBoardControlState,
	studyBoardPath,
} from "./study-board";

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

// Keep this test last: the restore guard deliberately lives for the module's
// lifetime (one page load), so it stays set once this test has run.
test("requests a remembered-board restore once per page load", () => {
	const previousDocument = globalThis.document;
	const previousWindow = globalThis.window;
	const remembered = "work00000000000";
	const root = {
		dataset: { studyBoard: "", studyBoardIds: "" } as Record<string, string>,
		querySelector: () => null,
	};
	globalThis.document = {
		querySelector: (selector: string) =>
			selector === "[data-study-board]" ? root : null,
		querySelectorAll: () => [],
		addEventListener: () => {},
	} as unknown as Document;
	const replacements: string[] = [];
	globalThis.window = {
		localStorage: { getItem: () => remembered },
		location: { replace: (url: string) => replacements.push(url) },
	} as unknown as Window & typeof globalThis;

	// Bootstrap calls the initialiser directly and again from the initial
	// htmx:load; a second replace would abort the first navigation.
	try {
		initialiseStudyBoard();
		initialiseStudyBoard();

		expect(replacements).toEqual([studyBoardPath([remembered])]);
	} finally {
		globalThis.document = previousDocument;
		globalThis.window = previousWindow;
	}
});
