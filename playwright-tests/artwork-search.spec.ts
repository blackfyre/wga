import { type Page, expect, test } from "@playwright/test";

test.describe.configure({ mode: "serial" });
test.setTimeout(60000);

async function expectArtworkResults(page: Page) {
  await expect(
    page.locator("#search-result-container .explore-artwork-card").first(),
  ).toBeVisible({
    timeout: 30000,
  });
}

test("artwork search", async ({ page }) => {
  await page.goto("/artworks");
  await expect(page.locator("h1")).toHaveText(/Explore Artworks/);
  await page.locator("[name='title']").fill("Synthetic Artwork 01-01");
  await page.getByRole("button", { name: "Apply filters" }).click();
  await expectArtworkResults(page);
  await expect(
    page.locator("#artwork-search-results [data-viewer]"),
  ).toBeVisible();
  await expect(page.locator("#search-result-container")).toContainText(
    "1 artwork found.",
  );
});

test("explore results include a pageable artwork grid", async ({ page }) => {
  await page.goto("/artworks/results");

  await expectArtworkResults(page);
  await expect(
    page.locator("#artwork-search-results [data-viewer]"),
  ).toBeVisible();
  await expect(
    page.getByRole("navigation", { name: "Pagination" }),
  ).toBeVisible();
});

test("artwork thumbnails open ViewerJS without navigation", async ({ page }) => {
  await page.goto("/artworks/results?title=Synthetic+Artwork+01-01");
  await expectArtworkResults(page);

  await page.locator(".explore-artwork-card img").first().click();

  await expect(page).toHaveURL(/\/artworks\/results/);
  await expect(page.locator(".viewer-title")).toContainText(
    "Synthetic Artwork 01-01",
  );
});

test("artform search", async ({ page }) => {
  await page.goto("/artworks");
  await page.locator("[name='art_form']").selectOption("painting");
  await page.getByRole("button", { name: "Apply filters" }).click();
  await expectArtworkResults(page);
});

test("art type search", async ({ page }) => {
  await page.goto("/artworks");
  await page.locator("[name='art_type']").selectOption("synthetic-test-type");
  await page.getByRole("button", { name: "Apply filters" }).click();
  await expectArtworkResults(page);
});

test("art school search", async ({ page }) => {
  await page.goto("/artworks");
  await page
    .locator("[name='art_school']")
    .selectOption("american");
  await page.getByRole("button", { name: "Apply filters" }).click();
  await expectArtworkResults(page);
});

test("art type and school combined search", async ({ page }) => {
  await page.goto("/artworks");
  await page.locator("[name='art_type']").selectOption("synthetic-test-type");
  await page
    .locator("[name='art_school']")
    .selectOption("american");
  await page.getByRole("button", { name: "Apply filters" }).click();
  await expectArtworkResults(page);
});

test("title search", async ({ page }) => {
  await page.goto("/artworks");
  await page.locator("[name='title']").fill("Synthetic Artwork 01-01");
  await page.getByRole("button", { name: "Apply filters" }).click();
  await expectArtworkResults(page);
});

test("artist name search", async ({ page }) => {
  await page.goto("/artworks");
  await page.locator("[name='artist']").fill("Synthetic Artist 01");
  await page.getByRole("button", { name: "Apply filters" }).click();
  await expectArtworkResults(page);
});

test("clear resets the artwork search form", async ({ page }) => {
  await page.goto("/artworks");
  await page.locator("[name='title']").fill("Synthetic Artwork 01-01");
  await page
    .locator("[name='art_school']")
    .selectOption("american");
  await page.getByRole("link", { name: "Clear all" }).click();

  await expect(page).toHaveURL(/\/artworks$/);
  await expect(page.locator("[name='title']").first()).toHaveValue("");
  await expect(page.locator("[name='art_school']").first()).toHaveValue("");
  await expect(page.locator("#search-result-container")).toContainText(
    /use a keyword or refine by artist/i,
  );
});
