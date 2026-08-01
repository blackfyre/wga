import { expect, test } from "@playwright/test";

test("opens keyboard help and command palette", async ({ page }) => {
	await page.goto("/");

	await page.getByRole("button", { name: "Keyboard shortcuts" }).click();
	await expect(page.locator("#keyboard-help")).toBeVisible();

	await page.keyboard.press("Escape");
	await page.keyboard.press("Control+k");
	await expect(page.getByRole("dialog", { name: "Go to" })).toBeVisible();
});

test("keyboard palette renders suggestions", async ({ page }) => {
	await page.goto("/");
	await page.getByRole("button", { name: "⌘K" }).click();

	const response = page.waitForResponse(
		(candidate) =>
			candidate.url().includes("/keyboard/suggestions") &&
			candidate.status() === 200,
	);
	await page
		.getByRole("searchbox", { name: "Search artists and artworks" })
		.fill("synthetic");
	await response;
	await expect(
		page.locator("[data-keyboard-suggestions] a").first(),
	).toBeVisible();
});

test("keyboard shortcuts do not interrupt editable controls", async ({
	page,
}) => {
	await page.goto("/artworks");
	const search = page.locator("[data-keyboard-page-search]");

	await search.fill("a");
	await search.press("a");
	await expect(search).toHaveValue("aa");
	await expect(page).toHaveURL(/\/artworks/);
});

test("keyboard traversal resets after artwork search replacement", async ({
	page,
}) => {
	await page.goto("/artworks");
	await expect(
		page.locator("[data-keyboard-list] [data-keyboard-item]").first(),
	).toBeVisible();

	await page.keyboard.press("ArrowDown");
	await expect(page.locator("[data-keyboard-current]")).toHaveCount(1);

	await page
		.locator("[data-keyboard-page-search]")
		.fill("Synthetic Artwork 01-01");
	await expect(page.locator("[data-keyboard-current]")).toHaveCount(0);
});
