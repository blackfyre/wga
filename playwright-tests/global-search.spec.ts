import { expect, test } from "@playwright/test";

test("groups artist and work matches", async ({ page }) => {
  await page.goto("/search?q=Synthetic");

  await expect(page.getByRole("heading", { name: "ARTISTS" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "WORKS" })).toBeVisible();
  const artist = page
    .getByRole("link", { name: /^Synthetic Artist 01/ })
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
  await page.goto("/search");

  const response = page.waitForResponse((candidate) => {
    const url = new URL(candidate.url());
    return url.pathname === "/search/results" && url.searchParams.get("q") === "Synthetic";
  });
  await page.locator("#search").getByRole("searchbox").fill("Synthetic");
  await response;
  await expect(page.locator("#global-search-results")).toContainText(
    "Synthetic Artist 01",
  );
});
