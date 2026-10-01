import { expect, test } from "@playwright/test";

test("groups artist and work matches", async ({ page }) => {
	await page.goto("/search?q=Synthetic");

	await expect(page.getByRole("heading", { name: "ARTISTS" })).toBeVisible();
	await expect(page.getByRole("heading", { name: "WORKS" })).toBeVisible();
	const artist = page
		.getByRole("link", { name: /^SYNTHETIC ARTIST 01/ })
		.first();
	await expect(artist).toBeVisible();
	await expect(
		page.getByRole("link", { name: /Synthetic Artwork 01-01/ }),
	).toBeVisible();
	await artist.click();
	await expect(page).toHaveURL(/\/artists\/synthetic-artist-01/);
});

test("shows an explicit empty state", async ({ page }) => {
	await page.goto("/search?q=No+Such+Record");

	await expect(page.getByText("No artist matches that.")).toBeVisible();
	await expect(page.getByText("No work matches that.")).toBeVisible();
});

test("updates grouped results while typing", async ({ page }) => {
	// Start from the empty state so the assertions below can only pass once the
	// debounced fragment has actually been swapped into the results region.
	await page.goto("/search?q=No+Such+Record");
	const results = page.locator("#global-search-results");
	await expect(results).toContainText("No artist matches that.");
	await expect(results).not.toContainText("SYNTHETIC ARTIST 01");

	await page.locator("#search").getByRole("searchbox").fill("Synthetic");

	await expect(results).toBeVisible();
	await expect(results).toContainText("SYNTHETIC ARTIST 01");
	await expect(results).not.toContainText("No artist matches that.");
});

test("debounced result swaps update the document title", async ({ page }) => {
	await page.goto("/search");
	await expect(page).toHaveTitle("Search - WGA");

	await page.locator("#search").getByRole("searchbox").fill("Synthetic");
	await expect(page).toHaveTitle("“Synthetic” · Search - WGA");
});
