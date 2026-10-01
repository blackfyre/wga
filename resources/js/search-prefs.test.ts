import { expect, test } from "bun:test";
import {
	actionsLabel,
	consentAllowsPreferences,
	cookieValue,
	DEFAULT_PREFS,
	parsePrefs,
	prefsFromHref,
	readCookie,
} from "./search-prefs";

test("parses stored preferences field by field", () => {
	expect(parsePrefs(null)).toBeNull();
	expect(parsePrefs("not json")).toBeNull();
	expect(parsePrefs("[]")).toBeNull();
	expect(
		parsePrefs('{"sort":"date","dir":"desc","view":"list","actions":true}'),
	).toEqual({ sort: "date", dir: "desc", view: "list", actions: true });
	expect(
		parsePrefs(
			'{"sort":"catalogue","dir":"down","view":"list","actions":"yes"}',
		),
	).toEqual({ ...DEFAULT_PREFS, view: "list" });
});

test("reads the addressed sort and view from a search link", () => {
	const current = { ...DEFAULT_PREFS, actions: true };
	const base = "https://wga.example/artworks";
	expect(prefsFromHref("/artworks?sort=date&dir=desc", current, base)).toEqual({
		sort: "date",
		dir: "desc",
		view: "grid",
		actions: true,
	});
	expect(
		prefsFromHref(
			"/artworks?sort=artist&view=list",
			{ ...current, sort: "date", dir: "desc" },
			base,
		),
	).toEqual({ sort: "artist", dir: "asc", view: "list", actions: true });
	expect(
		prefsFromHref("/artworks", { ...current, view: "list" }, base),
	).toEqual(current);
});

test("round-trips the cookie value", () => {
	const prefs = {
		sort: "date",
		dir: "desc",
		view: "list",
		actions: true,
	} as const;
	const value = cookieValue(prefs);
	expect(value).not.toContain('"');
	expect(value).not.toContain(",");
	expect(
		parsePrefs(readCookie(`other=1; wga_aw_prefs=${value}`, "wga_aw_prefs")),
	).toEqual(prefs);
	expect(readCookie("other=1", "wga_aw_prefs")).toBeNull();
});

test("labels the actions toggle", () => {
	expect(actionsLabel(false)).toBe("ACTIONS +");
	expect(actionsLabel(true)).toBe("ACTIONS ✓");
});

test("remembers only with the preferences consent category", () => {
	const consent = (categories: string[]) =>
		`cc_cookie=${encodeURIComponent(JSON.stringify({ categories, revision: 0 }))}`;
	expect(consentAllowsPreferences("")).toBe(false);
	expect(consentAllowsPreferences(consent(["necessary"]))).toBe(false);
	expect(
		consentAllowsPreferences(`a=1; ${consent(["necessary", "preferences"])}`),
	).toBe(true);
	expect(consentAllowsPreferences("cc_cookie=%7Bbroken")).toBe(false);
	expect(
		consentAllowsPreferences(
			`cc_cookie=${encodeURIComponent('{"categories":"preferences"}')}`,
		),
	).toBe(false);
});
