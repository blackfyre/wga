import { type Page, expect, test } from "@playwright/test";

const waitForKeyboard = (page: Page) =>
	page.waitForFunction(
		() => document.documentElement.dataset.keyboardNavigationReady === "true",
	);

test("opens keyboard help and command palette", async ({ page }) => {
	await page.goto("/");
	await waitForKeyboard(page);

	await page.getByRole("button", { name: "Keyboard shortcuts" }).click();
	await expect(page.locator("#keyboard-help")).toBeVisible();

	await page.keyboard.press("Escape");
	await page.keyboard.press("Control+k");
	await expect(page.getByRole("dialog", { name: "Go to" })).toBeVisible();
});

test("palette filters sections and bounds record suggestions", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("Control+k");

	const input = page.getByRole("searchbox", {
		name: "Search sections, artists and works",
	});
	await input.fill("artist index");
	await expect(
		page.getByRole("link", { name: /artist index/i }).first(),
	).toBeVisible();

	const response = page.waitForResponse((candidate) => {
		const url = new URL(candidate.url());
		return (
			url.pathname === "/keyboard/suggestions" &&
			url.searchParams.get("q") === "art"
		);
	});
	await input.fill("art");
	const suggestionURL = new URL((await response).url());
	expect(suggestionURL.searchParams.get("limit")).toBe("7");
	await page.keyboard.press("Enter");
	await expect(page).toHaveURL(/\/artists/);
});

test("section shortcuts navigate by letter and catalogue number", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("w");
	await expect(page).toHaveURL(/\/artworks/);

	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("1");
	await page.keyboard.press("2");
	await expect(page).toHaveURL(/\/artworks/);
});

test("search and palette shortcuts respect editing contexts", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);
	const search = page.locator("[data-kbd-search]").first();

	await search.fill("Vermeer");
	await page.getByRole("heading", { level: 1 }).click();
	await page.keyboard.press("/");
	await expect(search).toBeFocused();
	await expect(search).toHaveJSProperty("selectionStart", 0);
	await page.keyboard.press("Escape");
	await expect(search).not.toBeFocused();

	await search.focus();
	await search.press("Control+k");
	await expect(page.getByRole("dialog", { name: "Go to" })).toBeVisible();
});

test("mobile search shortcut opens the navigation disclosure", async ({
	page,
}) => {
	await page.setViewportSize({ width: 600, height: 900 });
	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("/");
	await expect(page.locator("[data-kbd-mobile-navigation]")).toHaveAttribute(
		"open",
		"",
	);
	await expect(
		page.locator("[data-kbd-mobile-navigation] [data-kbd-search]"),
	).toBeFocused();
	await page.getByRole("heading", { level: 1 }).click();
	await expect(
		page.locator("[data-kbd-mobile-navigation]"),
	).not.toHaveAttribute("open");

	await page.keyboard.press("/");
	await page.keyboard.press("Escape");
	await expect(
		page.locator("[data-kbd-mobile-navigation]"),
	).not.toHaveAttribute("open");
});

test("palette selection wraps between its first and last result", async ({ page }) => {
	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("Control+k");
	await page.keyboard.press("ArrowUp");
	await page.keyboard.press("Enter");
	await expect(page).toHaveURL(/\/glossary$/);
});

test("artwork traversal follows grid rows and resets after replacement", async ({
	page,
}) => {
	await page.goto("/artworks");
	await waitForKeyboard(page);
	await expect(
		page.locator("[data-kbd-list] [data-kbd-idx]").first(),
	).toBeVisible();

	await page.keyboard.press("ArrowDown");
	await page.keyboard.press("ArrowDown");
	await expect(page.locator("[data-kbd-caret]")).toHaveAttribute(
		"data-kbd-idx",
		"3",
	);
	await page.keyboard.press("ArrowRight");
	await expect(page.locator("[data-kbd-caret]")).toHaveAttribute(
		"data-kbd-idx",
		"4",
	);
	for (let index = 0; index < 5; index += 1) {
		await page.keyboard.press("ArrowUp");
	}
	await expect(page.locator("[data-kbd-caret]")).toHaveAttribute(
		"data-kbd-idx",
		"0",
	);

	await page
		.locator("[data-keyboard-page-search]")
		.fill("Synthetic Artwork 01-01");
	await expect(page.locator("[data-kbd-caret]")).toHaveCount(0);
});
