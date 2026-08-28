import { expect, test } from "@playwright/test";

const artistPath = "/artists/gozzoli-benozzo-r9fb82d431d2a5c";
const artworkPath =
	"/artists/gozzoli-benozzo-r9fb82d431d2a5c/the-mocking-of-christ-detail-r8c3a31f30aefc8";
const commentarySelectionPath =
	"/artists/gozzoli-benozzo-r9fb82d431d2a5c/selections/rfae1de58855628";
const unavailableSelectionPath =
	"/artists/gozzoli-benozzo-r9fb82d431d2a5c/selections/r16c7b98bc13fb4";

test.describe("production records without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("preserves supplied filing and short artist names with an ordinary holding link", async ({
		page,
	}) => {
		await page.goto(artistPath);

		await expect(page.locator("h1")).toHaveText("GOZZOLI, Benozzo");
		await expect(
			page.locator("nav[aria-label='Breadcrumb'] a").last(),
		).toHaveText("Benozzo Gozzoli");
		await expect(
			page.getByRole("link", {
				name: /FIND MORE BY Benozzo Gozzoli IN THE ARTWORK SEARCH/,
			}),
		).toHaveAttribute("href", "/artworks?artist=GOZZOLI%2C+Benozzo");
	});

	test("renders only evidence-backed artwork file facts", async ({ page }) => {
		await page.goto(artworkPath);

		const file = page.locator("figure dl");
		await expect(file).toContainText("FILE");
		await expect(file).toContainText(/\d+ × \d+ px · JPEG ·/);
		await expect(file).not.toContainText(/SOURCE|LICENCE|LICENSE/);
		await expect(
			page.getByRole("link", { name: /DOWNLOAD THE FULL FILE/ }),
		).toHaveAttribute("href", /\/api\/files\/artworks\//);
	});

	test("keeps sourced and unavailable selection commentary honest", async ({
		page,
	}) => {
		await page.goto(commentarySelectionPath);
		await expect(
			page.getByRole("heading", { name: "COMMENTARY" }),
		).toBeVisible();
		await expect(page.getByText("CITE THIS RECORD — BIBTEX")).toBeVisible();
		await expect(
			page.getByRole("link", { name: /VIEW FULL HOLDING/ }),
		).toHaveAttribute("href", "/artworks?artist=GOZZOLI%2C+Benozzo");
		await expect(
			page.locator("ul.grid.grid-cols-2.md\\:grid-cols-4"),
		).toHaveCount(1);

		await page.goto(unavailableSelectionPath);
		await expect(
			page.getByText("Commentary is unavailable for this selection."),
		).toBeVisible();
	});
});
