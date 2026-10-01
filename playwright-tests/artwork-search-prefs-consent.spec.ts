import { type BrowserContext, expect, type Page, test } from "@playwright/test";

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
