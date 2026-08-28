import { type Page, expect, test } from "@playwright/test";

const viewports = [390, 834, 1440];
const basisLabels = [
	"BY ARTIST",
	"SAME COLLECTION",
	"SAME PERIOD",
	"SIMILAR PALETTE",
];
let artworkPath = "";

async function discoverArtworkPath(page: Page) {
	const title = "Kolowrat Wedding";
	await page.goto(`/artworks?q=${encodeURIComponent(title)}`);
	const candidates = await page
		.locator("#artwork-search-results [data-kbd-href]")
		.filter({ hasText: title });
	await expect(candidates).toHaveCount(1);
	const candidate = await candidates.getAttribute("href");
	expect(candidate).toContain("rea135c19d9c553");
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

		test("renders the record and four keyboard-reachable basis links", async ({
			page,
		}) => {
			await page.goto(artworkPath);
			await expect(page.locator("h1")).toBeVisible();
			const basis = page.getByRole("navigation", {
				name: "Related works basis",
			});
			await expect(basis).toBeVisible();
			await expect(basis.getByRole("link")).toHaveCount(4);
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

test("basis navigation works as ordinary links without JavaScript", async ({
	browser,
}) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();
	await page.goto(artworkPath);
	const collection = page.getByRole("link", { name: "SAME COLLECTION" });
	await expect(collection).toHaveAttribute(
		"href",
		`${artworkPath}?basis=collection`,
	);
	const href = await collection.getAttribute("href");
	await page.goto(href as string);
	await expect(page).toHaveURL(`${artworkPath}?basis=collection`);
	await expect(
		page.getByRole("link", { name: "SAME COLLECTION" }),
	).toHaveAttribute("aria-current", "page");
	await context.close();
});
