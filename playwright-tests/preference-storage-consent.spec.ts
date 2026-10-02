import { expect, type Page, test } from "@playwright/test";
import {
	expectNoPageErrors,
	guardPageErrors,
	resetErrorCapture,
} from "./helpers/page-errors";
import { storedPreferences } from "./helpers/preference-consent";

// The colour scheme, palette, bionic reading and Study Board are remembered
// only with the optional "preferences" consent. The search toolbar has its own
// specification in ./artwork-search-prefs-consent.spec.ts.

test.setTimeout(60000);

const NOTHING_STORED = {
	"wga-aw-prefs": null,
	"wga-theme": null,
	"wga-palette": null,
	"wga-bionic": null,
	"wga-study-board": null,
};

test.beforeEach(async ({ context, page }) => {
	resetErrorCapture();
	guardPageErrors(page);
	await context.clearCookies();
	await page.emulateMedia({ colorScheme: "light" });
	// CookieConsent hides its dialog from automation; present as a visitor.
	await context.addInitScript(() => {
		Object.defineProperty(navigator, "webdriver", { get: () => false });
	});
});

test.afterEach(() => {
	expectNoPageErrors();
});

async function publishedArtworkID(page: Page): Promise<string> {
	await page.goto("/artworks");
	const link = page
		.locator("#artwork-search-results [data-view='grid'] a.wga-record-card")
		.first();
	await expect(link).toBeVisible({ timeout: 30000 });
	const path = await link.evaluate(
		(element) => (element as HTMLAnchorElement).pathname,
	);
	const id = path.match(/([A-Za-z0-9_-]{15})$/)?.[1];
	expect(id).toBeTruthy();
	return id ?? "";
}

// chooseAppearance picks DARK, the verdigris palette and bionic reading in
// the preferences panel, then closes it.
async function chooseAppearance(page: Page) {
	await page.locator("[data-wga-preferences-open]").click();
	const panel = page.locator("[data-wga-preferences-panel]");
	await expect(panel).toBeVisible();
	await panel.locator('[data-wga-scheme="dark"]').click();
	await panel.locator('[data-wga-palette="verdigris"]').click();
	await panel.locator("[data-wga-bionic-toggle]").click();
	await panel.locator("[data-wga-preferences-close]").first().click();
	await expect(panel).toBeHidden();
	await expectChosenAppearance(page);
}

async function expectChosenAppearance(page: Page) {
	const html = page.locator("html");
	await expect(html).toHaveAttribute("data-theme", "dark");
	await expect(html).toHaveAttribute("data-palette", "verdigris");
	await expect(html).toHaveAttribute("data-bionic-reading", "true");
}

async function expectDefaultAppearance(page: Page) {
	const html = page.locator("html");
	await expect(html).toHaveAttribute("data-theme", "light");
	await expect(html).toHaveAttribute("data-palette", "bone");
	await expect(html).toHaveAttribute("data-bionic-reading", "false");
}

async function savePreferenceStorage(page: Page, allowed: boolean) {
	const preferences = page.locator("#cc-main .pm");
	await expect(preferences).toBeVisible();
	const category = preferences.locator('input[value="preferences"]');
	if (allowed) {
		await category.check({ force: true });
	} else {
		await category.uncheck({ force: true });
	}
	await preferences.getByRole("button", { name: "SAVE PREFERENCES" }).click();
	await expect(preferences).toBeHidden();
}

const notice = (page: Page) => page.locator("#cc-main .cm");

test("without consent, choices apply to the page, nothing is stored, and stale copies are deleted", async ({
	page,
}) => {
	const id = await publishedArtworkID(page);
	await page.evaluate((board) => {
		window.localStorage.setItem("wga-theme", "dark");
		window.localStorage.setItem("wga-palette", "gothic");
		window.localStorage.setItem("wga-bionic", "on");
		window.localStorage.setItem("wga-study-board", board);
	}, id);

	// The head resolver ignores the stale copies, and the stores delete them.
	await page.goto("/");
	await expectDefaultAppearance(page);
	expect(await storedPreferences(page)).toEqual(NOTHING_STORED);

	await page.goto(`/study-board?board=${id}`);
	await expect(page.locator(`[data-study-board-work='${id}']`)).toHaveCount(1);
	await chooseAppearance(page);
	expect(await storedPreferences(page)).toEqual(NOTHING_STORED);

	// A full load starts from the defaults and no board.
	await page.goto("/");
	await expectDefaultAppearance(page);
	await page.goto("/study-board");
	await expect(page.locator("[data-study-board-empty]")).toBeVisible();
	await expect(page.locator("#study-board-shelf")).toBeEmpty();
	expect(await storedPreferences(page)).toEqual(NOTHING_STORED);
});

test("ACCEPT ALL stores the state the page shows, and withdrawing deletes every copy", async ({
	page,
}) => {
	const id = await publishedArtworkID(page);
	await page.goto(`/study-board?board=${id}`);
	await chooseAppearance(page);
	expect(await storedPreferences(page)).toEqual(NOTHING_STORED);

	await notice(page).getByRole("button", { name: "ACCEPT ALL" }).click();
	await expect(notice(page)).toBeHidden();
	expect(await storedPreferences(page)).toMatchObject({
		"wga-theme": "dark",
		"wga-palette": "verdigris",
		"wga-bionic": "on",
		"wga-study-board": id,
	});

	await page.reload();
	await expectChosenAppearance(page);
	await page.goto("/study-board");
	await expect(page).toHaveURL((url) => url.searchParams.get("board") === id);

	await page.getByRole("link", { name: "Cookie settings" }).click();
	await savePreferenceStorage(page, false);
	expect(await storedPreferences(page)).toEqual(NOTHING_STORED);
	// The open page keeps its state.
	await expectChosenAppearance(page);
	await expect(page.locator(`[data-study-board-work='${id}']`)).toHaveCount(1);

	await page.goto("/");
	await expectDefaultAppearance(page);
	await page.goto("/study-board");
	await expect(page.locator("[data-study-board-empty]")).toBeVisible();
});

test("a grant or withdrawal in another tab governs an already-open page", async ({
	context,
	page,
}) => {
	await page.goto("/");
	const otherTab = await context.newPage();
	await otherTab.goto("/");
	await notice(otherTab)
		.getByRole("button", { name: "PREFERENCES", exact: true })
		.click();
	await savePreferenceStorage(otherTab, true);

	// This page loaded before the grant; its next choices are stored.
	await chooseAppearance(page);
	expect(await storedPreferences(page)).toMatchObject({
		"wga-theme": "dark",
		"wga-palette": "verdigris",
		"wga-bionic": "on",
	});

	await otherTab.getByRole("link", { name: "Cookie settings" }).click();
	await savePreferenceStorage(otherTab, false);
	await otherTab.close();
	expect(await storedPreferences(page)).toEqual(NOTHING_STORED);

	// After the withdrawal this page keeps its appearance but stores nothing.
	await expectChosenAppearance(page);
	await page.locator("[data-wga-preferences-open]").click();
	await page.locator('[data-wga-palette="gothic"]').click();
	await page.locator("[data-wga-bionic-toggle]").click();
	await expect(page.locator("html")).toHaveAttribute("data-palette", "gothic");
	expect(await storedPreferences(page)).toEqual(NOTHING_STORED);
});

test("a shared board URL does not replace the remembered board", async ({
	page,
}) => {
	await page.goto("/artworks");
	const links = page.locator(
		"#artwork-search-results [data-view='grid'] a.wga-record-card",
	);
	await expect(links.nth(1)).toBeVisible({ timeout: 30000 });
	const ids = await links.evaluateAll((elements) =>
		elements
			.map(
				(element) =>
					(element as HTMLAnchorElement).pathname.match(
						/([A-Za-z0-9_-]{15})$/,
					)?.[1] ?? "",
			)
			.filter(Boolean),
	);
	const [remembered, shared] = [...new Set(ids)];

	await page.goto(`/study-board?board=${remembered}`);
	await notice(page).getByRole("button", { name: "ACCEPT ALL" }).click();
	expect(await storedPreferences(page)).toMatchObject({
		"wga-study-board": remembered,
	});

	await page.goto(`/study-board?board=${shared}`);
	await expect(page.locator(`[data-study-board-work='${shared}']`)).toHaveCount(
		1,
	);
	expect(await storedPreferences(page)).toMatchObject({
		"wga-study-board": remembered,
	});

	await page.goto("/study-board");
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("board") === remembered,
	);
});
