import { expect, test } from "bun:test";
import {
	isInPageFragmentLink,
	isMainAreaNavigation,
} from "./navigation-scroll";

test("only main-area swaps that update the URL count as navigations", () => {
	const main = { id: "mc-area" };
	expect(
		isMainAreaNavigation({ target: main, history: { type: "push" } }),
	).toBe(true);
	expect(
		isMainAreaNavigation({ target: main, history: { type: "replace" } }),
	).toBe(true);
	expect(isMainAreaNavigation({ target: main, history: {} })).toBe(false);
	expect(isMainAreaNavigation({ target: main })).toBe(false);
	for (const id of [
		"global-search-results",
		"artwork-search",
		"dual-left",
		"dual-area",
		"itinerary-tray",
		"d",
	]) {
		expect(
			isMainAreaNavigation({ target: { id }, history: { type: "push" } }),
		).toBe(false);
	}
	expect(isMainAreaNavigation(null)).toBe(false);
	expect(isMainAreaNavigation({ target: null })).toBe(false);
});

test("recognises fragment links within the current document only", () => {
	const current = "https://wga.test/artists/a?view=list#biography";
	expect(isInPageFragmentLink("#cite-this-record", current)).toBe(true);
	expect(
		isInPageFragmentLink("/artists/a?view=list#cite-this-record", current),
	).toBe(true);
	expect(isInPageFragmentLink("#top", current)).toBe(true);
	expect(isInPageFragmentLink("/artists/a#cite-this-record", current)).toBe(
		false,
	);
	expect(isInPageFragmentLink("/artists/b#cite-this-record", current)).toBe(
		false,
	);
	expect(
		isInPageFragmentLink(
			"https://other.test/artists/a?view=list#cite-this-record",
			current,
		),
	).toBe(false);
	expect(isInPageFragmentLink("/artists/a?view=list", current)).toBe(false);
	expect(isInPageFragmentLink("#", current)).toBe(false);
});
