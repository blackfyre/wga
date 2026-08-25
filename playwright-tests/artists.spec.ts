import { type Page, expect, test } from "@playwright/test";

const viewports = [
	{ width: 390, height: 900 },
	{ width: 834, height: 900 },
	{ width: 1440, height: 900 },
] as const;

async function assertNoHorizontalOverflow(page: Page) {
	const dimensions = await page.evaluate(() => ({
		clientWidth: document.documentElement.clientWidth,
		scrollWidth: document.documentElement.scrollWidth,
	}));
	expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
}

test("artist index renders its heading, kicker, and alphabet", async ({
	page,
}) => {
	await page.goto("/artists");

	await expect(page.locator("#artists h1")).toHaveText("Artists");
	await expect(page.getByText("01 — ARTIST INDEX")).toBeVisible();
	await expect(page.getByText("10 ARTISTS")).toBeVisible();

	const alphabet = page.locator("nav[aria-label='Filter artists by letter']");
	await expect(alphabet.getByRole("link", { name: "ALL" })).toBeVisible();
	// The synthetic fixture only publishes artists whose names begin with S.
	await expect(
		alphabet.getByRole("link", { name: "S", exact: true }),
	).toHaveAttribute("href", /letter=S/);
	await expect(alphabet.locator("span[aria-disabled='true']")).toHaveCount(25);
});

test("artist index combines name and school filters", async ({ page }) => {
	await page.emulateMedia({ reducedMotion: "reduce" });
	await page.goto("/artists");

	// Name query filter.
	await page
		.getByRole("searchbox", { name: "NAME CONTAINS" })
		.fill("Artist 04");
	await expect(page).toHaveURL(/q=Artist\+04/);
	await expect(page.locator("#artists")).toContainText("SYNTHETIC ARTIST 04");
	await expect(page.locator("#artists")).not.toContainText(
		"SYNTHETIC ARTIST 01",
	);

	// A school filter combines with the name query.
	await page
		.locator("form#artist-filters label", { hasText: "Bohemian" })
		.click();
	await expect(page).toHaveURL(/school=bohemian/);
	await expect(page).toHaveURL(/q=Artist\+04/);
	await expect(page.locator("#artists")).toContainText("SYNTHETIC ARTIST 04");
	await expect(page.locator("#artists")).not.toContainText(
		"SYNTHETIC ARTIST 05",
	);

	// Clicking a letter preserves the school filter and resets the page.
	await page.getByRole("link", { name: "S", exact: true }).click();
	await expect(page).toHaveURL(/letter=S/);
	await expect(page).toHaveURL(/school=bohemian/);
	await expect(page.locator("#artists")).toContainText("SYNTHETIC ARTIST 04");
});

test("artist index filters by explicit birth range", async ({ page }) => {
	await page.goto("/artists?born_from=1805&born_to=1810");

	await expect(page.locator("#artists")).toContainText("SYNTHETIC ARTIST 05");
	await expect(page.locator("#artists")).toContainText("SYNTHETIC ARTIST 10");
	await expect(page.locator("#artists")).not.toContainText(
		"SYNTHETIC ARTIST 04",
	);
	await expect(page.getByText(/6 ARTISTS/)).toBeVisible();
});

test("artist index toggles grid and list views and cycles sort", async ({
	page,
}) => {
	await page.goto("/artists");

	await expect(page.locator("#artists ul[data-kbd-list]")).toBeVisible();
	await expect(page.locator("#artists table")).toHaveCount(0);

	await page.getByRole("link", { name: "LIST", exact: true }).click();
	await expect(page).toHaveURL(/view=list/);
	await expect(page.locator("#artists table")).toBeVisible();
	await expect(page.locator("#artists table[data-kbd-list]")).toBeVisible();
	await expect(page.locator("#artists table caption")).toHaveText(
		"Artists in the collection",
	);
	for (const header of ["NAME", "DATES", "SCHOOL", "PERIOD", "FORM"]) {
		await expect(
			page.locator("#artists table thead th", { hasText: header }),
		).toBeVisible();
	}

	await page.getByRole("link", { name: /SORT: A–Z/ }).click();
	await expect(page).toHaveURL(/sort=za/);
	await page.getByRole("link", { name: /SORT: Z–A/ }).click();
	await expect(page).toHaveURL(/sort=birth/);
	await page.getByRole("link", { name: /SORT: BIRTH YEAR/ }).click();
	await expect(page).toHaveURL(/view=list/);
	await expect(page).not.toHaveURL(/sort=/);
});

test("artist index shows an honest empty state with reset", async ({
	page,
}) => {
	await page.goto("/artists?q=does-not-exist-anywhere");

	await expect(page.getByText("No artists match these filters.")).toBeVisible();
	await expect(
		page.getByRole("link", { name: /RESET FILTERS/ }),
	).toHaveAttribute("href", "/artists");
	await page.getByRole("link", { name: /RESET FILTERS/ }).click();
	await expect(page).toHaveURL("/artists");
	await expect(page.getByText("10 ARTISTS")).toBeVisible();
});

test("artist index highlights matched names as escaped text", async ({
	page,
}) => {
	await page.goto("/artists?q=Synthetic%20Artist%2002");

	await expect(page.locator("#artists mark")).toHaveText("SYNTHETIC ARTIST 02");
	const markHtml = await page.locator("#artists mark").first().innerHTML();
	expect(markHtml).not.toContain("<");
	expect(markHtml).not.toContain(">");
});

test("artist index responds by keyboard and resets after HTMX swaps", async ({
	page,
}) => {
	await page.emulateMedia({ reducedMotion: "reduce" });
	await page.goto("/artists");

	await page.waitForFunction(
		() => document.documentElement.dataset.keyboardNavigationReady === "true",
	);
	await page.keyboard.press("j");
	await expect(page.locator("#artists li[data-kbd-caret]")).toHaveCount(1);

	// A filter swap replaces #artists and clears the stale caret state.
	await page
		.locator("form#artist-filters label", { hasText: "Bohemian" })
		.click();
	await expect(page.locator("#artists")).toContainText("SYNTHETIC ARTIST 04");
	await expect(page.locator("#artists li[data-kbd-caret]")).toHaveCount(0);
});

test("artist index composes without overflow across viewports", async ({
	page,
}) => {
	for (const viewport of viewports) {
		await page.setViewportSize({
			width: viewport.width,
			height: viewport.height,
		});
		await page.goto("/artists");
		await expect(page.locator("#artists h1")).toBeVisible();
		const columns = await page
			.locator("#artists ul[data-kbd-list]")
			.evaluate(
				(element) =>
					getComputedStyle(element).gridTemplateColumns.split(" ").length,
			);
		expect(columns).toBe(viewport.width >= 768 ? 3 : 2);
		await assertNoHorizontalOverflow(page);
	}
});

test("artist index honours dark theme and reduced motion", async ({ page }) => {
	await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" });
	await page.goto("/artists");
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);
	await expect(page.locator("#artists h1")).toBeVisible();
});

test("artist index reflows at 200% text (400% reflow) without overflow", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/artists");
	await page.evaluate(() => {
		document.documentElement.style.fontSize = "2em";
	});
	await assertNoHorizontalOverflow(page);
	await expect(page.getByRole("main")).toBeVisible();
});

test.describe("artist index without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("keeps ordinary form and letter navigation", async ({ page }) => {
		await page.emulateMedia({ reducedMotion: "reduce" });
		await page.goto("/artists");

		await expect(page.locator("#artists h1")).toHaveText("Artists");
		await expect(page.locator("form#artist-filters")).toHaveAttribute(
			"action",
			"/artists",
		);

		await page
			.locator("form#artist-filters input[type='search']")
			.fill("Synthetic Artist 02");
		await page.getByRole("button", { name: "APPLY FILTERS" }).click();
		await expect(page).toHaveURL(/q=Synthetic\+Artist\+02/);
		await expect(page.locator("#artists")).toContainText("SYNTHETIC ARTIST 02");

		await page.getByRole("link", { name: "S", exact: true }).click();
		await expect(page).toHaveURL(/letter=S/);
	});
});
