import { expect, test } from "bun:test";
import {
	normaliseStudyBoardIDs,
	STUDY_BOARD_CAPACITY,
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

test("builds the stable canonical board path", () => {
	expect(studyBoardPath([])).toBe("/study-board");
	expect(studyBoardPath(["work00000000000", "work00000000001"])).toBe(
		"/study-board?board=work00000000000,work00000000001",
	);
});
