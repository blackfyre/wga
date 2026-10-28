import { expect, type Page, test } from "@playwright/test";
import {
	denyPreferenceConsent,
	grantPreferenceConsent,
	storedPreferences,
} from "./helpers/preference-consent";

test.setTimeout(60000);

// The grid/list choice is one remembered listing view: choosing it on one
// screen carries to every screen that offers the toggle, but only with the
// optional "preferences" consent.

async function chooseListOnArtworkSearch(page: Page) {
	await page.goto("/artworks");
	const toggle = page.locator(
		"#artwork-search-results [data-artwork-view-toggle]",
	);
	await expect(toggle).toHaveText("VIEW: GRID");
	await toggle.click();
	await expect(page).toHaveURL(/view=list/);
	await expect(toggle).toHaveText("VIEW: LIST");
}

const artistToggle = (page: Page, view: "grid" | "list") =>
	page.locator(`#artists a[data-wga-view-pref="${view}"]`);

const dualToggles = (page: Page, view: "grid" | "list") =>
	page.locator(`a[data-wga-view-pref="${view}"]`);

test("a list choice on artwork search carries to the artist index and Dual Mode", async ({
	page,
}) => {
	await grantPreferenceConsent(page);
	await chooseListOnArtworkSearch(page);
	expect(
		JSON.parse((await storedPreferences(page))["wga-aw-prefs"] ?? "null"),
	).toMatchObject({ view: "list" });

	await page.goto("/artists");
	await expect(page.locator("#artists table")).toBeVisible();
	await expect(artistToggle(page, "list")).toHaveAttribute(
		"aria-current",
		"page",
	);

	await page.goto("/dual-mode?wide=1");
	await expect(dualToggles(page, "list")).toHaveCount(2);
	for (const toggle of await dualToggles(page, "list").all()) {
		await expect(toggle).toHaveAttribute("aria-current", "true");
	}

	// Choosing the grid on the artist index is remembered for the next listing.
	await page.goto("/artists");
	await artistToggle(page, "grid").click();
	await expect(page.locator("#artists ul[data-kbd-list]")).toBeVisible();
	expect(
		JSON.parse((await storedPreferences(page))["wga-aw-prefs"] ?? "null"),
	).toMatchObject({ view: "grid" });
	await page.goto("/artworks");
	await expect(
		page.locator("#artwork-search-results [data-artwork-view-toggle]"),
	).toHaveText("VIEW: GRID");
});

test("without preference consent other listings stay on the grid", async ({
	page,
}) => {
	await denyPreferenceConsent(page);
	await chooseListOnArtworkSearch(page);
	expect((await storedPreferences(page))["wga-aw-prefs"]).toBeNull();

	await page.goto("/artists");
	await expect(page.locator("#artists ul[data-kbd-list]")).toBeVisible();
	await expect(artistToggle(page, "grid")).toHaveAttribute(
		"aria-current",
		"page",
	);

	await page.goto("/dual-mode?wide=1");
	await expect(dualToggles(page, "grid")).toHaveCount(2);
	for (const toggle of await dualToggles(page, "grid").all()) {
		await expect(toggle).toHaveAttribute("aria-current", "true");
	}
	const cookies = await page.context().cookies();
	expect(cookies.some((cookie) => cookie.name === "wga_aw_prefs")).toBe(false);
});
