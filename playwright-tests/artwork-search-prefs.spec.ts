import { expect, type Page, test } from "@playwright/test";
import { grantPreferenceConsent } from "./helpers/artwork-search-prefs";

test.setTimeout(60000);

// These specifications cover remembering, which needs preference consent; the
// consent gate itself is covered by artwork-search-prefs-consent.spec.ts.
test.beforeEach(async ({ page }) => {
	await grantPreferenceConsent(page);
});

const results = (page: Page) => page.locator("#artwork-search-results");
const actionsToggle = (page: Page) =>
	results(page).locator("[data-wga-aw-actions]");
const viewToggle = (page: Page) =>
	results(page).locator("[data-artwork-view-toggle]");
const firstGridActions = (page: Page) =>
	results(page)
		.locator("[data-view='grid'] article")
		.first()
		.locator(".wga-work-actions");
const firstListActions = (page: Page) =>
	results(page)
		.locator("[data-view='list'] li")
		.first()
		.locator(".wga-work-actions");

async function storedPrefs(page: Page) {
	return page.evaluate(() =>
		JSON.parse(window.localStorage.getItem("wga-aw-prefs") ?? "null"),
	);
}

test("result actions are hidden until the toolbar toggle shows them", async ({
	page,
}) => {
	await page.goto("/artworks");
	await expect(firstGridActions(page)).toBeHidden();
	await expect(actionsToggle(page)).toBeVisible();
	await expect(actionsToggle(page)).toHaveText("ACTIONS +");
	await expect(actionsToggle(page)).toHaveAttribute("aria-pressed", "false");

	await actionsToggle(page).click();
	await expect(actionsToggle(page)).toHaveText("ACTIONS ✓");
	await expect(actionsToggle(page)).toHaveAttribute("aria-pressed", "true");
	await expect(page.locator("html")).toHaveAttribute("data-aw-actions", "on");
	await expect(firstGridActions(page)).toBeVisible();
	await expect(
		firstGridActions(page).getByRole("button", { name: /ADD TO ITINERARY/ }),
	).toBeVisible();
	await expect(
		firstGridActions(page).getByRole("button", { name: /ADD TO STUDY BOARD/ }),
	).toBeVisible();
	expect(new URL(page.url()).search).toBe("");

	await actionsToggle(page).click();
	await expect(actionsToggle(page)).toHaveText("ACTIONS +");
	await expect(actionsToggle(page)).toHaveAttribute("aria-pressed", "false");
	await expect(firstGridActions(page)).toBeHidden();
});

test("one view control names the current view and switches to the other", async ({
	page,
}) => {
	await page.goto("/artworks");
	await expect(viewToggle(page)).toHaveCount(1);
	await expect(viewToggle(page)).toHaveAccessibleName(
		"VIEW: GRID, switch to list view",
	);
	await actionsToggle(page).click();

	await viewToggle(page).click();
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("view") === "list",
	);
	await expect(viewToggle(page)).toHaveText(/^VIEW: LIST/);
	await expect(results(page).locator("[data-view='list']")).toBeVisible();
	await expect(firstListActions(page)).toBeVisible();
	await expect(actionsToggle(page)).toHaveText("ACTIONS ✓");

	await viewToggle(page).click();
	await expect(page).toHaveURL((url) => !url.searchParams.has("view"));
	await expect(viewToggle(page)).toHaveText(/^VIEW: GRID/);
	await expect(firstGridActions(page)).toBeVisible();
});

test("sort, view and actions are remembered across page loads", async ({
	page,
}) => {
	await page.goto("/artworks");
	await results(page).getByRole("link", { name: "DATE", exact: true }).click();
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("sort") === "date",
	);
	await results(page)
		.getByRole("link", { name: "DATE EARLIEST", exact: true })
		.click();
	await expect(page).toHaveURL((url) => url.searchParams.get("dir") === "desc");
	await viewToggle(page).click();
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("view") === "list",
	);
	await actionsToggle(page).click();
	expect(await storedPrefs(page)).toEqual({
		sort: "date",
		dir: "desc",
		view: "list",
		actions: true,
	});

	const response = await page.goto("/artworks");
	const html = (await response?.text()) ?? "";
	expect(html).toContain('<html lang="en" data-aw-actions="on">');
	expect(new URL(page.url()).search).toBe("");
	await expect(
		results(page).getByRole("link", { name: "DATE LATEST", exact: true }),
	).toHaveAttribute("aria-current", "true");
	await expect(viewToggle(page)).toHaveText(/^VIEW: LIST/);
	await expect(actionsToggle(page)).toHaveText("ACTIONS ✓");
	await expect(firstListActions(page)).toBeVisible();

	await page.reload();
	await expect(viewToggle(page)).toHaveText(/^VIEW: LIST/);
	await expect(firstListActions(page)).toBeVisible();
});

test("explicit search URLs win over remembered choices", async ({ page }) => {
	await page.goto("/artworks?view=list&sort=date");
	await viewToggle(page).click();
	await expect(page).toHaveURL((url) => !url.searchParams.has("view"));
	await page.goto("/artworks?view=list");
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("view") === "list",
	);
	expect((await storedPrefs(page))?.view).toBe("grid");

	await page.goto("/artworks?sort=artist");
	await expect(
		results(page).getByRole("link", { name: "ARTIST A–Z", exact: true }),
	).toHaveAttribute("aria-current", "true");
	await expect(viewToggle(page)).toHaveText(/^VIEW: GRID/);
});

test("resetting filters keeps the remembered presentation", async ({
	page,
}) => {
	await page.goto("/artworks?sort=date&view=list");
	await actionsToggle(page).click();
	await results(page).getByRole("link", { name: "DATE EARLIEST" }).click();
	await expect(page).toHaveURL((url) => url.searchParams.get("dir") === "desc");

	await page.locator("#artwork-filters input[name='q']").fill("madonna");
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("q") === "madonna",
	);
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("view") === "list",
	);

	await page
		.locator("#artwork-filters")
		.getByRole("link", { name: "RESET" })
		.click();
	await expect(page).toHaveURL((url) => !url.searchParams.has("q"));
	await expect(page.locator("#artwork-filters input[name='q']")).toHaveValue(
		"",
	);
	await expect(
		results(page).getByRole("link", { name: "DATE LATEST", exact: true }),
	).toHaveAttribute("aria-current", "true");
	await expect(viewToggle(page)).toHaveText(/^VIEW: LIST/);
	await expect(firstListActions(page)).toBeVisible();
	expect(await storedPrefs(page)).toEqual({
		sort: "date",
		dir: "desc",
		view: "list",
		actions: true,
	});
});

test.describe("without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("sort and view stay ordinary links and the actions toggle is not offered", async ({
		page,
	}) => {
		await page.goto("/artworks");
		await expect(actionsToggle(page)).toBeHidden();
		await expect(firstGridActions(page)).toBeHidden();

		await results(page)
			.getByRole("link", { name: "DATE", exact: true })
			.click();
		await expect(page).toHaveURL(
			(url) => url.searchParams.get("sort") === "date",
		);
		await expect(
			results(page).getByRole("link", { name: "DATE EARLIEST", exact: true }),
		).toHaveAttribute("aria-current", "true");

		await viewToggle(page).click();
		await expect(page).toHaveURL(
			(url) =>
				url.searchParams.get("view") === "list" &&
				url.searchParams.get("sort") === "date",
		);
		await expect(results(page).locator("[data-view='list']")).toBeVisible();
	});

	test("a remembered cookie does not offer the toggle without its script", async ({
		page,
		context,
		baseURL,
	}) => {
		await context.addCookies([
			{
				name: "wga_aw_prefs",
				value: encodeURIComponent(
					JSON.stringify({
						sort: "title",
						dir: "asc",
						view: "grid",
						actions: false,
					}),
				),
				url: baseURL,
			},
		]);
		await page.goto("/artworks");
		await expect(page.locator("html")).toHaveAttribute(
			"data-aw-actions",
			"off",
		);
		await expect(actionsToggle(page)).toBeHidden();
	});
});
