import { expect, test } from "@playwright/test";
import {
	commentarySelection,
	curatedArtist,
	relationshipArtwork,
	uncommentedSelection,
} from "./helpers/synthetic-fixture";

const artistPath = curatedArtist.path;
const artworkPath = relationshipArtwork.path;
const commentarySelectionPath = commentarySelection.path;
const unavailableSelectionPath = uncommentedSelection.path;

test.describe("curated records without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("preserves supplied names and exposes curated record navigation", async ({
		page,
	}) => {
		await page.goto(artistPath);

		await expect(page.locator("h1")).toHaveText(curatedArtist.name);
		await expect(
			page.locator("nav[aria-label='Breadcrumb'] a").last(),
		).toHaveText(curatedArtist.breadcrumb);
		await expect(
			page.getByRole("navigation", { name: "On this page" }),
		).toBeVisible();
		await expect(page.getByRole("link", { name: /FIND MORE BY/ })).toHaveCount(
			0,
		);
	});

	test("renders only evidence-backed artwork file facts", async ({ page }) => {
		await page.goto(artworkPath);

		const file = page.locator("figure dl");
		await expect(file).toContainText("FILE");
		await expect(file).toContainText(/\d+ × \d+ px · JPEG/);
		await expect(file).not.toContainText(/\b\d+(?:\.\d+)? (?:kB|MB|GB)\b/);
		await expect(file).not.toContainText(/SOURCE|LICENCE|LICENSE/);
		const download = page.getByRole("link", { name: /DOWNLOAD THE FULL FILE/ });
		await expect(download).toHaveAttribute("href", /\/api\/files\/artworks\//);
		await expect(
			download.locator("xpath=following-sibling::p[1]"),
		).toContainText("CURRENT LOCATION ·");
	});

	test("keeps sourced and unavailable selection commentary honest", async ({
		page,
	}) => {
		await page.goto(commentarySelectionPath);
		await expect(
			page.getByRole("heading", { name: "COMMENTARY", exact: true }),
		).toBeVisible();
		await expect(page.getByText("CITE THIS RECORD — BIBTEX")).toBeVisible();
		await expect(
			page.getByRole("link", { name: /VIEW FULL HOLDING/ }),
		).toHaveAttribute("href", curatedArtist.holdingPath);
		await expect(
			page.locator("ul.grid.grid-cols-2.md\\:grid-cols-4"),
		).toHaveCount(1);

		await page.goto(unavailableSelectionPath);
		await expect(
			page.getByText("Commentary is unavailable for this selection."),
		).toBeVisible();
	});
});

test("artist record scope remains visible through live artwork refinement", async ({
	page,
}) => {
	const artistID = curatedArtist.id;
	await page.goto(`/artworks?artist_id=${artistID}`);

	const form = page.locator("#artwork-filters");
	await expect(page).toHaveURL(
		(url) =>
			url.pathname === "/artworks" &&
			url.searchParams.get("artist_id") === artistID,
	);
	await expect(form.locator("input[name='artist_id']")).toHaveValue(artistID);
	await expect(form.locator("[data-artwork-artist-scope]")).toContainText(
		curatedArtist.name,
	);

	const query = form.locator("#artwork-query");
	await expect(query).toBeEditable();
	const refinement = page.waitForResponse((response) => {
		const url = new URL(response.url());
		return (
			url.pathname === "/artworks" &&
			url.searchParams.get("artist_id") === artistID &&
			url.searchParams.get("q") === "02-01"
		);
	});
	await query.fill("02-01");
	await refinement;

	await expect(page).toHaveURL(
		(url) =>
			url.pathname === "/artworks" &&
			url.searchParams.get("artist_id") === artistID &&
			url.searchParams.get("q") === "02-01",
	);
	await expect(form.locator("input[name='artist_id']")).toHaveValue(artistID);
	await expect(form.locator("[data-artwork-artist-scope]")).toContainText(
		curatedArtist.name,
	);
	await expect(page.locator("#artwork-search-results")).toContainText(
		"Synthetic Artwork 02-01",
	);
});
