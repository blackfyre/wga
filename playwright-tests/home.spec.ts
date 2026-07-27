import { expect, test } from "@playwright/test";

test("homepage search loads the artwork results page", async ({ page }) => {
  await page.goto("/");
  await page.locator("#homepage-search").fill("Synthetic Artwork 01-01");
  await page.getByRole("button", { name: "Search" }).click();

  await expect(page).toHaveURL(/\/artworks\/results\?title=Synthetic/);
  await expect(page.locator("h1")).toHaveText(/Explore Artworks/);
  await expect(page.locator("#search-result-container")).toContainText(
    "1 artwork found.",
  );
});
