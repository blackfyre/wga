import { type Locator, type Page, expect, test } from "@playwright/test";

const artistRecordPath = "/artists/synthetic-artist-01-ad32608c6e36b2e";

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

// The `main#mc-area` element runs a 280ms entry animation. With the `backwards`
// fill it releases its transform once complete; wait for that so fixed tooltip
// descendants position against the viewport before measuring.
async function settleEntryAnimation(page: Page) {
	await expect(page.locator("main#mc-area")).toHaveCSS("transform", "none");
}

async function focusedTooltipBounds(
	tooltip: Locator,
	page: Page,
	width: number,
	height: number,
) {
	const box = await tooltip.boundingBox();
	if (!box) {
		throw new Error("Expected focused glossary tooltip geometry");
	}
	expect(box.x).toBeGreaterThanOrEqual(0);
	expect(box.y).toBeGreaterThanOrEqual(0);
	expect(box.x + box.width).toBeLessThanOrEqual(width);
	expect(box.y + box.height).toBeLessThanOrEqual(height);
	await assertNoHorizontalOverflow(page);
}

test("artist record renders metadata, biography, works, music, and citation", async ({
	page,
}) => {
	await page.goto(artistRecordPath);

	await expect(page.locator("h1")).toHaveText("SYNTHETIC ARTIST 01");
	await expect(page.getByText("02 — ARTIST")).toBeVisible();
	await expect(page.getByText("PORTRAIT — SYNTHETIC ARTIST 01")).toBeVisible();

	const aside = page.locator("aside");
	await expect(aside.getByText("LIFE")).toBeVisible();
	await expect(
		aside.getByText("b. 1801 Test City, d. 1851 Test City"),
	).toBeVisible();
	await expect(aside.getByText("American")).toBeVisible();
	await expect(aside.getByText("Romanticism")).toBeVisible();
	await expect(aside.getByText("Synthetic test profession")).toBeVisible();

	await expect(page.getByText("BIOGRAPHY")).toBeVisible();
	await expect(page.getByText("WORKS IN ARCHIVE")).toBeVisible();
	await expect(page.getByText("4 RECORDS")).toBeVisible();

	await expect(page.getByText("CITE THIS RECORD — BIBTEX")).toBeVisible();
	await expect(page.getByText("PERIOD MUSIC")).toBeVisible();
});

test("artist record shows the period-music card as an ordinary named-window link", async ({
	page,
}) => {
	await page.goto(artistRecordPath);

	const card = page.locator("a[target='wga-period-music']");
	await expect(card).toHaveCount(1);
	await expect(card).toContainText("PERIOD MUSIC");
	await expect(card).toContainText("Ballade No. 1, Op. 23");
	await expect(card).toHaveAttribute("href", /^\/api\/files\/music_song\//);
	await expect(card).not.toHaveAttribute("autoplay", /.*/);
});

test("artist record glossary terms are keyboard-reachable with an accessible name", async ({
	page,
}) => {
	await page.goto(artistRecordPath);

	const term = page.locator("dfn.wga-term").first();
	await expect(term).toHaveAttribute("role", "note");
	await expect(term).toHaveAttribute("tabindex", "0");
	await expect(term).toHaveAttribute("aria-label", /quadratura: /);

	await term.focus();
	await expect(term).toBeFocused();
	await expect(page).toHaveURL(artistRecordPath);
});

test("trailing glossary term tooltip stays viewport-contained at 834px", async ({
	page,
}) => {
	await page.setViewportSize({ width: 834, height: 900 });
	await page.goto(artistRecordPath);
	await settleEntryAnimation(page);

	const trailing = page.locator("dfn.wga-term:last-child");
	await expect(trailing).toHaveCount(1);
	await trailing.focus();
	await expect(trailing).toBeFocused();

	const tooltip = trailing.locator(".wga-term__tooltip");
	await expect(tooltip).toHaveCSS("visibility", "visible");
	await focusedTooltipBounds(tooltip, page, 834, 900);
});

test("focused glossary term tooltip is viewport-contained at 390px", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 900 });
	await page.goto(artistRecordPath);
	await settleEntryAnimation(page);

	const term = page.locator("dfn.wga-term").first();
	await expect(term).toHaveCount(1);
	await term.focus();
	await expect(term).toBeFocused();

	const tooltip = term.locator(".wga-term__tooltip");
	await expect(tooltip).toHaveCSS("visibility", "visible");
	await focusedTooltipBounds(tooltip, page, 390, 900);
});

test("focused glossary term tooltip is viewport-contained at 834px", async ({
	page,
}) => {
	await page.setViewportSize({ width: 834, height: 900 });
	await page.goto(artistRecordPath);
	await settleEntryAnimation(page);

	const term = page.locator("dfn.wga-term").first();
	await term.focus();
	await expect(term).toBeFocused();

	const tooltip = term.locator(".wga-term__tooltip");
	await expect(tooltip).toHaveCSS("visibility", "visible");
	await focusedTooltipBounds(tooltip, page, 834, 900);
});

test("focused glossary term tooltip is viewport-contained at 390px under reduced motion", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 900 });
	await page.emulateMedia({ reducedMotion: "reduce" });
	await page.goto(artistRecordPath);
	await settleEntryAnimation(page);

	const term = page.locator("dfn.wga-term").first();
	await term.focus();
	await expect(term).toBeFocused();

	const tooltip = term.locator(".wga-term__tooltip");
	await expect(tooltip).toHaveCSS("visibility", "visible");
	await focusedTooltipBounds(tooltip, page, 390, 900);
});

test("artist record works and breadcrumb use ordinary links", async ({
	page,
}) => {
	await page.goto(artistRecordPath);

	const breadcrumb = page.locator("nav[aria-label='Breadcrumb'] a", {
		hasText: "ARTISTS",
	});
	await expect(breadcrumb).toHaveAttribute("href", "/artists");
	await expect(breadcrumb).not.toHaveAttribute("hx-get", /.*/);

	const workLink = page
		.locator("a", { hasText: "Synthetic Artwork 01-01" })
		.first();
	await expect(workLink).toHaveAttribute(
		"href",
		/^\/artists\/synthetic-artist-01-ad32608c6e36b2e\//,
	);
});

test("artist record is reachable from the index by HTMX with history", async ({
	page,
}) => {
	await page.goto("/artists");
	await expect(page.locator("#artists")).toContainText("SYNTHETIC ARTIST 01");

	await page
		.locator("#artists a", { hasText: "SYNTHETIC ARTIST 01" })
		.first()
		.click();

	await expect(page).toHaveURL(artistRecordPath);
	await expect(page.locator("#mc-area h1")).toHaveText("SYNTHETIC ARTIST 01");

	await page.goBack();
	await expect(page).toHaveURL("/artists");
});

test("artist record composes without overflow across viewports", async ({
	page,
}) => {
	for (const viewport of viewports) {
		await page.setViewportSize({
			width: viewport.width,
			height: viewport.height,
		});
		await page.goto(artistRecordPath);
		await expect(page.locator("h1")).toBeVisible();
		await assertNoHorizontalOverflow(page);
	}
});

test("artist record honours dark theme and reduced motion", async ({
	page,
}) => {
	await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" });
	await page.goto(artistRecordPath);
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);
	await expect(page.locator("h1")).toBeVisible();
});

test("artist record reflows at 200% text without overflow", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto(artistRecordPath);
	await page.evaluate(() => {
		document.documentElement.style.fontSize = "2em";
	});
	await assertNoHorizontalOverflow(page);
	await expect(page.getByRole("main")).toBeVisible();
});

test("artist record renders without JavaScript", async ({ page }) => {
	await page.goto(artistRecordPath);

	await expect(page.locator("h1")).toHaveText("SYNTHETIC ARTIST 01");
	await expect(page.locator("a[target='wga-period-music']")).toHaveCount(1);
	await expect(page.getByText("CITE THIS RECORD — BIBTEX")).toBeVisible();

	const workLink = page
		.locator("a", { hasText: "Synthetic Artwork 01-01" })
		.first();
	await expect(workLink).toHaveAttribute("href", /^\/artists\//);
});

test.describe("artist record without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("keeps ordinary record links and metadata", async ({ page }) => {
		await page.goto(artistRecordPath);

		await expect(page.locator("h1")).toHaveText("SYNTHETIC ARTIST 01");
		await expect(page.getByText("PERIOD MUSIC")).toBeVisible();
		await expect(page.locator("a[target='wga-period-music']")).toHaveCount(1);
		await expect(page.getByText("CITE THIS RECORD — BIBTEX")).toBeVisible();
		await expect(page.getByText("4 RECORDS")).toBeVisible();
	});
});

test("artist record returns not found for a missing artist", async ({
	page,
}) => {
	const response = await page.goto("/artists/no-such-artist-000000000000000");
	expect(response?.status()).toBe(404);
});
