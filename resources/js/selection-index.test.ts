import { expect, test } from "bun:test";
import {
	currentFormId,
	foldPresentation,
	readingBandMargin,
} from "./selection-index";

const order = ["form-a", "form-b", "form-c"];

test("the current form is the first block crossing the reading band", () => {
	expect(currentFormId(order, new Set(["form-b"]), null)).toBe("form-b");
	expect(currentFormId(order, new Set(["form-c", "form-b"]), "form-a")).toBe(
		"form-b",
	);
});

test("between blocks the previous form stays current", () => {
	expect(currentFormId(order, new Set(), "form-b")).toBe("form-b");
	expect(currentFormId(order, new Set(), null)).toBeNull();
});

test("only form blocks count, so tiles never become sections", () => {
	// A tile id is never in the observed order, even if reported visible.
	expect(
		currentFormId(order, new Set(["selection-1", "selection-2"]), "form-a"),
	).toBe("form-a");
});

test("the reading band is measured in pixels from the viewport height", () => {
	// 1440×900: the band spans 90px to 360px whatever the viewport width.
	expect(readingBandMargin(900)).toBe("-90px 0px -540px 0px");
	expect(readingBandMargin(844)).toBe("-84px 0px -506px 0px");
});

test("fold controls report their state through aria-expanded", () => {
	expect(foldPresentation(true)).toEqual({ label: "−", expanded: "true" });
	expect(foldPresentation(false)).toEqual({ label: "+", expanded: "false" });
});
