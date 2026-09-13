import { expect, test } from "bun:test";
import {
	addStudyBoardID,
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
