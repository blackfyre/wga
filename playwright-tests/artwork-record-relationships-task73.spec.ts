import { expect, type Page, test } from "@playwright/test";
import { relationshipArtwork } from "./helpers/synthetic-fixture";

const viewports = [390, 834, 1440];
const basisLabels = ["BY ARTIST", "SAME COLLECTION", "SAME PERIOD"];
let artworkPath = "";

async function discoverArtworkPath(page: Page) {
	const title = relationshipArtwork.title;
	await page.goto(`/artworks?q=${encodeURIComponent(title)}`);
	const candidates = await page
		.locator("#artwork-search-results [data-kbd-href]")
		.filter({ hasText: title });
	await expect(candidates).toHaveCount(1);
	const candidate = await candidates.getAttribute("href");
	expect(candidate).toBe(relationshipArtwork.path);
	if (!candidate) throw new Error("Exact artwork search result has no href");

	const response = await page.goto(candidate);
	if (response?.status() !== 200) {
		throw new Error("Exact artwork search result did not return HTTP 200");
	}
	const basis = page.getByRole("navigation", { name: "Related works basis" });
	await expect(basis.getByRole("link")).toHaveCount(basisLabels.length);
	for (const label of basisLabels) {
		await expect(basis.getByRole("link", { name: label })).toHaveCount(1);
	}
	await expect(
		page.locator("#mc-area").getByText("PALETTE", { exact: true }),
	).toBeVisible();
	await expect(page.locator("[data-wga-palette-bar]")).toBeVisible();
	return candidate;
}

test.beforeAll(async ({ browser }) => {
	const page = await browser.newPage();
	artworkPath = await discoverArtworkPath(page);
	await page.close();
});

for (const width of viewports) {
	test.describe(`artwork record relationships at ${width}px`, () => {
		test.use({ viewport: { width, height: 900 } });
		test.setTimeout(60000);

		test("renders the record and three keyboard-reachable basis links", async ({
			page,
		}) => {
			await page.goto(artworkPath);
			await expect(page.locator("h1")).toBeVisible();
			const basis = page.getByRole("navigation", {
				name: "Related works basis",
			});
			await expect(basis).toBeVisible();
			await expect(basis.getByRole("link")).toHaveCount(3);
			for (const label of basisLabels) {
				const link = basis.getByRole("link", { name: label });
				await expect(link).toBeVisible();
				await link.focus();
				await expect(link).toBeFocused();
			}
			expect(
				await page.evaluate(
					() => document.documentElement.scrollWidth > window.innerWidth,
				),
			).toBe(false);
		});

		test("basis links preserve the canonical record and reload state", async ({
			page,
		}) => {
			await page.goto(artworkPath);
			const period = page.getByRole("link", { name: "SAME PERIOD" });
			await expect(period).toHaveAttribute(
				"href",
				`${artworkPath}?basis=period`,
			);
			await period.click();
			await expect(page).toHaveURL(`${artworkPath}?basis=period`);
			await expect(
				page.getByRole("link", { name: "SAME PERIOD" }),
			).toHaveAttribute("aria-current", "page");
			await page.reload();
			await expect(page).toHaveURL(`${artworkPath}?basis=period`);
			await expect(page.locator("#related-works-heading")).toContainText(
				/ARTISTS WORKING|SAME PERIOD/,
			);
		});
	});
}

test("each basis lists its pinned related record", async ({ page }) => {
	const expected = [
		["", relationshipArtwork.sameArtistPath],
		["?basis=collection", relationshipArtwork.sameCollectionPath],
		["?basis=period", relationshipArtwork.samePeriodPath],
	] as const;
	for (const [query, relatedPath] of expected) {
		await page.goto(`${artworkPath}${query}`);
		await expect(
			page.locator(`#mc-area a.wga-record-card[href="${relatedPath}"]`),
		).toHaveCount(1);
	}
});

test("basis navigation works as ordinary links without JavaScript", async ({
	browser,
}) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();
	await page.goto(artworkPath);
	const collection = page
		.getByRole("navigation", { name: "Related works basis" })
		.getByRole("link", { name: "SAME COLLECTION" });
	await expect(collection).toHaveAttribute(
		"href",
		`${artworkPath}?basis=collection`,
	);
	const href = await collection.getAttribute("href");
	await page.goto(href as string);
	await expect(page).toHaveURL(`${artworkPath}?basis=collection`);
	await expect(
		page
			.getByRole("navigation", { name: "Related works basis" })
			.getByRole("link", { name: "SAME COLLECTION" }),
	).toHaveAttribute("aria-current", "page");
	await context.close();
});

test("retired palette-basis URLs fall back to the default related works", async ({
	page,
}) => {
	await page.goto(`${artworkPath}?basis=palette`);
	await expect(page).toHaveURL(artworkPath);
	await expect(page.getByRole("link", { name: "BY ARTIST" })).toHaveAttribute(
		"aria-current",
		"page",
	);
});
