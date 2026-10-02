import { type BrowserContext, expect, type Page, test } from "@playwright/test";
import { grantPreferenceConsent } from "./helpers/preference-consent";

test.setTimeout(60000);

const actionsToggle = (page: Page) =>
	page.locator("#artwork-search-results [data-wga-aw-actions]");
const viewToggle = (page: Page) =>
	page.locator("#artwork-search-results [data-artwork-view-toggle]");

async function storedPrefs(page: Page, context: BrowserContext) {
	const cookies = await context.cookies();
	return {
		cookie: cookies.find((cookie) => cookie.name === "wga_aw_prefs") ?? null,
		local: await page.evaluate(() =>
			window.localStorage.getItem("wga-aw-prefs"),
		),
	};
}

async function chooseListWithActions(page: Page) {
	await page.goto("/artworks");
	await viewToggle(page).click();
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("view") === "list",
	);
	await actionsToggle(page).click();
	await expect(actionsToggle(page)).toHaveText("ACTIONS ✓");
}

async function savePreferenceStorage(page: Page, allowed: boolean) {
	const preferences = page.locator("#cc-main .pm");
	await expect(preferences).toBeVisible();
	await expect(preferences).toContainText("Preference storage");
	const category = preferences.locator('input[value="preferences"]');
	if (allowed) {
		await category.check({ force: true });
	} else {
		await category.uncheck({ force: true });
	}
	await preferences.getByRole("button", { name: "SAVE PREFERENCES" }).click();
	await expect(preferences).toBeHidden();
}

test.beforeEach(async ({ context }) => {
	await context.clearCookies();
	// CookieConsent hides its dialog from automation; present as a visitor.
	await context.addInitScript(() => {
		Object.defineProperty(navigator, "webdriver", { get: () => false });
	});
});

test("without preference consent, choices apply to the page but are not remembered", async ({
	context,
	page,
}) => {
	await chooseListWithActions(page);
	await expect(page.locator(".wga-work-actions").first()).toBeVisible();
	expect(await storedPrefs(page, context)).toEqual({
		cookie: null,
		local: null,
	});

	// An enhanced navigation away and back is a new page: defaults again.
	const header = page.locator("header");
	await header.getByRole("link", { name: "ARTISTS", exact: true }).click();
	await expect(page).toHaveURL(/\/artists/);
	await header.getByRole("link", { name: "ARTWORKS", exact: true }).click();
	await expect(page).toHaveURL((url) => url.pathname === "/artworks");
	await expect(actionsToggle(page)).toHaveText("ACTIONS +");
	await expect(page.locator(".wga-work-actions").first()).toBeHidden();

	await page.goto("/artworks");
	await expect(viewToggle(page)).toHaveText(/^VIEW: GRID/);
	await expect(actionsToggle(page)).toHaveText("ACTIONS +");
	await expect(page.locator(".wga-work-actions").first()).toBeHidden();
	expect(await storedPrefs(page, context)).toEqual({
		cookie: null,
		local: null,
	});
});

test("consent granted in another tab applies to an already-open page", async ({
	context,
	page,
}) => {
	await page.goto("/artworks");
	const otherTab = await context.newPage();
	await otherTab.goto("/");
	await otherTab
		.locator("#cc-main .cm")
		.getByRole("button", { name: "PREFERENCES", exact: true })
		.click();
	await savePreferenceStorage(otherTab, true);
	await otherTab.close();

	await actionsToggle(page).click();
	await expect(actionsToggle(page)).toHaveText("ACTIONS ✓");
	const stored = await storedPrefs(page, context);
	expect(stored.cookie).not.toBeNull();
	expect(JSON.parse(stored.local ?? "null")).toMatchObject({ actions: true });
});

test("accepted preference consent remembers choices, and withdrawing it deletes them", async ({
	context,
	page,
}) => {
	await page.goto("/artworks");
	await page
		.locator("#cc-main .cm")
		.getByRole("button", { name: "PREFERENCES", exact: true })
		.click();
	await savePreferenceStorage(page, true);

	await chooseListWithActions(page);
	const remembered = await storedPrefs(page, context);
	expect(remembered.cookie).not.toBeNull();
	expect(JSON.parse(remembered.local ?? "null")).toMatchObject({
		view: "list",
		actions: true,
	});

	await page.goto("/artworks");
	await expect(viewToggle(page)).toHaveText(/^VIEW: LIST/);
	await expect(actionsToggle(page)).toHaveText("ACTIONS ✓");
	await expect(page.locator("html")).toHaveAttribute("data-aw-actions", "on");

	const otherTab = await context.newPage();
	await otherTab.goto("/artworks");
	await expect(actionsToggle(otherTab)).toHaveText("ACTIONS ✓");

	await page.getByRole("link", { name: "Cookie settings" }).click();
	await savePreferenceStorage(page, false);
	expect(await storedPrefs(page, context)).toEqual({
		cookie: null,
		local: null,
	});
	await expect(actionsToggle(page)).toHaveText("ACTIONS ✓");

	// A tab opened before the withdrawal must not store choices again.
	await actionsToggle(otherTab).click();
	await expect(actionsToggle(otherTab)).toHaveText("ACTIONS +");
	expect(await storedPrefs(otherTab, context)).toEqual({
		cookie: null,
		local: null,
	});
	await otherTab.close();

	await page.goto("/artworks");
	await expect(viewToggle(page)).toHaveText(/^VIEW: GRID/);
	await expect(actionsToggle(page)).toHaveText("ACTIONS +");
});

test("granting consent remembers the choices the page shows", async ({
	context,
	page,
}) => {
	await page.goto("/artworks?sort=date&view=list");
	await expect(viewToggle(page)).toHaveText(/^VIEW: LIST/);
	await page
		.locator("#cc-main .cm")
		.getByRole("button", { name: "PREFERENCES", exact: true })
		.click();
	await savePreferenceStorage(page, true);

	const stored = await storedPrefs(page, context);
	expect(stored.cookie).not.toBeNull();
	expect(JSON.parse(stored.local ?? "null")).toEqual({
		sort: "date",
		dir: "asc",
		view: "list",
		actions: false,
	});

	await page.goto("/artworks");
	await expect(viewToggle(page)).toHaveText(/^VIEW: LIST/);
	await expect(
		page.locator(
			'#artwork-search-results a[data-wga-aw-pref][aria-current="true"]',
		),
	).toContainText("DATE");
});

test("the actions toggle inverts what the page shows after another tab changed it", async ({
	context,
	page,
}) => {
	await grantPreferenceConsent(page);
	await page.goto("/artworks");
	await expect(actionsToggle(page)).toHaveText("ACTIONS +");

	const otherTab = await context.newPage();
	await otherTab.goto("/artworks");
	await actionsToggle(otherTab).click();
	await expect(actionsToggle(otherTab)).toHaveText("ACTIONS ✓");
	await otherTab.close();

	// This page has not swapped, so it still shows the actions hidden.
	await expect(actionsToggle(page)).toHaveText("ACTIONS +");
	await actionsToggle(page).click();
	await expect(actionsToggle(page)).toHaveText("ACTIONS ✓");
	await expect(page.locator("html")).toHaveAttribute("data-aw-actions", "on");
	await expect(page.locator(".wga-work-actions").first()).toBeVisible();
	const stored = await storedPrefs(page, context);
	expect(JSON.parse(stored.local ?? "null")).toMatchObject({ actions: true });
});
